package application

import (
	"context"
	"fmt"
	"time"

	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/divinecoid/one-backend/internal/modules/warehouse/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type WarehouseUseCase interface {
	// Pick waves
	CreatePickWave(ctx context.Context, dto CreatePickWaveDTO) (*PickWaveResponseDTO, error)
	GetPickWaveByID(ctx context.Context, id uuid.UUID) (*PickWaveResponseDTO, error)
	ListPickWaves(ctx context.Context, query types.PaginationQuery) ([]PickWaveResponseDTO, types.PaginationMeta, error)
	RecordPick(ctx context.Context, id uuid.UUID, dto RecordPickDTO) (*PickWaveResponseDTO, error)
	CompletePickWave(ctx context.Context, id uuid.UUID) (*PickWaveResponseDTO, error)
	CancelPickWave(ctx context.Context, id uuid.UUID) (*PickWaveResponseDTO, error)

	// Packing sessions
	CreatePackingSession(ctx context.Context, dto CreatePackingSessionDTO) (*PackingSessionResponseDTO, error)
	ListPackingSessions(ctx context.Context, query types.PaginationQuery) ([]PackingSessionResponseDTO, types.PaginationMeta, error)
	CompletePackingSession(ctx context.Context, id uuid.UUID, dto CompletePackingSessionDTO) (*PackingSessionResponseDTO, error)

	// Shipments
	CreateShipment(ctx context.Context, dto CreateShipmentDTO) (*ShipmentResponseDTO, error)
	ListShipments(ctx context.Context, query types.PaginationQuery) ([]ShipmentResponseDTO, types.PaginationMeta, error)
	DispatchShipment(ctx context.Context, id uuid.UUID) (*ShipmentResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type warehouseUseCase struct {
	repo          domain.WarehouseRepository
	inventoryRepo inventoryDomain.InventoryRepository
	productRepo   productDomain.ProductRepository
}

func NewWarehouseUseCase(repo domain.WarehouseRepository, inventoryRepo inventoryDomain.InventoryRepository, productRepo productDomain.ProductRepository) WarehouseUseCase {
	return &warehouseUseCase{repo: repo, inventoryRepo: inventoryRepo, productRepo: productRepo}
}

func (uc *warehouseUseCase) productInfo(ctx context.Context, id uuid.UUID) (sku, name string) {
	p, err := uc.productRepo.GetByID(ctx, id)
	if err != nil || p == nil {
		return "", ""
	}
	return p.SKU, p.Name
}

func (uc *warehouseUseCase) warehouseName(ctx context.Context, id uuid.UUID) string {
	w, err := uc.inventoryRepo.GetWarehouseByID(ctx, id)
	if err != nil || w == nil {
		return ""
	}
	return w.Name
}

func (uc *warehouseUseCase) toPickWaveResponse(ctx context.Context, w *domain.PickWave) *PickWaveResponseDTO {
	lines := make([]PickWaveLineResponseDTO, len(w.Lines))
	for i, l := range w.Lines {
		sku, name := uc.productInfo(ctx, l.ProductID)
		lines[i] = PickWaveLineResponseDTO{
			ID: l.ID, ProductID: l.ProductID, ProductSKU: sku, ProductName: name,
			QuantityToPick: l.QuantityToPick, QuantityPicked: l.QuantityPicked,
		}
	}
	return &PickWaveResponseDTO{
		ID: w.ID, WarehouseID: w.WarehouseID, WarehouseName: uc.warehouseName(ctx, w.WarehouseID),
		SalesOrderID: w.SalesOrderID, Status: w.Status, AssignedTo: w.AssignedTo,
		Lines: lines, CreatedAt: w.CreatedAt,
	}
}

// Pick waves

// CreatePickWave creates a pick wave with the given lines, or, if no lines are
// given, auto-populates lines from current available stock at the warehouse.
func (uc *warehouseUseCase) CreatePickWave(ctx context.Context, dto CreatePickWaveDTO) (*PickWaveResponseDTO, error) {
	if dto.WarehouseID == uuid.Nil {
		return nil, apperrors.NewBadRequest("warehouseId is required")
	}
	wh, err := uc.inventoryRepo.GetWarehouseByID(ctx, dto.WarehouseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get warehouse")
	}
	if wh == nil {
		return nil, apperrors.NewNotFound("Warehouse not found")
	}

	lines := make([]domain.PickWaveLine, 0, len(dto.Lines))
	if len(dto.Lines) > 0 {
		for _, l := range dto.Lines {
			if l.ProductID == uuid.Nil || l.QuantityToPick <= 0 {
				return nil, apperrors.NewBadRequest("Each pick line requires a productId and a positive quantityToPick")
			}
			lines = append(lines, domain.PickWaveLine{ProductID: l.ProductID, QuantityToPick: l.QuantityToPick})
		}
	} else {
		levels, err := uc.inventoryRepo.ListStockLevelsByWarehouse(ctx, dto.WarehouseID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to list stock levels for warehouse")
		}
		for _, lvl := range levels {
			if lvl.Available() <= 0 {
				continue
			}
			lines = append(lines, domain.PickWaveLine{ProductID: lvl.ProductID, QuantityToPick: lvl.Available()})
		}
	}

	w := &domain.PickWave{
		WarehouseID:  dto.WarehouseID,
		SalesOrderID: dto.SalesOrderID,
		Status:       domain.PickWavePending,
		AssignedTo:   dto.AssignedTo,
		Lines:        lines,
	}
	if err := uc.repo.CreatePickWave(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create pick wave")
	}
	return uc.toPickWaveResponse(ctx, w), nil
}

func (uc *warehouseUseCase) GetPickWaveByID(ctx context.Context, id uuid.UUID) (*PickWaveResponseDTO, error) {
	w, err := uc.repo.GetPickWaveByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get pick wave")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Pick wave not found")
	}
	return uc.toPickWaveResponse(ctx, w), nil
}

func (uc *warehouseUseCase) ListPickWaves(ctx context.Context, query types.PaginationQuery) ([]PickWaveResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListPickWaves(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list pick waves")
	}
	result := make([]PickWaveResponseDTO, len(items))
	for i, w := range items {
		result[i] = *uc.toPickWaveResponse(ctx, &w)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

// RecordPick records picked quantities against a pick wave's lines and moves it in_progress.
func (uc *warehouseUseCase) RecordPick(ctx context.Context, id uuid.UUID, dto RecordPickDTO) (*PickWaveResponseDTO, error) {
	w, err := uc.repo.GetPickWaveByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get pick wave")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Pick wave not found")
	}
	if w.Status == domain.PickWaveCompleted || w.Status == domain.PickWaveCancelled {
		return nil, apperrors.NewConflict("Pick wave is already " + w.Status)
	}

	for _, pl := range dto.Lines {
		for i := range w.Lines {
			if w.Lines[i].ProductID == pl.ProductID {
				if pl.QuantityPicked > w.Lines[i].QuantityToPick {
					return nil, apperrors.NewBadRequest(fmt.Sprintf("Cannot pick %d units; only %d requested for this line", pl.QuantityPicked, w.Lines[i].QuantityToPick))
				}
				level, err := uc.inventoryRepo.GetStockLevel(ctx, pl.ProductID, w.WarehouseID)
				if err != nil {
					return nil, apperrors.NewInternal(err, "Failed to get stock level")
				}
				available := 0
				if level != nil {
					available = level.Available()
				}
				if pl.QuantityPicked > available {
					sku, name := uc.productInfo(ctx, pl.ProductID)
					return nil, apperrors.NewBadRequest(fmt.Sprintf("Cannot pick %d units of %s (%s); only %d available in stock", pl.QuantityPicked, name, sku, available))
				}
				w.Lines[i].QuantityPicked = pl.QuantityPicked
				if err := uc.repo.UpdatePickWaveLine(ctx, &w.Lines[i]); err != nil {
					return nil, apperrors.NewInternal(err, "Failed to update pick wave line")
				}
			}
		}
	}

	if w.Status == domain.PickWavePending {
		w.Status = domain.PickWaveInProgress
		if err := uc.repo.UpdatePickWave(ctx, w); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update pick wave status")
		}
	}
	return uc.toPickWaveResponse(ctx, w), nil
}

// CompletePickWave finalizes picking and deducts stock via the inventory module,
// writing an "out" movement per line (picked-for-shipment). Lines not explicitly
// picked are treated as fully picked at their requested quantity.
func (uc *warehouseUseCase) CompletePickWave(ctx context.Context, id uuid.UUID) (*PickWaveResponseDTO, error) {
	var result *domain.PickWave
	err := uc.repo.WithTransaction(ctx, func(txRepo domain.WarehouseRepository) error {
		w, err := txRepo.GetPickWaveByID(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get pick wave")
		}
		if w == nil {
			return apperrors.NewNotFound("Pick wave not found")
		}
		if w.Status == domain.PickWaveCompleted || w.Status == domain.PickWaveCancelled {
			return apperrors.NewConflict("Pick wave is already " + w.Status)
		}
		if len(w.Lines) == 0 {
			return apperrors.NewBadRequest("Pick wave has no lines to complete")
		}

		err = uc.inventoryRepo.WithTransaction(ctx, func(txInv inventoryDomain.InventoryRepository) error {
			for i := range w.Lines {
				qty := w.Lines[i].QuantityPicked
				if qty <= 0 {
					qty = w.Lines[i].QuantityToPick
				}
				if qty <= 0 {
					continue
				}
				level, err := txInv.GetStockLevel(ctx, w.Lines[i].ProductID, w.WarehouseID)
				if err != nil {
					return apperrors.NewInternal(err, "Failed to get stock level")
				}
				available := 0
				if level != nil {
					available = level.Available()
				}
				if qty > available {
					sku, name := uc.productInfo(ctx, w.Lines[i].ProductID)
					return apperrors.NewBadRequest(fmt.Sprintf("Cannot pick %d units of %s (%s); only %d available", qty, name, sku, available))
				}

				level.Quantity -= qty
				if level.Quantity < 0 {
					return apperrors.NewBadRequest("Pick would result in negative stock")
				}
				if err := txInv.UpsertStockLevel(ctx, level); err != nil {
					return apperrors.NewInternal(err, "Failed to update stock level")
				}
				movement := &inventoryDomain.StockMovement{
					ProductID: w.Lines[i].ProductID, WarehouseID: w.WarehouseID,
					Type: inventoryDomain.MovementOut, Quantity: -qty, Balance: level.Quantity,
					Reference: w.ID.String(), Reason: "Picked for shipment",
				}
				if err := txInv.CreateMovement(ctx, movement); err != nil {
					return apperrors.NewInternal(err, "Failed to record pick movement")
				}

				w.Lines[i].QuantityPicked = qty
				if err := txRepo.UpdatePickWaveLine(ctx, &w.Lines[i]); err != nil {
					return apperrors.NewInternal(err, "Failed to update pick wave line")
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		w.Status = domain.PickWaveCompleted
		if err := txRepo.UpdatePickWave(ctx, w); err != nil {
			return apperrors.NewInternal(err, "Failed to update pick wave status")
		}
		result = w
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uc.toPickWaveResponse(ctx, result), nil
}

func (uc *warehouseUseCase) CancelPickWave(ctx context.Context, id uuid.UUID) (*PickWaveResponseDTO, error) {
	w, err := uc.repo.GetPickWaveByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get pick wave")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Pick wave not found")
	}
	if w.Status == domain.PickWaveCompleted || w.Status == domain.PickWaveCancelled {
		return nil, apperrors.NewConflict("Pick wave is already " + w.Status)
	}
	w.Status = domain.PickWaveCancelled
	if err := uc.repo.UpdatePickWave(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to cancel pick wave")
	}
	return uc.toPickWaveResponse(ctx, w), nil
}

// Packing sessions

func (uc *warehouseUseCase) CreatePackingSession(ctx context.Context, dto CreatePackingSessionDTO) (*PackingSessionResponseDTO, error) {
	if dto.PickWaveID == uuid.Nil {
		return nil, apperrors.NewBadRequest("pickWaveId is required")
	}
	w, err := uc.repo.GetPickWaveByID(ctx, dto.PickWaveID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get pick wave")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Pick wave not found")
	}
	if w.Status != domain.PickWaveCompleted && w.Status != domain.PickWaveInProgress {
		return nil, apperrors.NewBadRequest("Pick wave must be in progress or completed before packing can start")
	}

	now := time.Now().Format(time.RFC3339)
	s := &domain.PackingSession{
		PickWaveID: dto.PickWaveID,
		Status:     domain.PackingPacking,
		Notes:      dto.Notes,
		StartedAt:  &now,
	}
	if err := uc.repo.CreatePackingSession(ctx, s); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create packing session")
	}
	return ToPackingSessionResponse(s), nil
}

func (uc *warehouseUseCase) ListPackingSessions(ctx context.Context, query types.PaginationQuery) ([]PackingSessionResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListPackingSessions(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list packing sessions")
	}
	result := make([]PackingSessionResponseDTO, len(items))
	for i, s := range items {
		result[i] = *ToPackingSessionResponse(&s)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

func (uc *warehouseUseCase) CompletePackingSession(ctx context.Context, id uuid.UUID, dto CompletePackingSessionDTO) (*PackingSessionResponseDTO, error) {
	s, err := uc.repo.GetPackingSessionByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get packing session")
	}
	if s == nil {
		return nil, apperrors.NewNotFound("Packing session not found")
	}
	if s.Status == domain.PackingCompleted {
		return nil, apperrors.NewConflict("Packing session is already completed")
	}
	if dto.PackageCount > 0 {
		s.PackageCount = dto.PackageCount
	}
	if dto.Notes != "" {
		s.Notes = dto.Notes
	}
	now := time.Now().Format(time.RFC3339)
	s.CompletedAt = &now
	s.Status = domain.PackingCompleted
	if err := uc.repo.UpdatePackingSession(ctx, s); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to complete packing session")
	}
	return ToPackingSessionResponse(s), nil
}

// Shipments

func (uc *warehouseUseCase) CreateShipment(ctx context.Context, dto CreateShipmentDTO) (*ShipmentResponseDTO, error) {
	if dto.PickWaveID == uuid.Nil {
		return nil, apperrors.NewBadRequest("pickWaveId is required")
	}
	w, err := uc.repo.GetPickWaveByID(ctx, dto.PickWaveID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get pick wave")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Pick wave not found")
	}
	if w.Status != domain.PickWaveCompleted {
		return nil, apperrors.NewBadRequest("Pick wave must be completed before booking a shipment")
	}

	if dto.PackingSessionID != nil {
		ps, err := uc.repo.GetPackingSessionByID(ctx, *dto.PackingSessionID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get packing session")
		}
		if ps == nil {
			return nil, apperrors.NewNotFound("Packing session not found")
		}
		if ps.Status != domain.PackingCompleted {
			return nil, apperrors.NewBadRequest("Packing session must be completed before booking a shipment")
		}
	}

	s := &domain.Shipment{
		PickWaveID:         dto.PickWaveID,
		PackingSessionID:   dto.PackingSessionID,
		Carrier:            dto.Carrier,
		TrackingNumber:     dto.TrackingNumber,
		Status:             domain.ShipmentBooked,
		DestinationAddress: dto.DestinationAddress,
	}
	if err := uc.repo.CreateShipment(ctx, s); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create shipment")
	}
	return ToShipmentResponse(s), nil
}

func (uc *warehouseUseCase) ListShipments(ctx context.Context, query types.PaginationQuery) ([]ShipmentResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListShipments(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list shipments")
	}
	result := make([]ShipmentResponseDTO, len(items))
	for i, s := range items {
		result[i] = *ToShipmentResponse(&s)
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

func (uc *warehouseUseCase) DispatchShipment(ctx context.Context, id uuid.UUID) (*ShipmentResponseDTO, error) {
	s, err := uc.repo.GetShipmentByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get shipment")
	}
	if s == nil {
		return nil, apperrors.NewNotFound("Shipment not found")
	}
	if s.Status != domain.ShipmentBooked {
		return nil, apperrors.NewConflict("Only booked shipments can be dispatched")
	}
	now := time.Now().Format(time.RFC3339)
	s.DispatchedAt = &now
	s.Status = domain.ShipmentDispatched
	if err := uc.repo.UpdateShipment(ctx, s); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to dispatch shipment")
	}
	return ToShipmentResponse(s), nil
}

// SeedInitialData creates one sample pick wave referencing a real warehouse/product if any exist.
func (uc *warehouseUseCase) SeedInitialData(ctx context.Context) error {
	existing, _, err := uc.repo.ListPickWaves(ctx, types.PaginationQuery{Page: 1, PerPage: 1})
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	warehouses, err := uc.inventoryRepo.ListAllWarehouses(ctx)
	if err != nil || len(warehouses) == 0 {
		return nil
	}
	wh := warehouses[0]

	levels, err := uc.inventoryRepo.ListStockLevelsByWarehouse(ctx, wh.ID)
	if err != nil || len(levels) == 0 {
		return nil
	}

	var lines []PickWaveLineDTO
	for _, lvl := range levels {
		if lvl.Available() <= 0 {
			continue
		}
		qty := lvl.Available()
		if qty > 5 {
			qty = 5
		}
		lines = append(lines, PickWaveLineDTO{ProductID: lvl.ProductID, QuantityToPick: qty})
		break
	}
	if len(lines) == 0 {
		return nil
	}

	_, err = uc.CreatePickWave(ctx, CreatePickWaveDTO{
		WarehouseID: wh.ID,
		AssignedTo:  "Warehouse Team",
		Lines:       lines,
	})
	return err
}
