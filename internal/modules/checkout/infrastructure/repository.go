package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/modules/checkout/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type checkoutRepository struct {
	db *gorm.DB
}

func NewCheckoutRepository(db *gorm.DB) domain.CheckoutOrderRepository {
	return &checkoutRepository{db: db}
}

func (r *checkoutRepository) Create(ctx context.Context, o *domain.CheckoutOrder) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *checkoutRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.CheckoutOrder, error) {
	var o domain.CheckoutOrder
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *checkoutRepository) GetByProviderRef(ctx context.Context, providerRef string) (*domain.CheckoutOrder, error) {
	var o domain.CheckoutOrder
	err := r.db.WithContext(ctx).Where("provider_ref = ?", providerRef).First(&o).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &o, nil
}

func (r *checkoutRepository) Update(ctx context.Context, o *domain.CheckoutOrder) error {
	return r.db.WithContext(ctx).Save(o).Error
}
