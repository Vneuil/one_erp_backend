package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/projectcost/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when PROJECTCOST_TEST_DSN is set (a scratch database).
func TestProjectCostRepositoryAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("PROJECTCOST_TEST_DSN")
	if dsn == "" {
		t.Skip("PROJECTCOST_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.BudgetItem{}, &domain.CostEntry{}, &domain.WorkOrder{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewRepository(db)
	pid, other := uuid.New(), uuid.New()
	mk := func(p uuid.UUID, kind, cat string, amt float64) domain.BudgetItem {
		return domain.BudgetItem{ProjectID: p, Kind: kind, Category: cat, Description: "d", Quantity: 1, UnitPrice: amt, Amount: amt}
	}
	if err := repo.AddBudgetItems(ctx, pid, nil, []domain.BudgetItem{mk(pid, "rab", "A", 10), mk(pid, "rap", "A", 6), mk(pid, "rab", "B", 20)}); err != nil {
		t.Fatal(err)
	}
	_ = repo.AddBudgetItems(ctx, other, nil, []domain.BudgetItem{mk(other, "rab", "Z", 99)})
	// Replace RAB only: RAP and other projects are untouched.
	if err := repo.AddBudgetItems(ctx, pid, []string{"rab"}, []domain.BudgetItem{mk(pid, "rab", "C", 5)}); err != nil {
		t.Fatal(err)
	}
	all, _ := repo.ListBudgetItems(ctx, pid, "")
	rab, _ := repo.ListBudgetItems(ctx, pid, "rab")
	if len(all) != 2 || len(rab) != 1 || rab[0].Category != "C" {
		t.Fatalf("after replace: all=%d rab=%+v", len(all), rab)
	}
	if o, _ := repo.ListBudgetItems(ctx, other, ""); len(o) != 1 {
		t.Fatalf("another project's lines must survive: %d", len(o))
	}
	// Deleting a line needs the right project.
	if ok, _ := repo.DeleteBudgetItem(ctx, other, rab[0].ID); ok {
		t.Fatal("must not delete a line through another project")
	}
	if ok, _ := repo.DeleteBudgetItem(ctx, pid, rab[0].ID); !ok {
		t.Fatal("delete own line")
	}
	// Work order numbers: unique, and deleted ones still count.
	w := &domain.WorkOrder{Number: "SPK-2026-001", CustomerName: "A", Title: "t", Status: "draft"}
	if err := repo.CreateWorkOrder(ctx, w); err != nil {
		t.Fatal(err)
	}
	dup := &domain.WorkOrder{Number: "SPK-2026-001", CustomerName: "B", Title: "t2", Status: "draft"}
	if err := repo.CreateWorkOrder(ctx, dup); err == nil {
		t.Fatal("duplicate SPK number must be rejected by the database")
	}
	db.Delete(w)
	if n, _ := repo.CountWorkOrdersWithPrefix(ctx, "SPK-2026-"); n != 1 {
		t.Fatalf("a deleted SPK must still occupy its number, count=%d", n)
	}
}
