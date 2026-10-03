package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/shipping/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type shippingMethodRepository struct {
	db *gorm.DB
}

func NewShippingMethodRepository(db *gorm.DB) domain.ShippingMethodRepository {
	return &shippingMethodRepository{db: db}
}

func (r *shippingMethodRepository) Create(ctx context.Context, shippingMethod *domain.ShippingMethod) error {
	tenantctx.SetTenantID(ctx, &shippingMethod.TenantID)
	return r.db.WithContext(ctx).Create(shippingMethod).Error
}

func (r *shippingMethodRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ShippingMethod, error) {
	var shippingMethod domain.ShippingMethod
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&shippingMethod).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &shippingMethod, nil
}

func (r *shippingMethodRepository) GetByCode(ctx context.Context, code string) (*domain.ShippingMethod, error) {
	var shippingMethod domain.ShippingMethod
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&shippingMethod).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &shippingMethod, nil
}

func (r *shippingMethodRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.ShippingMethod, int64, error) {
	var shippingMethods []domain.ShippingMethod
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ShippingMethod{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ? OR code ILIKE ? OR email ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortField := "created_at"
	sortDir := "desc"
	if query.SortBy != "" {
		sortField = query.SortBy
	}
	if query.SortDir != "" {
		sortDir = query.SortDir
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order(fmt.Sprintf("%s %s", sortField, sortDir)).
		Offset(offset).
		Limit(query.PerPage).
		Find(&shippingMethods).Error

	return shippingMethods, total, err
}

func (r *shippingMethodRepository) Update(ctx context.Context, shippingMethod *domain.ShippingMethod) error {
	return r.db.WithContext(ctx).Save(shippingMethod).Error
}

func (r *shippingMethodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.ShippingMethod{}, id).Error
}

func (r *shippingMethodRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.ShippingMethod{}).Count(&total).Error
	return total, err
}
