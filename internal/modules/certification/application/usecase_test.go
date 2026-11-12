package application

import (
	"context"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/certification/domain"
	"github.com/google/uuid"
)

type memRepo struct{ items []domain.Certificate }

func (m *memRepo) Create(_ context.Context, c *domain.Certificate) error {
	c.ID = uuid.New()
	m.items = append(m.items, *c)
	return nil
}
func (m *memRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Certificate, error) {
	for i := range m.items {
		if m.items[i].ID == id {
			c := m.items[i]
			return &c, nil
		}
	}
	return nil, nil
}
func (m *memRepo) Update(_ context.Context, c *domain.Certificate) error {
	for i := range m.items {
		if m.items[i].ID == c.ID {
			m.items[i] = *c
		}
	}
	return nil
}
func (m *memRepo) List(context.Context, domain.ListFilter) ([]domain.Certificate, error) {
	return m.items, nil
}

func newUC(day string) (*useCase, *memRepo) {
	r := &memRepo{}
	t, _ := time.Parse("2006-01-02", day)
	return &useCase{repo: r, now: func() time.Time { return t }}, r
}

func TestCertificateSwitchesOffAfterValidUntil(t *testing.T) {
	sup := uuid.New()
	uc, _ := newUC("2026-11-12")
	res, err := uc.Create(context.Background(), CertificateDTO{Scheme: "USDA Organic", Number: "A1", SupplierID: &sup, ValidFrom: "2026-01-01", ValidUntil: "2026-11-12"})
	if err != nil || res.EffectiveStatus != "active" {
		t.Fatalf("last valid day must still be active: %v %+v", err, res)
	}
	uc.now = func() time.Time { d, _ := time.Parse("2006-01-02", "2026-11-13"); return d }
	got, _ := uc.Get(context.Background(), res.ID)
	if got.EffectiveStatus != "expired" {
		t.Fatalf("day after validUntil must be expired, got %s", got.EffectiveStatus)
	}
}

func TestSuspendResumeAndEligibility(t *testing.T) {
	sup := uuid.New()
	uc, _ := newUC("2026-06-01")
	a, _ := uc.Create(context.Background(), CertificateDTO{Scheme: "USDA Organic", Number: "A", SupplierID: &sup, ValidUntil: "2027-01-01"})
	uc.Create(context.Background(), CertificateDTO{Scheme: "Halal", Number: "H", SupplierID: &sup, ValidUntil: "2027-01-01"})

	if _, err := uc.Suspend(context.Background(), a.ID, ""); err == nil {
		t.Fatal("suspend needs a reason")
	}
	if _, err := uc.Suspend(context.Background(), a.ID, "audit finding"); err != nil {
		t.Fatal(err)
	}
	el, _ := uc.Eligibility(context.Background(), "")
	if len(el) != 1 || len(el[0].Active) != 1 || el[0].Active[0] != "Halal" || len(el[0].Suspended) != 1 {
		t.Fatalf("eligibility wrong: %+v", el)
	}
	if _, err := uc.Resume(context.Background(), a.ID); err != nil {
		t.Fatal(err)
	}
	el, _ = uc.Eligibility(context.Background(), "")
	if len(el[0].Active) != 2 {
		t.Fatalf("resumed cert must be active again: %+v", el)
	}
}

func TestValidation(t *testing.T) {
	uc, _ := newUC("2026-06-01")
	sup := uuid.New()
	for _, bad := range []CertificateDTO{
		{Number: "1", SupplierID: &sup, ValidUntil: "2027-01-01"},
		{Scheme: "X", Number: "1", ValidUntil: "2027-01-01"},
		{Scheme: "X", Number: "1", SupplierID: &sup},
		{Scheme: "X", Number: "1", SupplierID: &sup, ValidFrom: "2027-02-01", ValidUntil: "2027-01-01"},
	} {
		if _, err := uc.Create(context.Background(), bad); err == nil {
			t.Fatalf("expected validation error for %+v", bad)
		}
	}
}
