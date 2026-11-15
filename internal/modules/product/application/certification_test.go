package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/google/uuid"
)

type certRepoStub struct {
	domain.ProductRepository
	certs map[uuid.UUID]*domain.Certification
	byCode map[string]*domain.Certification
}

func newCertRepoStub() *certRepoStub {
	return &certRepoStub{
		certs:  make(map[uuid.UUID]*domain.Certification),
		byCode: make(map[string]*domain.Certification),
	}
}

func (r *certRepoStub) CreateCertification(ctx context.Context, c *domain.Certification) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	r.certs[c.ID] = c
	r.byCode[c.Code] = c
	return nil
}

func (r *certRepoStub) GetCertificationByID(ctx context.Context, id uuid.UUID) (*domain.Certification, error) {
	return r.certs[id], nil
}

func (r *certRepoStub) GetCertificationByCode(ctx context.Context, code string) (*domain.Certification, error) {
	return r.byCode[code], nil
}

func (r *certRepoStub) ListCertifications(ctx context.Context) ([]domain.Certification, error) {
	var list []domain.Certification
	for _, c := range r.certs {
		list = append(list, *c)
	}
	return list, nil
}

func (r *certRepoStub) UpdateCertification(ctx context.Context, c *domain.Certification) error {
	r.certs[c.ID] = c
	r.byCode[c.Code] = c
	return nil
}

func (r *certRepoStub) DeleteCertification(ctx context.Context, id uuid.UUID) error {
	if c, ok := r.certs[id]; ok {
		delete(r.byCode, c.Code)
		delete(r.certs, id)
	}
	return nil
}

func TestCertificationAttachmentCreateAndUpdate(t *testing.T) {
	repo := newCertRepoStub()
	uc := NewProductUseCase(repo)

	// 1. Create certification with attachment
	created, err := uc.CreateCertification(context.Background(), CreateCertificationDTO{
		Code:           "ISO22000",
		Name:           "ISO 22000 Food Safety",
		Description:    "Food safety management standard",
		Attachment:     "product-certifications/abc/iso22000.pdf",
		AttachmentName: "iso22000.pdf",
	})
	if err != nil {
		t.Fatalf("unexpected error creating cert: %v", err)
	}
	if created.Code != "ISO22000" || created.Attachment != "product-certifications/abc/iso22000.pdf" || created.AttachmentName != "iso22000.pdf" {
		t.Fatalf("unexpected created cert: %+v", created)
	}

	// 2. Update certification attachment
	newAttachment := "product-certifications/xyz/iso22000_v2.pdf"
	newName := "iso22000_v2.pdf"
	updated, err := uc.UpdateCertification(context.Background(), created.ID, UpdateCertificationDTO{
		Attachment:     &newAttachment,
		AttachmentName: &newName,
	})
	if err != nil {
		t.Fatalf("unexpected error updating cert: %v", err)
	}
	if updated.Attachment != newAttachment || updated.AttachmentName != newName {
		t.Fatalf("unexpected updated cert: %+v", updated)
	}
}
