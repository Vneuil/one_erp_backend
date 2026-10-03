package application

import (
	"math"

	"github.com/divinecoid/one-backend/internal/modules/pos/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// SaleTotals is a sale's resolved amounts. Total is what the customer pays.
type SaleTotals struct {
	Subtotal float64
	Discount float64
	Tax      float64
	Total    float64
}

// ComputeSale resolves a cart into totals under the company's tax settings.
// Discount is taken off the subtotal first. With tax-inclusive pricing the tax is
// carved out of the discounted amount (total unchanged); otherwise it is added
// on top. RoundTo rounds the payable total to that step.
func ComputeSale(subtotal, discount float64, s domain.POSSettings) (SaleTotals, error) {
	if subtotal <= 0 {
		return SaleTotals{}, apperrors.NewBadRequest("The cart is empty")
	}
	if discount < 0 || discount > subtotal {
		return SaleTotals{}, apperrors.NewBadRequest("The discount must be between 0 and the subtotal")
	}
	if s.TaxPercent < 0 || s.TaxPercent > 100 {
		return SaleTotals{}, apperrors.NewBadRequest("Invalid tax rate in POS settings")
	}
	net := round2(subtotal - discount)
	rate := s.TaxPercent / 100
	var tax, total float64
	if s.TaxInclusive {
		total = net
		tax = round2(net - net/(1+rate))
	} else {
		tax = round2(net * rate)
		total = round2(net + tax)
	}
	if s.RoundTo > 0 {
		total = round2(math.Round(total/s.RoundTo) * s.RoundTo)
	}
	if total <= 0 {
		return SaleTotals{}, apperrors.NewBadRequest("The total must be positive")
	}
	return SaleTotals{Subtotal: round2(subtotal), Discount: round2(discount), Tax: tax, Total: total}, nil
}

// RefundShare is what handing back `qty` of one line returns to the customer.
type RefundShare struct {
	Amount float64 // total returned, tax included
	Tax    float64
	Cost   float64 // cost of goods coming back into stock
}

// RefundFor prices a line refund proportionally: the line's price share of the
// discount is given back too, and the tax on it follows the sale's own tax
// ratio, so refunding every unit returns (almost exactly) the sale's total.
func RefundFor(tx *domain.POSTransaction, l domain.POSTransactionLine, qty float64) RefundShare {
	if tx.Subtotal <= 0 || qty <= 0 {
		return RefundShare{}
	}
	gross := l.UnitPrice * qty
	// Share of the sale's payable total this line represents.
	share := gross / tx.Subtotal
	amount := round2(tx.TotalAmount * share)
	tax := round2(tx.TaxAmount * share)
	return RefundShare{Amount: amount, Tax: tax, Cost: round2(l.UnitCost * qty)}
}
