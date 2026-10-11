package application

import (
	"context"
	"errors"
	"testing"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	"github.com/google/uuid"
)

type purchaseReturnRepo struct {
	domain.ProcurementRepository
	gr       *domain.GoodsReceipt
	po       *domain.PurchaseOrder
	invoices []domain.PurchaseInvoice
	returns  []domain.PurchaseReturn
}

func (r *purchaseReturnRepo) GetGoodsReceiptByID(context.Context, uuid.UUID) (*domain.GoodsReceipt, error) {
	return r.gr, nil
}
func (r *purchaseReturnRepo) GetPurchaseOrderByID(context.Context, uuid.UUID) (*domain.PurchaseOrder, error) {
	return r.po, nil
}
func (r *purchaseReturnRepo) ListPurchaseReturnsByReceipt(context.Context, uuid.UUID) ([]domain.PurchaseReturn, error) {
	return r.returns, nil
}
func (r *purchaseReturnRepo) ListPurchaseInvoicesByOrder(context.Context, uuid.UUID) ([]domain.PurchaseInvoice, error) {
	return append([]domain.PurchaseInvoice(nil), r.invoices...), nil
}
func (r *purchaseReturnRepo) CreatePurchaseReturn(_ context.Context, s *domain.PurchaseReturn) error {
	s.ID = uuid.New()
	r.returns = append(r.returns, *s)
	return nil
}
func (r *purchaseReturnRepo) UpdatePurchaseInvoice(_ context.Context, inv *domain.PurchaseInvoice) error {
	for i := range r.invoices {
		if r.invoices[i].ID == inv.ID {
			r.invoices[i] = *inv
		}
	}
	return nil
}

type returnInventory struct {
	inventoryApp.InventoryUseCase
	stock  int
	failed bool
}

func (i *returnInventory) AdjustStock(_ context.Context, dto inventoryApp.AdjustStockDTO) (*inventoryApp.StockLevelResponseDTO, error) {
	if i.stock+dto.Quantity < 0 {
		i.failed = true
		return nil, errors.New("insufficient stock")
	}
	i.stock += dto.Quantity
	return nil, nil
}

type retLedger struct {
	financeApp.LedgerPoster
	entries map[string][]financeApp.LedgerLine
}

func (l *retLedger) PostEntry(_ context.Context, src, _ string, lines []financeApp.LedgerLine) error {
	l.entries[src] = lines
	return nil
}
func (l *retLedger) UpsertPayable(context.Context, financeApp.SubledgerDoc) error { return nil }

func setupPurchaseReturn(stock int) (*procurementUseCase, *purchaseReturnRepo, *retLedger, *returnInventory, uuid.UUID) {
	prod, poID, wh := uuid.New(), uuid.New(), uuid.New()
	po := &domain.PurchaseOrder{SupplierName: "CV S", Lines: []domain.PurchaseOrderLine{{ProductID: prod, Quantity: 10, UnitPrice: 1000}}}
	po.ID = poID
	gr := &domain.GoodsReceipt{PurchaseOrderID: poID, WarehouseID: &wh, ReceiptNo: "GR-1", Lines: []domain.GoodsReceiptLine{{ProductID: prod, QuantityReceived: 10}}}
	gr.ID = uuid.New()
	inv := domain.PurchaseInvoice{InvoiceNumber: "PI-1", TotalAmount: 11_000, TaxBase: 10_000, VATAmount: 1_000, VATCreditable: true, PurchaseOrderID: &poID}
	inv.ID = uuid.New()
	repo := &purchaseReturnRepo{gr: gr, po: po, invoices: []domain.PurchaseInvoice{inv}}
	led := &retLedger{entries: map[string][]financeApp.LedgerLine{}}
	inventory := &returnInventory{stock: stock}
	return &procurementUseCase{repo: repo, inventoryUC: inventory, ledger: led}, repo, led, inventory, prod
}

func TestPurchaseReturnDebitsInvoiceAndReversesVAT(t *testing.T) {
	uc, repo, led, inv, prod := setupPurchaseReturn(10)
	res, err := uc.CreatePurchaseReturn(context.Background(), CreatePurchaseReturnDTO{GoodsReceiptID: uuid.New(), Lines: []PurchaseReturnLineDTO{{ProductID: prod, Quantity: 4}}})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalAmount != 4000 || res.CreditedAmount != 4400 || res.VATAmount != 400 || inv.stock != 6 {
		t.Fatalf("return %+v stock %d", res, inv.stock)
	}
	var dr, cr float64
	for _, l := range led.entries[PurchaseReturnSourcePrefix+repo.returns[0].ID.String()] {
		dr += l.Debit
		cr += l.Credit
	}
	if dr != 4400 || cr != 4400 {
		t.Fatalf("journal dr %v cr %v", dr, cr)
	}
	if repo.invoices[0].CreditedAmount != 4400 {
		t.Fatalf("invoice credit %v", repo.invoices[0].CreditedAmount)
	}
}

func TestPurchaseReturnLimitsAndStock(t *testing.T) {
	uc, repo, _, inv, prod := setupPurchaseReturn(2) // only 2 left in the warehouse
	ctx := context.Background()
	if _, err := uc.CreatePurchaseReturn(ctx, CreatePurchaseReturnDTO{GoodsReceiptID: uuid.New(), Lines: []PurchaseReturnLineDTO{{ProductID: prod, Quantity: 11}}}); err == nil {
		t.Fatal("returning more than received must fail")
	}
	if _, err := uc.CreatePurchaseReturn(ctx, CreatePurchaseReturnDTO{GoodsReceiptID: uuid.New(), Lines: []PurchaseReturnLineDTO{{ProductID: prod, Quantity: 5}}}); err == nil {
		t.Fatal("returning stock that is no longer there must fail")
	}
	if len(repo.returns) != 0 || inv.stock != 2 {
		t.Fatalf("a failed return must leave nothing behind: returns %d stock %d", len(repo.returns), inv.stock)
	}
}
