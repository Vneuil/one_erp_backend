package application

import (
	"context"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

type fakeFundsRepo struct {
	*fakeLedgerRepo
	capital []*domain.CapitalTransaction
	income  []*domain.OtherIncome
}

func (f *fakeFundsRepo) CreateCapitalTransaction(_ context.Context, c *domain.CapitalTransaction) error {
	f.capital = append(f.capital, c)
	return nil
}
func (f *fakeFundsRepo) UpdateCapitalTransaction(context.Context, *domain.CapitalTransaction) error {
	return nil
}
func (f *fakeFundsRepo) CreateOtherIncome(_ context.Context, o *domain.OtherIncome) error {
	f.income = append(f.income, o)
	return nil
}
func (f *fakeFundsRepo) UpdateOtherIncome(context.Context, *domain.OtherIncome) error { return nil }

func entryTotals(e *domain.JournalEntry) (dr, cr float64) {
	for _, l := range e.Lines {
		dr += l.Debit
		cr += l.Credit
	}
	return
}

func TestCapitalInjectionAndDrawingPostBalancedEntries(t *testing.T) {
	repo := &fakeFundsRepo{fakeLedgerRepo: newFakeLedgerRepo()}
	uc := NewFinanceUseCase(repo)
	ctx := context.Background()

	inj, err := uc.RecordCapital(ctx, "injection", CapitalInput{OwnerName: "Budi", Amount: 50_000_000})
	if err != nil {
		t.Fatal(err)
	}
	if !inj.Posted {
		t.Fatal("record must be flagged as posted after its journal entry exists")
	}
	je := repo.entries[CapitalSourcePrefix+inj.ID.String()]
	if je == nil {
		t.Fatal("journal entry missing")
	}
	if dr, cr := entryTotals(je); dr != 50_000_000 || dr != cr {
		t.Fatalf("injection dr %v cr %v", dr, cr)
	}

	dr, err := uc.RecordCapital(ctx, "drawing", CapitalInput{OwnerName: "Budi", Amount: 5_000_000})
	if err != nil {
		t.Fatal(err)
	}
	je = repo.entries[CapitalSourcePrefix+dr.ID.String()]
	if d, c := entryTotals(je); d != 5_000_000 || d != c {
		t.Fatalf("drawing dr %v cr %v", d, c)
	}
	// Drawings debit the drawings account, not an expense.
	var drawingAcct uuid.UUID
	for _, a := range repo.accounts {
		if a.Code == AccountOwnerDrawings {
			drawingAcct = a.ID
		}
	}
	found := false
	for _, l := range je.Lines {
		if l.AccountID == drawingAcct && l.Debit == 5_000_000 {
			found = true
		}
	}
	if !found {
		t.Fatalf("drawing must debit %s: %+v", AccountOwnerDrawings, je.Lines)
	}
}

func TestOtherIncomePostsAgainstOtherIncomeAccount(t *testing.T) {
	repo := &fakeFundsRepo{fakeLedgerRepo: newFakeLedgerRepo()}
	o, err := NewFinanceUseCase(repo).RecordOtherIncome(context.Background(), OtherIncomeInput{Category: "Interest", Amount: 125_000.456})
	if err != nil {
		t.Fatal(err)
	}
	if o.Amount != 125_000.46 {
		t.Fatalf("amount should be rounded to sen, got %v", o.Amount)
	}
	if d, c := entryTotals(repo.entries[OtherIncomeSourcePrefix+o.ID.String()]); d != c || d != 125_000.46 {
		t.Fatalf("dr %v cr %v", d, c)
	}
}

func TestOwnerFundsValidation(t *testing.T) {
	repo := &fakeFundsRepo{fakeLedgerRepo: newFakeLedgerRepo()}
	uc := NewFinanceUseCase(repo)
	ctx := context.Background()
	future := time.Now().AddDate(0, 0, 3).Format("2006-01-02")
	bad := []func() error{
		func() error { _, e := uc.RecordCapital(ctx, "loan", CapitalInput{OwnerName: "B", Amount: 1}); return e },
		func() error {
			_, e := uc.RecordCapital(ctx, "injection", CapitalInput{OwnerName: "", Amount: 1})
			return e
		},
		func() error {
			_, e := uc.RecordCapital(ctx, "injection", CapitalInput{OwnerName: "B", Amount: 0})
			return e
		},
		func() error {
			_, e := uc.RecordCapital(ctx, "injection", CapitalInput{OwnerName: "B", Amount: 1, Date: future})
			return e
		},
		func() error {
			_, e := uc.RecordCapital(ctx, "injection", CapitalInput{OwnerName: "B", Amount: 1, PaymentAccountCode: "9999"})
			return e
		},
		func() error { _, e := uc.RecordOtherIncome(ctx, OtherIncomeInput{Category: "", Amount: 1}); return e },
	}
	for i, f := range bad {
		if f() == nil {
			t.Errorf("case %d should have been rejected", i)
		}
	}
	if len(repo.capital)+len(repo.income) != 0 || len(repo.entries) != 0 {
		t.Fatal("rejected requests must not save or post anything")
	}
}

func TestPaymentAccountMustBeAnAssetAccount(t *testing.T) {
	repo := &fakeFundsRepo{fakeLedgerRepo: newFakeLedgerRepo()}
	repo.accounts = append(repo.accounts,
		domain.Account{ID: uuid.New(), Code: "1010", Type: "asset", IsActive: true},
		domain.Account{ID: uuid.New(), Code: "2000", Type: "liability", IsActive: true})
	uc := NewFinanceUseCase(repo)
	if _, err := uc.RecordOtherIncome(context.Background(), OtherIncomeInput{Category: "Rent", Amount: 1, PaymentAccountCode: "1010"}); err != nil {
		t.Fatalf("bank asset account should be accepted: %v", err)
	}
	if _, err := uc.RecordOtherIncome(context.Background(), OtherIncomeInput{Category: "Rent", Amount: 1, PaymentAccountCode: "2000"}); err == nil {
		t.Fatal("a liability account must be refused as payment account")
	}
}

type cashFlowRepo struct {
	*fakeLedgerRepo
	sums []domain.JournalActualByAccount
}

func (f *cashFlowRepo) SumPostedByAccount(context.Context, string, string) ([]domain.JournalActualByAccount, error) {
	return f.sums, nil
}

func TestCashFlowCountsOnlyCashAndBankAccounts(t *testing.T) {
	base := newFakeLedgerRepo()
	base.accounts = []domain.Account{
		{ID: uuid.New(), Code: "1000", Type: "asset"},
		{ID: uuid.New(), Code: "1010", Type: "asset"},
		{ID: uuid.New(), Code: "1100", Type: "asset"}, // receivable: not cash
		{ID: uuid.New(), Code: "1200", Type: "asset"}, // inventory: not cash
	}
	repo := &cashFlowRepo{fakeLedgerRepo: base, sums: []domain.JournalActualByAccount{
		{AccountID: base.accounts[0].ID, TotalDebit: 1000, TotalCredit: 200},
		{AccountID: base.accounts[1].ID, TotalDebit: 500},
		{AccountID: base.accounts[2].ID, TotalDebit: 99_999, TotalCredit: 88_888},
		{AccountID: base.accounts[3].ID, TotalDebit: 77_777},
	}}
	cf, err := NewFinanceUseCase(repo).CashFlow(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if cf.CashInflow != 1500 || cf.CashOutflow != 200 || cf.NetCashFlow != 1300 {
		t.Fatalf("cash flow counted non-cash assets: %+v", cf)
	}
}
