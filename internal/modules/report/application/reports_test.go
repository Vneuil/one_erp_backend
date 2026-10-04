package application

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func f64(v float64) *float64 { return &v }

func TestAggregateByProductPrefersPOSCostAndRanks(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	lines := []salesLine{
		{OrderDate: "2026-03-01", ProductID: a, SKU: "A", Name: "Alpha", Category: "X", Quantity: 2, Subtotal: 200, CostPrice: 60},
		{OrderDate: "2026-03-02", ProductID: a, SKU: "A", Name: "Alpha", Category: "X", Quantity: 1, Subtotal: 100, CostPrice: 60, POSUnitCost: f64(50)},
		{OrderDate: "2026-03-02", ProductID: b, SKU: "B", Name: "Beta", Category: "Y", Quantity: 10, Subtotal: 1000, CostPrice: 0},
	}
	rows := aggregateByProduct(lines)
	if len(rows) != 2 || rows[0].Name != "Beta" || rows[0].Rank != 1 || rows[1].Rank != 2 {
		t.Fatalf("ranking: %+v", rows)
	}
	alpha := rows[1]
	// cost = 2*60 (master cost) + 1*50 (till snapshot)
	if alpha.Quantity != 3 || alpha.Revenue != 300 || alpha.Cost != 170 || alpha.GrossProfit != 130 || alpha.MarginPct != 43.33 || alpha.AvgPrice != 100 {
		t.Fatalf("alpha: %+v", alpha)
	}
	if rows[0].MarginPct != 100 {
		t.Fatalf("a product with no cost has a 100%% margin: %+v", rows[0])
	}
}

func TestDailyProductAndCategoryMargin(t *testing.T) {
	a := uuid.New()
	lines := []salesLine{
		{OrderDate: "2026-03-02", ProductID: a, Name: "Alpha", Category: "X", Quantity: 1, Subtotal: 100, CostPrice: 40},
		{OrderDate: "2026-03-02", ProductID: a, Name: "Alpha", Category: "X", Quantity: 2, Subtotal: 200, CostPrice: 40},
		{OrderDate: "2026-03-01", ProductID: a, Name: "Alpha", Category: "", Quantity: 1, Subtotal: 100, CostPrice: 40},
	}
	daily := aggregateDailyProduct(lines)
	if len(daily) != 2 || daily[0].Date != "2026-03-01" || daily[1].Quantity != 3 || daily[1].Revenue != 300 {
		t.Fatalf("daily: %+v", daily)
	}
	cats, total := marginByCategory(lines)
	if len(cats) != 2 || total.Revenue != 400 || total.Cost != 160 || total.MarginPct != 60 {
		t.Fatalf("categories: %+v total=%+v", cats, total)
	}
}

func TestAggregateByCustomerAndSummary(t *testing.T) {
	orders := []orderRow{
		{"2026-03-01", "PT Maju", 1000}, {"2026-03-03", "pt maju ", 500}, {"2026-03-03", "Toko Kecil", 2000}, {"2026-04-02", "Toko Kecil", 300},
	}
	lines := []salesLine{{CustomerName: "PT Maju", Subtotal: 1200, Quantity: 1, CostPrice: 300}}
	rows := aggregateByCustomer(orders, lines)
	if len(rows) != 2 || rows[0].Customer != "Toko Kecil" || rows[0].Revenue != 2300 || rows[0].Orders != 2 || rows[0].AvgOrder != 1150 {
		t.Fatalf("rows: %+v", rows)
	}
	maju := rows[1]
	if maju.Orders != 2 || maju.Revenue != 1500 || maju.FirstDate != "2026-03-01" || maju.LastDate != "2026-03-03" || maju.GrossProfit != 900 || maju.MarginPct != 75 {
		t.Fatalf("maju: %+v", maju)
	}

	s := summarizeSales("2026-03-01", "2026-04-04", orders)
	if s.Orders != 4 || s.Revenue != 3800 || s.AvgOrderValue != 950 || s.Days != 35 || s.ActiveDays != 3 || s.AvgPerActiveDay != 1266.67 || s.AvgPerDay != 108.57 {
		t.Fatalf("summary: %+v", s)
	}
	if len(s.Monthly) != 2 || s.Monthly[0].Period != "2026-03" || s.Monthly[0].Revenue != 3500 || s.Monthly[0].ActiveDays != 2 || s.Monthly[0].AvgPerDay != 1750 {
		t.Fatalf("monthly: %+v", s.Monthly)
	}
}

func arKind() partyKind {
	return partyKind{invoicePrefix: "sales-invoice:", paymentPrefix: "sales-invoice-payment:"}
}

func TestPartyLedgerAttributesPaymentsAndFlagsUndatedOnes(t *testing.T) {
	upd := time.Date(2026, 3, 20, 3, 0, 0, 0, time.UTC) // 10:00 WIB
	docs := []partyDoc{
		{SourceDoc: "sales-invoice:1", DocNo: "INV-1", Party: "PT Maju", IssueDate: "2026-02-10", DueDate: "2026-03-10", Total: 1000, Paid: 400, Updated: upd},
		{SourceDoc: "sales-invoice:2", DocNo: "INV-2", Party: "pt maju", IssueDate: "2026-03-05", DueDate: "2026-04-05", Total: 500, Paid: 0, Updated: upd},
		{SourceDoc: "", DocNo: "MAN-1", Party: "PT Maju", IssueDate: "2026-03-01", Total: 300, Paid: 100, Updated: upd},
		{SourceDoc: "sales-invoice:3", DocNo: "INV-3", Party: "Toko Kecil", IssueDate: "2026-03-02", Total: 200, Paid: 200, Updated: upd},
	}
	pays := []partyPayment{
		{Date: "2026-02-25", SourceDoc: "sales-invoice-payment:1:150.00", Amount: 150},
		{Date: "2026-03-12", SourceDoc: "sales-invoice-payment:1:400.00", Amount: 250},
		{Date: "2026-03-03", SourceDoc: "sales-invoice-payment:3:200.00", Amount: 200},
		{Date: "2026-03-03", SourceDoc: "sales-invoice-payment:unknown:10.00", Amount: 10},
	}
	entries := partyEntries(docs, pays, arKind())
	majuEntries := entries["pt maju"]
	var approx int
	for _, e := range majuEntries {
		if e.Approximate {
			approx++
			if e.Reference != "MAN-1" || e.Payment != 100 || e.Date != "2026-03-20" {
				t.Fatalf("approximate payment: %+v", e)
			}
		}
	}
	if len(majuEntries) != 6 || approx != 1 {
		t.Fatalf("maju entries: %d (approx %d) %+v", len(majuEntries), approx, majuEntries)
	}

	card := buildPartyCard("PT Maju", majuEntries, "2026-03-01", "2026-03-31")
	// Before March: INV-1 1000 charged, 150 paid => opening 850.
	if card.Opening != 850 || card.TotalCharges != 800 || card.TotalPayments != 350 || card.Closing != 1300 {
		t.Fatalf("card: %+v", card)
	}
	if len(card.Entries) != 4 || card.Entries[0].Reference != "MAN-1" || card.Entries[0].Balance != 1150 {
		t.Fatalf("entries: %+v", card.Entries)
	}
	last := card.Entries[len(card.Entries)-1]
	if last.Balance != card.Closing {
		t.Fatalf("running balance must end at the closing balance: %+v", last)
	}

	names := map[string]string{"pt maju": "PT Maju", "toko kecil": "Toko Kecil"}
	bal := buildPartyBalances(names, entries, "2026-03-31", false)
	if len(bal.Rows) != 1 || bal.Rows[0].Party != "PT Maju" || bal.Rows[0].Balance != 1300 || bal.Rows[0].OpenDocuments != 3 || bal.Rows[0].OldestDue != "2026-03-10" {
		t.Fatalf("balances: %+v", bal)
	}
	// As of 2026-02-28 only INV-1 existed, with 150 paid.
	early := buildPartyBalances(names, entries, "2026-02-28", false)
	if len(early.Rows) != 1 || early.Rows[0].Balance != 850 || early.TotalBalance != 850 {
		t.Fatalf("early balances: %+v", early)
	}
	if all := buildPartyBalances(names, entries, "2026-03-31", true); len(all.Rows) != 2 {
		t.Fatalf("includeZero: %+v", all.Rows)
	}
}

func TestPartyCardWithNoActivityInRangeKeepsOpening(t *testing.T) {
	docs := []partyDoc{{SourceDoc: "sales-invoice:1", DocNo: "INV-1", Party: "PT Maju", IssueDate: "2026-01-10", Total: 1000}}
	card := buildPartyCard("PT Maju", partyEntries(docs, nil, arKind())["pt maju"], "2026-03-01", "2026-03-31")
	if card.Opening != 1000 || card.Closing != 1000 || len(card.Entries) != 0 {
		t.Fatalf("card: %+v", card)
	}
}

func mv(day string, hour int, wh uuid.UUID, typ string, qty, bal int) stockMovementRow {
	d, _ := time.ParseInLocation("2006-01-02", day, wib)
	return stockMovementRow{CreatedAt: d.Add(time.Duration(hour) * time.Hour).UTC(), WarehouseID: wh, WarehouseName: "WH", Type: typ, Quantity: qty, Balance: bal}
}

func TestStockCardOpeningAcrossWarehouses(t *testing.T) {
	w1, w2 := uuid.New(), uuid.New()
	ms := []stockMovementRow{
		mv("2026-02-10", 9, w1, "in", 100, 100),
		mv("2026-02-11", 9, w2, "in", 50, 50),
		mv("2026-03-02", 9, w1, "out", -30, 70),
		mv("2026-03-02", 23, w2, "transfer_out", -20, 30), // 23:00 WIB is still 2 March in WIB
		mv("2026-03-03", 9, w1, "transfer_in", 20, 90),
		mv("2026-04-01", 9, w1, "out", -5, 85), // outside the range
	}
	card := buildStockCard(ms[:5], "2026-03-01", "2026-03-31")
	if card.Opening != 150 || card.TotalIn != 20 || card.TotalOut != 50 || card.Closing != 120 || len(card.Entries) != 3 {
		t.Fatalf("card: %+v", card)
	}
	if card.Entries[0].Out != 30 || card.Entries[0].Balance != 120 || card.Entries[2].In != 20 || card.Entries[2].Balance != 120 {
		t.Fatalf("entries: %+v", card.Entries)
	}
	// Single warehouse: opening is that warehouse's own last balance.
	one := buildStockCard([]stockMovementRow{ms[0], ms[2], ms[4]}, "2026-03-01", "2026-03-31")
	if one.Opening != 100 || one.Closing != 90 {
		t.Fatalf("single warehouse: %+v", one)
	}
}

func TestStockTransactionsTotalsByType(t *testing.T) {
	w := uuid.New()
	a := mv("2026-03-02", 9, w, "in", 10, 10)
	a.CostPrice = 5
	b := mv("2026-03-03", 9, w, "out", -4, 6)
	b.CostPrice = 5
	out := buildStockTransactions("2026-03-01", "2026-03-31", []stockMovementRow{a, b}, false)
	if len(out.Rows) != 2 || out.Rows[1].Value != -20 || len(out.ByType) != 2 || out.ByType[0].Type != "in" || out.ByType[0].Value != 50 {
		t.Fatalf("transactions: %+v", out)
	}
}

func TestStockAgingBucketsUntrackedAndExpired(t *testing.T) {
	p, w := uuid.New(), uuid.New()
	levels := []stockLevelRow{{ProductID: p, SKU: "A", ProductName: "Alpha", CostPrice: 10, WarehouseID: w, WarehouseName: "Main", Quantity: 100}}
	batches := []stockBatchRow{
		{ProductID: p, SKU: "A", ProductName: "Alpha", CostPrice: 10, WarehouseID: w, WarehouseName: "Main", BatchNo: "B1", ReceivedAt: "2026-03-10", Quantity: 10},
		{ProductID: p, SKU: "A", ProductName: "Alpha", CostPrice: 10, WarehouseID: w, WarehouseName: "Main", BatchNo: "B2", ReceivedAt: "2026-02-01", ExpiryDate: "2026-03-01", Quantity: 20},
		{ProductID: p, SKU: "A", ProductName: "Alpha", CostPrice: 10, WarehouseID: w, WarehouseName: "Main", BatchNo: "B3", ReceivedAt: "2025-10-01", Quantity: 30},
	}
	out := buildStockAging("2026-03-31", levels, batches)
	if len(out.Rows) != 1 {
		t.Fatalf("rows: %+v", out.Rows)
	}
	r := out.Rows[0]
	// B1 is 21 days old (0-30), B2 58 days (31-60), B3 181 days (>90); 40 units have no batch.
	if r.Buckets[0] != 10 || r.Buckets[1] != 20 || r.Buckets[2] != 0 || r.Buckets[3] != 30 || r.Untracked != 40 || r.Total != 100 || r.Value != 1000 || r.Expired != 20 || r.OldestDays != 181 {
		t.Fatalf("aging row: %+v", r)
	}
	if out.TotalQuantity != 100 || out.TotalValue != 1000 || out.UntrackedQty != 40 || out.BucketTotals[3] != 30 || out.BucketValues[3] != 300 {
		t.Fatalf("totals: %+v", out)
	}
}

func TestPurchaseReportBySupplier(t *testing.T) {
	inv := []purchaseInvoiceRow{
		{InvoiceNumber: "P1", SupplierName: "PT Pasok", InvoiceDate: "2026-03-01", Subtotal: 1000, DiscountAmount: 100, VATAmount: 99, TotalAmount: 999, PaidAmount: 500},
		{InvoiceNumber: "P2", SupplierName: "pt pasok", InvoiceDate: "2026-03-02", TotalAmount: 200}, // legacy: no subtotal
		{InvoiceNumber: "P3", SupplierName: "CV Lain", InvoiceDate: "2026-03-03", Subtotal: 5000, TotalAmount: 5000, PaidAmount: 5000},
	}
	a := uuid.New()
	out := buildPurchaseReport("2026-03-01", "2026-03-31", inv, []purchaseLineRow{{ProductID: a, Name: "Alpha", Quantity: 4, Subtotal: 400}, {ProductID: a, Name: "Alpha", Quantity: 1, Subtotal: 100}})
	if out.Total != 6199 || out.Paid != 5500 || out.Outstanding != 699 || out.VAT != 99 || len(out.BySupplier) != 2 {
		t.Fatalf("totals: %+v", out)
	}
	s := out.BySupplier[1]
	if s.Supplier != "PT Pasok" || s.Invoices != 2 || s.Subtotal != 1200 || s.Total != 1199 || s.Outstanding != 699 || out.BySupplier[0].Rank != 1 {
		t.Fatalf("supplier: %+v / %+v", s, out.BySupplier)
	}
	if len(out.ByProduct) != 1 || out.ByProduct[0].Quantity != 5 || out.ByProduct[0].AvgPrice != 100 {
		t.Fatalf("products: %+v", out.ByProduct)
	}
}

func TestResolveRange(t *testing.T) {
	from, to, err := resolveRange("", "")
	if err != nil || from != monthStart() || to != today() {
		t.Fatalf("default: %s %s %v", from, to, err)
	}
	if f, tt, _ := resolveRange("", "2026-03-15"); f != "2026-03-01" || tt != "2026-03-15" {
		t.Fatalf("missing start: %s %s", f, tt)
	}
	for _, bad := range [][2]string{{"2026-03-10", "2026-03-01"}, {"x", ""}, {"", "2026-13-01"}} {
		if _, _, err := resolveRange(bad[0], bad[1]); err == nil {
			t.Fatalf("%v should fail", bad)
		}
	}
}
