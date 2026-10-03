package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

const expiringSoonDays = 30

var batchZone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}()

func today() string { return time.Now().In(batchZone).Format("2006-01-02") }

// BatchStatus classifies a lot by its expiry date relative to today. daysLeft is
// nil for lots that never expire.
func BatchStatus(expiry, today string) (status string, daysLeft *int) {
	if expiry == "" {
		return "no_expiry", nil
	}
	e, err1 := time.Parse("2006-01-02", expiry)
	t, err2 := time.Parse("2006-01-02", today)
	if err1 != nil || err2 != nil {
		return "no_expiry", nil
	}
	d := int(e.Sub(t).Hours() / 24)
	switch {
	case d < 0:
		return "expired", &d
	case d <= expiringSoonDays:
		return "expiring_soon", &d
	default:
		return "ok", &d
	}
}

// expiredQuantity is how much of a product in a warehouse sits in expired lots.
func (uc *inventoryUseCase) expiredQuantity(ctx context.Context, productID, warehouseID uuid.UUID) int {
	batches, err := uc.repo.ListBatchesFEFO(ctx, productID, warehouseID)
	if err != nil {
		return 0
	}
	now := today()
	total := 0
	for _, b := range batches {
		if b.ExpiryDate != "" && b.ExpiryDate < now {
			total += b.Quantity
		}
	}
	return total
}

// applyBatches keeps lot quantities in step with a stock adjustment and returns
// the lot the movement should be tagged with ("" when none or several).
//
//   - An increase with a BatchNo is filed under that lot (created on first use).
//   - A decrease with a BatchID comes out of that exact lot.
//   - Any other decrease consumes unexpired lots soonest-expiry-first; stock that
//     is not tracked in any lot (older stock) is taken from the untracked balance.
func (uc *inventoryUseCase) applyBatches(ctx context.Context, repo domain.InventoryRepository, dto AdjustStockDTO) (string, error) {
	switch {
	case dto.Quantity > 0 && strings.TrimSpace(dto.BatchNo) != "":
		return uc.receiveIntoBatch(ctx, repo, dto)
	case dto.Quantity < 0 && dto.BatchID != nil:
		b, err := repo.GetBatch(ctx, *dto.BatchID)
		if err != nil {
			return "", apperrors.NewInternal(err, "Failed to load batch")
		}
		if b == nil || b.ProductID != dto.ProductID || b.WarehouseID != dto.WarehouseID {
			return "", apperrors.NewNotFound("Batch not found for this product and warehouse")
		}
		need := -dto.Quantity
		if b.Quantity < need {
			return "", apperrors.NewBadRequest(fmt.Sprintf("Batch %s has only %d left", b.BatchNo, b.Quantity))
		}
		b.Quantity -= need
		if err := repo.SaveBatch(ctx, b); err != nil {
			return "", apperrors.NewInternal(err, "Failed to update batch")
		}
		return b.BatchNo, nil
	case dto.Quantity < 0:
		batches, err := repo.ListBatchesFEFO(ctx, dto.ProductID, dto.WarehouseID)
		if err != nil {
			return "", apperrors.NewInternal(err, "Failed to load batches")
		}
		need, now := -dto.Quantity, today()
		var used []string
		for i := range batches {
			b := &batches[i]
			if need == 0 {
				break
			}
			if b.ExpiryDate != "" && b.ExpiryDate < now {
				continue // expired lots are never sold; write them off instead
			}
			take := min(need, b.Quantity)
			b.Quantity -= take
			need -= take
			if err := repo.SaveBatch(ctx, b); err != nil {
				return "", apperrors.NewInternal(err, "Failed to update batch")
			}
			used = append(used, b.BatchNo)
		}
		if len(used) == 1 {
			return used[0], nil
		}
		return "", nil
	}
	return "", nil
}

func (uc *inventoryUseCase) receiveIntoBatch(ctx context.Context, repo domain.InventoryRepository, dto AdjustStockDTO) (string, error) {
	no := strings.TrimSpace(dto.BatchNo)
	if dto.ExpiryDate != "" {
		if _, err := time.Parse("2006-01-02", dto.ExpiryDate); err != nil {
			return "", apperrors.NewBadRequest("expiryDate must be YYYY-MM-DD")
		}
	}
	b, err := repo.GetBatchByNo(ctx, dto.ProductID, dto.WarehouseID, no)
	if err != nil {
		return "", apperrors.NewInternal(err, "Failed to load batch")
	}
	if b == nil {
		b = &domain.StockBatch{ProductID: dto.ProductID, WarehouseID: dto.WarehouseID, BatchNo: no, ExpiryDate: dto.ExpiryDate, ReceivedAt: today()}
	} else if dto.ExpiryDate != "" && b.ExpiryDate != "" && b.ExpiryDate != dto.ExpiryDate {
		// One batch number is one lot with one expiry date.
		return "", apperrors.NewConflict(fmt.Sprintf("Batch %s already exists with expiry %s", no, b.ExpiryDate))
	} else if b.ExpiryDate == "" {
		b.ExpiryDate = dto.ExpiryDate
	}
	b.Quantity += dto.Quantity
	b.InitialQty += dto.Quantity
	if err := repo.SaveBatch(ctx, b); err != nil {
		return "", apperrors.NewInternal(err, "Failed to save batch")
	}
	return no, nil
}

func (uc *inventoryUseCase) toBatchDTO(ctx context.Context, b domain.StockBatch, names map[uuid.UUID][2]string, whs map[uuid.UUID]string) BatchResponseDTO {
	if _, ok := names[b.ProductID]; !ok {
		sku, name := uc.productInfo(ctx, b.ProductID)
		names[b.ProductID] = [2]string{sku, name}
	}
	if _, ok := whs[b.WarehouseID]; !ok {
		whs[b.WarehouseID] = uc.warehouseName(ctx, b.WarehouseID)
	}
	status, days := BatchStatus(b.ExpiryDate, today())
	return BatchResponseDTO{ID: b.ID, ProductID: b.ProductID, ProductSKU: names[b.ProductID][0], ProductName: names[b.ProductID][1],
		WarehouseID: b.WarehouseID, WarehouseName: whs[b.WarehouseID], BatchNo: b.BatchNo, ExpiryDate: b.ExpiryDate, ReceivedAt: b.ReceivedAt,
		InitialQty: b.InitialQty, Quantity: b.Quantity, Status: status, DaysToExpiry: days}
}

func (uc *inventoryUseCase) batchDTOs(ctx context.Context, list []domain.StockBatch) []BatchResponseDTO {
	names, whs := map[uuid.UUID][2]string{}, map[uuid.UUID]string{}
	out := make([]BatchResponseDTO, len(list))
	for i, b := range list {
		out[i] = uc.toBatchDTO(ctx, b, names, whs)
	}
	return out
}

// ReceiveBatch books goods into stock under a batch number, with an optional expiry.
func (uc *inventoryUseCase) ReceiveBatch(ctx context.Context, dto ReceiveBatchDTO) (*BatchResponseDTO, error) {
	if dto.Quantity <= 0 || strings.TrimSpace(dto.BatchNo) == "" {
		return nil, apperrors.NewBadRequest("batchNo and a positive quantity are required")
	}
	if dto.ExpiryDate != "" {
		if _, err := time.Parse("2006-01-02", dto.ExpiryDate); err != nil {
			return nil, apperrors.NewBadRequest("expiryDate must be YYYY-MM-DD")
		}
		if dto.ExpiryDate < today() {
			return nil, apperrors.NewBadRequest("The expiry date is already in the past")
		}
	}
	if _, err := uc.AdjustStock(ctx, AdjustStockDTO{ProductID: dto.ProductID, WarehouseID: dto.WarehouseID, Quantity: dto.Quantity,
		BatchNo: dto.BatchNo, ExpiryDate: dto.ExpiryDate, Reason: "Batch received", Reference: dto.Reference}); err != nil {
		return nil, err
	}
	b, err := uc.repo.GetBatchByNo(ctx, dto.ProductID, dto.WarehouseID, strings.TrimSpace(dto.BatchNo))
	if err != nil || b == nil {
		return nil, apperrors.NewInternal(err, "Batch received but could not be reloaded")
	}
	out := uc.batchDTOs(ctx, []domain.StockBatch{*b})
	return &out[0], nil
}

func (uc *inventoryUseCase) ListBatches(ctx context.Context, productID, warehouseID *uuid.UUID, includeEmpty bool) ([]BatchResponseDTO, error) {
	list, err := uc.repo.ListBatches(ctx, productID, warehouseID, includeEmpty)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list batches")
	}
	return uc.batchDTOs(ctx, list), nil
}

// ExpiringBatches lists lots with stock that are already expired or expire within withinDays.
func (uc *inventoryUseCase) ExpiringBatches(ctx context.Context, withinDays int) ([]BatchResponseDTO, error) {
	if withinDays < 0 || withinDays > 3650 {
		return nil, apperrors.NewBadRequest("days must be between 0 and 3650")
	}
	limit := time.Now().In(batchZone).AddDate(0, 0, withinDays).Format("2006-01-02")
	list, err := uc.repo.ListBatchesExpiringBy(ctx, limit)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list expiring batches")
	}
	return uc.batchDTOs(ctx, list), nil
}

// WriteOffBatch removes what is left of a lot (typically expired stock) from stock
// and books its cost as a loss.
func (uc *inventoryUseCase) WriteOffBatch(ctx context.Context, id uuid.UUID, reason string) (*BatchResponseDTO, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required")
	}
	b, err := uc.repo.GetBatch(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load batch")
	}
	if b == nil {
		return nil, apperrors.NewNotFound("Batch not found")
	}
	if b.Quantity <= 0 {
		return nil, apperrors.NewConflict("This batch has nothing left to write off")
	}
	qty := b.Quantity
	if _, err := uc.AdjustStock(ctx, AdjustStockDTO{ProductID: b.ProductID, WarehouseID: b.WarehouseID, Quantity: -qty, BatchID: &b.ID,
		Reason: "Write-off: " + reason, Reference: "batch " + b.BatchNo}); err != nil {
		return nil, err
	}
	uc.postWriteOff(ctx, b, qty)
	after, err := uc.repo.GetBatch(ctx, id)
	if err != nil || after == nil {
		return nil, apperrors.NewInternal(err, "Written off but could not reload the batch")
	}
	out := uc.batchDTOs(ctx, []domain.StockBatch{*after})
	return &out[0], nil
}

func (uc *inventoryUseCase) postWriteOff(ctx context.Context, b *domain.StockBatch, qty int) {
	if uc.ledger == nil {
		return
	}
	p, err := uc.productRepo.GetByID(ctx, b.ProductID)
	if err != nil || p == nil || p.CostPrice <= 0 {
		return
	}
	cost := float64(qty) * p.CostPrice
	err = uc.ledger.PostEntry(ctx, fmt.Sprintf("stock-writeoff:%s:%d", b.ID, b.InitialQty), "Stock write-off "+p.Name+" batch "+b.BatchNo, []financeApp.LedgerLine{
		{AccountCode: financeApp.AccountExpense, Debit: cost, Description: p.Name},
		{AccountCode: financeApp.AccountInventory, Credit: cost, Description: p.Name},
	})
	if err != nil {
		slog.Error("inventory: failed to post stock write-off", "batch", b.ID, "error", err)
	}
}
