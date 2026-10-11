package application

import (
	"context"
	"testing"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
)

type returnRepo struct {
	domain.SalesRepository
	delivery *domain.Delivery
	order    *domain.SalesOrder
	invoices []domain.Invoice
	returns  []domain.SalesReturn
}

func (r *returnRepo) GetDeliveryByID(context.Context, uuid.UUID) (*domain.Delivery, error) {
	return r.delivery, nil
}
func (r *returnRepo) GetOrderByID(context.Context, uuid.UUID) (*domain.SalesOrder, error) {
	return r.order, nil
}
func (r *returnRepo) ListSalesReturnsByDelivery(context.Context, uuid.UUID) ([]domain.SalesReturn, error) {
	return r.returns, nil
}
func (r *returnRepo) ListInvoicesByOrder(context.Context, uuid.UUID) ([]domain.Invoice, error) {
	return append([]domain.Invoice(nil), r.invoices...), nil
}
func (r *returnRepo) CreateSalesReturn(_ context.Context, s *domain.SalesReturn) error {
	s.ID = uuid.New()
	r.returns = append(r.returns, *s)
	return nil
}
func (r *returnRepo) UpdateInvoice(_ context.Context, inv *domain.Invoice) error {
	for i := range r.invoices {
		if r.invoices[i].ID == inv.ID {
			r.invoices[i] = *inv
		}
	}
	return nil
}

type returnLedger struct{ captureLedger }

func (returnLedger) UpsertReceivable(context.Context, financeApp.SubledgerDoc) error { return nil }

func setupReturn(paid float64) (*salesUseCase, *returnRepo, *returnLedger, uuid.UUID) {
	prod, orderID := uuid.New(), uuid.New()
	order := &domain.SalesOrder{Lines: []domain.SalesOrderLine{{ProductID: prod, Quantity: 10, UnitPrice: 1000}}}
	order.ID = orderID
	del := &domain.Delivery{SalesOrderID: &orderID, CustomerName: "PT A", Lines: []domain.DeliveryLine{{ProductID: prod, Quantity: 10}}}
	inv := domain.Invoice{InvoiceNumber: "INV-1", TotalAmount: 11_000, TaxBase: 10_000, VATAmount: 1_000, PaidAmount: paid, SalesOrderID: &orderID}
	inv.ID = uuid.New()
	repo := &returnRepo{delivery: del, order: order, invoices: []domain.Invoice{inv}}
	led := &returnLedger{captureLedger{entries: map[string][]financeApp.LedgerLine{}}}
	return &salesUseCase{repo: repo, ledger: led}, repo, led, prod
}

func TestSalesReturnCreditsInvoiceWithVAT(t *testing.T) {
	uc, repo, led, prod := setupReturn(0)
	res, err := uc.CreateSalesReturn(context.Background(), CreateSalesReturnDTO{DeliveryID: uuid.New(), Lines: []SalesReturnLineDTO{{ProductID: prod, Quantity: 4}}})
	if err != nil {
		t.Fatal(err)
	}
	// 4 x 1000 = 4000 + 10% PPN
	if res.TotalAmount != 4000 || res.CreditedAmount != 4400 || res.VATAmount != 400 {
		t.Fatalf("return: %+v", res)
	}
	lines := led.entries[SalesReturnSourcePrefix+repo.returns[0].ID.String()]
	var dr, cr float64
	for _, l := range lines {
		dr += l.Debit
		cr += l.Credit
	}
	if dr != cr || cr != 4400 {
		t.Fatalf("unbalanced %v/%v: %+v", dr, cr, lines)
	}
	if repo.invoices[0].CreditedAmount != 4400 {
		t.Fatalf("invoice credit %v", repo.invoices[0].CreditedAmount)
	}
}

func TestSalesReturnCannotExceedDelivered(t *testing.T) {
	uc, _, _, prod := setupReturn(0)
	ctx := context.Background()
	if _, err := uc.CreateSalesReturn(ctx, CreateSalesReturnDTO{DeliveryID: uuid.New(), Lines: []SalesReturnLineDTO{{ProductID: prod, Quantity: 7}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateSalesReturn(ctx, CreateSalesReturnDTO{DeliveryID: uuid.New(), Lines: []SalesReturnLineDTO{{ProductID: prod, Quantity: 4}}}); err == nil {
		t.Fatal("returning more than was shipped must fail")
	}
	if _, err := uc.CreateSalesReturn(ctx, CreateSalesReturnDTO{DeliveryID: uuid.New(), Lines: []SalesReturnLineDTO{{ProductID: uuid.New(), Quantity: 1}}}); err == nil {
		t.Fatal("a product not on the delivery must fail")
	}
}

func TestAllocateCreditSpillsAcrossInvoices(t *testing.T) {
	a := domain.Invoice{TotalAmount: 1000}
	b := domain.Invoice{TotalAmount: 5000}
	out := allocateCredit([]domain.Invoice{a, b}, 3000)
	if len(out) != 2 || out[0].Amount != 1000 || out[1].Amount != 2000 {
		t.Fatalf("%+v", out)
	}
	if got := allocateCredit(nil, 500); len(got) != 0 {
		t.Fatal("no invoice, no credit")
	}
}
