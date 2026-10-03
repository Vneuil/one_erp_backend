package application

import (
	"context"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	"github.com/google/uuid"
)

type fakeClaimRepo struct {
	domain.ReimbursementRepository
	claim *domain.ReimbursementClaim
}

func (f *fakeClaimRepo) CreateClaim(_ context.Context, c *domain.ReimbursementClaim) error {
	c.ID = uuid.New()
	f.claim = c
	return nil
}
func (f *fakeClaimRepo) GetClaimByID(context.Context, uuid.UUID) (*domain.ReimbursementClaim, error) {
	return f.claim, nil
}
func (f *fakeClaimRepo) UpdateClaim(context.Context, *domain.ReimbursementClaim) error { return nil }
func (f *fakeClaimRepo) CountClaims(context.Context) (int64, error)                    { return 0, nil }

func TestClaimStatusMachine(t *testing.T) {
	repo := &fakeClaimRepo{}
	uc := NewReimbursementUseCase(repo)
	ctx := context.Background()

	c, err := uc.CreateClaim(ctx, CreateClaimDTO{EmployeeName: "Ani", Amount: 100_000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.MarkPaid(ctx, c.ID); err == nil {
		t.Fatal("a pending claim must not be payable")
	}
	if _, err := uc.ApproveClaim(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.ApproveClaim(ctx, c.ID); err == nil {
		t.Fatal("approving twice must fail")
	}
	if _, err := uc.MarkPaid(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	// A paid claim is final: it must not be rejected or re-approved, otherwise the
	// status would contradict the journal entries already posted for it.
	if _, err := uc.RejectClaim(ctx, c.ID); err == nil {
		t.Fatal("a paid claim must not be rejectable")
	}
	if _, err := uc.ApproveClaim(ctx, c.ID); err == nil {
		t.Fatal("a paid claim must not be re-approvable")
	}
}

func TestCannotApproveOrPayOwnClaim(t *testing.T) {
	repo := &fakeClaimRepo{}
	uc := NewReimbursementUseCase(repo)

	ani := actor.WithEmail(context.Background(), "ani@x.com")
	c, err := uc.CreateClaim(ani, CreateClaimDTO{EmployeeName: "Ani", Amount: 100_000})
	if err != nil {
		t.Fatal(err)
	}
	if repo.claim.RequesterEmail != "ani@x.com" {
		t.Fatalf("requester email should be taken from the authenticated caller, got %q", repo.claim.RequesterEmail)
	}
	if _, err := uc.ApproveClaim(ani, c.ID); err == nil {
		t.Fatal("the requester must not approve their own claim")
	}

	boss := actor.WithEmail(context.Background(), "boss@x.com")
	if _, err := uc.ApproveClaim(boss, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.MarkPaid(ani, c.ID); err == nil {
		t.Fatal("the requester must not mark their own claim as paid")
	}
	if _, err := uc.MarkPaid(boss, c.ID); err != nil {
		t.Fatal(err)
	}
}
