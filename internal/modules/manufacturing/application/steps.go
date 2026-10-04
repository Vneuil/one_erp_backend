package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// LogStep records units finishing one routing step. A step can only be given
// units the previous step has already passed (the first step: the order
// quantity), and only while the order is released or in progress.
func (uc *manufacturingUseCase) LogStep(ctx context.Context, orderID, stepID uuid.UUID, dto LogStepDTO) (*ProductionOrderResponseDTO, error) {
	if dto.Quantity <= 0 {
		return nil, apperrors.NewBadRequest("quantity must be positive")
	}
	date := dto.Date
	today := time.Now().Format("2006-01-02")
	if date == "" {
		date = today
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	if date > today {
		return nil, apperrors.NewBadRequest("date cannot be in the future")
	}

	var result *domain.ProductionOrder
	err := uc.repo.WithTransaction(ctx, func(tx domain.ManufacturingRepository) error {
		o, err := tx.GetOrderByID(ctx, orderID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get production order")
		}
		if o == nil {
			return apperrors.NewNotFound("Production order not found")
		}
		if o.Status != domain.OrderReleased && o.Status != domain.OrderInProgress {
			return apperrors.NewConflict("Process output can only be logged on released or in-progress production orders")
		}
		idx := -1
		for i := range o.Steps {
			if o.Steps[i].ID == stepID {
				idx = i
			}
		}
		if idx < 0 {
			return apperrors.NewNotFound("Process step not found on this order")
		}
		step := &o.Steps[idx]
		upstream := o.QuantityToProduce
		if idx > 0 {
			upstream = o.Steps[idx-1].QuantityDone
		}
		if allowed := upstream - step.QuantityDone; dto.Quantity > allowed {
			where := "the order quantity"
			if idx > 0 {
				where = fmt.Sprintf("what step %d (%s) has finished", idx, o.Steps[idx-1].Name)
			}
			return apperrors.NewBadRequest(fmt.Sprintf("Only %d units can still be logged on %s: that is limited by %s", max(allowed, 0), step.Name, where))
		}

		step.QuantityDone += dto.Quantity
		if step.StartedDate == "" {
			step.StartedDate = date
		}
		if step.QuantityDone >= o.QuantityToProduce {
			step.Status, step.CompletedDate = domain.StepDone, date
		} else {
			step.Status = domain.StepInProgress
		}
		if err := tx.UpdateStep(ctx, step); err != nil {
			return apperrors.NewInternal(err, "Failed to update the process step")
		}
		if err := tx.CreateStepLog(ctx, &domain.ProductionStepLog{OrderID: o.ID, StepID: step.ID, Quantity: dto.Quantity, Date: date,
			Notes: strings.TrimSpace(dto.Notes), CreatedBy: actor.EmailFrom(ctx)}); err != nil {
			return apperrors.NewInternal(err, "Failed to record the process log")
		}
		if o.Status == domain.OrderReleased {
			o.Status = domain.OrderInProgress
			if err := tx.UpdateOrder(ctx, o); err != nil {
				return apperrors.NewInternal(err, "Failed to update production order")
			}
		}
		result = o
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uc.toOrderResponse(ctx, result), nil
}
