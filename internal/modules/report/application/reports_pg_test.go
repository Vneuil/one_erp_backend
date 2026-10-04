package application

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeDomain "github.com/divinecoid/one-backend/internal/modules/finance/domain"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	posDomain "github.com/divinecoid/one-backend/internal/modules/pos/domain"
	procApp "github.com/divinecoid/one-backend/internal/modules/procurement/application"
	procDomain "github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	salesApp "github.com/divinecoid/one-backend/internal/modules/sales/application"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when REPORT_TEST_DSN is set (point it at an EMPTY scratch database).
func TestReportsAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("REPORT_TEST_DSN")
	if dsn == "" {
		t.Skip("REPORT_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&financeDomain.Account{}, &financeDomain.JournalEntry{}, &financeDomain.JournalLine{}, &financeDomain.Receivable{}, &financeDomain.Payable{},
		&salesDomain.SalesOrder{}, &salesDomain.SalesOrderLine{}, &salesDomain.Invoice{}, &posDomain.POSTransaction{}, &posDomain.POSTransactionLine{},
		&productDomain.Product{}, &inventoryDomain.Warehouse{}, &inventoryDomain.StockLevel{}, &inventoryDomain.StockMovement{}, &inventoryDomain.StockBatch{},
		&procDomain.PurchaseInvoice{}, &procDomain.PurchaseOrder{}, &procDomain.PurchaseOrderLine{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	q := NewReportQuery(db)
	repo := financeInfra.NewFinanceRepository(db)
	if err := financeApp.NewFinanceUseCase(repo).SeedInitialData(ctx); err != nil {
		t.Fatal(err)
	}
	ledger := financeApp.NewLedgerPoster(repo)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	create := func(v any) {
		t.Helper()
		must(db.Create(v).Error)
	}

	at := func(day string, hour int) time.Time {
		d, _ := time.ParseInLocation("2006-01-02", day, wib)
		return d.Add(time.Duration(hour) * time.Hour).UTC()
	}

	// Products, orders and a POS sale.
	p1 := productDomain.Product{SKU: "P1", Name: "Alpha", Category: "Makanan", Unit: "Pcs", CostPrice: 60}
	p2 := productDomain.Product{SKU: "P2", Name: "Beta", Category: "Minuman", Unit: "Pcs", CostPrice: 10}
	p1.ID, p2.ID = uuid.New(), uuid.New()
	create(&p1)
	create(&p2)
	mkOrder := func(no, customer, channel, status, date string, total float64, lines ...salesDomain.SalesOrderLine) salesDomain.SalesOrder {
		o := salesDomain.SalesOrder{OrderNumber: no, CustomerName: customer, TotalAmount: total, Channel: channel, Status: status, OrderDate: date}
		o.ID = uuid.New()
		create(&o)
		for _, l := range lines {
			l.ID, l.SalesOrderID = uuid.New(), o.ID
			create(&l)
		}
		return o
	}
	mkOrder("SO-1", "PT Maju", "Direct B2B", "Confirmed", "2026-03-02", 300,
		salesDomain.SalesOrderLine{ProductID: p1.ID, Quantity: 2, UnitPrice: 100, Subtotal: 200}, salesDomain.SalesOrderLine{ProductID: p2.ID, Quantity: 5, UnitPrice: 20, Subtotal: 100})
	pos := mkOrder("POS-1", "Pelanggan Umum", "POS", "Completed", "2026-03-03", 100, salesDomain.SalesOrderLine{ProductID: p1.ID, Quantity: 1, UnitPrice: 100, Subtotal: 100})
	mkOrder("SO-X", "PT Maju", "Direct B2B", "Cancelled", "2026-03-04", 9999, salesDomain.SalesOrderLine{ProductID: p1.ID, Quantity: 99, UnitPrice: 100, Subtotal: 9900})
	mkOrder("SO-OLD", "PT Maju", "Direct B2B", "Confirmed", "2026-02-01", 50)
	tx := posDomain.POSTransaction{OrderNo: pos.OrderNumber, Outlet: "O", Cashier: "C", TotalItems: 1, TotalAmount: 100, PaymentMethod: "cash", SalesOrderID: &pos.ID}
	tx.ID = uuid.New()
	create(&tx)
	txl := posDomain.POSTransactionLine{TransactionID: tx.ID, ProductID: p1.ID, Quantity: 1, UnitPrice: 100, UnitCost: 50, Subtotal: 100}
	txl.ID = uuid.New()
	create(&txl)

	byProduct, err := q.SalesByProduct(ctx, "2026-03-01", "2026-03-31")
	must(err)
	if len(byProduct.Rows) != 2 || byProduct.Rows[0].SKU != "P1" || byProduct.Rows[0].Quantity != 3 || byProduct.Rows[0].Revenue != 300 || byProduct.Rows[0].Cost != 170 {
		t.Fatalf("by product (cancelled order must be excluded, POS cost preferred): %+v", byProduct.Rows)
	}
	byCustomer, err := q.SalesByCustomer(ctx, "2026-03-01", "2026-03-31")
	must(err)
	if byCustomer.Orders != 2 || byCustomer.Revenue != 400 || byCustomer.Rows[0].Customer != "PT Maju" || byCustomer.Rows[0].Revenue != 300 {
		t.Fatalf("by customer: %+v", byCustomer)
	}
	summary, err := q.SalesSummary(ctx, "2026-03-01", "2026-03-31")
	must(err)
	if summary.Orders != 2 || summary.Revenue != 400 || summary.ActiveDays != 2 || summary.Days != 31 || summary.AvgPerActiveDay != 200 {
		t.Fatalf("summary: %+v", summary)
	}
	daily, err := q.DailyProductSales(ctx, "2026-03-01", "2026-03-31")
	must(err)
	if len(daily.Rows) != 3 {
		t.Fatalf("daily product rows: %+v", daily.Rows)
	}

	// Ledger side: sales 400, discount 20, cost 170.
	must(ledger.PostEntryOn(ctx, "2026-03-05", "t-sales", "sales", []financeApp.LedgerLine{
		{AccountCode: financeApp.AccountCash, Debit: 380}, {AccountCode: financeApp.AccountSalesDiscount, Debit: 20}, {AccountCode: financeApp.AccountRevenue, Credit: 400}}))
	must(ledger.PostEntryOn(ctx, "2026-03-05", "t-cogs", "cogs", []financeApp.LedgerLine{
		{AccountCode: financeApp.AccountCOGS, Debit: 170}, {AccountCode: financeApp.AccountInventory, Credit: 170}}))
	gm, err := q.GrossMargin(ctx, "2026-03-01", "2026-03-31")
	must(err)
	if gm.Ledger.NetSales != 380 || gm.Ledger.CostOfGoodsSld != 170 || gm.Ledger.GrossProfit != 210 || gm.Ledger.MarginPct != 55.26 || len(gm.ByCategory) != 2 || gm.Items.Revenue != 400 {
		t.Fatalf("gross margin: %+v", gm)
	}

	// Receivable with a dated payment, plus a manual receivable with an undated one.
	inv := salesDomain.Invoice{InvoiceNumber: "INV-1", CustomerName: "PT Maju", InvoiceDate: "2026-03-02", DueDate: "2026-04-01", TotalAmount: 1000, Subtotal: 1000, PaidAmount: 400}
	inv.ID = uuid.New()
	e := salesApp.InvoiceLedgerEntry(&inv)
	must(ledger.PostEntryOn(ctx, "2026-03-02", e.SourceDoc, e.Memo, e.Lines))
	pe := salesApp.InvoicePaymentLedgerEntry(&inv, 400)
	must(ledger.PostEntryOn(ctx, "2026-03-12", pe.SourceDoc, pe.Memo, pe.Lines))
	must(ledger.UpsertReceivable(ctx, salesApp.InvoiceSubledgerDoc(&inv)))
	manual := financeDomain.Receivable{CustomerName: "PT Maju", InvoiceNo: "MAN-1", IssueDate: "2026-03-01", TotalInvoice: 300, PaidAmount: 100, Status: "partial"}
	manual.ID = uuid.New()
	// Its undated payment is placed at the last update, so pin that inside the period.
	manual.CreatedAt, manual.UpdatedAt = at("2026-03-01", 9), at("2026-03-15", 9)
	must(repo.CreateReceivable(ctx, &manual))

	ar, err := q.PartyBalances(ctx, true, false, "2026-03-31")
	must(err)
	if len(ar.Rows) != 1 || ar.Rows[0].Party != "PT Maju" || ar.Rows[0].Invoiced != 1300 || ar.Rows[0].Paid != 500 || ar.Rows[0].Balance != 800 || ar.Rows[0].OpenDocuments != 2 {
		t.Fatalf("ar balances: %+v", ar)
	}
	card, err := q.PartyCard(ctx, true, "pt maju", "2026-03-10", "2026-03-31")
	must(err)
	if card.Opening != 1000+300-0 || len(card.Entries) < 1 || card.Closing != 800 {
		// opening: both invoices charged before the 10th, no payment before it
		t.Fatalf("ar card: %+v", card)
	}
	var dated *PartyCardEntry
	for i := range card.Entries {
		if card.Entries[i].Reference == "INV-1" && card.Entries[i].Type == "payment" {
			dated = &card.Entries[i]
		}
	}
	if dated == nil || dated.Date != "2026-03-12" || dated.Payment != 400 || dated.Approximate {
		t.Fatalf("dated payment missing: %+v", card.Entries)
	}
	if _, err := q.PartyCard(ctx, true, "nobody", "", ""); err == nil {
		t.Fatal("unknown party must be a not-found error")
	}

	// Payable: invoice with PPN, paid in part.
	supplier := uuid.New()
	pinv := procDomain.PurchaseInvoice{SupplierID: supplier, SupplierName: "PT Pasok", InvoiceNumber: "PINV-1", InvoiceDate: "2026-03-04", DueDate: "2026-04-04",
		TotalAmount: 1110, Subtotal: 1000, TaxBase: 1000, VATAmount: 110, VATCreditable: true, PaidAmount: 500, Status: "partial"}
	pinv.ID = uuid.New()
	create(&pinv)
	pe1 := procApp.PurchaseInvoiceLedgerEntry(&pinv)
	must(ledger.PostEntryOn(ctx, "2026-03-04", pe1.SourceDoc, pe1.Memo, pe1.Lines))
	pp := procApp.PurchaseInvoicePaymentLedgerEntry(&pinv, 500)
	must(ledger.PostEntryOn(ctx, "2026-03-20", pp.SourceDoc, pp.Memo, pp.Lines))
	must(ledger.UpsertPayable(ctx, procApp.PurchaseInvoiceSubledgerDoc(&pinv)))
	ap, err := q.PartyBalances(ctx, false, false, "2026-03-31")
	must(err)
	if len(ap.Rows) != 1 || ap.Rows[0].Balance != 610 || ap.Rows[0].Paid != 500 {
		t.Fatalf("ap balances: %+v", ap)
	}
	apCard, err := q.PartyCard(ctx, false, "PT Pasok", "2026-03-01", "2026-03-31")
	must(err)
	if apCard.Closing != 610 || len(apCard.Entries) != 2 || apCard.Entries[1].Date != "2026-03-20" || apCard.Entries[1].Approximate {
		t.Fatalf("ap card: %+v", apCard)
	}

	po := procDomain.PurchaseOrder{OrderNo: "PO-1", SupplierID: supplier, SupplierName: "PT Pasok", OrderDate: "2026-03-03", Status: "approved", TotalAmount: 400}
	po.ID = uuid.New()
	create(&po)
	pol := procDomain.PurchaseOrderLine{PurchaseOrderID: po.ID, ProductID: p1.ID, Quantity: 4, UnitPrice: 100, Subtotal: 400}
	pol.ID = uuid.New()
	create(&pol)
	draftPO := procDomain.PurchaseOrder{OrderNo: "PO-2", SupplierID: supplier, OrderDate: "2026-03-03", Status: "draft"}
	draftPO.ID = uuid.New()
	create(&draftPO)
	pr, err := q.PurchaseReport(ctx, "2026-03-01", "2026-03-31")
	must(err)
	if len(pr.Invoices) != 1 || pr.Total != 1110 || pr.VAT != 110 || pr.Outstanding != 610 || len(pr.ByProduct) != 1 || pr.ByProduct[0].Quantity != 4 {
		t.Fatalf("purchase report: %+v", pr)
	}

	// Stock: a warehouse, movements across a month boundary, batches and a level.
	wh := inventoryDomain.Warehouse{Code: "WH1", Name: "Gudang Utama", IsActive: true}
	wh.ID = uuid.New()
	create(&wh)
	for _, m := range []inventoryDomain.StockMovement{
		{ProductID: p1.ID, WarehouseID: wh.ID, Type: "in", Quantity: 100, Balance: 100, Reference: "GRN-1"},
		{ProductID: p1.ID, WarehouseID: wh.ID, Type: "out", Quantity: -30, Balance: 70, Reference: "SO-1"},
		{ProductID: p1.ID, WarehouseID: wh.ID, Type: "out", Quantity: -5, Balance: 65, Reference: "SO-9"},
	} {
		m.ID = uuid.New()
		switch m.Reference {
		case "GRN-1":
			m.CreatedAt = at("2026-02-20", 9)
		case "SO-1":
			m.CreatedAt = at("2026-03-02", 9)
		default:
			m.CreatedAt = at("2026-03-31", 23) // late evening WIB, still 31 March
		}
		create(&m)
	}
	sc, err := q.StockCard(ctx, p1.ID, nil, "2026-03-01", "2026-03-31")
	must(err)
	if sc.Opening != 100 || sc.TotalOut != 35 || sc.Closing != 65 || len(sc.Entries) != 2 || sc.Entries[1].Date != "2026-03-31" || sc.SKU != "P1" {
		t.Fatalf("stock card: %+v", sc)
	}
	st, err := q.StockTransactions(ctx, "2026-03-01", "2026-03-31", "out", nil, &wh.ID)
	must(err)
	if len(st.Rows) != 2 || st.Rows[0].Value != -1800 || len(st.ByType) != 1 || st.ByType[0].Quantity != -35 {
		t.Fatalf("stock transactions: %+v", st)
	}
	level := inventoryDomain.StockLevel{ProductID: p1.ID, WarehouseID: wh.ID, Quantity: 65}
	level.ID = uuid.New()
	create(&level)
	b1 := inventoryDomain.StockBatch{ProductID: p1.ID, WarehouseID: wh.ID, BatchNo: "B1", ReceivedAt: "2026-02-20", InitialQty: 100, Quantity: 40}
	b1.ID = uuid.New()
	create(&b1)
	ag, err := q.StockAging(ctx, "2026-03-31", nil)
	must(err)
	if len(ag.Rows) != 1 || ag.Rows[0].Buckets[1] != 40 || ag.Rows[0].Untracked != 25 || ag.TotalQuantity != 65 || ag.TotalValue != 3900 {
		t.Fatalf("stock aging: %+v", ag)
	}
	if _, err := q.StockCard(ctx, uuid.New(), nil, "", ""); err == nil {
		t.Fatal("unknown product must be a not-found error")
	}

	// With an active tenant every query must still be valid SQL (qualified tenant_id on joined tables)
	// and, since none of the rows belong to it, return nothing.
	tid := uuid.New()
	tctx := tenantctx.WithTenantID(ctx, &tid)
	if r, err := q.SalesByProduct(tctx, "2026-03-01", "2026-03-31"); err != nil || len(r.Rows) != 0 {
		t.Fatalf("tenant-scoped sales by product: %+v err=%v", r, err)
	}
	if r, err := q.SalesByCustomer(tctx, "2026-03-01", "2026-03-31"); err != nil || len(r.Rows) != 0 {
		t.Fatalf("tenant-scoped sales by customer: %+v err=%v", r, err)
	}
	if r, err := q.PurchaseReport(tctx, "2026-03-01", "2026-03-31"); err != nil || len(r.Invoices) != 0 || len(r.ByProduct) != 0 {
		t.Fatalf("tenant-scoped purchases: %+v err=%v", r, err)
	}
	if r, err := q.StockTransactions(tctx, "2026-03-01", "2026-03-31", "", nil, nil); err != nil || len(r.Rows) != 0 {
		t.Fatalf("tenant-scoped stock transactions: %+v err=%v", r, err)
	}
	if r, err := q.StockAging(tctx, "2026-03-31", nil); err != nil || len(r.Rows) != 0 {
		t.Fatalf("tenant-scoped stock aging: %+v err=%v", r, err)
	}
	if r, err := q.StockCard(tctx, p1.ID, nil, "2026-03-01", "2026-03-31"); err != nil || len(r.Entries) != 0 {
		t.Fatalf("tenant-scoped stock card: %+v err=%v", r, err)
	}
}
