package application

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/certification/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// CommonSchemes are suggestions for the UI; any scheme name is accepted.
var CommonSchemes = []string{"USDA Organic", "EU Organic", "JAS Organic", "Halal", "HACCP", "ISO 22000", "FSSC 22000", "Fair Trade", "Kosher"}

type CertificateDTO struct {
	Scheme     string     `json:"scheme"`
	Number     string     `json:"number"`
	Issuer     string     `json:"issuer"`
	SupplierID *uuid.UUID `json:"supplierId,omitempty"`
	ProductID  *uuid.UUID `json:"productId,omitempty"`
	ValidFrom  string     `json:"validFrom"`
	ValidUntil string     `json:"validUntil"`
	Attachment string     `json:"attachment"`
	Notes      string     `json:"notes"`
}

type CertificateResponse struct {
	domain.Certificate
	// EffectiveStatus: active | suspended | revoked | expired | not_yet_valid
	EffectiveStatus string `json:"effectiveStatus"`
	DaysLeft        *int   `json:"daysLeft,omitempty"`
}

type SupplierEligibility struct {
	SupplierID uuid.UUID `json:"supplierId"`
	Active     []string  `json:"active"`
	Suspended  []string  `json:"suspended"`
	Expired    []string  `json:"expired"`
}

type UseCase interface {
	Create(ctx context.Context, dto CertificateDTO) (*CertificateResponse, error)
	Update(ctx context.Context, id uuid.UUID, dto CertificateDTO) (*CertificateResponse, error)
	Get(ctx context.Context, id uuid.UUID) (*CertificateResponse, error)
	List(ctx context.Context, f domain.ListFilter) ([]CertificateResponse, error)
	Suspend(ctx context.Context, id uuid.UUID, reason string) (*CertificateResponse, error)
	Resume(ctx context.Context, id uuid.UUID) (*CertificateResponse, error)
	Revoke(ctx context.Context, id uuid.UUID, reason string) (*CertificateResponse, error)
	// Eligibility groups every supplier's certificates by effective status. With scheme set,
	// the caller filters on Active containing that scheme (used by the vendor pickers in PR/PO).
	Eligibility(ctx context.Context, scheme string) ([]SupplierEligibility, error)
}

type useCase struct {
	repo domain.Repository
	now  func() time.Time
}

func NewUseCase(repo domain.Repository) UseCase { return &useCase{repo: repo, now: time.Now} }

func (uc *useCase) today() string { return uc.now().Format("2006-01-02") }

func (uc *useCase) respond(c *domain.Certificate) *CertificateResponse {
	r := &CertificateResponse{Certificate: *c, EffectiveStatus: c.EffectiveStatus(uc.today())}
	if c.ValidUntil != "" {
		if end, err := time.Parse("2006-01-02", c.ValidUntil); err == nil {
			d := int(end.Sub(uc.now().Truncate(24*time.Hour)).Hours() / 24)
			r.DaysLeft = &d
		}
	}
	return r
}

func validate(dto CertificateDTO) error {
	if strings.TrimSpace(dto.Scheme) == "" || strings.TrimSpace(dto.Number) == "" {
		return apperrors.NewBadRequest("Scheme and certificate number are required")
	}
	if dto.SupplierID == nil && dto.ProductID == nil {
		return apperrors.NewBadRequest("A certificate must belong to a supplier and/or a product")
	}
	var from, until time.Time
	var err error
	if dto.ValidFrom != "" {
		if from, err = time.Parse("2006-01-02", dto.ValidFrom); err != nil {
			return apperrors.NewBadRequest("validFrom must be YYYY-MM-DD")
		}
	}
	if dto.ValidUntil == "" {
		return apperrors.NewBadRequest("validUntil is required so the certificate can switch itself off")
	}
	if until, err = time.Parse("2006-01-02", dto.ValidUntil); err != nil {
		return apperrors.NewBadRequest("validUntil must be YYYY-MM-DD")
	}
	if !from.IsZero() && until.Before(from) {
		return apperrors.NewBadRequest("validUntil cannot be before validFrom")
	}
	return nil
}

func apply(c *domain.Certificate, dto CertificateDTO) {
	c.Scheme, c.Number, c.Issuer = strings.TrimSpace(dto.Scheme), strings.TrimSpace(dto.Number), strings.TrimSpace(dto.Issuer)
	c.SupplierID, c.ProductID = dto.SupplierID, dto.ProductID
	c.ValidFrom, c.ValidUntil = dto.ValidFrom, dto.ValidUntil
	c.Attachment, c.Notes = dto.Attachment, dto.Notes
}

func (uc *useCase) Create(ctx context.Context, dto CertificateDTO) (*CertificateResponse, error) {
	if err := validate(dto); err != nil {
		return nil, err
	}
	c := &domain.Certificate{Status: domain.StatusActive}
	apply(c, dto)
	if err := uc.repo.Create(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create certificate")
	}
	return uc.respond(c), nil
}

func (uc *useCase) load(ctx context.Context, id uuid.UUID) (*domain.Certificate, error) {
	c, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get certificate")
	}
	if c == nil {
		return nil, apperrors.NewNotFound("Certificate not found")
	}
	return c, nil
}

func (uc *useCase) Update(ctx context.Context, id uuid.UUID, dto CertificateDTO) (*CertificateResponse, error) {
	if err := validate(dto); err != nil {
		return nil, err
	}
	c, err := uc.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status == domain.StatusRevoked {
		return nil, apperrors.NewConflict("A revoked certificate cannot be edited")
	}
	apply(c, dto)
	if err := uc.repo.Update(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update certificate")
	}
	return uc.respond(c), nil
}

func (uc *useCase) Get(ctx context.Context, id uuid.UUID) (*CertificateResponse, error) {
	c, err := uc.load(ctx, id)
	if err != nil {
		return nil, err
	}
	return uc.respond(c), nil
}

func (uc *useCase) List(ctx context.Context, f domain.ListFilter) ([]CertificateResponse, error) {
	list, err := uc.repo.List(ctx, f)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list certificates")
	}
	out := make([]CertificateResponse, len(list))
	for i := range list {
		out[i] = *uc.respond(&list[i])
	}
	return out, nil
}

func (uc *useCase) Suspend(ctx context.Context, id uuid.UUID, reason string) (*CertificateResponse, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, apperrors.NewBadRequest("A reason is required to suspend a certificate")
	}
	c, err := uc.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status == domain.StatusRevoked {
		return nil, apperrors.NewConflict("A revoked certificate cannot be suspended")
	}
	now := uc.now()
	c.Status, c.SuspendReason, c.SuspendedAt, c.SuspendedBy = domain.StatusSuspended, strings.TrimSpace(reason), &now, actor.EmailFrom(ctx)
	if err := uc.repo.Update(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to suspend certificate")
	}
	return uc.respond(c), nil
}

func (uc *useCase) Resume(ctx context.Context, id uuid.UUID) (*CertificateResponse, error) {
	c, err := uc.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Status != domain.StatusSuspended {
		return nil, apperrors.NewConflict("Only a suspended certificate can be resumed")
	}
	c.Status, c.SuspendReason, c.SuspendedAt, c.SuspendedBy = domain.StatusActive, "", nil, ""
	if err := uc.repo.Update(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to resume certificate")
	}
	return uc.respond(c), nil
}

func (uc *useCase) Revoke(ctx context.Context, id uuid.UUID, reason string) (*CertificateResponse, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, apperrors.NewBadRequest("A reason is required to revoke a certificate")
	}
	c, err := uc.load(ctx, id)
	if err != nil {
		return nil, err
	}
	c.Status, c.SuspendReason = domain.StatusRevoked, strings.TrimSpace(reason)
	if err := uc.repo.Update(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to revoke certificate")
	}
	return uc.respond(c), nil
}

func (uc *useCase) Eligibility(ctx context.Context, scheme string) ([]SupplierEligibility, error) {
	list, err := uc.repo.List(ctx, domain.ListFilter{Scheme: scheme})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list certificates")
	}
	bySupplier := map[uuid.UUID]*SupplierEligibility{}
	var order []uuid.UUID
	today := uc.today()
	for _, c := range list {
		if c.SupplierID == nil {
			continue
		}
		e := bySupplier[*c.SupplierID]
		if e == nil {
			e = &SupplierEligibility{SupplierID: *c.SupplierID, Active: []string{}, Suspended: []string{}, Expired: []string{}}
			bySupplier[*c.SupplierID] = e
			order = append(order, *c.SupplierID)
		}
		switch c.EffectiveStatus(today) {
		case domain.StatusActive:
			e.Active = append(e.Active, c.Scheme)
		case domain.StatusSuspended:
			e.Suspended = append(e.Suspended, c.Scheme)
		default: // expired, not yet valid, revoked
			e.Expired = append(e.Expired, c.Scheme)
		}
	}
	out := make([]SupplierEligibility, 0, len(order))
	for _, id := range order {
		out = append(out, *bySupplier[id])
	}
	return out, nil
}
