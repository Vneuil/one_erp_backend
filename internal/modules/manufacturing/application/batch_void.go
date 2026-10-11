package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

const batchSourcePrefix = "production-batch:"
const batchVoidSourcePrefix = "production-batch-void:"

// ListBatches returns an order's completed batches, voided ones included.
func (uc *manufacturingUseCase) ListBatches(ctx context.Context, orderID uuid.UUID) ([]domain.ProductionBatch, error) {
	o, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get production order")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Production order not found")
	}
	out, err := uc.repo.ListBatchesByOrder(ctx, orderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list batches")
	}
	return out, nil
}

// VoidBatch cancels a completed batch: the finished units leave stock again,
// the materials a backflush order consumed return to stock, the order's
// completed quantity goes back (a completed order reopens), the finished-goods
// journal is reversed by an entry dated today, and the batch's cost is taken
// back out of the product's moving average. It is refused when the finished
// units are no longer in stock (already sold or moved).
func (uc *manufacturingUseCase) VoidBatch(ctx context.Context, orderID, batchID uuid.UUID, reason string) (*ProductionOrderResponseDTO, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required to void a batch")
	}
	var result *domain.ProductionOrder
	var batch domain.ProductionBatch
	var productID uuid.UUID
	var stockAfter int
	err := uc.repo.WithTransactionAndInventory(ctx, func(tx domain.ManufacturingRepository, txInv inventoryDomain.InventoryRepository) error {
		o, err := tx.GetOrderByID(ctx, orderID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get production order")
		}
		if o == nil {
			return apperrors.NewNotFound("Production order not found")
		}
		b, err := tx.GetBatch(ctx, batchID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get the batch")
		}
		if b == nil || b.ProductionOrderID != o.ID {
			return apperrors.NewNotFound("Batch not found on this order")
		}
		if b.Voided {
			return apperrors.NewConflict("This batch is already voided")
		}
		if o.Status == domain.OrderCancelled {
			return apperrors.NewConflict("A cancelled order's batches can no longer be voided")
		}
		bom, err := tx.GetBOMByID(ctx, o.BOMID)
		if err != nil || bom == nil {
			return apperrors.NewInternal(err, "Failed to get BOM")
		}

		fg, err := txInv.GetStockLevel(ctx, bom.ProductID, o.WarehouseID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get finished goods stock level")
		}
		onHand := 0
		if fg != nil {
			onHand = fg.Available()
		}
		if onHand < b.QuantityCompleted {
			sku, name := uc.productInfo(ctx, bom.ProductID)
			return apperrors.NewBadRequest(fmt.Sprintf("Cannot void: %s (%s) has only %d in stock, %d were produced in this batch and some have left the warehouse", name, sku, onHand, b.QuantityCompleted))
		}

		fg.Quantity -= b.QuantityCompleted
		if err := txInv.UpsertStockLevel(ctx, fg); err != nil {
			return apperrors.NewInternal(err, "Failed to update finished goods stock level")
		}
		if err := txInv.CreateMovement(ctx, &inventoryDomain.StockMovement{ProductID: bom.ProductID, WarehouseID: o.WarehouseID,
			Type: inventoryDomain.MovementOut, Quantity: -b.QuantityCompleted, Balance: fg.Quantity, Reference: o.ID.String(), Reason: "Manufacturing batch voided"}); err != nil {
			return apperrors.NewInternal(err, "Failed to record the reversal movement")
		}
		stockAfter = fg.Quantity + b.QuantityCompleted // stock with the batch still counted, for the cost average

		if o.MaterialMode != domain.MaterialIssued {
			for _, line := range bom.Lines {
				back := int(line.QuantityRequired * float64(b.QuantityCompleted))
				if back <= 0 {
					continue
				}
				level, err := txInv.GetStockLevel(ctx, line.ComponentProductID, o.WarehouseID)
				if err != nil {
					return apperrors.NewInternal(err, "Failed to get component stock level")
				}
				if level == nil {
					level = &inventoryDomain.StockLevel{ProductID: line.ComponentProductID, WarehouseID: o.WarehouseID}
				}
				level.Quantity += back
				if err := txInv.UpsertStockLevel(ctx, level); err != nil {
					return apperrors.NewInternal(err, "Failed to restore component stock")
				}
				if err := txInv.CreateMovement(ctx, &inventoryDomain.StockMovement{ProductID: line.ComponentProductID, WarehouseID: o.WarehouseID,
					Type: inventoryDomain.MovementIn, Quantity: back, Balance: level.Quantity, Reference: o.ID.String(), Reason: "Manufacturing batch voided"}); err != nil {
					return apperrors.NewInternal(err, "Failed to record the component return")
				}
			}
		}

		now := time.Now()
		b.Voided, b.VoidedAt, b.VoidedBy, b.VoidReason = true, &now, actor.EmailFrom(ctx), reason
		if err := tx.VoidBatch(ctx, b); err != nil {
			return apperrors.NewInternal(err, "Failed to mark the batch void")
		}
		o.QuantityCompleted -= b.QuantityCompleted
		if o.Status == domain.OrderCompleted {
			o.Status, o.ActualCompletionDate = domain.OrderInProgress, ""
		}
		if err := tx.UpdateOrder(ctx, o); err != nil {
			return apperrors.NewInternal(err, "Failed to update production order")
		}
		result, batch, productID = o, *b, bom.ProductID
		return nil
	})
	if err != nil {
		return nil, err
	}

	if batch.CostApplied {
		uc.revertProductCost(ctx, productID, batch.QuantityCompleted, stockAfter, batch.UnitCost)
	}
	if uc.ledger != nil {
		if _, err := uc.ledger.ReverseEntry(ctx, batchSourcePrefix+batch.ID.String(), batchVoidSourcePrefix+batch.ID.String(),
			time.Now().Format("2006-01-02"), "Pembatalan batch "+OrderLabel(result)); err != nil {
			slog.Error("manufacturing: failed to reverse the finished goods entry", "batch", batch.ID, "error", err)
		}
	}
	return uc.toOrderResponse(ctx, result), nil
}
