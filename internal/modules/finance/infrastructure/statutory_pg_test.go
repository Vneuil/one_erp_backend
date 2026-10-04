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

// Runs against a real Postgres when FINANCE_TEST_DSN is set (point it at an EMPTY scratch database: it asserts exact ledger balances).
func TestCashVoucherAndLedgerAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("FINANCE_TEST_DSN")
	if dsn == "" {
		t.Skip("FINANCE_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Account{}, &domain.JournalEntry{}, &domain.JournalLine{}, &domain.CashVoucher{}, &domain.CashVoucherLine{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewFinanceRepository(db)
	uc := application.NewFinanceUseCase(repo)
	if err := uc.SeedInitialData(ctx); err != nil {
		t.Fatal(err)
	}

	in := func(date string, amt float64) application.CashVoucherInput {
		return application.CashVoucherInput{Type: "receipt", Date: date, Counterparty: "Budi",
			Lines: []application.CashVoucherLineInput{{AccountCode: application.AccountRevenue, Amount: amt}}}
	}
	for _, v := range []application.CashVoucherInput{in("2026-02-20", 1000), in("2026-03-02", 500)} {
		if _, err := uc.CreateCashVoucher(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	pay, err := uc.CreateCashVoucher(ctx, application.CashVoucherInput{Type: "payment", Date: "2026-03-10",
		Lines: []application.CashVoucherLineInput{{AccountCode: application.AccountExpense, Amount: 200}}})
	if err != nil || !pay.Posted {
		t.Fatalf("payment voucher: %+v err=%v", pay, err)
	}

	got, err := uc.GetCashVoucher(ctx, pay.ID)
	if err != nil || len(got.Lines) != 1 || got.Number != "BKK-202603-0001" {
		t.Fatalf("get voucher: %+v err=%v", got, err)
	}
	list, err := uc.ListCashVouchers(ctx, "receipt", "2026-03-01", "2026-03-31")
	if err != nil || len(list) != 1 {
		t.Fatalf("list vouchers: %d err=%v", len(list), err)
	}

	cb, err := uc.CashBook(ctx, "", "2026-03-01", "2026-03-31")
	if err != nil || len(cb.Accounts) != 1 {
		t.Fatalf("cash book: %+v err=%v", cb, err)
	}
	a := cb.Accounts[0]
	if a.Opening != 1000 || a.Closing != 1300 || len(a.Lines) != 2 || a.Lines[1].Balance != 1300 {
		t.Fatalf("cash ledger wrong: %+v", a)
	}
	if len(cb.Days) != 2 || cb.Days[1].Closing != 1300 {
		t.Fatalf("days wrong: %+v", cb.Days)
	}
	gl, err := uc.GeneralLedger(ctx, nil, "2026-03-01", "2026-03-31")
	if err != nil || len(gl.Accounts) != 3 {
		t.Fatalf("general ledger: %d accounts err=%v", len(gl.Accounts), err)
	}
	eb, err := uc.ExpenseBreakdown(ctx, "2026-03-01", "2026-03-31")
	if err != nil || eb.Total != 200 {
		t.Fatalf("expense breakdown: %+v err=%v", eb, err)
	}
}
