package infrastructure

import (
	"context"

	"github.com/divinecoid/one-backend/internal/modules/workspace/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type tenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) domain.TenantRepository {
	return &tenantRepository{db: db}
}

func (r *tenantRepository) Create(ctx context.Context, t *domain.Tenant) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *tenantRepository) List(ctx context.Context) ([]domain.Tenant, error) {
	var tenants []domain.Tenant
	err := r.db.WithContext(ctx).Order("created_at asc").Find(&tenants).Error
	return tenants, err
}

func (r *tenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	var t domain.Tenant
	err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *tenantRepository) GetDefault(ctx context.Context) (*domain.Tenant, error) {
	var t domain.Tenant
	err := r.db.WithContext(ctx).First(&t, "is_default = ?", true).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
