package application

import (
	"testing"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
)

func balanced(t *testing.T, lines []financeApp.LedgerLine) (dr, cr float64) {
	t.Helper()
	for _, l := range lines {
		dr += l.Debit
		cr += l.Credit
	}
	if dr != cr {
		t.Fatalf("unbalanced: dr %v cr %v (%+v)", dr, cr, lines)
	}
	return
}

func TestInvoiceLedgerBalancesForEveryAdjustmentMix(t *testing.T) {
	for _, in := range []financeApp.InvoiceAdjustmentInput{
		{Subtotal: 1_000_000},
		{Subtotal: 1_000_000, DiscountPercent: 12.5},
		{Subtotal: 999_400, AdditionalCost: 300, RoundTo: 1000},
		{Subtotal: 1_234_530, DiscountAmount: 4_530, AdditionalCost: 75_000, RoundTo: 100},
		{Subtotal: 850_000, RoundTo: 1000},
	} {
		a, err := financeApp.ComputeInvoiceAmounts(in)
		if err != nil {
			t.Fatal(err)
		}
		inv := &domain.Invoice{TotalAmount: a.Total, Subtotal: a.Subtotal, DiscountAmount: a.Discount, AdditionalCost: a.AdditionalCost, RoundingAmount: a.Rounding}
		e := InvoiceLedgerEntry(inv)
		balanced(t, e.Lines)
		// Receivable always equals what the customer owes.
		if e.Lines[0].Debit != a.Total {
			t.Fatalf("receivable %v != total %v", e.Lines[0].Debit, a.Total)
		}
	}
}

func TestLegacyInvoiceWithoutAdjustmentColumnsStillPostsPlainEntry(t *testing.T) {
	inv := &domain.Invoice{TotalAmount: 500_000} // pre-migration row: all adjustment columns zero
	e := InvoiceLedgerEntry(inv)
	dr, _ := balanced(t, e.Lines)
	if dr != 500_000 || e.Lines[2].Credit != 500_000 {
		t.Fatalf("legacy entry changed: %+v", e.Lines)
	}
}

func TestInvoiceLedgerWithVATPostsPPNKeluaran(t *testing.T) {
	a, err := financeApp.ComputeInvoiceAmounts(financeApp.InvoiceAdjustmentInput{Subtotal: 1_000_000, DiscountAmount: 100_000, AdditionalCost: 50_000, RoundTo: 1000,
		VAT: financeApp.VATInput{Apply: true, OtherValueBase: true}})
	if err != nil {
		t.Fatal(err)
	}
	inv := &domain.Invoice{TotalAmount: a.Total, Subtotal: a.Subtotal, DiscountAmount: a.Discount, AdditionalCost: a.AdditionalCost,
		RoundingAmount: a.Rounding, TaxBase: a.VAT.TaxBase, VATAmount: a.VAT.Amount}
	e := InvoiceLedgerEntry(inv)
	balanced(t, e.Lines)
	var ppn, revenue float64
	for _, l := range e.Lines {
		switch l.AccountCode {
		case financeApp.AccountSalesTaxPayable:
			ppn += l.Credit
		case financeApp.AccountRevenue:
			revenue += l.Credit
		}
	}
	if ppn != 104_500 || revenue != 1_000_000 {
		t.Fatalf("ppn=%v revenue=%v, want 104500 / 1000000", ppn, revenue)
	}
}
