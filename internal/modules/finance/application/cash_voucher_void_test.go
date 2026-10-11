package application

import (
	"context"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

func (r *memRepo) GetCashVoucherByID(_ context.Context, id uuid.UUID) (*domain.CashVoucher, error) {
	for _, v := range r.vouchers {
		if v.ID == id {
			return v, nil
		}
	}
	return nil, nil
}
func (r *memRepo) VoidCashVoucher(context.Context, *domain.CashVoucher) error { return nil }

func TestVoidCashVoucherReversesTheJournalAndNetsToZero(t *testing.T) {
	repo := testRepo()
	uc := NewFinanceUseCase(repo)
	ctx := context.Background()
	v := voucher(t, uc, "payment", "2026-03-10", "1010",
		CashVoucherLineInput{AccountCode: AccountExpense, Amount: 60}, CashVoucherLineInput{AccountCode: AccountDeliveryExpense, Amount: 40})

	if _, err := uc.VoidCashVoucher(ctx, v.ID, " "); err == nil {
		t.Fatal("a reason is required")
	}
	out, err := uc.VoidCashVoucher(ctx, v.ID, "salah akun")
	if err != nil {
		t.Fatal(err)
	}
	if !out.Voided || out.VoidReason != "salah akun" || out.VoidedAt == nil {
		t.Fatalf("voucher: %+v", out)
	}
	if len(repo.entries) != 2 {
		t.Fatalf("expected the original and one reversal, got %d entries", len(repo.entries))
	}
	rev := repo.entries[1]
	if rev.SourceDoc != CashVoucherVoidSourcePrefix+v.ID.String() || rev.Date != time.Now().Format("2006-01-02") {
		t.Fatalf("reversal entry: %+v", rev)
	}
	// Every account nets to zero across the two entries.
	net := map[uuid.UUID]float64{}
	for _, e := range repo.entries {
		for _, l := range e.Lines {
			net[l.AccountID] += l.Debit - l.Credit
		}
	}
	for id, n := range net {
		if n != 0 {
			t.Fatalf("account %s nets to %v", id, n)
		}
	}
	if _, err := uc.VoidCashVoucher(ctx, v.ID, "lagi"); err == nil {
		t.Fatal("a voucher can only be voided once")
	}
	if _, err := uc.VoidCashVoucher(ctx, uuid.New(), "x"); err == nil {
		t.Fatal("unknown voucher")
	}
}

func (r *memRepo) GetJournalEntryByID(_ context.Context, id uuid.UUID) (*domain.JournalEntry, error) {
	for _, e := range r.entries {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}

func TestReverseEntryMirrorsAnyPostedEntryOnce(t *testing.T) {
	repo := testRepo()
	p := NewLedgerPoster(repo)
	ctx := context.Background()
	if err := p.PostEntryOn(ctx, "2026-03-05", "doc-1", "Original", []LedgerLine{{AccountCode: AccountExpense, Debit: 70}, {AccountCode: AccountCash, Credit: 70}}); err != nil {
		t.Fatal(err)
	}
	repo.entries[0].ID = uuid.New()

	if ok, err := p.ReverseEntry(ctx, "no-such-doc", "rev-x", "", "x"); err != nil || ok {
		t.Fatalf("nothing to reverse: %v %v", ok, err)
	}
	if ok, err := p.ReverseEntry(ctx, "doc-1", "rev-1", "2026-04-01", "Batal"); err != nil || !ok {
		t.Fatalf("reverse: %v %v", ok, err)
	}
	if ok, err := p.ReverseEntry(ctx, "doc-1", "rev-1", "2026-04-02", "Batal"); err != nil || !ok {
		t.Fatalf("reversing twice is a no-op: %v %v", ok, err)
	}
	if len(repo.entries) != 2 {
		t.Fatalf("exactly one reversal expected, got %d entries", len(repo.entries))
	}
	rev := repo.entries[1]
	if rev.SourceDoc != "rev-1" || rev.Date != "2026-04-01" || rev.Status != "posted" {
		t.Fatalf("reversal header: %+v", rev)
	}
	for i, l := range rev.Lines {
		o := repo.entries[0].Lines[i]
		if l.AccountID != o.AccountID || l.Debit != o.Credit || l.Credit != o.Debit {
			t.Fatalf("line %d is not the mirror: %+v vs %+v", i, l, o)
		}
	}
}
