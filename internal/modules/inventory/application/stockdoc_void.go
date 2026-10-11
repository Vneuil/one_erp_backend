package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

const stockDocVoidSourcePrefix = "stock-doc-void:"

// StockDocumentVoidEntry is the journal of a voided document: the original
// entry with debit and credit swapped.
func StockDocumentVoidEntry(d *domain.StockDocument) financeApp.LedgerEntry {
	e := StockDocumentLedgerEntry(d)
	lines := make([]financeApp.LedgerLine, len(e.Lines))
	for i, l := range e.Lines {
		lines[i] = financeApp.LedgerLine{AccountCode: l.AccountCode, Debit: l.Credit, Credit: l.Debit, Description: "Batal " + l.Description}
	}
	return financeApp.LedgerEntry{SourceDoc: stockDocVoidSourcePrefix + d.ID.String(), Memo: "Pembatalan " + e.Memo, Lines: lines}
}

// VoidStockDocument cancels a posted stock document: every line's stock
// movement is reversed, its journal entry is reversed (dated today, so closed
// periods are left alone), and the document stays on record marked void.
// Nothing changes unless the whole reversal is possible - for example a
// receipt cannot be voided after its stock has been used. To correct a
// document, void it and enter the right one.
func (uc *inventoryUseCase) VoidStockDocument(ctx context.Context, id uuid.UUID, reason string) (*StockDocumentResponseDTO, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required to void a stock document")
	}
	doc, err := uc.repo.GetStockDocument(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get the stock document")
	}
	if doc == nil {
		return nil, apperrors.NewNotFound("Stock document not found")
	}
	if doc.Voided {
		return nil, apperrors.NewConflict("This stock document is already voided")
	}
	kind := docKinds[doc.Type]
	back := -kind.sign // direction of the reversing movements

	if doc.ProductionOrderID != nil {
		used, err := uc.repo.OrderHasCompletedUnits(ctx, *doc.ProductionOrderID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check the production order")
		}
		if used {
			return nil, apperrors.NewConflict("This material issue belongs to a production order that has already completed units; voiding it would leave those units without issued materials")
		}
	}

	// Which lot each original outgoing movement took from, so an issue returns to the same lot.
	lotQueue := map[uuid.UUID][]string{}
	if kind.sign < 0 {
		moves, err := uc.repo.ListMovementsByReference(ctx, doc.Number)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to read the document's stock movements")
		}
		for _, m := range moves {
			if m.Quantity < 0 {
				lotQueue[m.ProductID] = append(lotQueue[m.ProductID], m.BatchNo)
			}
		}
	}

	// Validate the reversal before touching anything.
	if back < 0 {
		need := map[uuid.UUID]int{}
		for _, l := range doc.Lines {
			need[l.ProductID] += l.Quantity
			if l.BatchNo != "" {
				b, err := uc.repo.GetBatchByNo(ctx, l.ProductID, doc.WarehouseID, l.BatchNo)
				if err != nil {
					return nil, apperrors.NewInternal(err, "Failed to load batch")
				}
				if b == nil || b.Quantity < l.Quantity {
					sku, name := uc.productInfo(ctx, l.ProductID)
					left := 0
					if b != nil {
						left = b.Quantity
					}
					return nil, apperrors.NewBadRequest(fmt.Sprintf("Cannot void: batch %s of %s (%s) has only %d left, %d were received on this document", l.BatchNo, name, sku, left, l.Quantity))
				}
			}
		}
		for pid, n := range need {
			avail, err := uc.GetAvailability(ctx, pid, doc.WarehouseID)
			if err != nil {
				return nil, err
			}
			if n > avail {
				sku, name := uc.productInfo(ctx, pid)
				return nil, apperrors.NewBadRequest(fmt.Sprintf("Cannot void: %s (%s) has only %d in stock, %d came in on this document and part of it has been used", name, sku, avail, n))
			}
		}
	}

	by := actor.EmailFrom(ctx)
	moveReason := "Pembatalan " + doc.Number + ": " + reason
	var applied []AdjustStockDTO
	rollback := func() {
		for _, a := range applied {
			inv := a
			inv.Quantity = -a.Quantity
			inv.BatchID, inv.BatchNo, inv.ExpiryDate = nil, "", ""
			inv.Reason = "Pembatalan otomatis atas pembatalan " + doc.Number
			if _, err := uc.AdjustStock(ctx, inv); err != nil {
				slog.Error("inventory: failed to roll back a voided document line", "document", doc.Number, "product", a.ProductID, "error", err)
			}
		}
	}
	for _, l := range doc.Lines {
		dto := AdjustStockDTO{ProductID: l.ProductID, WarehouseID: doc.WarehouseID, Quantity: back * l.Quantity, Reason: moveReason, Reference: doc.Number, CreatedBy: by}
		if back < 0 && l.BatchNo != "" {
			if b, err := uc.repo.GetBatchByNo(ctx, l.ProductID, doc.WarehouseID, l.BatchNo); err == nil && b != nil {
				dto.BatchID = &b.ID
			}
		}
		if back > 0 {
			if q := lotQueue[l.ProductID]; len(q) > 0 {
				dto.BatchNo, lotQueue[l.ProductID] = q[0], q[1:]
			}
		}
		if _, err := uc.AdjustStock(ctx, dto); err != nil {
			rollback()
			return nil, err
		}
		applied = append(applied, dto)
	}

	now := time.Now()
	doc.Voided, doc.VoidedAt, doc.VoidedBy, doc.VoidReason = true, &now, by, reason
	if err := uc.repo.VoidStockDocument(ctx, doc); err != nil {
		rollback()
		return nil, apperrors.NewInternal(err, "Failed to mark the document void; the stock reversal was rolled back")
	}

	var postErr error
	if doc.Posted && doc.TotalValue > 0 {
		if uc.ledger == nil {
			postErr = fmt.Errorf("no ledger configured")
		} else {
			e := StockDocumentVoidEntry(doc)
			postErr = uc.ledger.PostEntryOn(ctx, today(), e.SourceDoc, e.Memo, e.Lines)
		}
	}
	resp := uc.toStockDocumentResponse(ctx, doc)
	if postErr != nil {
		return resp, apperrors.NewInternal(postErr, "Document voided and stock restored, but its journal could not be reversed; reverse it manually")
	}
	return resp, nil
}
