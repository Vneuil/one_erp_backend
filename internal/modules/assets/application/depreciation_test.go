package application

import (
	"context"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/assets/domain"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/google/uuid"
)

type fakeAssetsRepo struct {
	domain.AssetsRepository
	assets   []domain.FixedAsset
	postings []domain.DepreciationPosting
}

func (r *fakeAssetsRepo) ListAllAssets(context.Context) ([]domain.FixedAsset, error) {
	return append([]domain.FixedAsset(nil), r.assets...), nil
}
func (r *fakeAssetsRepo) ListDepreciationPostings(_ context.Context, period string) ([]domain.DepreciationPosting, error) {
	var out []domain.DepreciationPosting
	for _, p := range r.postings {
		if period == "" || p.Period == period {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *fakeAssetsRepo) CreateDepreciationPosting(_ context.Context, p *domain.DepreciationPosting) error {
	r.postings = append(r.postings, *p)
	return nil
}

type fakeLedger struct {
	financeApp.LedgerPoster
	docs map[string][]financeApp.LedgerLine
}

func (l *fakeLedger) PostEntryOn(_ context.Context, _, doc, _ string, lines []financeApp.LedgerLine) error {
	if _, dup := l.docs[doc]; !dup {
		l.docs[doc] = lines
	}
	return nil
}

func monthsAgo(n int) string {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), 15, 0, 0, 0, 0, time.UTC).AddDate(0, -n, 0).Format("2006-01-02")
}

func newDepUC(a ...domain.FixedAsset) (AssetsUseCase, *fakeAssetsRepo, *fakeLedger) {
	repo := &fakeAssetsRepo{assets: a}
	led := &fakeLedger{docs: map[string][]financeApp.LedgerLine{}}
	return NewAssetsUseCase(repo, WithLedger(led)), repo, led
}

func asset(price float64, years int, purchased string) domain.FixedAsset {
	a := domain.FixedAsset{AssetCode: "A1", Name: "Machine", PurchasePrice: price, UsefulLifeYears: years, PurchaseDate: purchased, Status: "In Use"}
	a.ID = uuid.New()
	return a
}

func TestPostDepreciationSingleMonthIsIdempotent(t *testing.T) {
	uc, repo, led := newDepUC(asset(12000, 1, monthsAgo(2)))
	period := time.Now().Format("2006-01")

	res, err := uc.PostDepreciation(context.Background(), period, false)
	if err != nil || res.PostedCount != 1 || res.TotalAmount != 1000 {
		t.Fatalf("first post: %+v err=%v", res, err)
	}
	lines := led.docs["depreciation:"+repo.assets[0].ID.String()+":"+period]
	if len(lines) != 2 || lines[0].Debit != 1000 || lines[1].Credit != 1000 ||
		lines[0].AccountCode != financeApp.AccountDepreciationExpense || lines[1].AccountCode != financeApp.AccountAccumDepreciation {
		t.Fatalf("bad ledger lines: %+v", lines)
	}
	res, _ = uc.PostDepreciation(context.Background(), period, false)
	if res.PostedCount != 0 || res.AlreadyDone != 1 || len(repo.postings) != 1 {
		t.Fatalf("second post should be a no-op: %+v postings=%d", res, len(repo.postings))
	}
}

func TestPostDepreciationCatchUpStopsAtCost(t *testing.T) {
	// 3-month life bought 2 months ago: months 0..2 exist, all three get posted, then nothing more.
	uc, repo, _ := newDepUC(asset(1000, 0, monthsAgo(2)))
	repo.assets[0].UsefulLifeYears = 1
	repo.assets[0].PurchasePrice = 1001 // 83.42/month → not evenly divisible, last month must be capped
	period := time.Now().Format("2006-01")

	res, err := uc.PostDepreciation(context.Background(), period, true)
	if err != nil || res.PostedCount != 3 {
		t.Fatalf("catch-up should post 3 months: %+v err=%v", res, err)
	}

	// A fully depreciated asset posts nothing further.
	full := asset(120, 1, monthsAgo(11))
	uc2, repo2, _ := newDepUC(full)
	for i := 0; i < 12; i++ {
		m := time.Now().AddDate(0, -11+i, 0).Format("2006-01")
		repo2.postings = append(repo2.postings, domain.DepreciationPosting{AssetID: full.ID, Period: m, Amount: 10})
	}
	res, _ = uc2.PostDepreciation(context.Background(), time.Now().Format("2006-01"), true)
	if res.PostedCount != 0 {
		t.Fatalf("fully depreciated asset must not post: %+v", res)
	}
}

func TestPostDepreciationSkipsDisposedAndFutureAssets(t *testing.T) {
	disposed := asset(1200, 1, monthsAgo(1))
	disposed.Status = "Disposed"
	future := asset(1200, 1, time.Now().AddDate(0, 3, 0).Format("2006-01-02"))
	uc, _, _ := newDepUC(disposed, future)

	res, err := uc.PostDepreciation(context.Background(), time.Now().Format("2006-01"), true)
	if err != nil || res.PostedCount != 0 {
		t.Fatalf("nothing should post: %+v err=%v", res, err)
	}
	if _, err := uc.PostDepreciation(context.Background(), time.Now().AddDate(0, 2, 0).Format("2006-01"), false); err == nil {
		t.Fatal("future period must be rejected")
	}
	if _, err := uc.PostDepreciation(context.Background(), "2026/01", false); err == nil {
		t.Fatal("bad period format must be rejected")
	}
}

func TestDepreciationReportBookValue(t *testing.T) {
	a := asset(12000, 1, monthsAgo(1))
	uc, repo, _ := newDepUC(a)
	period := time.Now().Format("2006-01")
	if _, err := uc.PostDepreciation(context.Background(), period, true); err != nil {
		t.Fatal(err)
	}
	rep, err := uc.DepreciationReport(context.Background(), period)
	if err != nil || len(rep.Lines) != 1 {
		t.Fatalf("report: %+v err=%v", rep, err)
	}
	l := rep.Lines[0]
	if l.PostedInPeriod != 1000 || l.AccumulatedPosted != 2000 || l.BookValue != 10000 || l.UnpostedMonths != 0 {
		t.Fatalf("bad line: %+v", l)
	}
	_ = repo
}
