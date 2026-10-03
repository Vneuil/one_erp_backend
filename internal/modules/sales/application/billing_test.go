package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
)

func TestSplitByPercentAlwaysAddsUpToTheTotal(t *testing.T) {
	for _, tc := range []struct {
		total    float64
		percents []float64
	}{
		{1_000_000, []float64{30, 40, 30}},
		{1_000_001, []float64{33.333, 33.333, 33.334}},
		{999.99, []float64{50, 50}},
		{10, []float64{100}},
		{1234567.89, []float64{10, 20, 30, 40}},
	} {
		parts := SplitByPercent(tc.total, tc.percents)
		var sum float64
		for _, p := range parts {
			sum += p
		}
		if round2(sum) != round2(tc.total) {
			t.Errorf("%v split %v = %v (sum %v)", tc.total, tc.percents, parts, sum)
		}
	}
	if got := SplitByPercent(1_000_000, []float64{30, 70}); got[0] != 300_000 || got[1] != 700_000 {
		t.Fatalf("plain split wrong: %v", got)
	}
}

type billingRepo struct {
	domain.SalesRepository
	order    *domain.SalesOrder
	terms    []domain.BillingTerm
	invoices []*domain.Invoice
}

func (r *billingRepo) GetOrderByID(context.Context, uuid.UUID) (*domain.SalesOrder, error) {
	return r.order, nil
}
func (r *billingRepo) ReplaceBillingTerms(_ context.Context, _ uuid.UUID, terms []domain.BillingTerm) error {
	for i := range terms {
		terms[i].ID = uuid.New()
	}
	r.terms = terms
	return nil
}
func (r *billingRepo) ListBillingTerms(context.Context, uuid.UUID) ([]domain.BillingTerm, error) {
	return append([]domain.BillingTerm(nil), r.terms...), nil
}
func (r *billingRepo) GetBillingTerm(_ context.Context, id uuid.UUID) (*domain.BillingTerm, error) {
	for i := range r.terms {
		if r.terms[i].ID == id {
			return &r.terms[i], nil
		}
	}
	return nil, nil
}
func (r *billingRepo) UpdateBillingTerm(_ context.Context, t *domain.BillingTerm) error {
	for i := range r.terms {
		if r.terms[i].ID == t.ID {
			r.terms[i] = *t
		}
	}
	return nil
}
func (r *billingRepo) CreateInvoice(_ context.Context, inv *domain.Invoice) error {
	inv.ID = uuid.New()
	r.invoices = append(r.invoices, inv)
	return nil
}

func newBillingUC(status string) (*salesUseCase, *billingRepo) {
	order := &domain.SalesOrder{OrderNumber: "SO-202609-0007", CustomerName: "PT Maju", TotalAmount: 10_000_000, BaseAmount: 10_000_000, Status: status}
	order.ID = uuid.New()
	repo := &billingRepo{order: order}
	return &salesUseCase{repo: repo}, repo
}

func TestBillingScheduleMustSumTo100(t *testing.T) {
	uc, repo := newBillingUC("Confirmed")
	ctx := context.Background()
	for _, bad := range [][]BillingTermInput{
		nil,
		{{Percent: 50}, {Percent: 40}},
		{{Percent: 60}, {Percent: 60}},
		{{Percent: 0}, {Percent: 100}},
		{{Percent: 100, DueDate: "31-12-2026"}},
	} {
		if _, err := uc.SetBillingSchedule(ctx, repo.order.ID, bad); err == nil {
			t.Errorf("schedule should be rejected: %+v", bad)
		}
	}
	terms, err := uc.SetBillingSchedule(ctx, repo.order.ID, []BillingTermInput{{Label: "DP", Percent: 30}, {Percent: 70, DueInDays: 30}})
	if err != nil {
		t.Fatal(err)
	}
	if terms[0].Amount != 3_000_000 || terms[1].Amount != 7_000_000 || terms[1].Label != "Termin 2" || terms[0].Seq != 1 {
		t.Fatalf("unexpected schedule: %+v", terms)
	}
}

func TestTermsAreInvoicedInOrderAndOnlyOnce(t *testing.T) {
	uc, repo := newBillingUC("Confirmed")
	ctx := context.Background()
	terms, err := uc.SetBillingSchedule(ctx, repo.order.ID, []BillingTermInput{{Percent: 30}, {Percent: 70}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.InvoiceBillingTerm(ctx, terms[1].ID); err == nil {
		t.Fatal("term 2 must not be invoiced before term 1")
	}
	inv, err := uc.InvoiceBillingTerm(ctx, terms[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if inv.TotalAmount != 3_000_000 || inv.InvoiceNumber != "INV-202609-0007-T1" || inv.SalesOrderID == nil {
		t.Fatalf("bad sub-invoice: %+v", inv)
	}
	if repo.terms[0].Status != "invoiced" || repo.terms[0].InvoiceID == nil {
		t.Fatalf("term not marked invoiced: %+v", repo.terms[0])
	}
	if _, err := uc.InvoiceBillingTerm(ctx, terms[0].ID); err == nil {
		t.Fatal("a term must not be invoiced twice")
	}
	if _, err := uc.InvoiceBillingTerm(ctx, terms[1].ID); err != nil {
		t.Fatalf("term 2 should now be allowed: %v", err)
	}
	var billed float64
	for _, i := range repo.invoices {
		billed += i.TotalAmount
	}
	if billed != 10_000_000 {
		t.Fatalf("sub-invoices add up to %v, want the order total", billed)
	}
	// Once invoiced, the schedule is frozen.
	if _, err := uc.SetBillingSchedule(ctx, repo.order.ID, []BillingTermInput{{Percent: 100}}); err == nil {
		t.Fatal("schedule must be frozen after invoicing")
	}
}

func TestUnapprovedOrdersCannotBeBilled(t *testing.T) {
	for _, status := range []string{"pending_approval", "rejected"} {
		uc, repo := newBillingUC("Confirmed")
		terms, _ := uc.SetBillingSchedule(context.Background(), repo.order.ID, []BillingTermInput{{Percent: 100}})
		repo.order.Status = status
		if _, err := uc.InvoiceBillingTerm(context.Background(), terms[0].ID); err == nil {
			t.Errorf("a %s order must not be billed", status)
		}
	}
	uc, repo := newBillingUC("rejected")
	if _, err := uc.SetBillingSchedule(context.Background(), repo.order.ID, []BillingTermInput{{Percent: 100}}); err == nil {
		t.Fatal("a rejected order cannot get a schedule")
	}
}
