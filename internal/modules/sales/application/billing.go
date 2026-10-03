package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

const maxBillingTerms = 12

// BillingTermInput describes one instalment. Give DueDate (YYYY-MM-DD) or
// DueInDays (from today); with neither, the term is due today.
type BillingTermInput struct {
	Label     string  `json:"label"`
	Percent   float64 `json:"percent"`
	DueDate   string  `json:"dueDate"`
	DueInDays int     `json:"dueInDays"`
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// SplitByPercent divides total across percentages that add up to 100. Each
// share is rounded to 2 decimals and the last one absorbs the rounding
// remainder, so the shares always add up to exactly total.
func SplitByPercent(total float64, percents []float64) []float64 {
	out := make([]float64, len(percents))
	var assigned float64
	for i, p := range percents {
		if i == len(percents)-1 {
			out[i] = round2(total - assigned)
			break
		}
		out[i] = round2(total * p / 100)
		assigned += out[i]
	}
	return out
}

// SetBillingSchedule replaces an order's billing schedule. It refuses once any
// term has been invoiced, since invoices already reference the old amounts.
func (uc *salesUseCase) SetBillingSchedule(ctx context.Context, orderID uuid.UUID, in []BillingTermInput) ([]domain.BillingTerm, error) {
	order, err := uc.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}
	if order.Status == "rejected" || order.Status == "cancelled" {
		return nil, apperrors.NewConflict("A " + order.Status + " order cannot be billed")
	}
	if len(in) == 0 || len(in) > maxBillingTerms {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Provide between 1 and %d billing terms", maxBillingTerms))
	}
	existing, err := uc.repo.ListBillingTerms(ctx, orderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load billing schedule")
	}
	for _, t := range existing {
		if t.Status == "invoiced" {
			return nil, apperrors.NewConflict("The schedule cannot change once a term has been invoiced")
		}
	}

	percents := make([]float64, len(in))
	var sum float64
	for i, t := range in {
		if t.Percent <= 0 || t.Percent > 100 {
			return nil, apperrors.NewBadRequest("Every term needs a percent between 0 and 100")
		}
		if t.DueDate != "" {
			if _, err := time.Parse("2006-01-02", t.DueDate); err != nil {
				return nil, apperrors.NewBadRequest("dueDate must be YYYY-MM-DD")
			}
		}
		if t.DueInDays < 0 {
			return nil, apperrors.NewBadRequest("dueInDays cannot be negative")
		}
		percents[i] = t.Percent
		sum += t.Percent
	}
	if math.Abs(sum-100) > 0.001 {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Term percentages must add up to 100 (currently %.3f)", sum))
	}

	total := order.BaseAmount
	if total <= 0 {
		total = order.TotalAmount
	}
	amounts := SplitByPercent(total, percents)
	terms := make([]domain.BillingTerm, len(in))
	for i, t := range in {
		due := t.DueDate
		if due == "" {
			due = time.Now().AddDate(0, 0, t.DueInDays).Format("2006-01-02")
		}
		label := strings.TrimSpace(t.Label)
		if label == "" {
			label = fmt.Sprintf("Termin %d", i+1)
		}
		terms[i] = domain.BillingTerm{SalesOrderID: orderID, Seq: i + 1, Label: label, Percent: t.Percent, Amount: amounts[i], DueDate: due, Status: "scheduled"}
	}
	if err := uc.repo.ReplaceBillingTerms(ctx, orderID, terms); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save billing schedule")
	}
	return uc.repo.ListBillingTerms(ctx, orderID)
}

func (uc *salesUseCase) ListBillingSchedule(ctx context.Context, orderID uuid.UUID) ([]domain.BillingTerm, error) {
	out, err := uc.repo.ListBillingTerms(ctx, orderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load billing schedule")
	}
	return out, nil
}

// InvoiceBillingTerm issues the sub-invoice for one term. Terms must be invoiced
// in order, so a later instalment cannot be billed ahead of an earlier one.
func (uc *salesUseCase) InvoiceBillingTerm(ctx context.Context, termID uuid.UUID) (*InvoiceResponseDTO, error) {
	term, err := uc.repo.GetBillingTerm(ctx, termID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get billing term")
	}
	if term == nil {
		return nil, apperrors.NewNotFound("Billing term not found")
	}
	if term.Status == "invoiced" {
		return nil, apperrors.NewConflict("This term has already been invoiced")
	}
	order, err := uc.repo.GetOrderByID(ctx, term.SalesOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get sales order")
	}
	if order == nil {
		return nil, apperrors.NewNotFound("Sales order not found")
	}
	if order.Status == "pending_approval" || order.Status == "rejected" || order.Status == "cancelled" {
		return nil, apperrors.NewConflict("An order that is " + order.Status + " cannot be billed")
	}
	all, err := uc.repo.ListBillingTerms(ctx, term.SalesOrderID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load billing schedule")
	}
	for _, t := range all {
		if t.Seq < term.Seq && t.Status != "invoiced" {
			return nil, apperrors.NewConflict(fmt.Sprintf("Invoice %q first", t.Label))
		}
	}

	inv, err := uc.CreateInvoice(ctx, CreateInvoiceDTO{
		InvoiceNumber: fmt.Sprintf("%s-T%d", strings.Replace(order.OrderNumber, "SO-", "INV-", 1), term.Seq),
		CustomerName:  order.CustomerName,
		TotalAmount:   term.Amount,
		DueDate:       term.DueDate,
		SalesOrderID:  &order.ID,
	})
	if err != nil {
		return nil, err
	}
	term.InvoiceID, term.Status = &inv.ID, "invoiced"
	if err := uc.repo.UpdateBillingTerm(ctx, term); err != nil {
		return inv, apperrors.NewInternal(err, "Invoice "+inv.InvoiceNumber+" was issued but the term could not be marked as invoiced")
	}
	return inv, nil
}
