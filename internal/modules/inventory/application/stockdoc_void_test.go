package application

import (
	"context"
	"strings"
	"testing"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/google/uuid"
)

func (r *docRepo) GetStockDocument(_ context.Context, id uuid.UUID) (*domain.StockDocument, error) {
	for _, d := range r.docs {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}
func (r *docRepo) VoidStockDocument(_ context.Context, d *domain.StockDocument) error { return nil }
func (r *docRepo) ListMovementsByReference(_ context.Context, ref string) ([]domain.StockMovement, error) {
	var out []domain.StockMovement
	for _, m := range r.moves {
		if m.Reference == ref {
			out = append(out, m)
		}
	}
	return out, nil
}
func (r *docRepo) OrderHasCompletedUnits(_ context.Context, id uuid.UUID) (bool, error) {
	return r.completedOrders[id], nil
}

func sides(lines []financeApp.LedgerLine) (dr, cr float64) {
	for _, l := range lines {
		dr += l.Debit
		cr += l.Credit
	}
	return
}

func TestVoidingAnIssueRestoresStockAndReversesTheJournal(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	d, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: domain.DocScrap, Reason: "rusak", WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 30}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.VoidStockDocument(ctx, d.ID, "  "); err == nil {
		t.Fatal("a reason is required")
	}
	v, err := f.uc.VoidStockDocument(ctx, d.ID, "salah input")
	if err != nil {
		t.Fatal(err)
	}
	if !v.Voided || v.VoidReason != "salah input" || f.qty(f.a) != 100 {
		t.Fatalf("void: %+v stock %d", v, f.qty(f.a))
	}
	orig, rev := f.ledger.entries[stockDocSourcePrefix+d.ID.String()], f.ledger.entries[stockDocVoidSourcePrefix+d.ID.String()]
	if len(rev) != 2 || rev[0].AccountCode != orig[0].AccountCode || rev[0].Credit != orig[0].Debit || rev[1].Debit != orig[1].Credit {
		t.Fatalf("reversal: %+v vs %+v", rev, orig)
	}
	if dr, cr := sides(rev); dr != cr || dr != 150_000 {
		t.Fatalf("reversal totals %v/%v", dr, cr)
	}
	if f.ledger.dates[stockDocVoidSourcePrefix+d.ID.String()] != time.Now().Format("2006-01-02") {
		t.Fatalf("the reversal is dated today, got %s", f.ledger.dates[stockDocVoidSourcePrefix+d.ID.String()])
	}
	if _, err := f.uc.VoidStockDocument(ctx, d.ID, "lagi"); err == nil {
		t.Fatal("a document can only be voided once")
	}
}

func TestVoidingAReceiptNeedsTheStockToStillBeThere(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	d, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: domain.DocMemoIn, Reason: "temuan", WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 40}}})
	if err != nil {
		t.Fatal(err)
	}
	// 100 + 40 = 140; 120 of it is then used, leaving 20.
	if _, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: domain.DocMemoOut, Reason: "pakai", WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 120}}}); err != nil {
		t.Fatal(err)
	}
	moves := len(f.repo.moves)
	if _, err := f.uc.VoidStockDocument(ctx, d.ID, "salah"); err == nil || !strings.Contains(err.Error(), "Cannot void") {
		t.Fatalf("want a refusal, got %v", err)
	}
	if f.qty(f.a) != 20 || len(f.repo.moves) != moves {
		t.Fatalf("a refused void must change nothing: stock %d, moves %d->%d", f.qty(f.a), moves, len(f.repo.moves))
	}
}

func TestVoidingAnIssueTiedToAnOrderWithCompletedUnitsIsRefused(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	order := uuid.New()
	d, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: domain.DocMaterialIssue, WarehouseID: f.wh, ProductionOrderID: &order, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 10}}})
	if err != nil {
		t.Fatal(err)
	}
	f.repo.completedOrders = map[uuid.UUID]bool{order: true}
	if _, err := f.uc.VoidStockDocument(ctx, d.ID, "salah"); err == nil || !strings.Contains(err.Error(), "already completed units") {
		t.Fatalf("got %v", err)
	}
	f.repo.completedOrders[order] = false
	if _, err := f.uc.VoidStockDocument(ctx, d.ID, "salah"); err != nil {
		t.Fatal(err)
	}
}

func TestVoidingABatchReceiptTakesItFromThatBatch(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	d, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: domain.DocFinishedGoods, WarehouseID: f.wh,
		Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 10, BatchNo: "L1", ExpiryDate: "2099-01-01"}, {ProductID: f.a, Quantity: 5, BatchNo: "L2"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.VoidStockDocument(ctx, d.ID, "salah lot"); err != nil {
		t.Fatal(err)
	}
	for _, b := range f.repo.batches {
		if b.Quantity != 0 {
			t.Fatalf("batch %s still holds %d", b.BatchNo, b.Quantity)
		}
	}
	if f.qty(f.a) != 100 {
		t.Fatalf("stock %d", f.qty(f.a))
	}
}

func TestVoidingAnIssueReturnsTheStockToTheLotItCameFrom(t *testing.T) {
	f := newDocFixture()
	ctx := context.Background()
	if _, err := f.uc.AdjustStock(ctx, AdjustStockDTO{ProductID: f.a, WarehouseID: f.wh, Quantity: 20, BatchNo: "L9", ExpiryDate: "2099-01-01"}); err != nil {
		t.Fatal(err)
	}
	d, err := f.uc.CreateStockDocument(ctx, CreateStockDocumentDTO{Type: domain.DocScrap, Reason: "rusak", WarehouseID: f.wh, Lines: []StockDocumentLineInput{{ProductID: f.a, Quantity: 8}}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := f.repo.GetBatchByNo(ctx, f.a, f.wh, "L9")
	if b.Quantity != 12 {
		t.Fatalf("the scrap should have taken from L9: %d", b.Quantity)
	}
	if _, err := f.uc.VoidStockDocument(ctx, d.ID, "salah"); err != nil {
		t.Fatal(err)
	}
	if b, _ = f.repo.GetBatchByNo(ctx, f.a, f.wh, "L9"); b.Quantity != 20 {
		t.Fatalf("lot L9 should be back to 20, got %d", b.Quantity)
	}
}
