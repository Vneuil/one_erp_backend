package application

import (
	"context"
	"fmt"
	"strings"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// DeliveryCostSourcePrefix keys the journal entry for a delivery's shipping cost.
const DeliveryCostSourcePrefix = "delivery-cost:"

// DeliveryCostLedgerEntry is Dr Delivery Expense / Cr Cash (paid now) or Cr
// Accounts Payable (owed to the carrier).
func DeliveryCostLedgerEntry(d *domain.Delivery) financeApp.LedgerEntry {
	credit := financeApp.AccountPayable
	if d.ShippingPaid {
		credit = financeApp.AccountCash
	}
	label := "Delivery " + d.DeliveryNumber
	return financeApp.LedgerEntry{
		SourceDoc: DeliveryCostSourcePrefix + d.ID.String(),
		Memo:      "Shipping cost " + d.DeliveryNumber,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountDeliveryExpense, Debit: d.ShippingCost, Description: label},
			{AccountCode: credit, Credit: d.ShippingCost, Description: label},
		},
	}
}

// RecordShippingCost books the carrier's charge for a delivery. The cost can be
// recorded once; the carrier and tracking number can be filled in at any time.
func (uc *salesUseCase) RecordShippingCost(ctx context.Context, id uuid.UUID, dto ShippingCostDTO) (*DeliveryResponseDTO, error) {
	if dto.Amount < 0 {
		return nil, apperrors.NewBadRequest("amount cannot be negative")
	}
	d, err := uc.repo.GetDeliveryByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get delivery")
	}
	if d == nil {
		return nil, apperrors.NewNotFound("Delivery not found")
	}
	if dto.Amount > 0 && d.ShippingCost > 0 {
		return nil, apperrors.NewConflict(fmt.Sprintf("The shipping cost was already recorded (%.2f)", d.ShippingCost))
	}
	if c := strings.TrimSpace(dto.Carrier); c != "" {
		d.Carrier = c
	}
	if t := strings.TrimSpace(dto.TrackingNumber); t != "" {
		d.TrackingNumber = t
	}
	if dto.Amount > 0 {
		d.ShippingCost, d.ShippingPaid = round2(dto.Amount), dto.PaidNow
	}
	if err := uc.repo.UpdateDelivery(ctx, d); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save the shipping details")
	}
	if dto.Amount > 0 {
		e := DeliveryCostLedgerEntry(d)
		uc.postLedger(ctx, e.SourceDoc, e.Memo, e.Lines)
	}
	return ToDeliveryResponse(d), nil
}
