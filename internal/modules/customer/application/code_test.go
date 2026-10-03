package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	"github.com/google/uuid"
)

type fakeCustomerRepo struct {
	domain.CustomerRepository
	byCode map[string]*domain.Customer
}

func (f *fakeCustomerRepo) GetByCode(_ context.Context, code string) (*domain.Customer, error) {
	return f.byCode[code], nil
}
func (f *fakeCustomerRepo) Count(context.Context) (int64, error) { return int64(len(f.byCode)), nil }
func (f *fakeCustomerRepo) Create(_ context.Context, c *domain.Customer) error {
	c.ID = uuid.New()
	f.byCode[c.Code] = c
	return nil
}

func TestCustomerCodeIsGeneratedAndNeverCollides(t *testing.T) {
	// CUST-001 and CUST-003 exist (e.g. after a delete), so the next free code after
	// "count + 1" must be picked without reusing a taken one.
	repo := &fakeCustomerRepo{byCode: map[string]*domain.Customer{"CUST-001": {}, "CUST-003": {}}}
	uc := NewCustomerUseCase(repo)

	res, err := uc.Create(context.Background(), CreateCustomerDTO{Name: "Acme"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != "CUST-004" {
		t.Fatalf("expected the first free code after the count, CUST-004, got %s", res.Code)
	}

	if _, err := uc.Create(context.Background(), CreateCustomerDTO{Code: "cust-001", Name: "Dup"}); err == nil {
		t.Fatal("an explicit duplicate code must still be rejected")
	}
	if _, err := uc.Create(context.Background(), CreateCustomerDTO{}); err == nil {
		t.Fatal("a name is still required")
	}
}
