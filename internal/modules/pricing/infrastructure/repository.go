package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/pricing/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type pricingRepository struct {
	db *gorm.DB
}

func NewPricingRepository(db *gorm.DB) domain.PricingRepository {
	return &pricingRepository{db: db}
}

func (r *pricingRepository) CreatePaymentTerm(ctx context.Context, p *domain.PaymentTerm) error {
	tenantctx.SetTenantID(ctx, &p.TenantID)
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *pricingRepository) ListPaymentTerms(ctx context.Context) ([]domain.PaymentTerm, error) {
	var items []domain.PaymentTerm
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PaymentTerm{}))
	err := db.Order("days asc").Find(&items).Error
	return items, err
}

func (r *pricingRepository) GetPaymentTermByID(ctx context.Context, id uuid.UUID) (*domain.PaymentTerm, error) {
	var p domain.PaymentTerm
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *pricingRepository) UpdatePaymentTerm(ctx context.Context, p *domain.PaymentTerm) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *pricingRepository) DeletePaymentTerm(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.PaymentTerm{}, id).Error
}

func (r *pricingRepository) CreateCustomerType(ctx context.Context, c *domain.CustomerType) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *pricingRepository) ListCustomerTypes(ctx context.Context) ([]domain.CustomerType, error) {
	var items []domain.CustomerType
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.CustomerType{}))
	err := db.Order("name asc").Find(&items).Error
	return items, err
}

func (r *pricingRepository) GetCustomerTypeByID(ctx context.Context, id uuid.UUID) (*domain.CustomerType, error) {
	var c domain.CustomerType
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *pricingRepository) UpdateCustomerType(ctx context.Context, c *domain.CustomerType) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *pricingRepository) DeleteCustomerType(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.CustomerType{}, id).Error
}

func (r *pricingRepository) CreateTaxRate(ctx context.Context, t *domain.TaxRate) error {
	tenantctx.SetTenantID(ctx, &t.TenantID)
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *pricingRepository) ListTaxRates(ctx context.Context) ([]domain.TaxRate, error) {
	var items []domain.TaxRate
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.TaxRate{}))
	err := db.Order("rate asc").Find(&items).Error
	return items, err
}

func (r *pricingRepository) GetTaxRateByID(ctx context.Context, id uuid.UUID) (*domain.TaxRate, error) {
	var t domain.TaxRate
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *pricingRepository) UpdateTaxRate(ctx context.Context, t *domain.TaxRate) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *pricingRepository) DeleteTaxRate(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.TaxRate{}, id).Error
}
