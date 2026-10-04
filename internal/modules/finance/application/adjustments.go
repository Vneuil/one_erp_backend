package application

import (
	"math"

	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
)

// InvoiceAdjustmentInput is what a caller supplies on top of the subtotal.
// DiscountPercent and DiscountAmount are alternatives (supplying both is an
// error); RoundTo is the step the final total is rounded to (0 = no rounding,
// e.g. 100 rounds to the nearest Rp100).
type InvoiceAdjustmentInput struct {
	Subtotal        float64
	DiscountPercent float64
	DiscountAmount  float64
	AdditionalCost  float64
	RoundTo         float64
	// VAT optionally adds PPN on (subtotal - discount + additional cost).
	VAT VATInput
}

// InvoiceAmounts is the resolved breakdown. Total = Subtotal - Discount +
// AdditionalCost + VATAmount + Rounding; Rounding is signed (negative when
// rounded down). The VAT fields are zero for invoices without PPN.
type InvoiceAmounts struct {
	Subtotal       float64   `json:"subtotal"`
	Discount       float64   `json:"discountAmount"`
	AdditionalCost float64   `json:"additionalCost"`
	Rounding       float64   `json:"roundingAmount"`
	Total          float64   `json:"totalAmount"`
	VAT            VATResult `json:"vat"`
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ComputeInvoiceAmounts resolves discount, additional cost and rounding into a
// final invoice total, validating the inputs.
func ComputeInvoiceAmounts(in InvoiceAdjustmentInput) (InvoiceAmounts, error) {
	if in.Subtotal <= 0 {
		return InvoiceAmounts{}, apperrors.NewBadRequest("A positive amount is required")
	}
	if in.DiscountPercent < 0 || in.DiscountPercent > 100 || in.DiscountAmount < 0 || in.AdditionalCost < 0 || in.RoundTo < 0 {
		return InvoiceAmounts{}, apperrors.NewBadRequest("Discount, additional cost and rounding step cannot be negative, and a discount percent is at most 100")
	}
	if in.DiscountPercent > 0 && in.DiscountAmount > 0 {
		return InvoiceAmounts{}, apperrors.NewBadRequest("Give either a discount percent or a discount amount, not both")
	}
	discount := in.DiscountAmount
	if in.DiscountPercent > 0 {
		discount = round2(in.Subtotal * in.DiscountPercent / 100)
	}
	if discount > in.Subtotal {
		return InvoiceAmounts{}, apperrors.NewBadRequest("The discount cannot exceed the invoice amount")
	}
	vat, err := ComputeVAT(round2(in.Subtotal-discount+in.AdditionalCost), in.VAT)
	if err != nil {
		return InvoiceAmounts{}, err
	}
	total := round2(vat.TaxBase + vat.Amount)
	var rounding float64
	if in.RoundTo > 0 {
		rounded := math.Round(total/in.RoundTo) * in.RoundTo
		rounding = round2(rounded - total)
		total = round2(total + rounding)
	}
	if total <= 0 {
		return InvoiceAmounts{}, apperrors.NewBadRequest("The invoice total must stay positive after adjustments")
	}
	return InvoiceAmounts{Subtotal: round2(in.Subtotal), Discount: discount, AdditionalCost: round2(in.AdditionalCost), Rounding: rounding, Total: total, VAT: vat}, nil
}
