package application

import (
	"context"
	"testing"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
)

type deliveryRepo struct {
	domain.SalesRepository
	d *domain.Delivery
}

func (r *deliveryRepo) GetDeliveryByID(context.Context, uuid.UUID) (*domain.Delivery, error) {
	return r.d, nil
}
func (r *deliveryRepo) UpdateDelivery(_ context.Context, d *domain.Delivery) error {
	r.d = d
	return nil
}

type captureLedger struct {
	financeApp.LedgerPoster
	entries map[string][]financeApp.LedgerLine
}

func (c *captureLedger) PostEntry(_ context.Context, src, _ string, l []financeApp.LedgerLine) error {
	c.entries[src] = l
	return nil
}

func TestShippingCostIsBookedOnceAndBalanced(t *testing.T) {
	d := &domain.Delivery{DeliveryNumber: "DO-1"}
	d.ID = uuid.New()
	led := &captureLedger{entries: map[string][]financeApp.LedgerLine{}}
	uc := &salesUseCase{repo: &deliveryRepo{d: d}, ledger: led}
	ctx := context.Background()

	// Tracking number alone books nothing.
	if _, err := uc.RecordShippingCost(ctx, d.ID, ShippingCostDTO{TrackingNumber: " JNE123 ", Carrier: "JNE"}); err != nil {
		t.Fatal(err)
	}
	if d.TrackingNumber != "JNE123" || d.Carrier != "JNE" || d.ShippingCost != 0 || len(led.entries) != 0 {
		t.Fatalf("tracking only: %+v entries %v", d, led.entries)
	}
	// Owed to the carrier: credit payables.
	if _, err := uc.RecordShippingCost(ctx, d.ID, ShippingCostDTO{Amount: 45_000.456}); err != nil {
		t.Fatal(err)
	}
	if d.ShippingCost != 45_000.46 || d.ShippingPaid {
		t.Fatalf("cost: %+v", d)
	}
	lines := led.entries[DeliveryCostSourcePrefix+d.ID.String()]
	if len(lines) != 2 || lines[0].AccountCode != financeApp.AccountDeliveryExpense || lines[0].Debit != lines[1].Credit || lines[1].AccountCode != financeApp.AccountPayable {
		t.Fatalf("journal: %+v", lines)
	}
	// Recording the cost twice would double the expense.
	if _, err := uc.RecordShippingCost(ctx, d.ID, ShippingCostDTO{Amount: 10_000}); err == nil {
		t.Fatal("the cost must not be recorded twice")
	}
	if _, err := uc.RecordShippingCost(ctx, d.ID, ShippingCostDTO{Amount: -1}); err == nil {
		t.Fatal("negative cost must be refused")
	}
	// Paid in cash credits the cash account.
	d2 := &domain.Delivery{DeliveryNumber: "DO-2"}
	d2.ID = uuid.New()
	uc.repo = &deliveryRepo{d: d2}
	if _, err := uc.RecordShippingCost(ctx, d2.ID, ShippingCostDTO{Amount: 20_000, PaidNow: true}); err != nil {
		t.Fatal(err)
	}
	if l := led.entries[DeliveryCostSourcePrefix+d2.ID.String()]; l[1].AccountCode != financeApp.AccountCash {
		t.Fatalf("cash payment: %+v", l)
	}
}
