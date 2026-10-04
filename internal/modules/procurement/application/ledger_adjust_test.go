package application

import (
	"testing"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
)

func TestPurchaseInvoiceLedgerBalancesAndCarriesInventoryAtCost(t *testing.T) {
	for _, in := range []financeApp.InvoiceAdjustmentInput{
		{Subtotal: 2_000_000},
		{Subtotal: 2_000_000, DiscountPercent: 5, AdditionalCost: 40_000},
		{Subtotal: 1_999_550, RoundTo: 1000},
		{Subtotal: 1_234_530, DiscountAmount: 4_530, RoundTo: 100},
	} {
		a, err := financeApp.ComputeInvoiceAmounts(in)
		if err != nil {
			t.Fatal(err)
		}
		inv := &domain.PurchaseInvoice{TotalAmount: a.Total, Subtotal: a.Subtotal, DiscountAmount: a.Discount, AdditionalCost: a.AdditionalCost, RoundingAmount: a.Rounding}
		e := PurchaseInvoiceLedgerEntry(inv)
		var dr, cr float64
		for _, l := range e.Lines {
			dr += l.Debit
			cr += l.Credit
		}
		if dr != cr {
			t.Fatalf("unbalanced for %+v: dr %v cr %v", in, dr, cr)
		}
		wantInv := a.Subtotal - a.Discount + a.AdditionalCost
		if e.Lines[0].Debit != wantInv {
			t.Fatalf("inventory %v want %v", e.Lines[0].Debit, wantInv)
		}
		if e.Lines[1].Credit != a.Total {
			t.Fatalf("payable %v want %v", e.Lines[1].Credit, a.Total)
		}
	}
}

func TestLegacyPurchaseInvoiceEntryUnchanged(t *testing.T) {
	e := PurchaseInvoiceLedgerEntry(&domain.PurchaseInvoice{TotalAmount: 700_000})
	if len(e.Lines) != 2 || e.Lines[0].Debit != 700_000 || e.Lines[1].Credit != 700_000 {
		t.Fatalf("legacy entry changed: %+v", e.Lines)
	}
}

func TestPurchaseInvoiceLedgerInputVAT(t *testing.T) {
	a, err := financeApp.ComputeInvoiceAmounts(financeApp.InvoiceAdjustmentInput{Subtotal: 2_000_000, VAT: financeApp.VATInput{Apply: true, OtherValueBase: true}})
	if err != nil {
		t.Fatal(err)
	}
	inv := &domain.PurchaseInvoice{TotalAmount: a.Total, Subtotal: a.Subtotal, TaxBase: a.VAT.TaxBase, VATAmount: a.VAT.Amount, VATCreditable: true}
	byAcct := func(e financeApp.LedgerEntry) map[string]float64 {
		dr, cr := 0.0, 0.0
		m := map[string]float64{}
		for _, l := range e.Lines {
			dr += l.Debit
			cr += l.Credit
			m[l.AccountCode] += l.Debit
		}
		if dr != cr {
			t.Fatalf("unbalanced: %+v", e.Lines)
		}
		return m
	}
	m := byAcct(PurchaseInvoiceLedgerEntry(inv))
	if m[financeApp.AccountInputVAT] != 220_000 || m[financeApp.AccountInventory] != 2_000_000 {
		t.Fatalf("creditable: %+v", m)
	}
	inv.VATCreditable = false // not backed by a valid faktur: the VAT becomes part of the cost
	m = byAcct(PurchaseInvoiceLedgerEntry(inv))
	if m[financeApp.AccountInputVAT] != 0 || m[financeApp.AccountInventory] != 2_220_000 {
		t.Fatalf("non-creditable: %+v", m)
	}
}
