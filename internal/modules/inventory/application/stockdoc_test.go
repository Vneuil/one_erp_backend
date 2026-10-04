package application

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/google/uuid"
)

// docRepo adds stock-document and opname storage to the in-memory repo.
type docRepo struct {
	*memRepo
	docs       []*domain.StockDocument
	opname     *domain.StockOpname
	failUpsert map[uuid.UUID]bool // product ids whose stock-level update fails
}

func (r *docRepo) WithTransaction(_ context.Context, fn func(domain.InventoryRepository) error) error {
	return fn(r)
}
func (r *docRepo) UpsertStockLevel(ctx context.Context, s *domain.StockLevel) error {
	if r.failUpsert[s.ProductID] {
		return fmt.Errorf("injected failure")
	}
	return r.memRepo.UpsertStockLevel(ctx, s)
}
func (r *docRepo) CreateStockDocument(_ context.Context, d *domain.StockDocument) error {
	r.docs = append(r.docs, d)
	return nil
}
func (r *docRepo) UpdateStockDocument(context.Context, *domain.StockDocument) error { return nil }
func (r *docRepo) CountStockDocuments(_ context.Context, typ, prefix string) (int64, error) {
	var n int64
	for _, d := range r.docs {
		if d.Type == typ && strings.HasPrefix(d.Number, prefix) {
			n++
		}
	}
	return n, nil
}
func (r *docRepo) GetOpnameByID(_ context.Context, id uuid.UUID) (*domain.StockOpname, error) {
	if r.opname != nil && r.opname.ID == id {
		return r.opname, nil
	}
	return nil, nil
}
func (r *docRepo) UpdateOpnameLine(context.Context, *domain.StockOpnameLine) error { return nil }
func (r *docRepo) UpdateOpname(context.Context, *domain.StockOpname) error         { return nil }

type products map[uuid.UUID]*productDomain.Product

type productRepoMap struct {
	productDomain.ProductRepository
	byID products
}

func (p productRepoMap) GetByID(_ context.Context, id uuid.UUID) (*productDomain.Product, error) {
	return p.byID[id], nil
}

type dateLedger struct {
	financeApp.LedgerPoster
	entries map[string][]financeApp.LedgerLine
	dates   map[string]string
}

func (l *dateLedger) PostEntryOn(_ context.Context, date, src, _ string, lines []financeApp.LedgerLine) error {
	l.entries[src] = lines
	l.dates[src] = date
	return nil
}

type docFixture struct {
	uc     *inventoryUseCase
	repo   *docRepo
	ledger *dateLedger
	wh     uuid.UUID
	a, b   uuid.UUID
}

func newDocFixture() *docFixture {
	a, b := uuid.New(), uuid.New()
	repo := &docRepo{memRepo: newMem(), failUpsert: map[uuid.UUID]bool{}}
	led := &dateLedger{entries: map[string][]financeApp.LedgerLine{}, dates: map[string]string{}}
	uc := &inventoryUseCase{repo: repo, ledger: led, productRepo: productRepoMap{byID: products{
		a: {SKU: "A", Name: "Tepung", CostPrice: 5_000}, b: {SKU: "B", Name: "Gula", CostPrice: 2_000}}}}
	wh := uuid.New()
	for _, p := range []uuid.UUID{a, b} {
		repo.levels[[2]uuid.UUID{p, wh}] = &domain.StockLevel{ProductID: p, WarehouseID: wh, Quantity: 100}
	}
	return &docFixture{uc: uc, repo: repo, ledger: led, wh: wh, a: a, b: b}
}

func (f *docFixture) qty(p uuid.UUID) int { return f.repo.levels[[2]uuid.UUID{p, f.wh}].Quantity }

func (f *docFixture) entryOf(d *StockDocumentResponseDTO) []financeApp.LedgerLine {
	return f.ledger.entries[stockDocSourcePrefix+d.ID.String()]
}

func TestMaterialIssueMovesStockNumbersAndPostsToWIP(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	d, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: domain.DocMaterialIssue, WarehouseID: f.wh, Reference: "PO-7",
		Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 30}, {ProductID: f.b, Quantity: 10}}})
	if err != nil {
		t.Fatal(err)
	}
	prefix := "PBH-" + time.Now().In(batchZone).Format("200601") + "-"
	if d.Number != prefix+"0001" || d.Direction != "out" || d.TotalValue != 170_000 || !d.Posted || len(d.Lines) != 2 {
		t.Fatalf("document: %+v", d)
	}
	if f.qty(f.a) != 70 || f.qty(f.b) != 90 {
		t.Fatalf("stock: a=%d b=%d", f.qty(f.a), f.qty(f.b))
	}
	if mv := f.repo.moves[0]; mv.Quantity != -30 || mv.Reference != d.Number || !strings.HasPrefix(mv.Reason, "Pengambilan bahan") {
		t.Fatalf("movement: %+v", mv)
	}
	e := f.entryOf(d)
	if len(e) != 2 || e[0].AccountCode != financeApp.AccountWIP || e[0].Debit != 170_000 || e[1].AccountCode != financeApp.AccountInventory || e[1].Credit != 170_000 {
		t.Fatalf("journal: %+v", e)
	}
	d2, _ := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: domain.DocMaterialIssue, WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 1}}})
	if d2.Number != prefix+"0002" {
		t.Fatalf("second number: %s", d2.Number)
	}
}

func TestStockOutDocumentsRefuseInsufficientStockWithoutSideEffects(t *testing.T) {
	f := newDocFixture()
	_, err := f.uc.CreateStockDocument(context.Background(), CreateStockDocumentDTO{Type: domain.DocScrap, Reason: "rusak", WarehouseID: f.wh,
		Lines: []StockDocumentLineInput{{ProductID: f.b, Quantity: 5}, {ProductID: f.a, Quantity: 60}, {ProductID: f.a, Quantity: 60}}})
	if err == nil || !strings.Contains(err.Error(), "Insufficient stock of Tepung") {
		t.Fatalf("want an insufficient-stock error naming the product (duplicate lines are summed), got %v", err)
	}
	if f.qty(f.a) != 100 || f.qty(f.b) != 100 || len(f.repo.moves) != 0 || len(f.repo.docs) != 0 || len(f.ledger.entries) != 0 {
		t.Fatal("a refused document must leave nothing behind")
	}
}

func TestDocumentIsAllOrNothingWhenALineFailsMidway(t *testing.T) {
	f := newDocFixture()
	f.repo.failUpsert[f.b] = true
	_, err := f.uc.CreateStockDocument(context.Background(), CreateStockDocumentDTO{Type: domain.DocMemoOut, Reason: "selisih", WarehouseID: f.wh,
		Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 10}, {ProductID: f.b, Quantity: 10}}})
	if err == nil {
		t.Fatal("expected the injected failure")
	}
	if f.qty(f.a) != 100 {
		t.Fatalf("the first line must be put back, stock is %d", f.qty(f.a))
	}
	if len(f.repo.docs) != 0 || len(f.ledger.entries) != 0 {
		t.Fatal("no document or journal may exist")
	}
	last := f.repo.moves[len(f.repo.moves)-1]
	if last.Quantity != 10 || !strings.HasPrefix(last.Reason, "Pembatalan otomatis") {
		t.Fatalf("the rollback must be visible in the movement history: %+v", last)
	}
}

func TestFinishedGoodsReceiptUsesProductionCostAndBatch(t *testing.T) {
	f := newDocFixture()
	cost := 7_500.0
	d, err := f.uc.CreateStockDocument(context.Background(), CreateStockDocumentDTO{Type: domain.DocFinishedGoods, WarehouseID: f.wh, Reference: "PO-7",
		Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 20, UnitCost: &cost, BatchNo: "L1", ExpiryDate: "2030-01-01"}, {ProductID: f.b, Quantity: 4}}})
	if err != nil {
		t.Fatal(err)
	}
	// 20 x 7.500 (production cost) + 4 x 2.000 (product cost price)
	if d.TotalValue != 158_000 || d.Lines[0].UnitCost != 7_500 || d.Lines[1].UnitCost != 2_000 || d.Direction != "in" {
		t.Fatalf("document: %+v", d)
	}
	if f.qty(f.a) != 120 || qtyOf(f.repo.memRepo, "L1") != 20 {
		t.Fatalf("stock %d, lot L1 %d", f.qty(f.a), qtyOf(f.repo.memRepo, "L1"))
	}
	e := f.entryOf(d)
	if e[0].AccountCode != financeApp.AccountInventory || e[0].Debit != 158_000 || e[1].AccountCode != financeApp.AccountWIP || e[1].Credit != 158_000 {
		t.Fatalf("journal: %+v", e)
	}
}

func TestScrapAndMemoJournalsAndReasons(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	line := []StockDocumentLineInput{{ProductID: f.a, Quantity: 2}}
	for _, typ := range []string{domain.DocScrap, domain.DocMemoIn, domain.DocMemoOut} {
		if _, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: typ, WarehouseID: f.wh, Lines: line}); err == nil {
			t.Errorf("%s must require a reason", typ)
		}
	}
	want := map[string][2]string{
		domain.DocScrap:   {financeApp.AccountScrapLoss, financeApp.AccountInventory},
		domain.DocMemoIn:  {financeApp.AccountInventory, financeApp.AccountInventoryAdjustment},
		domain.DocMemoOut: {financeApp.AccountInventoryAdjustment, financeApp.AccountInventory},
	}
	for typ, accts := range want {
		d, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: typ, Reason: "uji", WarehouseID: f.wh, Lines: line})
		if err != nil {
			t.Fatal(err)
		}
		e := f.entryOf(d)
		if e[0].AccountCode != accts[0] || e[1].AccountCode != accts[1] || e[0].Debit != 10_000 || e[1].Credit != 10_000 {
			t.Fatalf("%s journal: %+v", typ, e)
		}
	}
}

func TestStockDocumentValidation(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	ok := []StockDocumentLineInput{{ProductID: f.a, Quantity: 1}}
	cost := 1.0
	future := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	cases := map[string]CreateStockDocumentDTO{
		"unknown type":      {Type: "x", WarehouseID: f.wh, Lines: ok},
		"no lines":          {Type: domain.DocMaterialIssue, WarehouseID: f.wh},
		"zero quantity":     {Type: domain.DocMaterialIssue, WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a}}},
		"no product":        {Type: domain.DocMaterialIssue, WarehouseID: f.wh, Lines: []StockDocumentLineInput{{Quantity: 1}}},
		"future date":       {Type: domain.DocMaterialIssue, WarehouseID: f.wh, Date: future, Lines: ok},
		"bad date":          {Type: domain.DocMaterialIssue, WarehouseID: f.wh, Date: "10-10-2026", Lines: ok},
		"cost on issue":     {Type: domain.DocMaterialIssue, WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 1, UnitCost: &cost}}},
		"batch on issue":    {Type: domain.DocMaterialIssue, WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 1, BatchNo: "X"}}},
		"unknown product":   {Type: domain.DocMaterialIssue, WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: uuid.New(), Quantity: 1}}},
		"bad expiry format": {Type: domain.DocFinishedGoods, WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 1, BatchNo: "X", ExpiryDate: "soon"}}},
	}
	for name, dto := range cases {
		if _, err := f.uc.CreateStockDocument(ctx, dto); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if len(f.repo.moves) != 0 {
		t.Fatal("validation failures must not move stock")
	}
}

func TestOpnameImportAppliesValidRowsAndReportsTheRest(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	f.repo.opname = &domain.StockOpname{WarehouseID: f.wh, Status: domain.OpnameDraft, Lines: []domain.StockOpnameLine{
		{ProductID: f.a, SystemQty: 100}, {ProductID: f.b, SystemQty: 100}}}
	f.repo.opname.ID = uuid.New()

	res, err := f.uc.ImportOpnameCounts(ctx, f.repo.opname.ID, []OpnameCountRow{
		{SKU: "a", CountedQty: 95}, {SKU: "B ", CountedQty: 103}, {SKU: "ZZZ", CountedQty: 1}, {SKU: "A", CountedQty: 7}, {SKU: "", CountedQty: 1}, {SKU: "B", CountedQty: -1}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Applied != 2 || len(res.Errors) != 4 {
		t.Fatalf("result: %+v", res)
	}
	msgs := map[int]string{}
	for _, e := range res.Errors {
		msgs[e.Row] = e.Message
	}
	if !strings.Contains(msgs[3], "not part") || !strings.Contains(msgs[4], "more than once") || !strings.Contains(msgs[5], "empty") || !strings.Contains(msgs[6], "negative") {
		t.Fatalf("errors: %+v", res.Errors)
	}
	if l := f.repo.opname.Lines; l[0].CountedQty != 95 || !l[0].Counted || l[1].CountedQty != 103 || f.repo.opname.Status != domain.OpnameInProgress {
		t.Fatalf("lines: %+v status %s", l, f.repo.opname.Status)
	}

	// Finalizing books the variance at cost: A short by 5 (25,000), B over by 3 (6,000).
	if _, err := f.uc.FinalizeOpname(ctx, f.repo.opname.ID); err != nil {
		t.Fatal(err)
	}
	if f.qty(f.a) != 95 || f.qty(f.b) != 103 {
		t.Fatalf("stock after finalize: a=%d b=%d", f.qty(f.a), f.qty(f.b))
	}
	e := f.ledger.entries["stock-opname:"+f.repo.opname.ID.String()]
	var dr, cr, loss, gain float64
	for _, l := range e {
		dr, cr = dr+l.Debit, cr+l.Credit
		switch {
		case l.AccountCode == financeApp.AccountInventoryAdjustment && l.Debit > 0:
			loss = l.Debit
		case l.AccountCode == financeApp.AccountInventoryAdjustment && l.Credit > 0:
			gain = l.Credit
		}
	}
	if dr != cr || loss != 25_000 || gain != 6_000 {
		t.Fatalf("opname journal: %+v", e)
	}
	if _, err := f.uc.ImportOpnameCounts(ctx, f.repo.opname.ID, []OpnameCountRow{{SKU: "A", CountedQty: 1}}); err == nil {
		t.Fatal("a completed opname cannot take counts")
	}
}
