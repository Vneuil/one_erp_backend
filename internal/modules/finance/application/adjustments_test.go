package application

import "testing"

// amts builds the expected breakdown of an invoice without PPN.
func amts(sub, disc, add, rounding, total float64) InvoiceAmounts {
	return InvoiceAmounts{Subtotal: sub, Discount: disc, AdditionalCost: add, Rounding: rounding, Total: total, VAT: VATResult{TaxBase: sub - disc + add}}
}

func TestComputeInvoiceAmounts(t *testing.T) {
	cases := []struct {
		name string
		in   InvoiceAdjustmentInput
		want InvoiceAmounts
	}{
		{"no adjustments keeps the original behaviour", InvoiceAdjustmentInput{Subtotal: 1_000_000}, amts(1_000_000, 0, 0, 0, 1_000_000)},
		{"percent discount", InvoiceAdjustmentInput{Subtotal: 1_000_000, DiscountPercent: 10}, amts(1_000_000, 100_000, 0, 0, 900_000)},
		{"amount discount plus shipping", InvoiceAdjustmentInput{Subtotal: 1_000_000, DiscountAmount: 50_000, AdditionalCost: 25_000}, amts(1_000_000, 50_000, 25_000, 0, 975_000)},
		{"round up to 1000", InvoiceAdjustmentInput{Subtotal: 999_400, AdditionalCost: 300, RoundTo: 1000}, amts(999_400, 0, 300, 300, 1_000_000)},
		{"round down to 100", InvoiceAdjustmentInput{Subtotal: 1_234_530, RoundTo: 100}, amts(1_234_530, 0, 0, -30, 1_234_500)},
	}
	for _, c := range cases {
		got, err := ComputeInvoiceAmounts(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
		if got.Subtotal-got.Discount+got.AdditionalCost+got.VAT.Amount+got.Rounding != got.Total {
			t.Errorf("%s: components do not add up to the total", c.name)
		}
	}
}

func TestComputeInvoiceAmountsRejectsBadInput(t *testing.T) {
	bad := []InvoiceAdjustmentInput{
		{Subtotal: 0},
		{Subtotal: 100, DiscountPercent: 101},
		{Subtotal: 100, DiscountPercent: 5, DiscountAmount: 5},
		{Subtotal: 100, DiscountAmount: 150},
		{Subtotal: 100, AdditionalCost: -1},
		{Subtotal: 100, RoundTo: -100},
	}
	for i, in := range bad {
		if _, err := ComputeInvoiceAmounts(in); err == nil {
			t.Errorf("case %d should be rejected: %+v", i, in)
		}
	}
}

func TestComputeInvoiceAmountsWithVAT(t *testing.T) {
	// Effective 11%: 12% on the 11/12 "nilai lain" base.
	got, err := ComputeInvoiceAmounts(InvoiceAdjustmentInput{Subtotal: 1_000_000, DiscountAmount: 100_000, AdditionalCost: 50_000,
		VAT: VATInput{Apply: true, OtherValueBase: true}})
	if err != nil {
		t.Fatal(err)
	}
	v := got.VAT
	if v.TaxBase != 950_000 || v.DPPOtherValue != 870_833.33 || v.Rate != 12 || v.Amount != 104_500 || got.Total != 1_054_500 {
		t.Fatalf("unexpected VAT breakdown: %+v total=%v", v, got.Total)
	}

	// Full-base 12%, then rounding applies to the VAT-inclusive total.
	got, _ = ComputeInvoiceAmounts(InvoiceAdjustmentInput{Subtotal: 1_000_000, RoundTo: 1000, VAT: VATInput{Apply: true, Rate: 11}})
	if got.VAT.Amount != 110_000 || got.VAT.DPPOtherValue != 0 || got.Total != 1_110_000 || got.Rounding != 0 {
		t.Fatalf("unexpected: %+v", got)
	}
	got, _ = ComputeInvoiceAmounts(InvoiceAdjustmentInput{Subtotal: 333_333, RoundTo: 1000, VAT: VATInput{Apply: true, OtherValueBase: true}})
	if int64(got.Total)%1000 != 0 {
		t.Fatalf("total not rounded: %+v", got)
	}
	if got.Subtotal-got.Discount+got.AdditionalCost+got.VAT.Amount+got.Rounding != got.Total {
		t.Fatalf("components do not add up: %+v", got)
	}

	if _, err := ComputeInvoiceAmounts(InvoiceAdjustmentInput{Subtotal: 100, VAT: VATInput{Apply: true, Rate: 150}}); err == nil {
		t.Fatal("rate above 100 must be rejected")
	}
}
