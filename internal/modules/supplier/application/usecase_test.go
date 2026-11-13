package application

import (
	"context"
	"strings"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type memSupplierRepo struct {
	items map[uuid.UUID]*domain.Supplier
}

func newMemSupplierRepo() *memSupplierRepo {
	return &memSupplierRepo{items: make(map[uuid.UUID]*domain.Supplier)}
}

func (m *memSupplierRepo) Create(_ context.Context, s *domain.Supplier) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	cp := *s
	m.items[cp.ID] = &cp
	return nil
}

func (m *memSupplierRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Supplier, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, nil
	}
	cp := *s
	return &cp, nil
}

func (m *memSupplierRepo) GetByCode(_ context.Context, code string) (*domain.Supplier, error) {
	for _, s := range m.items {
		if strings.EqualFold(s.Code, code) {
			cp := *s
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memSupplierRepo) List(_ context.Context, _ types.PaginationQuery) ([]domain.Supplier, int64, error) {
	out := make([]domain.Supplier, 0, len(m.items))
	for _, s := range m.items {
		out = append(out, *s)
	}
	return out, int64(len(out)), nil
}

func (m *memSupplierRepo) Update(_ context.Context, s *domain.Supplier) error {
	cp := *s
	m.items[cp.ID] = &cp
	return nil
}

func (m *memSupplierRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.items, id)
	return nil
}

func (m *memSupplierRepo) Count(_ context.Context) (int64, error) {
	return int64(len(m.items)), nil
}

func (m *memSupplierRepo) CodesWithPrefix(_ context.Context, prefix string) ([]string, error) {
	var codes []string
	for _, s := range m.items {
		if strings.HasPrefix(strings.ToUpper(s.Code), strings.ToUpper(prefix)) {
			codes = append(codes, s.Code)
		}
	}
	return codes, nil
}

func (m *memSupplierRepo) HasReferences(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

func TestUpdateLegacyCodePreservesDerivedVendorType(t *testing.T) {
	repo := newMemSupplierRepo()
	uc := NewSupplierUseCase(repo)
	ctx := context.Background()

	legacy := &domain.Supplier{
		Code:   "PCK-001",
		Name:   "PT Kemas Jaya",
		City:   "Semarang",
		Status: "Active",
	}
	if err := repo.Create(ctx, legacy); err != nil {
		t.Fatal(err)
	}

	// Frontend sends empty vendorType alongside the updated code VD-PCK-001.
	newCode := "VD-PCK-001"
	emptyType := ""
	updated, err := uc.Update(ctx, legacy.ID, UpdateSupplierDTO{
		Code:       &newCode,
		VendorType: &emptyType,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Code != "VD-PCK-001" {
		t.Fatalf("code = %q, want VD-PCK-001", updated.Code)
	}
	if updated.VendorType != "PCK" {
		t.Fatalf("vendorType = %q, want PCK", updated.VendorType)
	}

	// Invalid prefix V-PCK-001 must be rejected.
	badCode := "V-PCK-001"
	if _, err := uc.Update(ctx, legacy.ID, UpdateSupplierDTO{Code: &badCode}); err == nil {
		t.Fatal("expected error for V-PCK-001")
	}
}

func TestCreateDefaultsCityToSemarangAndSyncsVendorType(t *testing.T) {
	repo := newMemSupplierRepo()
	uc := NewSupplierUseCase(repo)
	ctx := context.Background()

	created, err := uc.Create(ctx, CreateSupplierDTO{
		VendorType:  "PRD",
		Code:        "VD-ATK-002",
		Name:        "CV Kertas Makmur",
		PaymentTerm: "Net 30",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.City != "Semarang" {
		t.Fatalf("city = %q, want Semarang", created.City)
	}
	if created.VendorType != "ATK" {
		t.Fatalf("vendorType = %q, want ATK", created.VendorType)
	}
	if created.PaymentTerm != "Net 30" {
		t.Fatalf("paymentTerm = %q, want Net 30", created.PaymentTerm)
	}
}

func TestUpdateBlocksStaffFromChangingStatus(t *testing.T) {
	repo := newMemSupplierRepo()
	uc := NewSupplierUseCase(repo)
	ctx := context.Background()

	created, err := uc.Create(ctx, CreateSupplierDTO{
		VendorType: "PRD",
		Name:       "PT Sumber Bahan",
	})
	if err != nil {
		t.Fatal(err)
	}

	inactive := "Inactive"
	staffCtx := actor.WithRoles(ctx, []string{"staff"})
	if _, err := uc.Update(staffCtx, created.ID, UpdateSupplierDTO{Status: &inactive}); err == nil {
		t.Fatal("expected staff status change via Update to be forbidden")
	}

	managerCtx := actor.WithRoles(ctx, []string{"staff", "Manager"})
	updated, err := uc.Update(managerCtx, created.ID, UpdateSupplierDTO{Status: &inactive})
	if err != nil {
		t.Fatalf("manager status change failed: %v", err)
	}
	if updated.Status != "Inactive" {
		t.Fatalf("status = %q, want Inactive", updated.Status)
	}
}

