package application

import (
	"context"
	"fmt"
	"time"

	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type ManufacturingUseCase interface {
	// BOM
	CreateBOM(ctx context.Context, dto CreateBOMDTO) (*BOMResponseDTO, error)
	GetBOMByID(ctx context.Context, id uuid.UUID) (*BOMResponseDTO, error)
	ListBOMs(ctx context.Context, query types.PaginationQuery) ([]BOMResponseDTO, types.PaginationMeta, error)

	// Production orders
	CreateOrder(ctx context.Context, dto CreateProductionOrderDTO) (*ProductionOrderResponseDTO, error)
	GetOrderByID(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error)
	ListOrders(ctx context.Context, query types.PaginationQuery) ([]ProductionOrderResponseDTO, types.PaginationMeta, error)
	ReleaseOrder(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error)
	PauseOrder(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error)
	ResumeOrder(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error)
	CancelOrder(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error)
	CompleteBatch(ctx context.Context, orderID uuid.UUID, dto CompleteBatchDTO) (*ProductionOrderResponseDTO, error)

	GetDashboardSummary(ctx context.Context) (*DashboardSummaryDTO, error)

	SeedInitialData(ctx context.Context) error
}

type manufacturingUseCase struct {
	repo          domain.ManufacturingRepository
	inventoryRepo inventoryDomain.InventoryRepository
	productRepo   productDomain.ProductRepository
}

func NewManufacturingUseCase(repo domain.ManufacturingRepository, inventoryRepo inventoryDomain.InventoryRepository, productRepo productDomain.ProductRepository) ManufacturingUseCase {
	return &manufacturingUseCase{repo: repo, inventoryRepo: inventoryRepo, productRepo: productRepo}
}

func (uc *manufacturingUseCase) productInfo(ctx context.Context, id uuid.UUID) (sku, name string) {
	p, err := uc.productRepo.GetByID(ctx, id)
	if err != nil || p == nil {
		return "", ""
	}
	return p.SKU, p.Name
}

func (uc *manufacturingUseCase) warehouseName(ctx context.Context, id uuid.UUID) string {
	w, err := uc.inventoryRepo.GetWarehouseByID(ctx, id)
	if err != nil || w == nil {
		return ""
	}
	return w.Name
}

// BOM

func (uc *manufacturingUseCase) toBOMResponse(ctx context.Context, b *domain.BillOfMaterial) *BOMResponseDTO {
	sku, name := uc.productInfo(ctx, b.ProductID)
	lines := make([]BOMLineResponseDTO, len(b.Lines))
	for i, l := range b.Lines {
		csku, cname := uc.productInfo(ctx, l.ComponentProductID)
		lines[i] = ToBOMLineResponse(&l, csku, cname)
	}
	return &BOMResponseDTO{
		ID: b.ID, ProductID: b.ProductID, ProductSKU: sku, ProductName: name,
		Name: b.Name, Version: b.Version, IsActive: b.IsActive, Lines: lines, CreatedAt: b.CreatedAt,
	}
}

func (uc *manufacturingUseCase) CreateBOM(ctx context.Context, dto CreateBOMDTO) (*BOMResponseDTO, error) {
	if dto.ProductID == uuid.Nil || dto.Name == "" {
		return nil, apperrors.NewBadRequest("productId and name are required")
	}
	if len(dto.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one BOM line is required")
	}
	lines := make([]domain.BOMLine, 0, len(dto.Lines))
	for _, l := range dto.Lines {
		if l.ComponentProductID == uuid.Nil || l.QuantityRequired <= 0 {
			return nil, apperrors.NewBadRequest("Each BOM line requires a componentProductId and a positive quantityRequired")
		}
		lines = append(lines, domain.BOMLine{
			ComponentProductID: l.ComponentProductID,
			QuantityRequired:   l.QuantityRequired,
			Unit:               l.Unit,
		})
	}
	version := dto.Version
	if version == "" {
		version = "v1.0"
	}
	isActive := true
	if dto.IsActive != nil {
		isActive = *dto.IsActive
	}
	b := &domain.BillOfMaterial{
		ProductID: dto.ProductID,
		Name:      dto.Name,
		Version:   version,
		IsActive:  isActive,
		Lines:     lines,
	}
	if err := uc.repo.CreateBOM(ctx, b); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create BOM")
	}
	return uc.toBOMResponse(ctx, b), nil
}

func (uc *manufacturingUseCase) GetBOMByID(ctx context.Context, id uuid.UUID) (*BOMResponseDTO, error) {
	b, err := uc.repo.GetBOMByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get BOM")
	}
	if b == nil {
		return nil, apperrors.NewNotFound("BOM not found")
	}
	return uc.toBOMResponse(ctx, b), nil
}

func (uc *manufacturingUseCase) ListBOMs(ctx context.Context, query types.PaginationQuery) ([]BOMResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListBOMs(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list BOMs")
	}
	result := make([]BOMResponseDTO, len(items))
	for i, b := range items {
		result[i] = *uc.toBOMResponse(ctx, &b)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

// Production orders

func (uc *manufacturingUseCase) toOrderResponse(ctx context.Context, o *domain.ProductionOrder) *ProductionOrderResponseDTO {
	bom, err := uc.repo.GetBOMByID(ctx, o.BOMID)
	bomName := ""
	var productID uuid.UUID
	if err == nil && bom != nil {
		bomName = bom.Name
		productID = bom.ProductID
	}
	sku, name := uc.productInfo(ctx, productID)
	wName := uc.warehouseName(ctx, o.WarehouseID)
	return ToProductionOrderResponse(o, bomName, productID, sku, name, wName)
}

func (uc *manufacturingUseCase) CreateOrder(ctx context.Context, dto CreateProductionOrderDTO) (*ProductionOrderResponseDTO, error) {
	if dto.BOMID == uuid.Nil || dto.WarehouseID == uuid.Nil || dto.QuantityToProduce <= 0 {
		return nil, apperrors.NewBadRequest("bomId, warehouseId and a positive quantityToProduce are required")
	}
	bom, err := uc.repo.GetBOMByID(ctx, dto.BOMID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get BOM")
	}
	if bom == nil {
		return nil, apperrors.NewNotFound("BOM not found")
	}
	warehouse, err := uc.inventoryRepo.GetWarehouseByID(ctx, dto.WarehouseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get warehouse")
	}
	if warehouse == nil {
		return nil, apperrors.NewNotFound("Warehouse not found")
	}

	o := &domain.ProductionOrder{
		BOMID:             dto.BOMID,
		WarehouseID:       dto.WarehouseID,
		QuantityToProduce: dto.QuantityToProduce,
		Status:            domain.OrderPlanned,
		PlannedDate:       dto.PlannedDate,
	}
	if err := uc.repo.CreateOrder(ctx, o); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create production order")
	}
	return uc.toOrderResponse(ctx, o), nil
}

func (uc *manufacturingUseCase) GetOrderByID(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error) {
	o, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get production order")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Production order not found")
	}
	return uc.toOrderResponse(ctx, o), nil
}

func (uc *manufacturingUseCase) ListOrders(ctx context.Context, query types.PaginationQuery) ([]ProductionOrderResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListOrders(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list production orders")
	}
	result := make([]ProductionOrderResponseDTO, len(items))
	for i, o := range items {
		result[i] = *uc.toOrderResponse(ctx, &o)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

func (uc *manufacturingUseCase) ReleaseOrder(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error) {
	o, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get production order")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Production order not found")
	}
	if o.Status != domain.OrderPlanned {
		return nil, apperrors.NewConflict("Only planned production orders can be released")
	}
	o.Status = domain.OrderReleased
	if err := uc.repo.UpdateOrder(ctx, o); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to release production order")
	}
	return uc.toOrderResponse(ctx, o), nil
}

func (uc *manufacturingUseCase) PauseOrder(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error) {
	o, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get production order")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Production order not found")
	}
	if o.Status != domain.OrderReleased && o.Status != domain.OrderInProgress {
		return nil, apperrors.NewConflict("Only released or in-progress production orders can be paused")
	}
	o.Status = domain.OrderPaused
	if err := uc.repo.UpdateOrder(ctx, o); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to pause production order")
	}
	return uc.toOrderResponse(ctx, o), nil
}

func (uc *manufacturingUseCase) ResumeOrder(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error) {
	o, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get production order")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Production order not found")
	}
	if o.Status != domain.OrderPaused {
		return nil, apperrors.NewConflict("Only paused production orders can be resumed")
	}
	o.Status = domain.OrderInProgress
	if err := uc.repo.UpdateOrder(ctx, o); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to resume production order")
	}
	return uc.toOrderResponse(ctx, o), nil
}

// CancelOrder cancels a production order that has not yet completed. Since
// inventory is only ever touched at successful batch completion (see
// CompleteBatch), cancelling here makes no inventory changes — materials are
// only consumed once a batch has actually been produced.
func (uc *manufacturingUseCase) CancelOrder(ctx context.Context, id uuid.UUID) (*ProductionOrderResponseDTO, error) {
	o, err := uc.repo.GetOrderByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get production order")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Production order not found")
	}
	if o.Status == domain.OrderCompleted {
		return nil, apperrors.NewConflict("Completed production orders cannot be cancelled")
	}
	if o.Status == domain.OrderCancelled {
		return nil, apperrors.NewConflict("Production order is already cancelled")
	}
	o.Status = domain.OrderCancelled
	if err := uc.repo.UpdateOrder(ctx, o); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to cancel production order")
	}
	return uc.toOrderResponse(ctx, o), nil
}

// CompleteBatch consumes raw materials per the BOM and produces finished goods,
// writing stock movements through the inventory module's repository so the
// movement ledger stays consistent with the rest of the system.
func (uc *manufacturingUseCase) CompleteBatch(ctx context.Context, orderID uuid.UUID, dto CompleteBatchDTO) (*ProductionOrderResponseDTO, error) {
	if dto.QuantityCompleted <= 0 {
		return nil, apperrors.NewBadRequest("quantityCompleted must be positive")
	}

	o, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get production order")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Production order not found")
	}
	if o.Status != domain.OrderReleased && o.Status != domain.OrderInProgress {
		return nil, apperrors.NewConflict("Only released or in-progress production orders can log batch completions")
	}
	if dto.QuantityCompleted > o.RemainingQuantity() {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Cannot complete %d units; only %d remaining on this production order", dto.QuantityCompleted, o.RemainingQuantity()))
	}

	bom, err := uc.repo.GetBOMByID(ctx, o.BOMID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get BOM")
	}
	if bom == nil {
		return nil, apperrors.NewNotFound("BOM not found")
	}

	completionDate := dto.CompletionDate
	if completionDate == "" {
		completionDate = time.Now().Format("2006-01-02")
	}

	// Consume raw materials, produce finished goods, record the production
	// batch, and update the order's completed-quantity/status all inside a
	// SINGLE database transaction, so a failure partway through can never
	// leave inventory adjusted without the order reflecting completion (or
	// vice versa).
	var result *domain.ProductionOrder
	err = uc.repo.WithTransactionAndInventory(ctx, func(txRepo domain.ManufacturingRepository, txInv inventoryDomain.InventoryRepository) error {
		// Validate sufficient raw material stock before mutating anything.
		for _, line := range bom.Lines {
			required := int(line.QuantityRequired * float64(dto.QuantityCompleted))
			level, err := txInv.GetStockLevel(ctx, line.ComponentProductID, o.WarehouseID)
			if err != nil {
				return apperrors.NewInternal(err, "Failed to get component stock level")
			}
			available := 0
			if level != nil {
				available = level.Available()
			}
			if required > available {
				sku, name := uc.productInfo(ctx, line.ComponentProductID)
				return apperrors.NewBadRequest(fmt.Sprintf("Insufficient stock of %s (%s): need %d, only %d available", name, sku, required, available))
			}
		}

		// Consume raw materials.
		for _, line := range bom.Lines {
			required := int(line.QuantityRequired * float64(dto.QuantityCompleted))
			if required <= 0 {
				continue
			}
			level, err := txInv.GetStockLevel(ctx, line.ComponentProductID, o.WarehouseID)
			if err != nil {
				return apperrors.NewInternal(err, "Failed to get component stock level")
			}
			if level == nil {
				level = &inventoryDomain.StockLevel{ProductID: line.ComponentProductID, WarehouseID: o.WarehouseID}
			}
			level.Quantity -= required
			if level.Quantity < 0 {
				return apperrors.NewBadRequest("Adjustment would result in negative stock")
			}
			if err := txInv.UpsertStockLevel(ctx, level); err != nil {
				return apperrors.NewInternal(err, "Failed to update component stock level")
			}
			movement := &inventoryDomain.StockMovement{
				ProductID: line.ComponentProductID, WarehouseID: o.WarehouseID,
				Type: inventoryDomain.MovementOut, Quantity: -required, Balance: level.Quantity,
				Reference: o.ID.String(), Reason: "Manufacturing consumption",
			}
			if err := txInv.CreateMovement(ctx, movement); err != nil {
				return apperrors.NewInternal(err, "Failed to record consumption movement")
			}
		}

		// Produce finished goods.
		outLevel, err := txInv.GetStockLevel(ctx, bom.ProductID, o.WarehouseID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get finished goods stock level")
		}
		if outLevel == nil {
			outLevel = &inventoryDomain.StockLevel{ProductID: bom.ProductID, WarehouseID: o.WarehouseID}
		}
		outLevel.Quantity += dto.QuantityCompleted
		if err := txInv.UpsertStockLevel(ctx, outLevel); err != nil {
			return apperrors.NewInternal(err, "Failed to update finished goods stock level")
		}
		outMovement := &inventoryDomain.StockMovement{
			ProductID: bom.ProductID, WarehouseID: o.WarehouseID,
			Type: inventoryDomain.MovementIn, Quantity: dto.QuantityCompleted, Balance: outLevel.Quantity,
			Reference: o.ID.String(), Reason: "Manufacturing production",
		}
		if err := txInv.CreateMovement(ctx, outMovement); err != nil {
			return apperrors.NewInternal(err, "Failed to record production movement")
		}

		// Record the production batch and update the order's status, in the
		// same transaction as the inventory mutations above.
		batch := &domain.ProductionBatch{
			ProductionOrderID: o.ID,
			QuantityCompleted: dto.QuantityCompleted,
			CompletionDate:    completionDate,
			Notes:             dto.Notes,
		}
		if err := txRepo.CreateBatch(ctx, batch); err != nil {
			return apperrors.NewInternal(err, "Failed to record production batch")
		}

		o.QuantityCompleted += dto.QuantityCompleted
		if o.QuantityCompleted >= o.QuantityToProduce {
			o.Status = domain.OrderCompleted
			o.ActualCompletionDate = completionDate
		} else {
			o.Status = domain.OrderInProgress
		}
		if err := txRepo.UpdateOrder(ctx, o); err != nil {
			return apperrors.NewInternal(err, "Failed to update production order")
		}
		result = o
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uc.toOrderResponse(ctx, result), nil
}

func (uc *manufacturingUseCase) GetDashboardSummary(ctx context.Context) (*DashboardSummaryDTO, error) {
	activeBOMs, err := uc.repo.CountActiveBOMs(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to count active BOMs")
	}

	ordersByStatus, err := uc.repo.CountOrdersByStatus(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to count orders by status")
	}

	monthStart := time.Now().Format("2006-01") + "-01"
	unitsCompleted, err := uc.repo.SumQuantityCompletedSince(ctx, monthStart)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to sum completed quantity")
	}

	upcoming, err := uc.repo.ListUpcomingOrders(ctx, 5)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list upcoming orders")
	}
	upcomingDTOs := make([]ProductionOrderResponseDTO, len(upcoming))
	for i := range upcoming {
		upcomingDTOs[i] = *uc.toOrderResponse(ctx, &upcoming[i])
	}

	return &DashboardSummaryDTO{
		ActiveBOMs:              activeBOMs,
		OrdersByStatus:          ordersByStatus,
		UnitsCompletedThisMonth: unitsCompleted,
		UpcomingOrders:          upcomingDTOs,
	}, nil
}

// SeedInitialData creates a sample BOM referencing the first available product, if any.
func (uc *manufacturingUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountBOMs(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	products, _, err := uc.productRepo.List(ctx, types.PaginationQuery{Page: 1, PerPage: 2})
	if err != nil || len(products) < 2 {
		return nil
	}

	b := &domain.BillOfMaterial{
		ProductID: products[0].ID,
		Name:      products[0].Name + " Recipe",
		Version:   "v1.0",
		IsActive:  true,
		Lines: []domain.BOMLine{
			{
				ComponentProductID: products[1].ID,
				QuantityRequired:   1,
				Unit:               products[1].Unit,
			},
		},
	}
	return uc.repo.CreateBOM(ctx, b)
}
