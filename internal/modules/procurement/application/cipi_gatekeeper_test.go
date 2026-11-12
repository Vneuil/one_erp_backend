package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	salesDomain "github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
)

type fakeProcurementSalesRepo struct {
	salesDomain.SalesRepository
	order *salesDomain.SalesOrder
}

func (f *fakeProcurementSalesRepo) GetOrderByID(ctx context.Context, id uuid.UUID) (*salesDomain.SalesOrder, error) {
	if f.order != nil && f.order.ID == id {
		return f.order, nil
	}
	return nil, nil
}

type fakeCreatePRRepo struct {
	domain.ProcurementRepository
	created *domain.PurchaseRequest
}

func (f *fakeCreatePRRepo) CreatePurchaseRequest(ctx context.Context, pr *domain.PurchaseRequest) error {
	f.created = pr
	return nil
}

func (f *fakeCreatePRRepo) UpdatePurchaseRequest(ctx context.Context, pr *domain.PurchaseRequest) error {
	return nil
}

func TestPRGatekeeper_SalesOrderEligibility(t *testing.T) {
	soID := uuid.New()
	prodID := uuid.New()

	baseDTO := CreatePurchaseRequestDTO{
		RequestedBy:  "buyer@company.com",
		Department:   "Purchasing",
		NeededByDate: "2026-11-20",
		Category:     "inventory",
		SalesOrderID: &soID,
		SalesOrderNo: "SO-TEST-001",
		Lines: []PurchaseRequestLineDTO{
			{ProductID: prodID, Quantity: 10, Notes: "Test line"},
		},
	}

	t.Run("Reject if SO is still pending_approval", func(t *testing.T) {
		so := &salesDomain.SalesOrder{
			OrderNumber:          "SO-TEST-001",
			Status:               "pending_approval",
			InvoiceDocAttachment: "sales/orders/ci.pdf",
		}
		so.ID = soID

		fakeSales := &fakeProcurementSalesRepo{order: so}
		fakePR := &fakeCreatePRRepo{}
		uc := NewProcurementUseCase(fakePR, nil, nil, nil, WithSalesRepo(fakeSales))

		_, err := uc.CreatePurchaseRequest(context.Background(), baseDTO)
		if err == nil {
			t.Fatal("expected error when SO is pending_approval, got nil")
		}
		if !strings.Contains(err.Error(), "belum disetujui") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("Reject if SO is Confirmed but missing CI/PI document", func(t *testing.T) {
		so := &salesDomain.SalesOrder{
			OrderNumber:          "SO-TEST-001",
			Status:               "Confirmed",
			InvoiceDocAttachment: "",
			CustomerPOAttachment: "",
		}
		so.ID = soID

		fakeSales := &fakeProcurementSalesRepo{order: so}
		fakePR := &fakeCreatePRRepo{}
		uc := NewProcurementUseCase(fakePR, nil, nil, nil, WithSalesRepo(fakeSales))

		_, err := uc.CreatePurchaseRequest(context.Background(), baseDTO)
		if err == nil {
			t.Fatal("expected error when SO has no CI/PI document, got nil")
		}
		if !strings.Contains(err.Error(), "belum memiliki dokumen Commercial Invoice (CI) atau Proforma Invoice (PI)") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("Accept if SO is Confirmed and has CI/PI document", func(t *testing.T) {
		so := &salesDomain.SalesOrder{
			OrderNumber:          "SO-TEST-001",
			Status:               "Confirmed",
			InvoiceDocType:       "CI",
			InvoiceDocAttachment: "sales/orders/ci-document.pdf",
		}
		so.ID = soID

		fakeSales := &fakeProcurementSalesRepo{order: so}
		fakePR := &fakeCreatePRRepo{}
		uc := NewProcurementUseCase(fakePR, nil, nil, nil, WithSalesRepo(fakeSales))

		res, err := uc.CreatePurchaseRequest(context.Background(), baseDTO)
		if err != nil {
			t.Fatalf("expected PR creation to succeed, got error: %v", err)
		}
		if res == nil {
			t.Fatal("expected PR response, got nil")
		}
		if fakePR.created == nil || fakePR.created.SalesOrderID == nil || *fakePR.created.SalesOrderID != soID {
			t.Fatal("expected PR to be created and linked to the Sales Order")
		}
	})
}
