package application

import (
	"context"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

type kbsRepo struct {
	*memRepo
	advances []*domain.TempAdvance
}

func (r *kbsRepo) CreateTempAdvance(_ context.Context, a *domain.TempAdvance) error {
	cp := *a
	r.advances = append(r.advances, &cp)
	return nil
}
func (r *kbsRepo) find(id uuid.UUID) *domain.TempAdvance {
	for _, a := range r.advances {
		if a.ID == id {
			return a
		}
	}
	return nil
}
func (r *kbsRepo) GetTempAdvanceByID(_ context.Context, id uuid.UUID) (*domain.TempAdvance, error) {
	if a := r.find(id); a != nil {
		cp := *a
		return &cp, nil
	}
	return nil, nil
}
func (r *kbsRepo) ListTempAdvances(_ context.Context, status, from, to string) ([]domain.TempAdvance, error) {
	var out []domain.TempAdvance
	for _, a := range r.advances {
		if (status != "" && a.Status != status) || (from != "" && a.Date < from) || (to != "" && a.Date > to) {
			continue
		}
		out = append(out, *a)
	}
	return out, nil
}
func (r *kbsRepo) CountTempAdvances(context.Context, string) (int64, error) {
	return int64(len(r.advances)), nil
}
func (r *kbsRepo) SaveTempAdvanceState(_ context.Context, a *domain.TempAdvance) error {
	if x := r.find(a.ID); x != nil {
		x.Status, x.Posted, x.SettledDate, x.SpentTotal = a.Status, a.Posted, a.SettledDate, a.SpentTotal
		x.ReturnedAmount, x.ExtraPaid, x.SettlementPosted, x.CancelledDate = a.ReturnedAmount, a.ExtraPaid, a.SettlementPosted, a.CancelledDate
	}
	return nil
}
func (r *kbsRepo) SettleTempAdvance(ctx context.Context, a *domain.TempAdvance) error {
	if x := r.find(a.ID); x != nil {
		x.Lines = a.Lines
	}
	return r.SaveTempAdvanceState(ctx, a)
}

func newKBS() (*kbsRepo, FinanceUseCase) {
	r := &kbsRepo{memRepo: newMemRepo(
		domain.Account{Code: AccountCash, Name: "Cash", Type: "asset"},
		domain.Account{Code: AccountExpense, Name: "Opex", Type: "expense"},
	)}
	return r, NewFinanceUseCase(r)
}

func lastEntry(r *kbsRepo, doc string) *domain.JournalEntry {
	for _, e := range r.entries {
		if e.SourceDoc == doc {
			return e
		}
	}
	return nil
}

func TestTempAdvanceSettleReturnsLeftover(t *testing.T) {
	r, uc := newKBS()
	ctx := context.Background()
	a, err := uc.IssueTempAdvance(ctx, TempAdvanceInput{RecipientName: "Budi", Amount: 1_000_000, Purpose: "Dinas"})
	if err != nil || !a.Posted {
		t.Fatalf("issue: %v %+v", err, a)
	}
	if e := lastEntry(r, tempAdvanceIssuePrefix+a.ID.String()); e == nil {
		t.Fatal("issue journal missing")
	}
	s, err := uc.SettleTempAdvance(ctx, a.ID, TempAdvanceSettleInput{Lines: []CashVoucherLineInput{{AccountCode: AccountExpense, Amount: 700_000}}})
	if err != nil {
		t.Fatal(err)
	}
	if s.Status != "settled" || s.ReturnedAmount != 300_000 || s.ExtraPaid != 0 || !s.SettlementPosted {
		t.Fatalf("settle: %+v", s)
	}
	e := lastEntry(r, tempAdvanceSettlePrefix+a.ID.String())
	if dr, cr := entryTotals(e); dr != 1_000_000 || cr != 1_000_000 {
		t.Fatalf("settlement dr %v cr %v", dr, cr)
	}
	if _, err := uc.SettleTempAdvance(ctx, a.ID, TempAdvanceSettleInput{}); err == nil {
		t.Fatal("a settled advance must not settle again")
	}
}

func TestTempAdvanceSettleOverspendPaysShortfall(t *testing.T) {
	r, uc := newKBS()
	ctx := context.Background()
	a, _ := uc.IssueTempAdvance(ctx, TempAdvanceInput{RecipientName: "Sari", Amount: 500_000})
	s, err := uc.SettleTempAdvance(ctx, a.ID, TempAdvanceSettleInput{Lines: []CashVoucherLineInput{{AccountCode: AccountExpense, Amount: 650_000}}})
	if err != nil || s.ExtraPaid != 150_000 || s.ReturnedAmount != 0 {
		t.Fatalf("overspend: %v %+v", err, s)
	}
	if dr, cr := entryTotals(lastEntry(r, tempAdvanceSettlePrefix+a.ID.String())); dr != 650_000 || cr != 650_000 {
		t.Fatalf("dr %v cr %v", dr, cr)
	}
}

func TestOutstandingReportAndCancel(t *testing.T) {
	_, uc := newKBS()
	ctx := context.Background()
	old := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	due := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	a1, _ := uc.IssueTempAdvance(ctx, TempAdvanceInput{RecipientName: "A", Amount: 100, Date: old, DueDate: due})
	a2, _ := uc.IssueTempAdvance(ctx, TempAdvanceInput{RecipientName: "B", Amount: 200})
	a3, _ := uc.IssueTempAdvance(ctx, TempAdvanceInput{RecipientName: "C", Amount: 400})
	if _, err := uc.SettleTempAdvance(ctx, a2.ID, TempAdvanceSettleInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CancelTempAdvance(ctx, a3.ID); err != nil {
		t.Fatal(err)
	}
	rep, err := uc.OutstandingTempAdvances(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Rows) != 1 || rep.Rows[0].ID != a1.ID || rep.Total != 100 || rep.OverdueTotal != 100 || rep.Rows[0].AgeDays != 10 {
		t.Fatalf("report: %+v", rep)
	}
	if _, err := uc.CancelTempAdvance(ctx, a2.ID); err == nil {
		t.Fatal("settled advance must not be cancellable")
	}
}
