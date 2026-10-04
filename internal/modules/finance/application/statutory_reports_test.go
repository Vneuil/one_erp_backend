package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

// memRepo is an in-memory ledger: journal entries saved through the poster are
// what the reports read back, so these tests cover voucher → ledger → report.
type memRepo struct {
	domain.FinanceRepository
	accounts []domain.Account
	entries  []*domain.JournalEntry
	vouchers []*domain.CashVoucher
}

func newMemRepo(accs ...domain.Account) *memRepo {
	r := &memRepo{}
	for _, a := range accs {
		a.ID = uuid.New()
		a.IsActive = true
		r.accounts = append(r.accounts, a)
	}
	return r
}

func (r *memRepo) ListAllAccounts(context.Context) ([]domain.Account, error) { return r.accounts, nil }
func (r *memRepo) CreateAccount(_ context.Context, a *domain.Account) error {
	a.ID = uuid.New()
	r.accounts = append(r.accounts, *a)
	return nil
}
func (r *memRepo) GetJournalEntryBySourceDoc(_ context.Context, s string) (*domain.JournalEntry, error) {
	for _, e := range r.entries {
		if e.SourceDoc == s {
			return e, nil
		}
	}
	return nil, nil
}
func (r *memRepo) CreateJournalEntry(_ context.Context, je *domain.JournalEntry) error {
	r.entries = append(r.entries, je)
	return nil
}
func (r *memRepo) SumPostedByAccount(_ context.Context, from, to string) ([]domain.JournalActualByAccount, error) {
	m := map[uuid.UUID]*domain.JournalActualByAccount{}
	for _, e := range r.entries {
		if (from != "" && e.Date < from) || (to != "" && e.Date > to) {
			continue
		}
		for _, l := range e.Lines {
			s := m[l.AccountID]
			if s == nil {
				s = &domain.JournalActualByAccount{AccountID: l.AccountID}
				m[l.AccountID] = s
			}
			s.TotalDebit += l.Debit
			s.TotalCredit += l.Credit
		}
	}
	var out []domain.JournalActualByAccount
	for _, s := range m {
		out = append(out, *s)
	}
	return out, nil
}
func (r *memRepo) ListPostedLines(_ context.Context, from, to string, ids []uuid.UUID) ([]domain.JournalLineDetail, error) {
	want := map[uuid.UUID]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []domain.JournalLineDetail
	for _, e := range r.entries {
		if (from != "" && e.Date < from) || (to != "" && e.Date > to) {
			continue
		}
		for _, l := range e.Lines {
			if ids == nil || want[l.AccountID] {
				out = append(out, domain.JournalLineDetail{AccountID: l.AccountID, Date: e.Date, EntryNumber: e.EntryNumber,
					Memo: e.Memo, SourceDoc: e.SourceDoc, Description: l.Description, Debit: l.Debit, Credit: l.Credit})
			}
		}
	}
	return out, nil
}
func (r *memRepo) CountCashVouchers(_ context.Context, typ, prefix string) (int64, error) {
	var n int64
	for _, v := range r.vouchers {
		if v.Type == typ && strings.HasPrefix(v.Number, prefix) {
			n++
		}
	}
	return n, nil
}
func (r *memRepo) CreateCashVoucher(_ context.Context, v *domain.CashVoucher) error {
	r.vouchers = append(r.vouchers, v)
	return nil
}
func (r *memRepo) UpdateCashVoucher(context.Context, *domain.CashVoucher) error { return nil }

func testRepo() *memRepo {
	return newMemRepo(
		domain.Account{Code: AccountCash, Name: "Cash", Type: "asset"},
		domain.Account{Code: "1010", Name: "Bank BCA", Type: "asset"},
		domain.Account{Code: AccountRevenue, Name: "Sales", Type: "revenue"},
		domain.Account{Code: AccountExpense, Name: "Operating", Type: "expense"},
		domain.Account{Code: AccountDeliveryExpense, Name: "Delivery", Type: "expense"},
		domain.Account{Code: AccountOtherIncome, Name: "Other Income", Type: "revenue"},
		domain.Account{Code: AccountRounding, Name: "Rounding", Type: "expense"},
	)
}

func voucher(t *testing.T, uc FinanceUseCase, typ, date, cash string, lines ...CashVoucherLineInput) *domain.CashVoucher {
	t.Helper()
	v, err := uc.CreateCashVoucher(context.Background(), CashVoucherInput{Type: typ, Date: date, CashAccountCode: cash, Lines: lines})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCashVoucherPostsBalancedEntryAndNumbers(t *testing.T) {
	repo := testRepo()
	uc := NewFinanceUseCase(repo)
	a := voucher(t, uc, "receipt", "2026-03-02", "1010", CashVoucherLineInput{AccountCode: AccountRevenue, Amount: 500})
	b := voucher(t, uc, "receipt", "2026-03-09", "1010", CashVoucherLineInput{AccountCode: AccountRevenue, Amount: 100})
	p := voucher(t, uc, "payment", "2026-03-10", "1010",
		CashVoucherLineInput{AccountCode: AccountExpense, Amount: 60}, CashVoucherLineInput{AccountCode: AccountDeliveryExpense, Amount: 40})

	if a.Number != "BKM-202603-0001" || b.Number != "BKM-202603-0002" || p.Number != "BKK-202603-0001" {
		t.Fatalf("unexpected numbers %s %s %s", a.Number, b.Number, p.Number)
	}
	if !p.Posted || p.Total != 100 || len(repo.entries) != 3 {
		t.Fatalf("voucher not posted correctly: %+v entries=%d", p, len(repo.entries))
	}
}

func TestCashVoucherValidation(t *testing.T) {
	uc := NewFinanceUseCase(testRepo())
	ctx := context.Background()
	line := CashVoucherLineInput{AccountCode: AccountRevenue, Amount: 10}
	cases := map[string]CashVoucherInput{
		"bad type":        {Type: "x", Lines: []CashVoucherLineInput{line}},
		"no lines":        {Type: "receipt"},
		"non-cash":        {Type: "receipt", CashAccountCode: AccountRevenue, Lines: []CashVoucherLineInput{line}},
		"unknown counter": {Type: "receipt", Lines: []CashVoucherLineInput{{AccountCode: "9999", Amount: 10}}},
		"self counter":    {Type: "receipt", Lines: []CashVoucherLineInput{{AccountCode: AccountCash, Amount: 10}}},
		"zero amount":     {Type: "receipt", Lines: []CashVoucherLineInput{{AccountCode: AccountRevenue}}},
		"future date":     {Type: "receipt", Date: "2999-01-01", Lines: []CashVoucherLineInput{line}},
	}
	for name, in := range cases {
		if _, err := uc.CreateCashVoucher(ctx, in); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestGeneralLedgerRunningBalanceAndOpening(t *testing.T) {
	repo := testRepo()
	uc := NewFinanceUseCase(repo)
	voucher(t, uc, "receipt", "2026-02-20", "1010", CashVoucherLineInput{AccountCode: AccountRevenue, Amount: 1000})
	voucher(t, uc, "receipt", "2026-03-02", "1010", CashVoucherLineInput{AccountCode: AccountRevenue, Amount: 500})
	voucher(t, uc, "payment", "2026-03-10", "1010", CashVoucherLineInput{AccountCode: AccountExpense, Amount: 200})

	var bankID uuid.UUID
	for _, a := range repo.accounts {
		if a.Code == "1010" {
			bankID = a.ID
		}
	}
	gl, err := uc.GeneralLedger(context.Background(), &bankID, "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(gl.Accounts) != 1 {
		t.Fatalf("want 1 account, got %d", len(gl.Accounts))
	}
	acc := gl.Accounts[0]
	if acc.Opening != 1000 || acc.TotalDebit != 500 || acc.TotalCredit != 200 || acc.Closing != 1300 {
		t.Fatalf("bad ledger totals: %+v", acc)
	}
	if len(acc.Lines) != 2 || acc.Lines[0].Balance != 1500 || acc.Lines[1].Balance != 1300 {
		t.Fatalf("bad running balance: %+v", acc.Lines)
	}

	// Credit-normal accounts grow on credits.
	all, _ := uc.GeneralLedger(context.Background(), nil, "2026-03-01", "2026-03-31")
	for _, a := range all.Accounts {
		if a.Code == AccountRevenue && (a.Opening != 1000 || a.Closing != 1500) {
			t.Fatalf("revenue ledger wrong: %+v", a)
		}
	}
	if _, err := uc.GeneralLedger(context.Background(), nil, "2026-04-01", "2026-03-01"); err == nil {
		t.Fatal("expected error for inverted range")
	}
}

func TestCashBookDailySummary(t *testing.T) {
	repo := testRepo()
	uc := NewFinanceUseCase(repo)
	voucher(t, uc, "receipt", "2026-02-20", "1010", CashVoucherLineInput{AccountCode: AccountRevenue, Amount: 1000})
	voucher(t, uc, "receipt", "2026-03-02", "1000", CashVoucherLineInput{AccountCode: AccountRevenue, Amount: 500})
	voucher(t, uc, "payment", "2026-03-02", "1010", CashVoucherLineInput{AccountCode: AccountExpense, Amount: 200})
	voucher(t, uc, "payment", "2026-03-05", "1010", CashVoucherLineInput{AccountCode: AccountExpense, Amount: 100})

	cb, err := uc.CashBook(context.Background(), "", "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(cb.Days) != 2 {
		t.Fatalf("want 2 active days, got %+v", cb.Days)
	}
	d1, d2 := cb.Days[0], cb.Days[1]
	if d1.Opening != 1000 || d1.In != 500 || d1.Out != 200 || d1.Closing != 1300 || d2.Opening != 1300 || d2.Closing != 1200 {
		t.Fatalf("bad daily summary: %+v", cb.Days)
	}
	if _, err := uc.CashBook(context.Background(), AccountRevenue, "2026-03-01", "2026-03-31"); err == nil {
		t.Fatal("expected error for non-cash account")
	}
}

func TestExpenseBreakdownAndNonOperating(t *testing.T) {
	repo := testRepo()
	uc := NewFinanceUseCase(repo)
	voucher(t, uc, "payment", "2026-03-10", "1000",
		CashVoucherLineInput{AccountCode: AccountExpense, Amount: 300},
		CashVoucherLineInput{AccountCode: AccountDeliveryExpense, Amount: 100},
		CashVoucherLineInput{AccountCode: AccountRounding, Amount: 5})
	voucher(t, uc, "receipt", "2026-03-11", "1000", CashVoucherLineInput{AccountCode: AccountOtherIncome, Amount: 50})

	eb, err := uc.ExpenseBreakdown(context.Background(), "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(eb.Groups) != 2 || eb.Groups[0].Total != 100 || eb.Groups[1].Total != 300 || eb.Total != 400 {
		t.Fatalf("bad breakdown (rounding must be excluded): %+v", eb)
	}
	no, _ := uc.NonOperating(context.Background(), "2026-03-01", "2026-03-31")
	if no.TotalIncome != 50 || no.TotalExpenses != 5 || no.Net != 45 {
		t.Fatalf("bad non-operating: %+v", no)
	}
}

func TestClassifyAccountExplicitCategoryWins(t *testing.T) {
	a := domain.Account{Code: "5000", Type: "expense", Category: CategoryMarketing}
	if ClassifyAccount(a) != CategoryMarketing {
		t.Fatal("explicit category should win")
	}
	if ClassifyAccount(domain.Account{Code: "1000", Type: "asset", Category: "marketing"}) != "" {
		t.Fatal("balance-sheet accounts have no category")
	}
}
