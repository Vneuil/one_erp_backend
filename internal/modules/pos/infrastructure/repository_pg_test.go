package infrastructure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/pos/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when POS_TEST_DSN is set (a scratch database).
func TestPOSRepositoryAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("POS_TEST_DSN")
	if dsn == "" {
		t.Skip("POS_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.POSTransaction{}, &domain.POSTransactionLine{}, &domain.POSRefund{}, &domain.POSSettings{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewPOSRepository(db)

	// Settings default until saved, then persisted (one row, updated in place).
	if s, _ := repo.GetSettings(ctx); s.TaxPercent != 0 || !s.TaxInclusive || s.DefaultOutlet != "Outlet Utama" {
		t.Fatalf("default settings: %+v", s)
	}
	s, _ := repo.GetSettings(ctx)
	s.TaxPercent = 11
	if err := repo.SaveSettings(ctx, s); err != nil {
		t.Fatal(err)
	}
	s2, _ := repo.GetSettings(ctx)
	s2.TaxPercent = 12
	if err := repo.SaveSettings(ctx, s2); err != nil {
		t.Fatal(err)
	}
	var n int64
	db.Model(&domain.POSSettings{}).Count(&n)
	if got, _ := repo.GetSettings(ctx); got.TaxPercent != 12 || n != 1 {
		t.Fatalf("settings persisted: %+v rows=%d", got, n)
	}

	// A sale with lines round-trips, and updating the header does not touch the lines.
	tx := &domain.POSTransaction{OrderNo: "POS-1", Outlet: "Pusat", Cashier: "ani", TotalAmount: 30_000, Subtotal: 30_000, PaymentMethod: "cash", Status: "Completed", TotalItems: 3}
	if err := repo.Create(ctx, tx); err != nil {
		t.Fatal(err)
	}
	lines := []domain.POSTransactionLine{
		{TransactionID: tx.ID, ProductID: uuid.New(), Name: "Kopi", Quantity: 2, UnitPrice: 10_000, Subtotal: 20_000},
		{TransactionID: tx.ID, ProductID: uuid.New(), Name: "Roti", Quantity: 1, UnitPrice: 10_000, Subtotal: 10_000},
	}
	if err := repo.CreateLines(ctx, lines); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, tx.ID)
	if err != nil || len(got.Lines) != 2 {
		t.Fatalf("preloaded lines: %v %v", got, err)
	}
	got.Status, got.RefundedAmount = "Partially Refunded", 10_000
	if err := repo.Update(ctx, got); err != nil {
		t.Fatal(err)
	}
	got.Lines[0].RefundedQty = 1
	if err := repo.UpdateLine(ctx, &got.Lines[0]); err != nil {
		t.Fatal(err)
	}
	again, _ := repo.GetByID(ctx, tx.ID)
	if again.Status != "Partially Refunded" || again.RefundedAmount != 10_000 || len(again.Lines) != 2 {
		t.Fatalf("after update: %+v", again)
	}
	refunded := 0.0
	for _, l := range again.Lines {
		refunded += l.RefundedQty
	}
	if refunded != 1 {
		t.Fatalf("line refunded qty: %v", refunded)
	}

	// Refunds and range queries (Jakarta days), with the outlet filter reaching refunds.
	if err := repo.CreateRefund(ctx, &domain.POSRefund{TransactionID: tx.ID, Kind: "refund", Amount: 10_000, Reason: "rusak"}); err != nil {
		t.Fatal(err)
	}
	other := &domain.POSTransaction{OrderNo: "POS-2", Outlet: "Cabang", Cashier: "bob", TotalAmount: 5_000, PaymentMethod: "cash", Status: "Completed", TotalItems: 1}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	_ = repo.CreateRefund(ctx, &domain.POSRefund{TransactionID: other.ID, Kind: "refund", Amount: 5_000, Reason: "x"})
	today := time.Now().In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02")
	all, err := repo.ListInRange(ctx, today, today, "")
	if err != nil || len(all) != 2 {
		t.Fatalf("range: %d %v", len(all), err)
	}
	if pusat, _ := repo.ListInRange(ctx, today, today, "Pusat"); len(pusat) != 1 || len(pusat[0].Lines) != 2 {
		t.Fatalf("outlet filter with lines: %+v", pusat)
	}
	if rf, _ := repo.ListRefundsInRange(ctx, today, today, "Pusat"); len(rf) != 1 || rf[0].Amount != 10_000 {
		t.Fatalf("refunds for one outlet: %+v", rf)
	}
	if rf, _ := repo.ListRefundsInRange(ctx, today, today, ""); len(rf) != 2 {
		t.Fatalf("all refunds: %d", len(rf))
	}
	if none, _ := repo.ListInRange(ctx, "2020-01-01", "2020-01-02", ""); len(none) != 0 {
		t.Fatal("a range in the past must be empty")
	}
}
