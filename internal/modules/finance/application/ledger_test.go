package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

type fakeLedgerRepo struct {
	domain.FinanceRepository
	receivables map[string]*domain.Receivable
	accounts    []domain.Account
	entries     map[string]*domain.JournalEntry
}

func (f *fakeLedgerRepo) ListAllAccounts(context.Context) ([]domain.Account, error) {
	return f.accounts, nil
}
func (f *fakeLedgerRepo) GetJournalEntryBySourceDoc(_ context.Context, s string) (*domain.JournalEntry, error) {
	return f.entries[s], nil
}
func (f *fakeLedgerRepo) CreateJournalEntry(_ context.Context, je *domain.JournalEntry) error {
	f.entries[je.SourceDoc] = je
	return nil
}

func (f *fakeLedgerRepo) CreateAccount(_ context.Context, a *domain.Account) error {
	a.ID = uuid.New()
	f.accounts = append(f.accounts, *a)
	return nil
}

func newFakeLedgerRepo() *fakeLedgerRepo {
	r := &fakeLedgerRepo{entries: map[string]*domain.JournalEntry{}}
	for _, c := range []string{AccountCash, AccountReceivble, AccountRevenue} {
		r.accounts = append(r.accounts, domain.Account{ID: uuid.New(), Code: c})
	}
	return r
}

func TestLedgerPosterPostsBalancedEntryOnce(t *testing.T) {
	repo := newFakeLedgerRepo()
	p := NewLedgerPoster(repo)
	lines := []LedgerLine{{AccountCode: AccountReceivble, Debit: 100}, {AccountCode: AccountRevenue, Credit: 100}}

	if err := p.PostEntry(context.Background(), "doc-1", "memo", lines); err != nil {
		t.Fatal(err)
	}
	if err := p.PostEntry(context.Background(), "doc-1", "memo", lines); err != nil {
		t.Fatal(err)
	}
	if len(repo.entries) != 1 || repo.entries["doc-1"].Status != "posted" {
		t.Fatalf("expected one posted entry, got %+v", repo.entries)
	}
}

func TestLedgerPosterRejectsUnbalancedAndUnknownAccount(t *testing.T) {
	p := NewLedgerPoster(newFakeLedgerRepo())
	if err := p.PostEntry(context.Background(), "a", "", []LedgerLine{{AccountCode: AccountCash, Debit: 50}, {AccountCode: AccountRevenue, Credit: 40}}); err == nil {
		t.Fatal("expected unbalanced error")
	}
	if err := p.PostEntry(context.Background(), "b", "", []LedgerLine{{AccountCode: "9999", Debit: 1}, {AccountCode: AccountRevenue, Credit: 1}}); err == nil {
		t.Fatal("expected unknown account error")
	}
}

func TestLedgerPosterCreatesMissingStandardAccount(t *testing.T) {
	repo := newFakeLedgerRepo() // has no Inventory (1200) or Payable (2000)
	p := NewLedgerPoster(repo)
	err := p.PostEntry(context.Background(), "pinv-1", "", []LedgerLine{{AccountCode: AccountInventory, Debit: 10}, {AccountCode: AccountPayable, Credit: 10}})
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, a := range repo.accounts {
		codes[a.Code] = true
	}
	if !codes[AccountInventory] || !codes[AccountPayable] {
		t.Fatalf("expected standard accounts to be created, have %v", codes)
	}
}

func TestLedgerPosterDropsZeroLines(t *testing.T) {
	repo := newFakeLedgerRepo()
	p := NewLedgerPoster(repo)
	err := p.PostEntry(context.Background(), "payroll-1", "", []LedgerLine{
		{AccountCode: AccountSalaryExpense, Debit: 100},
		{AccountCode: AccountSalaryPayable, Credit: 100},
		{AccountCode: AccountTaxPayable, Credit: 0},
		{AccountCode: AccountCoopPayable, Credit: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := len(repo.entries["payroll-1"].Lines); n != 2 {
		t.Fatalf("expected 2 lines after dropping zero amounts, got %d", n)
	}
}

func (f *fakeLedgerRepo) GetReceivableBySourceDoc(_ context.Context, s string) (*domain.Receivable, error) {
	return f.receivables[s], nil
}
func (f *fakeLedgerRepo) CreateReceivable(_ context.Context, r *domain.Receivable) error {
	r.ID = uuid.New()
	f.receivables[r.SourceDoc] = r
	return nil
}
func (f *fakeLedgerRepo) UpdateReceivable(_ context.Context, r *domain.Receivable) error {
	f.receivables[r.SourceDoc] = r
	return nil
}

func TestUpsertReceivableCreatesThenRefreshes(t *testing.T) {
	repo := newFakeLedgerRepo()
	repo.receivables = map[string]*domain.Receivable{}
	p := NewLedgerPoster(repo)
	d := SubledgerDoc{SourceDoc: "sales-invoice:1", PartyName: "Acme", DocNo: "INV-1", DueDate: "2999-01-01", Total: 100}

	if err := p.UpsertReceivable(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	if r := repo.receivables["sales-invoice:1"]; r.Status != "pending" || r.TotalInvoice != 100 {
		t.Fatalf("unexpected receivable after create: %+v", r)
	}
	id := repo.receivables["sales-invoice:1"].ID

	d.Paid = 100
	if err := p.UpsertReceivable(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	r := repo.receivables["sales-invoice:1"]
	if r.ID != id || len(repo.receivables) != 1 || r.Status != "paid" || r.PaidAmount != 100 {
		t.Fatalf("expected the same row refreshed to paid, got %+v", r)
	}
}
