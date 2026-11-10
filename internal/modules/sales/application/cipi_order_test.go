package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
)

type fakeSalesCIPIRepo struct {
	domain.SalesRepository
	order *domain.SalesOrder
}

func (f *fakeSalesCIPIRepo) CreateOrder(ctx context.Context, order *domain.SalesOrder) error {
	order.ID = uuid.New()
	f.order = order
	return nil
}

func (f *fakeSalesCIPIRepo) GetOrderByID(ctx context.Context, id uuid.UUID) (*domain.SalesOrder, error) {
	if f.order != nil && f.order.ID == id {
		return f.order, nil
	}
	return nil, nil
}

func (f *fakeSalesCIPIRepo) UpdateOrder(ctx context.Context, order *domain.SalesOrder) error {
	f.order = order
	return nil
}

func TestSalesOrder_CIPILogic(t *testing.T) {
	prodID := uuid.New()

	t.Run("CreateOrder without DP routes to pending_approval", func(t *testing.T) {
		repo := &fakeSalesCIPIRepo{}
		uc := NewSalesUseCase(repo, nil, nil, nil)

		dto := CreateSalesOrderDTO{
			OrderNumber:  "SO-NODP-001",
			CustomerName: "PT Maju Jaya",
			TotalAmount:  5_000_000,
			DPPaid:       false,
			Lines: []SalesOrderLineDTO{
				{ProductID: prodID, Quantity: 5, UnitPrice: 1_000_000},
			},
		}

		res, err := uc.CreateOrder(context.Background(), dto)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != "pending_approval" {
			t.Fatalf("expected order without DP to have status pending_approval, got %s", res.Status)
		}
	})

	t.Run("AttachInvoiceDocument successfully attaches CI or PI", func(t *testing.T) {
		orderID := uuid.New()
		order := &domain.SalesOrder{
			OrderNumber:  "SO-CONFIRMED-001",
			CustomerName: "PT Sukses",
			Status:       "Confirmed",
		}
		order.ID = orderID
		repo := &fakeSalesCIPIRepo{order: order}
		uc := NewSalesUseCase(repo, nil, nil, nil)

		dto := AttachInvoiceDocDTO{
			InvoiceDocType:       "CI",
			InvoiceDocNumber:     "CI-2026/001",
			InvoiceDocAttachment: "sales/orders/ci-document.pdf",
		}

		res, err := uc.AttachInvoiceDocument(context.Background(), orderID, dto)
		if err != nil {
			t.Fatalf("unexpected error attaching invoice document: %v", err)
		}
		if res.InvoiceDocType != "CI" {
			t.Fatalf("expected CI, got %s", res.InvoiceDocType)
		}
		if res.InvoiceDocAttachment != "sales/orders/ci-document.pdf" {
			t.Fatalf("expected attachment key to match, got %s", res.InvoiceDocAttachment)
		}
		if res.InvoiceDocNumber != "CI-2026/001" {
			t.Fatalf("expected doc number CI-2026/001, got %s", res.InvoiceDocNumber)
		}
	})

	t.Run("AttachInvoiceDocument rejects invalid doc type", func(t *testing.T) {
		orderID := uuid.New()
		order := &domain.SalesOrder{
			OrderNumber: "SO-CONFIRMED-001",
			Status:      "Confirmed",
		}
		order.ID = orderID
		repo := &fakeSalesCIPIRepo{order: order}
		uc := NewSalesUseCase(repo, nil, nil, nil)

		dto := AttachInvoiceDocDTO{
			InvoiceDocType:       "INVALID_DOC",
			InvoiceDocAttachment: "sales/orders/test.pdf",
		}

		_, err := uc.AttachInvoiceDocument(context.Background(), orderID, dto)
		if err == nil {
			t.Fatal("expected error for invalid invoice doc type, got nil")
		}
		if !strings.Contains(err.Error(), "must be CI") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("AttachInvoiceDocument rejects empty attachment", func(t *testing.T) {
		orderID := uuid.New()
		order := &domain.SalesOrder{
			OrderNumber: "SO-CONFIRMED-001",
			Status:      "Confirmed",
		}
		order.ID = orderID
		repo := &fakeSalesCIPIRepo{order: order}
		uc := NewSalesUseCase(repo, nil, nil, nil)

		dto := AttachInvoiceDocDTO{
			InvoiceDocType:       "PI",
			InvoiceDocAttachment: "   ",
		}

		_, err := uc.AttachInvoiceDocument(context.Background(), orderID, dto)
		if err == nil {
			t.Fatal("expected error for empty attachment key, got nil")
		}
	})
}

