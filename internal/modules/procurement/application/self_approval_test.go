package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/google/uuid"
)

type fakePRRepo struct {
	domain.ProcurementRepository
	pr *domain.PurchaseRequest
}

func (f *fakePRRepo) GetPurchaseRequestByID(context.Context, uuid.UUID) (*domain.PurchaseRequest, error) {
	return f.pr, nil
}
func (f *fakePRRepo) UpdatePurchaseRequest(context.Context, *domain.PurchaseRequest) error {
	return nil
}

func TestCannotApproveOwnPurchaseRequest(t *testing.T) {
	pr := &domain.PurchaseRequest{RequestedBy: "ani@x.com", Status: "submitted"}
	pr.ID = uuid.New()
	uc := NewProcurementUseCase(&fakePRRepo{pr: pr}, nil, nil, nil)

	own := actor.WithEmail(context.Background(), "Ani@x.com")
	if _, err := uc.ApprovePurchaseRequest(own, pr.ID); err == nil {
		t.Fatal("a requester must not approve their own purchase request")
	}
	if pr.Status != "submitted" {
		t.Fatalf("status changed to %q by a blocked approval", pr.Status)
	}
	// Rejecting your own request is allowed: it is effectively withdrawing it.
	if _, err := uc.RejectPurchaseRequest(own, pr.ID); err != nil {
		t.Fatalf("withdrawing your own request should be allowed: %v", err)
	}

	pr.Status = "submitted"
	boss := actor.WithEmail(context.Background(), "boss@x.com")
	if _, err := uc.ApprovePurchaseRequest(boss, pr.ID); err != nil {
		t.Fatalf("a different approver must be able to approve: %v", err)
	}
}

type fakePORepo struct {
	domain.ProcurementRepository
	po *domain.PurchaseOrder
}

func (f *fakePORepo) GetPurchaseOrderByID(context.Context, uuid.UUID) (*domain.PurchaseOrder, error) {
	return f.po, nil
}

func TestCannotApproveOwnPurchaseOrder(t *testing.T) {
	po := &domain.PurchaseOrder{Status: "draft", CreatedByEmail: "ani@x.com"}
	po.ID = uuid.New()
	uc := NewProcurementUseCase(&fakePORepo{po: po}, nil, nil, nil)

	own := actor.WithEmail(context.Background(), "ani@x.com")
	if _, err := uc.ApprovePurchaseOrder(own, po.ID, ApprovalActionDTO{}); err == nil {
		t.Fatal("creator must not approve their own purchase order")
	}
	if po.Status != "draft" {
		t.Fatalf("blocked approval changed status to %q", po.Status)
	}
}
