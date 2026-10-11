package application

import (
	"context"
	"strings"
	"testing"

	inventoryDomain "github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/google/uuid"
)

func (f *fixture) stock(p uuid.UUID) int {
	if l := f.inv.levels[[2]uuid.UUID{p, f.wh}]; l != nil {
		return l.Quantity
	}
	return 0
}

func passAll(t *testing.T, f *fixture, o *domain.ProductionOrder, qty int) {
	t.Helper()
	for i := range o.Steps {
		if err := f.log(t, o, i, qty); err != nil {
			t.Fatal(err)
		}
	}
}

func TestVoidBatchReturnsMaterialsRemovesFinishedGoodsAndReopensTheOrder(t *testing.T) {
	f := newFixture(t) // backflush: 2 x RM1 + 1 x RM2 per unit
	ctx := context.Background()
	o := f.newOrder(t, 10, "")
	f.release(o)
	passAll(t, f, o, 10)
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 10}); err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.OrderCompleted || f.stock(f.fg) != 10 || f.stock(f.rm1) != 980 || f.stock(f.rm2) != 990 {
		t.Fatalf("after completion: status %s fg %d rm1 %d rm2 %d", o.Status, f.stock(f.fg), f.stock(f.rm1), f.stock(f.rm2))
	}
	batch := f.repo.batches[0]
	if _, err := f.uc.VoidBatch(ctx, o.ID, batch.ID, " "); err == nil {
		t.Fatal("a reason is required")
	}
	if _, err := f.uc.VoidBatch(ctx, o.ID, batch.ID, "salah hitung"); err != nil {
		t.Fatal(err)
	}
	if f.stock(f.fg) != 0 || f.stock(f.rm1) != 1000 || f.stock(f.rm2) != 1000 {
		t.Fatalf("stock after void: fg %d rm1 %d rm2 %d", f.stock(f.fg), f.stock(f.rm1), f.stock(f.rm2))
	}
	if o.QuantityCompleted != 0 || o.Status != domain.OrderInProgress || o.ActualCompletionDate != "" {
		t.Fatalf("the order should reopen: %+v", o)
	}
	if !f.repo.batches[0].Voided {
		t.Fatal("the batch should be marked void")
	}
	if _, err := f.uc.VoidBatch(ctx, o.ID, batch.ID, "lagi"); err == nil {
		t.Fatal("a batch can only be voided once")
	}
	// The same units can be completed again.
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 10}); err != nil {
		t.Fatalf("after the void the batch can be redone: %v", err)
	}
}

func TestVoidBatchRefusedWhenTheFinishedGoodsAreGone(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	o := f.newOrder(t, 10, "")
	f.release(o)
	passAll(t, f, o, 10)
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 6}); err != nil {
		t.Fatal(err)
	}
	f.inv.levels[[2]uuid.UUID{f.fg, f.wh}].Quantity = 2 // 4 of the 6 were sold
	_, err := f.uc.VoidBatch(ctx, o.ID, f.repo.batches[0].ID, "salah")
	if err == nil || !strings.Contains(err.Error(), "Cannot void") {
		t.Fatalf("got %v", err)
	}
	if f.stock(f.fg) != 2 || f.stock(f.rm1) != 988 || o.QuantityCompleted != 6 || f.repo.batches[0].Voided {
		t.Fatal("a refused void must change nothing")
	}
}

func TestVoidBatchLeavesIssuedMaterialsAloneAndReversesTheJournalAndCost(t *testing.T) {
	f := costedFixture(t)
	ctx := context.Background()
	f.inv.levels[[2]uuid.UUID{f.fg, f.wh}] = &inventoryDomain.StockLevel{ProductID: f.fg, WarehouseID: f.wh, Quantity: 6}
	f.uc.productRepo.(memProducts).byID[f.fg].CostPrice = 10_000
	o := f.newOrder(t, 10, domain.MaterialIssued)
	f.release(o)
	f.inv.issued[o.ID] = map[uuid.UUID]int{f.rm1: 20, f.rm2: 10}
	passAll(t, f, o, 10)
	if _, err := f.uc.CompleteBatch(ctx, o.ID, CompleteBatchDTO{QuantityCompleted: 4}); err != nil {
		t.Fatal(err)
	}
	cost := f.uc.productRepo.(memProducts).byID[f.fg].CostPrice
	// 6 on hand @ 10,000 + 4 new @ (2,500 + 19,000): 14,600.
	if cost != 14_600 {
		t.Fatalf("moving average after completion: %v", cost)
	}
	rm1 := f.stock(f.rm1)
	b := f.repo.batches[0]
	if b.UnitCost != 21_500 || !b.CostApplied {
		t.Fatalf("batch cost not recorded: %+v", b)
	}
	if _, err := f.uc.VoidBatch(ctx, o.ID, b.ID, "salah"); err != nil {
		t.Fatal(err)
	}
	if f.stock(f.rm1) != rm1 {
		t.Fatal("issued materials were never consumed by the batch, so none come back")
	}
	if got := f.uc.productRepo.(memProducts).byID[f.fg].CostPrice; got != 10_000 {
		t.Fatalf("cost price should return to 10,000, got %v", got)
	}
	orig, rev := f.ledger.entries[batchSourcePrefix+b.ID.String()], f.ledger.entries[batchVoidSourcePrefix+b.ID.String()]
	if len(orig) != 2 || len(rev) != 2 || rev[0].Debit != orig[0].Credit || rev[1].Credit != orig[1].Debit {
		t.Fatalf("journal reversal: %+v vs %+v", rev, orig)
	}
}
