package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when INVENTORY_TEST_DSN is set (a scratch database).
func TestBatchRepositoryAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("INVENTORY_TEST_DSN")
	if dsn == "" {
		t.Skip("INVENTORY_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.StockBatch{}, &domain.StockMovement{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewInventoryRepository(db)
	p, w := uuid.New(), uuid.New()
	mk := func(no, exp string, qty int) *domain.StockBatch {
		b := &domain.StockBatch{ProductID: p, WarehouseID: w, BatchNo: no, ExpiryDate: exp, InitialQty: qty, Quantity: qty}
		if err := repo.SaveBatch(ctx, b); err != nil {
			t.Fatal(err)
		}
		return b
	}
	late, soon, none, empty := mk("LATE", "2027-06-01", 5), mk("SOON", "2026-11-01", 3), mk("NOEXP", "", 2), mk("EMPTY", "2026-10-01", 0)

	fefo, err := repo.ListBatchesFEFO(ctx, p, w)
	if err != nil || len(fefo) != 3 || fefo[0].BatchNo != "SOON" || fefo[1].BatchNo != "LATE" || fefo[2].BatchNo != "NOEXP" {
		t.Fatalf("FEFO order (empty lots excluded, no-expiry last): %+v %v", fefo, err)
	}
	if all, _ := repo.ListBatches(ctx, &p, &w, true); len(all) != 4 {
		t.Fatalf("include empty: %d", len(all))
	}
	if open, _ := repo.ListBatches(ctx, &p, nil, false); len(open) != 3 {
		t.Fatalf("open lots: %d", len(open))
	}
	if exp, _ := repo.ListBatchesExpiringBy(ctx, "2026-12-31"); len(exp) != 1 || exp[0].BatchNo != "SOON" {
		t.Fatalf("expiring by year end (lots with stock only): %+v", exp)
	}
	got, err := repo.GetBatchByNo(ctx, p, w, "LATE")
	if err != nil || got == nil || got.ID != late.ID {
		t.Fatalf("by number: %+v %v", got, err)
	}
	if miss, _ := repo.GetBatchByNo(ctx, p, w, "NOPE"); miss != nil {
		t.Fatal("unknown lot must be nil")
	}
	// The same lot number in the same warehouse is unique.
	dup := &domain.StockBatch{ProductID: p, WarehouseID: w, BatchNo: "SOON", InitialQty: 1, Quantity: 1}
	if err := repo.SaveBatch(ctx, dup); err == nil {
		t.Fatal("duplicate (product, warehouse, batch number) must be rejected by the database")
	}
	// Update in place.
	soon.Quantity = 1
	if err := repo.SaveBatch(ctx, soon); err != nil {
		t.Fatal(err)
	}
	if b, _ := repo.GetBatch(ctx, soon.ID); b.Quantity != 1 {
		t.Fatalf("update: %+v", b)
	}
	_, _ = none, empty
}
