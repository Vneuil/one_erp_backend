package application

import "testing"

func TestComputeInvoiceAmounts(t *testing.T) {
	cases := []struct {
		name string
		in   InvoiceAdjustmentInput
		want InvoiceAmounts
	}{
		{"no adjustments keeps the original behaviour", InvoiceAdjustmentInput{Subtotal: 1_000_000}, InvoiceAmounts{1_000_000, 0, 0, 0, 1_000_000}},
		{"percent discount", InvoiceAdjustmentInput{Subtotal: 1_000_000, DiscountPercent: 10}, InvoiceAmounts{1_000_000, 100_000, 0, 0, 900_000}},
		{"amount discount plus shipping", InvoiceAdjustmentInput{Subtotal: 1_000_000, DiscountAmount: 50_000, AdditionalCost: 25_000}, InvoiceAmounts{1_000_000, 50_000, 25_000, 0, 975_000}},
		{"round up to 1000", InvoiceAdjustmentInput{Subtotal: 999_400, AdditionalCost: 300, RoundTo: 1000}, InvoiceAmounts{999_400, 0, 300, 300, 1_000_000}},
		{"round down to 100", InvoiceAdjustmentInput{Subtotal: 1_234_530, RoundTo: 100}, InvoiceAmounts{1_234_530, 0, 0, -30, 1_234_500}},
	}
	for _, c := range cases {
		got, err := ComputeInvoiceAmounts(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: got %+v want %+v", c.name, got, c.want)
		}
		if got.Subtotal-got.Discount+got.AdditionalCost+got.Rounding != got.Total {
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
