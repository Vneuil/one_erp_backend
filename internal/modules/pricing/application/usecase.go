package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/pricing/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

type PricingUseCase interface {
	CreatePaymentTerm(ctx context.Context, dto CreatePaymentTermDTO) (*PaymentTermResponseDTO, error)
	ListPaymentTerms(ctx context.Context) ([]PaymentTermResponseDTO, error)
	UpdatePaymentTerm(ctx context.Context, id uuid.UUID, dto UpdatePaymentTermDTO) (*PaymentTermResponseDTO, error)
	DeletePaymentTerm(ctx context.Context, id uuid.UUID) error

	CreateCustomerType(ctx context.Context, dto CreateCustomerTypeDTO) (*CustomerTypeResponseDTO, error)
	ListCustomerTypes(ctx context.Context) ([]CustomerTypeResponseDTO, error)
	UpdateCustomerType(ctx context.Context, id uuid.UUID, dto UpdateCustomerTypeDTO) (*CustomerTypeResponseDTO, error)
	DeleteCustomerType(ctx context.Context, id uuid.UUID) error

	CreateTaxRate(ctx context.Context, dto CreateTaxRateDTO) (*TaxRateResponseDTO, error)
	ListTaxRates(ctx context.Context) ([]TaxRateResponseDTO, error)
	UpdateTaxRate(ctx context.Context, id uuid.UUID, dto UpdateTaxRateDTO) (*TaxRateResponseDTO, error)
	DeleteTaxRate(ctx context.Context, id uuid.UUID) error
}

type pricingUseCase struct {
	repo domain.PricingRepository
}

func NewPricingUseCase(repo domain.PricingRepository) PricingUseCase {
	return &pricingUseCase{repo: repo}
}

func (uc *pricingUseCase) CreatePaymentTerm(ctx context.Context, dto CreatePaymentTermDTO) (*PaymentTermResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Payment term name is required")
	}
	if dto.Days < 0 {
		return nil, apperrors.NewBadRequest("Days cannot be negative")
	}
	p := &domain.PaymentTerm{Name: name, Days: dto.Days}
	if err := uc.repo.CreatePaymentTerm(ctx, p); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create payment term")
	}
	return &PaymentTermResponseDTO{ID: p.ID, Name: p.Name, Days: p.Days}, nil
}

func (uc *pricingUseCase) ListPaymentTerms(ctx context.Context) ([]PaymentTermResponseDTO, error) {
	items, err := uc.repo.ListPaymentTerms(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list payment terms")
	}
	return ToPaymentTermResponseList(items), nil
}

func (uc *pricingUseCase) UpdatePaymentTerm(ctx context.Context, id uuid.UUID, dto UpdatePaymentTermDTO) (*PaymentTermResponseDTO, error) {
	p, err := uc.repo.GetPaymentTermByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get payment term")
	}
	if p == nil {
		return nil, apperrors.NewNotFound("Payment term not found")
	}
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Payment term name is required")
	}
	if dto.Days < 0 {
		return nil, apperrors.NewBadRequest("Days cannot be negative")
	}
	p.Name = name
	p.Days = dto.Days
	if err := uc.repo.UpdatePaymentTerm(ctx, p); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update payment term")
	}
	return &PaymentTermResponseDTO{ID: p.ID, Name: p.Name, Days: p.Days}, nil
}

func (uc *pricingUseCase) DeletePaymentTerm(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeletePaymentTerm(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete payment term")
	}
	return nil
}

func (uc *pricingUseCase) CreateCustomerType(ctx context.Context, dto CreateCustomerTypeDTO) (*CustomerTypeResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Customer type name is required")
	}
	c := &domain.CustomerType{Name: name}
	if err := uc.repo.CreateCustomerType(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create customer type")
	}
	return &CustomerTypeResponseDTO{ID: c.ID, Name: c.Name}, nil
}

func (uc *pricingUseCase) ListCustomerTypes(ctx context.Context) ([]CustomerTypeResponseDTO, error) {
	items, err := uc.repo.ListCustomerTypes(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list customer types")
	}
	return ToCustomerTypeResponseList(items), nil
}

func (uc *pricingUseCase) UpdateCustomerType(ctx context.Context, id uuid.UUID, dto UpdateCustomerTypeDTO) (*CustomerTypeResponseDTO, error) {
	c, err := uc.repo.GetCustomerTypeByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get customer type")
	}
	if c == nil {
		return nil, apperrors.NewNotFound("Customer type not found")
	}
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Customer type name is required")
	}
	c.Name = name
	if err := uc.repo.UpdateCustomerType(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update customer type")
	}
	return &CustomerTypeResponseDTO{ID: c.ID, Name: c.Name}, nil
}

func (uc *pricingUseCase) DeleteCustomerType(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteCustomerType(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete customer type")
	}
	return nil
}

func (uc *pricingUseCase) CreateTaxRate(ctx context.Context, dto CreateTaxRateDTO) (*TaxRateResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Tax rate name is required")
	}
	if dto.Rate < 0 {
		return nil, apperrors.NewBadRequest("Rate cannot be negative")
	}
	t := &domain.TaxRate{Name: name, Rate: dto.Rate, IsDefault: dto.IsDefault}
	if err := uc.repo.CreateTaxRate(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create tax rate")
	}
	return &TaxRateResponseDTO{ID: t.ID, Name: t.Name, Rate: t.Rate, IsDefault: t.IsDefault}, nil
}

func (uc *pricingUseCase) ListTaxRates(ctx context.Context) ([]TaxRateResponseDTO, error) {
	items, err := uc.repo.ListTaxRates(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list tax rates")
	}
	return ToTaxRateResponseList(items), nil
}

func (uc *pricingUseCase) UpdateTaxRate(ctx context.Context, id uuid.UUID, dto UpdateTaxRateDTO) (*TaxRateResponseDTO, error) {
	t, err := uc.repo.GetTaxRateByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get tax rate")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Tax rate not found")
	}
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Tax rate name is required")
	}
	if dto.Rate < 0 {
		return nil, apperrors.NewBadRequest("Rate cannot be negative")
	}
	t.Name = name
	t.Rate = dto.Rate
	t.IsDefault = dto.IsDefault
	if err := uc.repo.UpdateTaxRate(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update tax rate")
	}
	return &TaxRateResponseDTO{ID: t.ID, Name: t.Name, Rate: t.Rate, IsDefault: t.IsDefault}, nil
}

func (uc *pricingUseCase) DeleteTaxRate(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteTaxRate(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete tax rate")
	}
	return nil
}
