package application

import (
	"context"
	"testing"

	"github.com/google/uuid"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
)

func sumSide(lines []financeApp.LedgerLine) (dr, cr float64) {
	for _, l := range lines {
		dr += l.Debit
		cr += l.Credit
	}
	return
}

// A step costs standard hours x rate: Potong 6 min (labour 30,000/h, overhead 10,000/h) = 3,000 + 1,000 per unit;
// Jahit 30 min (20,000/h, 10,000/h) = 10,000 + 5,000 per unit. Conversion per finished unit: 19,000.
func costedFixture(t *testing.T) *fixture {
	f := newFixture(t)
	b, err := f.uc.CreateBOM(context.Background(), CreateBOMDTO{ProductID: f.fg, Name: "Kemeja costed",
		Lines: []BOMLineDTO{{ComponentProductID: f.rm1, QuantityRequired: 2}, {ComponentProductID: f.rm2, QuantityRequired: 1}},
		Processes: []BOMProcessDTO{
			{Name: "Potong", StandardMinutes: 6, LaborRatePerHour: 30_000, OverheadRatePerHour: 10_000},
			{Name: "Jahit", StandardMinutes: 30, LaborRatePerHour: 20_000, OverheadRatePerHour: 10_000}}})
	if err != nil {
		t.Fatal(err)
	}
	f.bom = f.repo.boms[b.ID]
	return f
}

func TestBOMRejectsRatesWithoutStandardTime(t *testing.T) {
	f := newFixture(t)
	bad := []BOMProcessDTO{{Name: "Potong", LaborRatePerHour: 30_000}}
	if _, err := f.uc.CreateBOM(context.Background(), CreateBOMDTO{ProductID: f.fg, Name: "x", Lines: []BOMLineDTO{{ComponentProductID: f.rm1, QuantityRequired: 1}}, Processes: bad}); err == nil {
		t.Fatal("a rate without a standard time has nothing to multiply")
	}
	bad = []BOMProcessDTO{{Name: "Potong", StandardMinutes: 5, OverheadRatePerHour: -1}}
	if _, err := f.uc.CreateBOM(context.Background(), CreateBOMDTO{ProductID: f.fg, Name: "x", Lines: []BOMLineDTO{{ComponentProductID: f.rm1, QuantityRequired: 1}}, Processes: bad}); err == nil {
		t.Fatal("negative rates must be refused")
	}
}

func TestStepLogChargesLabourAndOverheadToWIPAndBatchRelievesIt(t *testing.T) {
	f := costedFixture(t)
	ctx := context.Background()
	o := f.newOrder(t, 10, "")
	f.release(o)
	if o.Steps[0].LaborRatePerHour != 30_000 {
		t.Fatalf("the order snapshots the BOM rates: %+v", o.Steps[0])
	}
	if err := f.log(t, o, 0, 10); err != nil {
		t.Fatal(err)
	}
	if err := f.log(t, o, 1, 10); err != nil {
		t.Fatal(err)
	}
	var wipDr, labor, overhead float64
	for _, lines := range f.ledger.entries {
		dr, cr := sumSide(lines)
		if dr != cr {
			t.Fatalf("unbalanced entry: %+v", lines)
		}
		for _, l := range lines {
			switch l.AccountCode {
			case financeApp.AccountWIP:
				wipDr += l.Debit
			case financeApp.AccountLaborApplied:
				labor += l.Credit
			case financeApp.AccountOverheadApplied:
				overhead += l.Credit
			}
		}
	}
	// Potong 10 units: 30,000 + 10,000. Jahit 10 units: 100,000 + 50,000.
	if wipDr != 190_000 || labor != 130_000 || overhead != 60_000 || len(f.ledger.entries) != 2 {
		t.Fatalf("wip %v labour %v overhead %v entries %d", wipDr, labor, overhead, len(f.ledger.entries))
	}

	orders, _, _ := f.uc.loadReportOrders(ctx)
	in := wipInputs{costPerUnit: map[uuid.UUID]float64{f.bom.ID: 2500}}
	wip := buildWIPReport(orders, in)
	// 10 units past step 1, none finished: material 25,000 + labour 130,000 + overhead 60,000.
	if len(wip.Orders) != 1 || wip.Orders[0].MaterialValue != 25_000 || wip.Orders[0].LaborValue != 130_000 || wip.Orders[0].OverheadValue != 60_000 || wip.TotalValue != 215_000 {
		t.Fatalf("wip before completion: %+v", wip)
	}

	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 4}); err != nil {
		t.Fatal(err)
	}
	// Backflush mode: only the 4 units' conversion (4 x 19,000) leaves WIP for Inventory; materials never entered WIP.
	var fg []financeApp.LedgerLine
	for src, lines := range f.ledger.entries {
		if len(src) > 17 && src[:17] == "production-batch:" {
			fg = lines
		}
	}
	if dr, cr := sumSide(fg); dr != 76_000 || cr != 76_000 {
		t.Fatalf("finished goods entry: %+v", fg)
	}
	// The finished product now carries materials (2,500) plus conversion (19,000) as its cost: moving average over 0 on hand + 4 new.
	if p, _ := f.uc.productRepo.GetByID(ctx, f.fg); p.CostPrice != 21_500 {
		t.Fatalf("finished product cost price: %v", p.CostPrice)
	}
	orders, _, _ = f.uc.loadReportOrders(ctx)
	wip = buildWIPReport(orders, in)
	// 6 units remain in process: labour 130,000 - 4 x 13,000, overhead 60,000 - 4 x 6,000.
	if wip.Orders[0].LaborValue != 78_000 || wip.Orders[0].OverheadValue != 36_000 || wip.TotalLabor != 78_000 || wip.TotalOverhead != 36_000 {
		t.Fatalf("wip after completion: %+v", wip)
	}
}

func TestOrdersWithoutRatesPostNothingExtra(t *testing.T) {
	f := newFixture(t)
	o := f.newOrder(t, 5, "")
	f.release(o)
	if err := f.log(t, o, 0, 5); err != nil {
		t.Fatal(err)
	}
	if len(f.ledger.entries) != 0 {
		t.Fatalf("no rates, no conversion entry: %+v", f.ledger.entries)
	}
}

var _ = domain.OrderReleased
