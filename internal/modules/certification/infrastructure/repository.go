package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/certification/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) domain.Repository { return &repository{db: db} }

func (r *repository) Create(ctx context.Context, c *domain.Certificate) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Certificate, error) {
	var c domain.Certificate
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

func (r *repository) Update(ctx context.Context, c *domain.Certificate) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *repository) List(ctx context.Context, f domain.ListFilter) ([]domain.Certificate, error) {
	var out []domain.Certificate
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Certificate{}))
	if f.SupplierID != nil {
		db = db.Where("supplier_id = ?", *f.SupplierID)
	}
	if f.ProductID != nil {
		db = db.Where("product_id = ?", *f.ProductID)
	}
	if f.Scheme != "" {
		db = db.Where("LOWER(scheme) = LOWER(?)", f.Scheme)
	}
	return out, db.Order("valid_until asc").Find(&out).Error
}
