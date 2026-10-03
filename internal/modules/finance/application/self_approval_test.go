package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/google/uuid"
)

type fakePettyRepo struct {
	domain.FinanceRepository
	fund *domain.PettyCashFund
	tx   *domain.PettyCashTransaction
}

func (f *fakePettyRepo) GetPettyCashFundByID(context.Context, uuid.UUID) (*domain.PettyCashFund, error) {
	return f.fund, nil
}
func (f *fakePettyRepo) GetPettyCashTransactionByID(context.Context, uuid.UUID) (*domain.PettyCashTransaction, error) {
	return f.tx, nil
}
func (f *fakePettyRepo) UpdatePettyCashTransaction(context.Context, *domain.PettyCashTransaction) error {
	return nil
}
func (f *fakePettyRepo) UpdatePettyCashFund(context.Context, *domain.PettyCashFund) error { return nil }
func (f *fakePettyRepo) ListPettyCashTransactions(context.Context, uuid.UUID) ([]domain.PettyCashTransaction, error) {
	return nil, nil
}

func TestCannotApproveOwnPettyCashTransaction(t *testing.T) {
	fund := &domain.PettyCashFund{CurrentBalance: 1000}
	fund.ID = uuid.New()
	tx := &domain.PettyCashTransaction{FundID: fund.ID, Type: "out", Amount: 100, ApprovalStatus: "pending", CreatedByEmail: "ani@x.com"}
	tx.ID = uuid.New()
	uc := NewFinanceUseCase(&fakePettyRepo{fund: fund, tx: tx})

	own := actor.WithEmail(context.Background(), "ANI@x.com")
	if _, err := uc.ApprovePettyCashTransaction(own, fund.ID, tx.ID); err == nil {
		t.Fatal("creator must not approve their own petty cash transaction")
	}
	if tx.ApprovalStatus != "pending" || fund.CurrentBalance != 1000 {
		t.Fatalf("blocked approval changed state: %s / %v", tx.ApprovalStatus, fund.CurrentBalance)
	}
	if _, err := uc.RejectPettyCashTransaction(own, fund.ID, tx.ID); err != nil {
		t.Fatalf("withdrawing your own entry should be allowed: %v", err)
	}

	tx.ApprovalStatus = "pending"
	boss := actor.WithEmail(context.Background(), "boss@x.com")
	if _, err := uc.ApprovePettyCashTransaction(boss, fund.ID, tx.ID); err != nil {
		t.Fatalf("another approver must be able to approve: %v", err)
	}
}
