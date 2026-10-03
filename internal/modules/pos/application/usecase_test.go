package application

import (
	"context"
	"fmt"
	"testing"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/pos/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/google/uuid"
)

// ---- pure calculation

func TestComputeSale(t *testing.T) {
	incl := domain.POSSettings{TaxPercent: 11, TaxInclusive: true}
	excl := domain.POSSettings{TaxPercent: 11, TaxInclusive: false}
	cases := []struct {
		name       string
		sub, disc  float64
		s          domain.POSSettings
		total, tax float64
	}{
		{"no tax", 100_000, 0, domain.POSSettings{}, 100_000, 0},
		{"inclusive keeps the total and carves tax out", 111_000, 0, incl, 111_000, 11_000},
		{"exclusive adds tax on top", 100_000, 0, excl, 111_000, 11_000},
		{"discount comes off before tax", 100_000, 10_000, excl, 99_900, 9_900},
		{"rounding to 500", 99_900, 0, domain.POSSettings{RoundTo: 500}, 100_000, 0},
	}
	for _, c := range cases {
		got, err := ComputeSale(c.sub, c.disc, c.s)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.Total != c.total || got.Tax != c.tax {
			t.Errorf("%s: total %v tax %v, want %v / %v", c.name, got.Total, got.Tax, c.total, c.tax)
		}
	}
	for _, bad := range [][2]float64{{0, 0}, {100, 200}, {100, -1}} {
		if _, err := ComputeSale(bad[0], bad[1], domain.POSSettings{}); err == nil {
			t.Errorf("should reject %v", bad)
		}
	}
}

func TestLedgerEntriesBalance(t *testing.T) {
	tx := &domain.POSTransaction{OrderNo: "POS-1", TotalAmount: 111_000, TaxAmount: 11_000, PaymentMethod: "qris"}
	tx.ID = uuid.New()
	s := domain.POSSettings{NonCashAccountCode: "1010"}
	check := func(name string, e financeApp.LedgerEntry) {
		var dr, cr float64
		for _, l := range e.Lines {
			dr += l.Debit
			cr += l.Credit
		}
		if dr != cr {
			t.Fatalf("%s unbalanced: dr %v cr %v", name, dr, cr)
		}
	}
	e := SaleLedgerEntry(tx, s)
	check("sale", e)
	if e.Lines[0].AccountCode != "1010" {
		t.Fatalf("a QRIS sale must settle into the configured account, got %s", e.Lines[0].AccountCode)
	}
	if cash := SaleLedgerEntry(&domain.POSTransaction{PaymentMethod: "cash", TotalAmount: 1}, s); cash.Lines[0].AccountCode != financeApp.AccountCash {
		t.Fatal("cash goes to the cash account")
	}
	cogs, ok := SaleCOGSEntry(tx, []domain.POSTransactionLine{{Quantity: 2, UnitCost: 30_000}, {Quantity: 1, UnitCost: 0}})
	if !ok {
		t.Fatal("cost expected")
	}
	check("cogs", cogs)
	if _, ok := SaleCOGSEntry(tx, []domain.POSTransactionLine{{Quantity: 1}}); ok {
		t.Fatal("no cost, no COGS entry")
	}
	rf := &domain.POSRefund{Kind: "refund", Amount: 55_500, TaxAmount: 5_500, CostReturned: 30_000, Restocked: true}
	rf.ID = uuid.New()
	check("refund", RefundLedgerEntry(tx, rf, s))
}

// ---- use case with fakes

type fakePOSRepo struct {
	domain.POSRepository
	settings domain.POSSettings
	txs      map[uuid.UUID]*domain.POSTransaction
	lines    map[uuid.UUID][]domain.POSTransactionLine
	refunds  []domain.POSRefund
}

func newRepo(s domain.POSSettings) *fakePOSRepo {
	return &fakePOSRepo{settings: s, txs: map[uuid.UUID]*domain.POSTransaction{}, lines: map[uuid.UUID][]domain.POSTransactionLine{}}
}
func (f *fakePOSRepo) GetSettings(context.Context) (*domain.POSSettings, error) {
	c := f.settings
	return &c, nil
}
func (f *fakePOSRepo) Create(_ context.Context, tx *domain.POSTransaction) error {
	tx.ID = uuid.New()
	f.txs[tx.ID] = tx
	return nil
}
func (f *fakePOSRepo) CreateLines(_ context.Context, lines []domain.POSTransactionLine) error {
	for i := range lines {
		lines[i].ID = uuid.New()
		f.lines[lines[i].TransactionID] = append(f.lines[lines[i].TransactionID], lines[i])
	}
	return nil
}
func (f *fakePOSRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.POSTransaction, error) {
	tx := f.txs[id]
	if tx == nil {
		return nil, nil
	}
	c := *tx
	c.Lines = append([]domain.POSTransactionLine(nil), f.lines[id]...)
	return &c, nil
}
func (f *fakePOSRepo) Update(_ context.Context, tx *domain.POSTransaction) error {
	c := *tx
	c.Lines = nil
	f.txs[tx.ID] = &c
	return nil
}
func (f *fakePOSRepo) UpdateLine(_ context.Context, l *domain.POSTransactionLine) error {
	ls := f.lines[l.TransactionID]
	for i := range ls {
		if ls[i].ID == l.ID {
			ls[i] = *l
		}
	}
	return nil
}
func (f *fakePOSRepo) CreateRefund(_ context.Context, r *domain.POSRefund) error {
	r.ID = uuid.New()
	f.refunds = append(f.refunds, *r)
	return nil
}
func (f *fakePOSRepo) ListRefunds(_ context.Context, id uuid.UUID) ([]domain.POSRefund, error) {
	var out []domain.POSRefund
	for _, r := range f.refunds {
		if r.TransactionID == id {
			out = append(out, r)
		}
	}
	return out, nil
}

type fakeSales struct {
	salesDomain.SalesRepository
	orders map[uuid.UUID]*salesDomain.SalesOrder
}

func (f *fakeSales) CreateOrder(_ context.Context, o *salesDomain.SalesOrder) error {
	o.ID = uuid.New()
	f.orders[o.ID] = o
	return nil
}
func (f *fakeSales) UpdateOrder(context.Context, *salesDomain.SalesOrder) error { return nil }
func (f *fakeSales) GetOrderByID(_ context.Context, id uuid.UUID) (*salesDomain.SalesOrder, error) {
	return f.orders[id], nil
}

type fakeInv struct {
	inventoryApp.InventoryUseCase
	stock   map[uuid.UUID]int
	failFor uuid.UUID
}

func (f *fakeInv) GetAvailability(_ context.Context, p, _ uuid.UUID) (int, error) {
	return f.stock[p], nil
}
func (f *fakeInv) AdjustStock(_ context.Context, d inventoryApp.AdjustStockDTO) (*inventoryApp.StockLevelResponseDTO, error) {
	if d.ProductID == f.failFor && d.Quantity < 0 {
		return nil, fmt.Errorf("simulated stock failure")
	}
	f.stock[d.ProductID] += d.Quantity
	return nil, nil
}

type fakeProducts struct {
	productDomain.ProductRepository
	byID map[uuid.UUID]*productDomain.Product
}

func (f *fakeProducts) GetByID(_ context.Context, id uuid.UUID) (*productDomain.Product, error) {
	return f.byID[id], nil
}

type fakeLedger struct {
	financeApp.LedgerPoster
	posted map[string][]financeApp.LedgerLine
}

func (f *fakeLedger) PostEntry(_ context.Context, src, _ string, lines []financeApp.LedgerLine) error {
	f.posted[src] = lines
	return nil
}

type env struct {
	uc     POSUseCase
	repo   *fakePOSRepo
	inv    *fakeInv
	ledger *fakeLedger
	sales  *fakeSales
	a, b   uuid.UUID
	wh     uuid.UUID
}

func newEnv(s domain.POSSettings) *env {
	e := &env{a: uuid.New(), b: uuid.New(), wh: uuid.New()}
	e.repo = newRepo(s)
	e.sales = &fakeSales{orders: map[uuid.UUID]*salesDomain.SalesOrder{}}
	e.inv = &fakeInv{stock: map[uuid.UUID]int{e.a: 10, e.b: 5}}
	e.ledger = &fakeLedger{posted: map[string][]financeApp.LedgerLine{}}
	prods := &fakeProducts{byID: map[uuid.UUID]*productDomain.Product{
		e.a: {SKU: "A-1", Name: "Kopi", CostPrice: 6_000, SellingPrice: 10_000},
		e.b: {SKU: "B-1", Name: "Roti", CostPrice: 4_000, SellingPrice: 8_000},
	}}
	e.uc = NewPOSUseCase(e.repo, e.sales, nil, e.inv, WithLedger(e.ledger), WithProducts(prods))
	return e
}

func cashier(email string) context.Context { return actor.WithEmail(context.Background(), email) }

func (e *env) sell(t *testing.T, qtyA, qtyB float64, method string, tendered float64) *POSTransactionResponseDTO {
	t.Helper()
	res, err := e.uc.Checkout(cashier("kasir@x.com"), CheckoutDTO{WarehouseID: &e.wh, PaymentMethod: method, AmountTendered: tendered,
		Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: qtyA}, {ProductID: e.b, Quantity: qtyB}}})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestCheckoutComputesTotalsStoresLinesAndPostsToLedger(t *testing.T) {
	e := newEnv(domain.POSSettings{TaxPercent: 11, TaxInclusive: true})
	// 2 x 10,000 + 1 x 8,000 = 28,000 inclusive of 11% tax.
	res := e.sell(t, 2, 1, "cash", 50_000)
	if res.TotalAmount != 28_000 || res.TaxAmount != 2_774.77 || res.ChangeAmount != 22_000 || res.Cashier != "kasir@x.com" {
		t.Fatalf("checkout: %+v", res)
	}
	if len(res.Lines) != 2 || res.Lines[0].Name != "Kopi" || res.Lines[0].UnitCost != 6_000 {
		t.Fatalf("lines must carry the product snapshot: %+v", res.Lines)
	}
	if e.inv.stock[e.a] != 8 || e.inv.stock[e.b] != 4 {
		t.Fatalf("stock after sale: %v", e.inv.stock)
	}
	sale := e.ledger.posted[SaleSourcePrefix+res.ID.String()]
	cogs := e.ledger.posted[COGSSourcePrefix+res.ID.String()]
	if len(sale) != 3 || len(cogs) != 2 || cogs[0].Debit != 16_000 { // 2*6000 + 1*4000
		t.Fatalf("ledger: sale %+v cogs %+v", sale, cogs)
	}
}

func TestCheckoutValidation(t *testing.T) {
	e := newEnv(domain.POSSettings{})
	ctx := cashier("kasir@x.com")
	bad := []CheckoutDTO{
		{WarehouseID: &e.wh, PaymentMethod: "cash", AmountTendered: 5_000, Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: 1}}},  // short cash
		{WarehouseID: &e.wh, PaymentMethod: "bitcoin", Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: 1}}},                      // unknown method
		{WarehouseID: &e.wh, PaymentMethod: "card", Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: 11}}},                        // not enough stock
		{WarehouseID: &e.wh, PaymentMethod: "card", TotalAmount: 5_000, Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: 1}}},     // total disagrees with the cart
		{PaymentMethod: "card", Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: 1}}},                                             // no warehouse
		{WarehouseID: &e.wh, PaymentMethod: "card", Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: 0}}},                         // zero quantity
		{WarehouseID: &e.wh, PaymentMethod: "card", DiscountAmount: 99_999, Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: 1}}}, // discount above subtotal
		{PaymentMethod: "cash"}, // nothing to sell
	}
	for i, dto := range bad {
		if _, err := e.uc.Checkout(ctx, dto); err == nil {
			t.Errorf("case %d should be rejected", i)
		}
	}
	if e.inv.stock[e.a] != 10 || len(e.repo.txs) != 0 || len(e.sales.orders) != 0 {
		t.Fatalf("rejected sales must not change anything: stock %v txs %d orders %d", e.inv.stock, len(e.repo.txs), len(e.sales.orders))
	}
	// Legacy quick sale (no lines) still works.
	if res, err := e.uc.Checkout(ctx, CheckoutDTO{TotalAmount: 25_000, TotalItems: 3, PaymentMethod: "cash"}); err != nil || res.TotalAmount != 25_000 {
		t.Fatalf("quick sale: %+v %v", res, err)
	}
}

func TestFailedStockDeductionRestoresWhatWasTaken(t *testing.T) {
	e := newEnv(domain.POSSettings{})
	e.inv.failFor = e.b // the second line's deduction fails
	_, err := e.uc.Checkout(cashier("kasir@x.com"), CheckoutDTO{WarehouseID: &e.wh, PaymentMethod: "card",
		Lines: []CheckoutLineDTO{{ProductID: e.a, Quantity: 3}, {ProductID: e.b, Quantity: 1}}})
	if err == nil {
		t.Fatal("expected failure")
	}
	if e.inv.stock[e.a] != 10 || e.inv.stock[e.b] != 5 {
		t.Fatalf("stock must be back where it started: %v", e.inv.stock)
	}
	if len(e.repo.txs) != 0 || len(e.ledger.posted) != 0 {
		t.Fatal("no transaction or journal may exist for a failed sale")
	}
}

func TestVoidNeedsAnotherPersonAndReversesEverything(t *testing.T) {
	e := newEnv(domain.POSSettings{TaxPercent: 11, TaxInclusive: true})
	res := e.sell(t, 2, 1, "card", 0)
	if _, err := e.uc.Void(cashier("kasir@x.com"), res.ID, "salah input"); err == nil {
		t.Fatal("the cashier must not void their own sale")
	}
	if _, err := e.uc.Void(cashier("spv@x.com"), res.ID, " "); err == nil {
		t.Fatal("a reason is required")
	}
	got, err := e.uc.Void(cashier("spv@x.com"), res.ID, "pelanggan batal")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "Voided" || got.RefundedAmount != 28_000 || got.VoidReason != "pelanggan batal" {
		t.Fatalf("voided sale: %+v", got)
	}
	if e.inv.stock[e.a] != 10 || e.inv.stock[e.b] != 5 {
		t.Fatalf("void must restock: %v", e.inv.stock)
	}
	var reversal []financeApp.LedgerLine
	for src, lines := range e.ledger.posted {
		if len(src) > len(RefundSourcePrefix) && src[:len(RefundSourcePrefix)] == RefundSourcePrefix {
			reversal = lines
		}
	}
	var dr, cr float64
	for _, l := range reversal {
		dr += l.Debit
		cr += l.Credit
	}
	if dr != cr || cr < 28_000 {
		t.Fatalf("reversal journal: dr %v cr %v %+v", dr, cr, reversal)
	}
	if _, err := e.uc.Void(cashier("spv@x.com"), res.ID, "lagi"); err == nil {
		t.Fatal("a voided sale cannot be voided again")
	}
	if _, err := e.uc.Refund(cashier("spv@x.com"), res.ID, RefundDTO{Reason: "x", Lines: []RefundLineDTO{{LineID: uuid.New(), Quantity: 1}}}); err == nil {
		t.Fatal("a voided sale cannot be refunded")
	}
}

func TestPartialRefundsAddUpToTheSaleTotalExactly(t *testing.T) {
	e := newEnv(domain.POSSettings{TaxPercent: 11, TaxInclusive: true, RoundTo: 100})
	res := e.sell(t, 3, 3, "card", 0) // 30,000 + 24,000 = 54,000
	if res.TotalAmount != 54_000 {
		t.Fatalf("total %v", res.TotalAmount)
	}
	spv := cashier("spv@x.com")
	lineA, lineB := res.Lines[0].ID, res.Lines[1].ID
	var refunded float64
	step := func(line uuid.UUID, qty float64, restock *bool) *POSTransactionResponseDTO {
		t.Helper()
		out, err := e.uc.Refund(spv, res.ID, RefundDTO{Reason: "rusak", Restock: restock, Lines: []RefundLineDTO{{LineID: line, Quantity: qty}}})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	out := step(lineA, 1, nil)
	if out.Status != "Partially Refunded" || out.RefundedAmount != 10_000 {
		t.Fatalf("first refund: %+v", out)
	}
	refunded = out.RefundedAmount
	no := false
	out = step(lineB, 1, &no)
	refunded = out.RefundedAmount
	if e.inv.stock[e.a] != 8 || e.inv.stock[e.b] != 2 { // start 10/5, sold 3/3 -> 7/2, restocked 1 of A only
		t.Fatalf("restock flag ignored: %v", e.inv.stock)
	}
	if _, err := e.uc.Refund(spv, res.ID, RefundDTO{Reason: "x", Lines: []RefundLineDTO{{LineID: lineA, Quantity: 3}}}); err == nil {
		t.Fatal("cannot refund more than remains")
	}
	step(lineA, 2, nil)
	out = step(lineB, 2, nil)
	if out.Status != "Refunded" || out.RefundedAmount != 54_000 {
		t.Fatalf("a fully refunded sale must hand back exactly its total: %+v (was %v)", out, refunded)
	}
	var taxBack float64
	for _, r := range e.repo.refunds {
		taxBack += r.TaxAmount
	}
	if taxBack != res.TaxAmount {
		t.Fatalf("tax returned %v, sale had %v", taxBack, res.TaxAmount)
	}
	if _, err := e.uc.Refund(cashier("kasir@x.com"), res.ID, RefundDTO{Reason: "x", Lines: []RefundLineDTO{{LineID: lineA, Quantity: 1}}}); err == nil {
		t.Fatal("the cashier must not refund their own sale")
	}
}

func TestLegacySaleCanOnlyBeVoided(t *testing.T) {
	e := newEnv(domain.POSSettings{})
	res, err := e.uc.Checkout(cashier("kasir@x.com"), CheckoutDTO{TotalAmount: 25_000, PaymentMethod: "cash"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.uc.Refund(cashier("spv@x.com"), res.ID, RefundDTO{Reason: "x", Lines: []RefundLineDTO{{LineID: uuid.New(), Quantity: 1}}}); err == nil {
		t.Fatal("a sale with no lines cannot be partially refunded")
	}
	got, err := e.uc.Void(cashier("spv@x.com"), res.ID, "salah")
	if err != nil || got.Status != "Voided" || got.RefundedAmount != 25_000 {
		t.Fatalf("legacy void: %+v %v", got, err)
	}
}

func TestSalesReport(t *testing.T) {
	d1 := time.Date(2026, 9, 10, 10, 0, 0, 0, zone)
	d2 := time.Date(2026, 9, 11, 22, 0, 0, 0, zone)
	mk := func(at time.Time, total, tax float64, method, cashier, status string, lines ...domain.POSTransactionLine) domain.POSTransaction {
		tx := domain.POSTransaction{TotalAmount: total, TaxAmount: tax, PaymentMethod: method, Cashier: cashier, Status: status, Lines: lines}
		tx.ID = uuid.New()
		tx.CreatedAt = at
		return tx
	}
	p1 := uuid.New()
	txs := []domain.POSTransaction{
		mk(d1, 10_000, 1_000, "cash", "ani", "Completed", domain.POSTransactionLine{ProductID: p1, Name: "Kopi", Quantity: 2, UnitPrice: 5_000, RefundedQty: 1}),
		mk(d1, 20_000, 2_000, "qris", "bob", "Completed"),
		mk(d1, 99_999, 0, "cash", "ani", "Voided"),
		mk(d2, 30_000, 3_000, "cash", "ani", "Partially Refunded"),
	}
	refunds := []domain.POSRefund{
		{TransactionID: txs[0].ID, Kind: "refund", Amount: 5_000, CreatedAt: d2},
		{TransactionID: txs[2].ID, Kind: "void", Amount: 99_999, CreatedAt: d2},
	}
	rep := BuildSalesReport("2026-09-10", "2026-09-11", txs, refunds)
	if len(rep.Days) != 2 || rep.Days[0].Date != "2026-09-10" || rep.Days[0].GrossSales != 30_000 || rep.Days[0].Voided != 1 || rep.Days[0].Transactions != 2 {
		t.Fatalf("day 1: %+v", rep.Days)
	}
	if rep.Days[1].Refunds != 5_000 || rep.Days[1].NetSales != 25_000 { // refund lowers the day it was made
		t.Fatalf("day 2: %+v", rep.Days[1])
	}
	if rep.Totals.GrossSales != 60_000 || rep.Totals.Refunds != 5_000 || rep.Totals.NetSales != 55_000 || rep.Totals.Voided != 1 {
		t.Fatalf("totals: %+v", rep.Totals)
	}
	if rep.ByPayment[0].Name != "cash" || rep.ByPayment[0].Amount != 40_000 {
		t.Fatalf("by payment: %+v", rep.ByPayment)
	}
	if len(rep.TopProducts) != 1 || rep.TopProducts[0].Quantity != 1 || rep.TopProducts[0].Revenue != 5_000 { // net of the returned unit
		t.Fatalf("top products: %+v", rep.TopProducts)
	}
}
