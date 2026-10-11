package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
	"log/slog"
)

const conversionVoidSourcePrefix = "production-conversion-void:"

// ListStepLogs returns an order's process logs, voided ones included.
func (uc *manufacturingUseCase) ListStepLogs(ctx context.Context, orderID uuid.UUID) ([]domain.ProductionStepLog, error) {
	o, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get production order")
	}
	if o == nil {
		return nil, apperrors.NewNotFound("Production order not found")
	}
	out, err := uc.repo.ListOrderStepLogs(ctx, orderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list process logs")
	}
	return out, nil
}

// VoidStepLog cancels a process log entered by mistake: the step goes back by
// the logged units, and the labour and overhead it charged to Work in Process
// is reversed. The log stays on record, marked void, and no longer counts in
// reports. It is refused when later work depends on those units - the next
// step has already taken them, or finished goods were completed from them.
func (uc *manufacturingUseCase) VoidStepLog(ctx context.Context, orderID, logID uuid.UUID, reason string) (*ProductionOrderResponseDTO, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required to void a process log")
	}
	var result *domain.ProductionOrder
	var step domain.ProductionStep
	var logCopy domain.ProductionStepLog
	err := uc.repo.WithTransaction(ctx, func(tx domain.ManufacturingRepository) error {
		o, err := tx.GetOrderByID(ctx, orderID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get production order")
		}
		if o == nil {
			return apperrors.NewNotFound("Production order not found")
		}
		l, err := tx.GetStepLog(ctx, logID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get the process log")
		}
		if l == nil || l.OrderID != o.ID {
			return apperrors.NewNotFound("Process log not found on this order")
		}
		if l.Voided {
			return apperrors.NewConflict("This process log is already voided")
		}
		if o.Status != domain.OrderReleased && o.Status != domain.OrderInProgress && o.Status != domain.OrderPaused {
			return apperrors.NewConflict("A process log can only be voided while the order is released, in progress or paused")
		}
		idx := -1
		for i := range o.Steps {
			if o.Steps[i].ID == l.StepID {
				idx = i
			}
		}
		if idx < 0 {
			return apperrors.NewNotFound("The step of this log no longer exists on the order")
		}
		s := &o.Steps[idx]
		left := s.QuantityDone - l.Quantity
		if left < 0 {
			return apperrors.NewConflict(fmt.Sprintf("Step %s counts only %d units, fewer than this log's %d", s.Name, s.QuantityDone, l.Quantity))
		}
		if idx+1 < len(o.Steps) && o.Steps[idx+1].QuantityDone > left {
			return apperrors.NewConflict(fmt.Sprintf("Cannot void: step %d (%s) has already taken %d units that passed %s; void its logs first", idx+2, o.Steps[idx+1].Name, o.Steps[idx+1].QuantityDone, s.Name))
		}
		if idx == len(o.Steps)-1 && o.QuantityCompleted > left {
			return apperrors.NewConflict(fmt.Sprintf("Cannot void: %d units were already completed as finished goods and only %d would remain through %s", o.QuantityCompleted, left, s.Name))
		}

		s.QuantityDone = left
		switch {
		case left <= 0:
			s.Status, s.StartedDate, s.CompletedDate = domain.StepPending, "", ""
		case left >= o.QuantityToProduce:
			s.Status = domain.StepDone
		default:
			s.Status, s.CompletedDate = domain.StepInProgress, ""
		}
		if err := tx.UpdateStep(ctx, s); err != nil {
			return apperrors.NewInternal(err, "Failed to update the process step")
		}
		now := time.Now()
		l.Voided, l.VoidedAt, l.VoidedBy, l.VoidReason = true, &now, actor.EmailFrom(ctx), reason
		if err := tx.VoidStepLog(ctx, l); err != nil {
			return apperrors.NewInternal(err, "Failed to mark the log void")
		}
		result, step, logCopy = o, *s, *l
		return nil
	})
	if err != nil {
		return nil, err
	}
	uc.reverseConversion(ctx, result, step, logCopy)
	return uc.toOrderResponse(ctx, result), nil
}

// reverseConversion reverses the labour and overhead a voided log charged to WIP.
func (uc *manufacturingUseCase) reverseConversion(ctx context.Context, o *domain.ProductionOrder, step domain.ProductionStep, l domain.ProductionStepLog) {
	if uc.ledger == nil {
		return
	}
	labor, overhead := step.UnitLabor()*float64(l.Quantity), step.UnitOverhead()*float64(l.Quantity)
	if round2(labor)+round2(overhead) <= 0 {
		return
	}
	e := ConversionLedgerEntry(l.ID, OrderLabel(o), step.Name, labor, overhead)
	lines := make([]financeApp.LedgerLine, len(e.Lines))
	for i, ln := range e.Lines {
		lines[i] = financeApp.LedgerLine{AccountCode: ln.AccountCode, Debit: ln.Credit, Credit: ln.Debit, Description: "Batal " + ln.Description}
	}
	if err := uc.ledger.PostEntryOn(ctx, time.Now().Format("2006-01-02"), conversionVoidSourcePrefix+l.ID.String(), "Pembatalan "+e.Memo, lines); err != nil {
		slog.Error("manufacturing: failed to reverse conversion cost", "order", o.ID, "log", l.ID, "error", err)
	}
}
