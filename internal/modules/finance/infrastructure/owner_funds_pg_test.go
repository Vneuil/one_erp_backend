package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when FINANCE_TEST_DSN is set (point it at a scratch database).
func TestOwnerFundsAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("FINANCE_TEST_DSN")
	if dsn == "" {
		t.Skip("FINANCE_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Account{}, &domain.JournalEntry{}, &domain.JournalLine{}, &domain.CapitalTransaction{}, &domain.OtherIncome{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewFinanceRepository(db)
	uc := application.NewFinanceUseCase(repo)

	if _, err := uc.RecordCapital(ctx, "injection", application.CapitalInput{OwnerName: "Budi", Amount: 10_000_000, Date: "2026-09-01"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.RecordCapital(ctx, "drawing", application.CapitalInput{OwnerName: "Budi", Amount: 2_000_000, Date: "2026-09-15"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.RecordOtherIncome(ctx, application.OtherIncomeInput{Category: "Interest", Amount: 50_000, Date: "2026-08-31"}); err != nil {
		t.Fatal(err)
	}

	if l, _ := uc.ListCapital(ctx, "drawing", ""); len(l) != 1 || !l[0].Posted {
		t.Fatalf("drawings: %+v", l)
	}
	if l, _ := uc.ListCapital(ctx, "", "2026-09"); len(l) != 2 {
		t.Fatalf("september capital: %d", len(l))
	}
	if l, _ := uc.ListOtherIncome(ctx, "", "2026-09"); len(l) != 0 {
		t.Fatalf("interest was posted in August, got %d in September", len(l))
	}

	// The books balance and equity reflects capital minus drawings.
	bs, err := uc.BalanceSheet(ctx, "2026-12-31")
	if err != nil {
		t.Fatal(err)
	}
	// Equity: +10,000,000 injected, -2,000,000 drawn. Other income (revenue) is not equity yet.
	if bs.Equity != 8_000_000 || bs.RetainedEarnings != 50_000 {
		t.Fatalf("equity = %v earnings = %v, want 8,000,000 and 50,000", bs.Equity, bs.RetainedEarnings)
	}
	// Assets = 10,000,000 - 2,000,000 + 50,000; the sheet must balance.
	if bs.Assets != bs.TotalLiabEq {
		t.Fatalf("balance sheet does not balance: assets %v vs liabilities+equity %v", bs.Assets, bs.TotalLiabEq)
	}
	cf, err := uc.CashFlow(ctx, "2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatal(err)
	}
	if cf.CashInflow != 10_000_000 || cf.CashOutflow != 2_000_000 {
		t.Fatalf("cash flow: %+v", cf)
	}
}
