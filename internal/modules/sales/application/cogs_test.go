package application

import (
	"context"
	"testing"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
)

type costRepo struct {
	productDomain.ProductRepository
	costs map[uuid.UUID]float64
}

func (r costRepo) GetByID(_ context.Context, id uuid.UUID) (*productDomain.Product, error) {
	if c, ok := r.costs[id]; ok {
		return &productDomain.Product{CostPrice: c}, nil
	}
	return nil, nil
}

func TestPostCOGSBooksCostOfGoodsOncePerOrder(t *testing.T) {
	a, b, unknown := uuid.New(), uuid.New(), uuid.New()
	led := &captureLedger{entries: map[string][]financeApp.LedgerLine{}}
	uc := &salesUseCase{ledger: led, products: costRepo{costs: map[uuid.UUID]float64{a: 6_000, b: 0}}}
	order := uuid.New()
	lines := []domain.SalesOrderLine{{ProductID: a, Quantity: 3}, {ProductID: b, Quantity: 5}, {ProductID: unknown, Quantity: 2}}
	uc.postCOGS(context.Background(), order, "SO-1", lines)
	got := led.entries[COGSSourcePrefix+order.String()]
	if len(got) != 2 || got[0].AccountCode != financeApp.AccountCOGS || got[0].Debit != 18_000 || got[1].AccountCode != financeApp.AccountInventory || got[1].Credit != 18_000 {
		t.Fatalf("cost entry: %+v", got)
	}
	// Free or unknown goods book nothing; no ledger or no product costs is a no-op.
	led2 := &captureLedger{entries: map[string][]financeApp.LedgerLine{}}
	(&salesUseCase{ledger: led2, products: costRepo{costs: map[uuid.UUID]float64{b: 0}}}).postCOGS(context.Background(), uuid.New(), "SO-2", lines[1:])
	(&salesUseCase{products: costRepo{}}).postCOGS(context.Background(), uuid.New(), "SO-3", lines)
	(&salesUseCase{ledger: led2}).postCOGS(context.Background(), uuid.New(), "SO-4", lines)
	if len(led2.entries) != 0 {
		t.Fatalf("nothing should have been booked: %v", led2.entries)
	}
}
