package application

import apperrors "github.com/divinecoid/one-backend/internal/shared/errors"

// PPN (VAT) parameters. Since 2025 the statutory rate is 12%; for non-luxury
// goods and services the tax base (DPP) is the "nilai lain" of 11/12 of the
// price, which makes the effective PPN 11% of the price.
const (
	StatutoryVATRate      = 12.0
	OtherValueNumerator   = 11.0
	OtherValueDenominator = 12.0
	maxVATRate            = 100.0
	InputVATAccountName   = "PPN Masukan"
	OutputVATAccountName  = "PPN Keluaran"
)

// VATInput asks for PPN on top of an invoice. Rate 0 means the statutory rate.
type VATInput struct {
	Apply          bool
	Rate           float64
	OtherValueBase bool
}

// VATResult is the resolved PPN. TaxBase is the price-based DPP (after
// discount, including additional charges); DPPOtherValue is only set when the
// "nilai lain" base is used, and then PPN = DPPOtherValue x Rate.
type VATResult struct {
	Rate           float64 `json:"vatRate"`
	OtherValueBase bool    `json:"vatOtherValueBase"`
	TaxBase        float64 `json:"taxBase"`
	DPPOtherValue  float64 `json:"dppOtherValue"`
	Amount         float64 `json:"vatAmount"`
}

// ComputeVAT returns the PPN on a taxable base. With Apply false it returns the
// zero result carrying only the base, so callers can treat all invoices alike.
func ComputeVAT(taxBase float64, in VATInput) (VATResult, error) {
	res := VATResult{TaxBase: round2(taxBase)}
	if !in.Apply {
		return res, nil
	}
	if in.Rate < 0 || in.Rate > maxVATRate {
		return VATResult{}, apperrors.NewBadRequest("VAT rate must be between 0 and 100")
	}
	rate := in.Rate
	if rate == 0 {
		rate = StatutoryVATRate
	}
	basis := res.TaxBase
	if in.OtherValueBase {
		res.DPPOtherValue = round2(res.TaxBase * OtherValueNumerator / OtherValueDenominator)
		basis = res.DPPOtherValue
	}
	res.Rate, res.OtherValueBase = rate, in.OtherValueBase
	res.Amount = round2(basis * rate / 100)
	return res, nil
}
