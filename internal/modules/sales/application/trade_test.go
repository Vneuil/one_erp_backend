package application

import (
	"testing"

	"github.com/google/uuid"

	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
)

func TestTradeTotalsVATOnAndExempt(t *testing.T) {
	inv := &domain.TradeInvoice{VATRate: DefaultTradeVATRate, Lines: []domain.TradeInvoiceLine{{Subtotal: 1000}, {Subtotal: 500}}}
	computeTradeTotals(inv)
	if inv.Subtotal != 1500 || inv.VATAmount != 165 || inv.Total != 1665 {
		t.Fatalf("PPN 11%% totals wrong: %+v", inv)
	}
	inv.VATExempt = true
	computeTradeTotals(inv)
	if inv.VATAmount != 0 || inv.Total != 1500 {
		t.Fatalf("export (exempt) must carry no PPN: %+v", inv)
	}
}

func TestLinesChanged(t *testing.T) {
	a := []domain.SalesOrderLine{{Quantity: 1, UnitPrice: 10}}
	if linesChanged(a, []domain.SalesOrderLine{{Quantity: 1, UnitPrice: 10}}) {
		t.Fatal("identical lines are not a revision")
	}
	if !linesChanged(a, []domain.SalesOrderLine{{Quantity: 2, UnitPrice: 10}}) {
		t.Fatal("changed quantity is a revision")
	}
}

func TestNormalizeState(t *testing.T) {
	if v, err := normalizeState("x", "", "pending", packagingStates); err != nil || v != "pending" {
		t.Fatal("empty falls back to default")
	}
	if _, err := normalizeState("x", "bogus", "pending", packagingStates); err == nil {
		t.Fatal("unknown state must be rejected")
	}
}

func TestSubmitNeedsDocumentOrDP(t *testing.T) {
	o := &domain.SalesOrder{Channel: "Direct B2B"}
	if checkSubmitDocuments(o) == nil {
		t.Fatal("no PO/PI document and no DP must block submit")
	}
	o.CustomerPOAttachment = "https://x/po.pdf"
	if err := checkSubmitDocuments(o); err != nil {
		t.Fatal(err)
	}
	o = &domain.SalesOrder{Channel: "Direct B2B", DPPaid: true}
	if err := checkSubmitDocuments(o); err != nil {
		t.Fatal("DP received satisfies the rule")
	}
	if err := checkSubmitDocuments(&domain.SalesOrder{Channel: "POS"}); err != nil {
		t.Fatal("POS orders are exempt")
	}
}

func TestOrderFromPI(t *testing.T) {
	pid := uuid.New()
	pi := &domain.TradeInvoice{Kind: domain.TradeKindPI, Status: domain.TradeStatusDraft, Number: "PI-1", CustomerName: "Perusahaan A",
		Total: 1110, DPPercent: 50, DPAmount: 555, CustomerPONumber: "PO-018", VATExempt: true,
		Lines: []domain.TradeInvoiceLine{{ProductID: &pid, Quantity: 100, UnitPrice: 10}}}
	if _, err := OrderFromPI(pi, OrderFromPIDTO{}); err == nil {
		t.Fatal("a draft PI cannot become a Sales Order")
	}
	pi.Status = domain.TradeStatusIssued
	// DP menyusul: the order is created with DP not yet received.
	so, err := OrderFromPI(pi, OrderFromPIDTO{DPPaid: false})
	if err != nil {
		t.Fatal(err)
	}
	if so.DPPaid || so.DPPercent != 50 || so.DPAmount != 555 || so.CustomerPONumber != "PO-018" || !so.VATExempt || len(so.Lines) != 1 || so.Lines[0].UnitPrice != 10 {
		t.Fatalf("order must mirror the PI: %+v", so)
	}
	linked := uuid.New()
	pi.SalesOrderID = &linked
	if _, err := OrderFromPI(pi, OrderFromPIDTO{}); err == nil {
		t.Fatal("a PI can only produce one Sales Order")
	}
	pi.SalesOrderID = nil
	pi.Lines[0].ProductID = nil
	if _, err := OrderFromPI(pi, OrderFromPIDTO{}); err == nil {
		t.Fatal("free-text lines cannot become order lines")
	}
}
