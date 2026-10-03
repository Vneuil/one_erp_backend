package application

import (
	"context"
	"fmt"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type InventoryUseCase interface {
	// Warehouses
	CreateWarehouse(ctx context.Context, dto CreateWarehouseDTO) (*WarehouseResponseDTO, error)
	ListWarehouses(ctx context.Context, query types.PaginationQuery) ([]WarehouseResponseDTO, types.PaginationMeta, error)
	UpdateWarehouse(ctx context.Context, id uuid.UUID, dto CreateWarehouseDTO) (*WarehouseResponseDTO, error)
	DeleteWarehouse(ctx context.Context, id uuid.UUID) error

	// Stock levels
	ListStockLevels(ctx context.Context, query types.PaginationQuery) ([]StockLevelResponseDTO, types.PaginationMeta, error)
	AdjustStock(ctx context.Context, dto AdjustStockDTO) (*StockLevelResponseDTO, error)
	ReceiveBatch(ctx context.Context, dto ReceiveBatchDTO) (*BatchResponseDTO, error)
	ListBatches(ctx context.Context, productID, warehouseID *uuid.UUID, includeEmpty bool) ([]BatchResponseDTO, error)
	ExpiringBatches(ctx context.Context, withinDays int) ([]BatchResponseDTO, error)
	WriteOffBatch(ctx context.Context, id uuid.UUID, reason string) (*BatchResponseDTO, error)
	SetBufferStock(ctx context.Context, dto SetBufferStockDTO) (*StockLevelResponseDTO, error)
	// GetAvailability returns the available-to-sell quantity for a
	// product at a warehouse (0 if no stock level row exists yet).
	GetAvailability(ctx context.Context, productID, warehouseID uuid.UUID) (int, error)

	// Movements
	ListMovements(ctx context.Context, query types.PaginationQuery) ([]MovementResponseDTO, types.PaginationMeta, error)
	MovementSummary(ctx context.Context, from, to string, warehouseID *uuid.UUID) (*MovementSummary, error)

	// Transfers
	CreateTransfer(ctx context.Context, dto CreateTransferDTO) (*TransferResponseDTO, error)
	ListTransfers(ctx context.Context, query types.PaginationQuery) ([]TransferResponseDTO, types.PaginationMeta, error)
	CompleteTransfer(ctx context.Context, id uuid.UUID) (*TransferResponseDTO, error)
	CancelTransfer(ctx context.Context, id uuid.UUID) (*TransferResponseDTO, error)

	// Stock opname
	CreateOpname(ctx context.Context, dto CreateOpnameDTO) (*OpnameResponseDTO, error)
	GetOpnameByID(ctx context.Context, id uuid.UUID) (*OpnameResponseDTO, error)
	ListOpnames(ctx context.Context, query types.PaginationQuery) ([]OpnameResponseDTO, types.PaginationMeta, error)
	CountOpnameLine(ctx context.Context, opnameID uuid.UUID, dto CountOpnameLineDTO) (*OpnameResponseDTO, error)
	FinalizeOpname(ctx context.Context, id uuid.UUID) (*OpnameResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type inventoryUseCase struct {
	repo        domain.InventoryRepository
	productRepo productDomain.ProductRepository
	ledger      financeApp.LedgerPoster // optional: posts write-offs of expired stock
}

// Option customises optional collaborators.
type Option func(*inventoryUseCase)

// WithLedger posts write-offs to the general ledger.
func WithLedger(l financeApp.LedgerPoster) Option {
	return func(uc *inventoryUseCase) { uc.ledger = l }
}

func NewInventoryUseCase(repo domain.InventoryRepository, productRepo productDomain.ProductRepository, opts ...Option) InventoryUseCase {
	uc := &inventoryUseCase{repo: repo, productRepo: productRepo}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

func (uc *inventoryUseCase) productInfo(ctx context.Context, id uuid.UUID) (sku, name string) {
	p, err := uc.productRepo.GetByID(ctx, id)
	if err != nil || p == nil {
		return "", ""
	}
	return p.SKU, p.Name
}

func (uc *inventoryUseCase) warehouseName(ctx context.Context, id uuid.UUID) string {
	w, err := uc.repo.GetWarehouseByID(ctx, id)
	if err != nil || w == nil {
		return ""
	}
	return w.Name
}

// Warehouses

func (uc *inventoryUseCase) CreateWarehouse(ctx context.Context, dto CreateWarehouseDTO) (*WarehouseResponseDTO, error) {
	if dto.Code == "" || dto.Name == "" {
		return nil, apperrors.NewBadRequest("Warehouse code and name are required")
	}
	isActive := true
	if dto.IsActive != nil {
		isActive = *dto.IsActive
	}
	w := &domain.Warehouse{
		Code:     dto.Code,
		Name:     dto.Name,
		Address:  dto.Address,
		IsActive: isActive,
	}
	if err := uc.repo.CreateWarehouse(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create warehouse")
	}
	return ToWarehouseResponse(w), nil
}

func (uc *inventoryUseCase) ListWarehouses(ctx context.Context, query types.PaginationQuery) ([]WarehouseResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListWarehouses(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list warehouses")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToWarehouseResponseList(items), meta, nil
}

func (uc *inventoryUseCase) UpdateWarehouse(ctx context.Context, id uuid.UUID, dto CreateWarehouseDTO) (*WarehouseResponseDTO, error) {
	w, err := uc.repo.GetWarehouseByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get warehouse")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Warehouse not found")
	}
	if dto.Name != "" {
		w.Name = dto.Name
	}
	if dto.Address != "" {
		w.Address = dto.Address
	}
	if dto.IsActive != nil {
		w.IsActive = *dto.IsActive
	}
	if err := uc.repo.UpdateWarehouse(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update warehouse")
	}
	return ToWarehouseResponse(w), nil
}

// DeleteWarehouse removes a warehouse, but only if it has no stock level
// records (even zero-quantity ones) referencing it, mirroring the "block
// delete if referenced" convention used elsewhere for FK dependents.
func (uc *inventoryUseCase) DeleteWarehouse(ctx context.Context, id uuid.UUID) error {
	w, err := uc.repo.GetWarehouseByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to get warehouse")
	}
	if w == nil {
		return apperrors.NewNotFound("Warehouse not found")
	}
	levels, err := uc.repo.ListStockLevelsByWarehouse(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to check warehouse stock levels")
	}
	if len(levels) > 0 {
		return apperrors.NewConflict("Cannot delete warehouse with existing stock level records")
	}
	if err := uc.repo.DeleteWarehouse(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete warehouse")
	}
	return nil
}

// Stock levels

func (uc *inventoryUseCase) ListStockLevels(ctx context.Context, query types.PaginationQuery) ([]StockLevelResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListStockLevels(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list stock levels")
	}
	result := make([]StockLevelResponseDTO, len(items))
	for i, s := range items {
		sku, name := uc.productInfo(ctx, s.ProductID)
		wName := uc.warehouseName(ctx, s.WarehouseID)
		result[i] = *ToStockLevelResponse(&s, sku, name, wName)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

// AdjustStock creates/updates a stock level and appends a matching movement.
// A positive quantity increases stock (type "in"), a negative one decreases it (type "out"),
// unless a reason is given, in which case it is recorded as an "adjustment".
func (uc *inventoryUseCase) AdjustStock(ctx context.Context, dto AdjustStockDTO) (*StockLevelResponseDTO, error) {
	if dto.ProductID == uuid.Nil || dto.WarehouseID == uuid.Nil || dto.Quantity == 0 {
		return nil, apperrors.NewBadRequest("productId, warehouseId and a non-zero quantity are required")
	}

	var result *StockLevelResponseDTO
	err := uc.repo.WithTransaction(ctx, func(txRepo domain.InventoryRepository) error {
		level, err := txRepo.GetStockLevel(ctx, dto.ProductID, dto.WarehouseID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get stock level")
		}
		if level == nil {
			level = &domain.StockLevel{ProductID: dto.ProductID, WarehouseID: dto.WarehouseID}
			// First adjustment ever recorded for this product+warehouse. If
			// no OTHER warehouse has claimed this product's legacy flat
			// Stock figure either, seed from it so this matches what
			// GetAvailability already told the caller was available (see
			// legacyFallbackStock/HasAnyStockLevel), instead of starting a
			// silent new count from zero and immediately going negative.
			hasAny, hasErr := txRepo.HasAnyStockLevel(ctx, dto.ProductID)
			if hasErr == nil && !hasAny {
				if seed, seedErr := uc.legacyFallbackStock(ctx, dto.ProductID); seedErr == nil {
					level.Quantity = seed
				}
			}
		}

		newQty := level.Quantity + dto.Quantity
		if newQty < 0 {
			return apperrors.NewBadRequest("Adjustment would result in negative stock")
		}
		// Lots first: a request the lots refuse must not have touched the stock level.
		batchNo, err := uc.applyBatches(ctx, txRepo, dto)
		if err != nil {
			return err
		}
		level.Quantity = newQty
		if err := txRepo.UpsertStockLevel(ctx, level); err != nil {
			return apperrors.NewInternal(err, "Failed to update stock level")
		}

		movementType := domain.MovementIn
		if dto.Quantity < 0 {
			movementType = domain.MovementOut
		}
		if dto.Reason != "" {
			movementType = domain.MovementAdjustment
		}
		movement := &domain.StockMovement{
			ProductID:   dto.ProductID,
			WarehouseID: dto.WarehouseID,
			Type:        movementType,
			Quantity:    dto.Quantity,
			Balance:     level.Quantity,
			Reference:   dto.Reference,
			Reason:      dto.Reason,
			CreatedBy:   dto.CreatedBy,
			BatchNo:     batchNo,
		}
		if err := txRepo.CreateMovement(ctx, movement); err != nil {
			return apperrors.NewInternal(err, "Failed to record stock movement")
		}

		sku, name := uc.productInfo(ctx, dto.ProductID)
		wName := uc.warehouseName(ctx, dto.WarehouseID)
		result = ToStockLevelResponse(level, sku, name, wName)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *inventoryUseCase) GetAvailability(ctx context.Context, productID, warehouseID uuid.UUID) (int, error) {
	level, err := uc.repo.GetStockLevel(ctx, productID, warehouseID)
	if err != nil {
		return 0, apperrors.NewInternal(err, "Failed to get stock level")
	}
	if level != nil {
		// Expired lots cannot be sold, so they do not count as available.
		return max(0, level.Available()-uc.expiredQuantity(ctx, productID, warehouseID)), nil
	}
	// No per-warehouse stock record exists yet for THIS warehouse. If the
	// product has never been counted into ANY warehouse either, treat its
	// legacy flat Product.Stock figure (set via the Product Catalog, before
	// per-warehouse tracking) as this warehouse's starting on-hand quantity
	// rather than silently reporting zero - otherwise every product looks
	// "out of stock" for Sales Order/POS creation the moment a warehouse is
	// introduced, even though the product master record says stock exists.
	// Once any warehouse has claimed it, later warehouses start at a real
	// zero instead - reusing the same legacy figure for a second warehouse
	// would double-count stock that only ever existed once.
	hasAny, err := uc.repo.HasAnyStockLevel(ctx, productID)
	if err != nil {
		return 0, apperrors.NewInternal(err, "Failed to check existing stock levels")
	}
	if hasAny {
		return 0, nil
	}
	return uc.legacyFallbackStock(ctx, productID)
}

// legacyFallbackStock returns a product's flat Stock field, used only when
// no inventory.StockLevel row exists yet for a given product+warehouse pair.
// Callers are responsible for confirming (via HasAnyStockLevel) that this
// product hasn't already claimed its legacy stock into a different
// warehouse, to avoid double-counting.
func (uc *inventoryUseCase) legacyFallbackStock(ctx context.Context, productID uuid.UUID) (int, error) {
	product, err := uc.productRepo.GetByID(ctx, productID)
	if err != nil {
		return 0, apperrors.NewInternal(err, "Failed to get product")
	}
	if product == nil {
		return 0, nil
	}
	return product.Stock, nil
}

func (uc *inventoryUseCase) SetBufferStock(ctx context.Context, dto SetBufferStockDTO) (*StockLevelResponseDTO, error) {
	if dto.ProductID == uuid.Nil || dto.WarehouseID == uuid.Nil {
		return nil, apperrors.NewBadRequest("productId and warehouseId are required")
	}
	if dto.MinStock < 0 || dto.MaxStock < 0 {
		return nil, apperrors.NewBadRequest("minStock and maxStock cannot be negative")
	}
	if dto.MaxStock > 0 && dto.MinStock > dto.MaxStock {
		return nil, apperrors.NewBadRequest("minStock cannot be greater than maxStock")
	}

	level, err := uc.repo.GetStockLevel(ctx, dto.ProductID, dto.WarehouseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get stock level")
	}
	if level == nil {
		level = &domain.StockLevel{ProductID: dto.ProductID, WarehouseID: dto.WarehouseID}
	}
	level.MinStock = dto.MinStock
	level.MaxStock = dto.MaxStock

	if err := uc.repo.UpsertStockLevel(ctx, level); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update buffer stock")
	}

	sku, name := uc.productInfo(ctx, dto.ProductID)
	wName := uc.warehouseName(ctx, dto.WarehouseID)
	return ToStockLevelResponse(level, sku, name, wName), nil
}

// Movements

func (uc *inventoryUseCase) ListMovements(ctx context.Context, query types.PaginationQuery) ([]MovementResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListMovements(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list stock movements")
	}
	result := make([]MovementResponseDTO, len(items))
	for i, m := range items {
		sku, name := uc.productInfo(ctx, m.ProductID)
		wName := uc.warehouseName(ctx, m.WarehouseID)
		result[i] = *ToMovementResponse(&m, sku, name, wName)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

// Transfers

func (uc *inventoryUseCase) CreateTransfer(ctx context.Context, dto CreateTransferDTO) (*TransferResponseDTO, error) {
	if dto.ProductID == uuid.Nil || dto.FromWarehouseID == uuid.Nil || dto.ToWarehouseID == uuid.Nil {
		return nil, apperrors.NewBadRequest("productId, fromWarehouseId and toWarehouseId are required")
	}
	if dto.FromWarehouseID == dto.ToWarehouseID {
		return nil, apperrors.NewBadRequest("Source and destination warehouses must be different")
	}
	if dto.Quantity <= 0 {
		return nil, apperrors.NewBadRequest("Transfer quantity must be positive")
	}

	level, err := uc.repo.GetStockLevel(ctx, dto.ProductID, dto.FromWarehouseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get stock level")
	}
	available := 0
	if level != nil {
		available = level.Available()
	}
	if dto.Quantity > available {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Cannot transfer %d units; only %d available at source warehouse", dto.Quantity, available))
	}

	t := &domain.StockTransfer{
		ProductID:       dto.ProductID,
		FromWarehouseID: dto.FromWarehouseID,
		ToWarehouseID:   dto.ToWarehouseID,
		Quantity:        dto.Quantity,
		Status:          domain.TransferPending,
		RequestedBy:     dto.RequestedBy,
		Notes:           dto.Notes,
	}
	if err := uc.repo.CreateTransfer(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create stock transfer")
	}
	return uc.toTransferResponse(ctx, t), nil
}

func (uc *inventoryUseCase) toTransferResponse(ctx context.Context, t *domain.StockTransfer) *TransferResponseDTO {
	sku, name := uc.productInfo(ctx, t.ProductID)
	fromName := uc.warehouseName(ctx, t.FromWarehouseID)
	toName := uc.warehouseName(ctx, t.ToWarehouseID)
	return ToTransferResponse(t, sku, name, fromName, toName)
}

func (uc *inventoryUseCase) ListTransfers(ctx context.Context, query types.PaginationQuery) ([]TransferResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListTransfers(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list stock transfers")
	}
	result := make([]TransferResponseDTO, len(items))
	for i, t := range items {
		result[i] = *uc.toTransferResponse(ctx, &t)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

// CompleteTransfer decrements stock at the source and increments it at the destination,
// each producing a stock movement record (transfer_out / transfer_in).
func (uc *inventoryUseCase) CompleteTransfer(ctx context.Context, id uuid.UUID) (*TransferResponseDTO, error) {
	var result *domain.StockTransfer
	err := uc.repo.WithTransaction(ctx, func(txRepo domain.InventoryRepository) error {
		t, err := txRepo.GetTransferByID(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get stock transfer")
		}
		if t == nil {
			return apperrors.NewNotFound("Stock transfer not found")
		}
		if t.Status == domain.TransferCompleted || t.Status == domain.TransferCancelled {
			return apperrors.NewConflict("Stock transfer is already " + t.Status)
		}

		fromLevel, err := txRepo.GetStockLevel(ctx, t.ProductID, t.FromWarehouseID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get source stock level")
		}
		available := 0
		if fromLevel != nil {
			available = fromLevel.Available()
		}
		if t.Quantity > available {
			return apperrors.NewBadRequest(fmt.Sprintf("Cannot complete transfer of %d units; only %d available at source warehouse", t.Quantity, available))
		}

		fromLevel.Quantity -= t.Quantity
		if err := txRepo.UpsertStockLevel(ctx, fromLevel); err != nil {
			return apperrors.NewInternal(err, "Failed to update source stock level")
		}
		outMovement := &domain.StockMovement{
			ProductID: t.ProductID, WarehouseID: t.FromWarehouseID,
			Type: domain.MovementTransferOut, Quantity: -t.Quantity, Balance: fromLevel.Quantity,
			Reference: t.ID.String(), Reason: "Stock transfer", CreatedBy: t.RequestedBy,
		}
		if err := txRepo.CreateMovement(ctx, outMovement); err != nil {
			return apperrors.NewInternal(err, "Failed to record transfer-out movement")
		}

		toLevel, err := txRepo.GetStockLevel(ctx, t.ProductID, t.ToWarehouseID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get destination stock level")
		}
		if toLevel == nil {
			toLevel = &domain.StockLevel{ProductID: t.ProductID, WarehouseID: t.ToWarehouseID}
		}
		toLevel.Quantity += t.Quantity
		if err := txRepo.UpsertStockLevel(ctx, toLevel); err != nil {
			return apperrors.NewInternal(err, "Failed to update destination stock level")
		}
		inMovement := &domain.StockMovement{
			ProductID: t.ProductID, WarehouseID: t.ToWarehouseID,
			Type: domain.MovementTransferIn, Quantity: t.Quantity, Balance: toLevel.Quantity,
			Reference: t.ID.String(), Reason: "Stock transfer", CreatedBy: t.RequestedBy,
		}
		if err := txRepo.CreateMovement(ctx, inMovement); err != nil {
			return apperrors.NewInternal(err, "Failed to record transfer-in movement")
		}

		t.Status = domain.TransferCompleted
		if err := txRepo.UpdateTransfer(ctx, t); err != nil {
			return apperrors.NewInternal(err, "Failed to update stock transfer status")
		}
		result = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uc.toTransferResponse(ctx, result), nil
}

func (uc *inventoryUseCase) CancelTransfer(ctx context.Context, id uuid.UUID) (*TransferResponseDTO, error) {
	t, err := uc.repo.GetTransferByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get stock transfer")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Stock transfer not found")
	}
	if t.Status == domain.TransferCompleted || t.Status == domain.TransferCancelled {
		return nil, apperrors.NewConflict("Stock transfer is already " + t.Status)
	}
	t.Status = domain.TransferCancelled
	if err := uc.repo.UpdateTransfer(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to cancel stock transfer")
	}
	return uc.toTransferResponse(ctx, t), nil
}

// Stock opname

func (uc *inventoryUseCase) CreateOpname(ctx context.Context, dto CreateOpnameDTO) (*OpnameResponseDTO, error) {
	if dto.WarehouseID == uuid.Nil {
		return nil, apperrors.NewBadRequest("warehouseId is required")
	}
	warehouse, err := uc.repo.GetWarehouseByID(ctx, dto.WarehouseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get warehouse")
	}
	if warehouse == nil {
		return nil, apperrors.NewNotFound("Warehouse not found")
	}

	auditDate := dto.AuditDate
	if auditDate == "" {
		auditDate = time.Now().Format("2006-01-02")
	}

	productIDs := dto.ProductIDs
	if len(productIDs) == 0 {
		levels, err := uc.repo.ListStockLevelsByWarehouse(ctx, dto.WarehouseID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to list stock levels for warehouse")
		}
		for _, l := range levels {
			productIDs = append(productIDs, l.ProductID)
		}
	}

	lines := make([]domain.StockOpnameLine, 0, len(productIDs))
	for _, pid := range productIDs {
		level, err := uc.repo.GetStockLevel(ctx, pid, dto.WarehouseID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get stock level")
		}
		systemQty := 0
		if level != nil {
			systemQty = level.Quantity
		}
		lines = append(lines, domain.StockOpnameLine{
			ProductID: pid,
			SystemQty: systemQty,
		})
	}

	o := &domain.StockOpname{
		WarehouseID: dto.WarehouseID,
		AuditDate:   auditDate,
		Status:      domain.OpnameDraft,
		Lines:       lines,
	}
	if err := uc.repo.CreateOpname(ctx, o); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create stock opname")
	}
	return uc.toOpnameResponse(ctx, o), nil
}

func (uc *inventoryUseCase) toOpnameResponse(ctx context.Context, o *domain.StockOpname) *OpnameResponseDTO {
	lines := make([]OpnameLineResponseDTO, len(o.Lines))
	for i, l := range o.Lines {
		sku, name := uc.productInfo(ctx, l.ProductID)
		lines[i] = OpnameLineResponseDTO{
			ID: l.ID, ProductID: l.ProductID, ProductSKU: sku, ProductName: name,
			SystemQty: l.SystemQty, CountedQty: l.CountedQty, Variance: l.Variance(), Counted: l.Counted,
		}
	}
	return &OpnameResponseDTO{
		ID: o.ID, WarehouseID: o.WarehouseID, WarehouseName: uc.warehouseName(ctx, o.WarehouseID),
		AuditDate: o.AuditDate, Status: o.Status, Lines: lines, CreatedAt: o.CreatedAt,
	}
}

func (uc *inventoryUseCase) GetOpnameByID(ctx context.Context, id uuid.UUID) (*OpnameResponseDTO, error) {
	o, err := uc.repo.GetOpnameByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get stock opname")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Stock opname not found")
	}
	return uc.toOpnameResponse(ctx, o), nil
}

func (uc *inventoryUseCase) ListOpnames(ctx context.Context, query types.PaginationQuery) ([]OpnameResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListOpnames(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list stock opnames")
	}
	result := make([]OpnameResponseDTO, len(items))
	for i, o := range items {
		result[i] = *uc.toOpnameResponse(ctx, &o)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

func (uc *inventoryUseCase) CountOpnameLine(ctx context.Context, opnameID uuid.UUID, dto CountOpnameLineDTO) (*OpnameResponseDTO, error) {
	o, err := uc.repo.GetOpnameByID(ctx, opnameID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get stock opname")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Stock opname not found")
	}
	if o.Status == domain.OpnameCompleted {
		return nil, apperrors.NewConflict("Cannot record counts on a completed opname")
	}

	found := false
	for i := range o.Lines {
		if o.Lines[i].ProductID == dto.ProductID {
			o.Lines[i].CountedQty = dto.CountedQty
			o.Lines[i].Counted = true
			if err := uc.repo.UpdateOpnameLine(ctx, &o.Lines[i]); err != nil {
				return nil, apperrors.NewInternal(err, "Failed to update opname line")
			}
			found = true
			break
		}
	}
	if !found {
		return nil, apperrors.NewNotFound("Product is not part of this opname")
	}

	if o.Status == domain.OpnameDraft {
		o.Status = domain.OpnameInProgress
		if err := uc.repo.UpdateOpname(ctx, o); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update stock opname status")
		}
	}
	return uc.toOpnameResponse(ctx, o), nil
}

// FinalizeOpname reconciles variances by writing adjustment stock movements and
// updating stock levels; each line is reconciled atomically.
func (uc *inventoryUseCase) FinalizeOpname(ctx context.Context, id uuid.UUID) (*OpnameResponseDTO, error) {
	var result *domain.StockOpname
	err := uc.repo.WithTransaction(ctx, func(txRepo domain.InventoryRepository) error {
		o, err := txRepo.GetOpnameByID(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get stock opname")
		}
		if o == nil {
			return apperrors.NewNotFound("Stock opname not found")
		}
		if o.Status == domain.OpnameCompleted {
			return apperrors.NewConflict("Stock opname is already completed")
		}

		for _, l := range o.Lines {
			if !l.Counted || l.Variance() == 0 {
				continue
			}
			level, err := txRepo.GetStockLevel(ctx, l.ProductID, o.WarehouseID)
			if err != nil {
				return apperrors.NewInternal(err, "Failed to get stock level")
			}
			if level == nil {
				level = &domain.StockLevel{ProductID: l.ProductID, WarehouseID: o.WarehouseID}
			}
			level.Quantity = l.CountedQty
			if level.Quantity < 0 {
				return apperrors.NewBadRequest("Counted quantity cannot be negative")
			}
			if err := txRepo.UpsertStockLevel(ctx, level); err != nil {
				return apperrors.NewInternal(err, "Failed to update stock level")
			}
			movement := &domain.StockMovement{
				ProductID: l.ProductID, WarehouseID: o.WarehouseID,
				Type: domain.MovementAdjustment, Quantity: l.Variance(), Balance: level.Quantity,
				Reference: o.ID.String(), Reason: "Stock opname reconciliation",
			}
			if err := txRepo.CreateMovement(ctx, movement); err != nil {
				return apperrors.NewInternal(err, "Failed to record adjustment movement")
			}
		}

		o.Status = domain.OpnameCompleted
		if err := txRepo.UpdateOpname(ctx, o); err != nil {
			return apperrors.NewInternal(err, "Failed to update stock opname status")
		}
		result = o
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uc.toOpnameResponse(ctx, result), nil
}

// SeedInitialData populates default warehouses on first boot
func (uc *inventoryUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountWarehouses(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	warehouses := []CreateWarehouseDTO{
		{Code: "WH-CKR", Name: "Gudang Utama Cikarang", Address: "Cikarang, Bekasi, Jawa Barat"},
		{Code: "WH-SBY", Name: "Hub Surabaya", Address: "Surabaya, Jawa Timur"},
	}
	for _, w := range warehouses {
		_, _ = uc.CreateWarehouse(ctx, w)
	}
	return nil
}
