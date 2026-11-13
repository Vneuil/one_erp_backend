package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type memCustomerRepo struct {
	items map[uuid.UUID]*domain.Customer
}

func newMemCustomerRepo() *memCustomerRepo {
	return &memCustomerRepo{items: make(map[uuid.UUID]*domain.Customer)}
}

func (m *memCustomerRepo) Create(_ context.Context, c *domain.Customer) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	cp := *c
	m.items[cp.ID] = &cp
	return nil
}

func (m *memCustomerRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Customer, error) {
	c, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (m *memCustomerRepo) GetByCode(_ context.Context, code string) (*domain.Customer, error) {
	for _, c := range m.items {
		if strings.EqualFold(c.Code, code) {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memCustomerRepo) GetByName(_ context.Context, name string) (*domain.Customer, error) {
	for _, c := range m.items {
		if strings.EqualFold(c.Name, strings.TrimSpace(name)) {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memCustomerRepo) List(_ context.Context, _ types.PaginationQuery) ([]domain.Customer, int64, error) {
	out := make([]domain.Customer, 0, len(m.items))
	for _, c := range m.items {
		out = append(out, *c)
	}
	return out, int64(len(out)), nil
}

func (m *memCustomerRepo) Update(_ context.Context, c *domain.Customer) error {
	cp := *c
	m.items[cp.ID] = &cp
	return nil
}

func (m *memCustomerRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.items, id)
	return nil
}

func (m *memCustomerRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.items)), nil
}

func (m *memCustomerRepo) CodesWithPrefix(_ context.Context, prefix string) ([]string, error) {
	var out []string
	for _, c := range m.items {
		if strings.HasPrefix(strings.ToUpper(c.Code), strings.ToUpper(prefix)) {
			out = append(out, c.Code)
		}
	}
	return out, nil
}

func (m *memCustomerRepo) ReplaceShippingAddresses(_ context.Context, _ uuid.UUID, _ []domain.CustomerAddress) error {
	return nil
}

func (m *memCustomerRepo) HasReferences(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func TestUpdateCustomerCodeAndBlockStaffStatusChange(t *testing.T) {
	repo := newMemCustomerRepo()
	uc := NewCustomerUseCase(repo)
	ctx := context.Background()

	legacy := &domain.Customer{
		Code:          "CUST-001",
		CustomerType:  "B2B",
		Name:          "PT Lama Sejahtera",
		ContactPerson: "Budi",
		ContactMobile: "08123456789",
		City:          "Semarang",
		Status:        "Active",
	}
	if err := repo.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}

	// Updating legacy code to B2C-0001 syncs CustomerType and saves Code.
	newCode := "B2C-0001"
	updated, err := uc.Update(ctx, legacy.ID, UpdateCustomerDTO{Code: &newCode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Code != "B2C-0001" || updated.CustomerType != "B2C" {
		t.Fatalf("got code=%q type=%q, want B2C-0001 / B2C", updated.Code, updated.CustomerType)
	}

	// Invalid manual code on Update or Create must be rejected.
	bad := "CUST-99"
	if _, err := uc.Update(ctx, legacy.ID, UpdateCustomerDTO{Code: &bad}); err == nil {
		t.Fatal("expected invalid code on update to fail")
	}
	if _, err := uc.Create(ctx, CreateCustomerDTO{Code: "CUST-99", CustomerType: "B2C", Name: "Bad"}); err == nil {
		t.Fatal("expected invalid code on create to fail")
	}

	// Staff cannot change status via Update, Manager can.
	inactive := "Inactive"
	staffCtx := actor.WithRoles(ctx, []string{"staff"})
	if _, err := uc.Update(staffCtx, legacy.ID, UpdateCustomerDTO{Status: &inactive}); err == nil {
		t.Fatal("expected staff status change via Update to be forbidden")
	}
	mgrCtx := actor.WithRoles(ctx, []string{"staff", "Manager"})
	res, err := uc.Update(mgrCtx, legacy.ID, UpdateCustomerDTO{Status: &inactive})
	if err != nil {
		t.Fatalf("manager status change failed: %v", err)
	}
	if res.Status != "Inactive" {
		t.Fatalf("status = %q, want Inactive", res.Status)
	}
}

