package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/assets/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type assetsRepository struct {
	db *gorm.DB
}

func NewAssetsRepository(db *gorm.DB) domain.AssetsRepository {
	return &assetsRepository{db: db}
}

func (r *assetsRepository) CreateAsset(ctx context.Context, a *domain.FixedAsset) error {
	tenantctx.SetTenantID(ctx, &a.TenantID)
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *assetsRepository) GetAssetByID(ctx context.Context, id uuid.UUID) (*domain.FixedAsset, error) {
	var a domain.FixedAsset
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *assetsRepository) ListAssets(ctx context.Context, query types.PaginationQuery) ([]domain.FixedAsset, int64, error) {
	var assets []domain.FixedAsset
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.FixedAsset{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("asset_code ILIKE ? OR name ILIKE ? OR location ILIKE ?", pattern, pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&assets).Error
	return assets, total, err
}

func (r *assetsRepository) ListAllAssets(ctx context.Context) ([]domain.FixedAsset, error) {
	var assets []domain.FixedAsset
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Order("created_at desc").Find(&assets).Error
	return assets, err
}

func (r *assetsRepository) UpdateAsset(ctx context.Context, a *domain.FixedAsset) error {
	return tenantctx.Scope(ctx, r.db.WithContext(ctx)).Save(a).Error
}

// CountAssets intentionally returns a platform-wide count across all tenants.
// It is only called from SeedInitialData (internal/modules/assets/application/usecase.go)
// to decide whether sample data has already been seeded on first boot, and is
// never exposed via a tenant-facing handler or endpoint, so it must remain unscoped.
func (r *assetsRepository) CountAssets(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.FixedAsset{}).Count(&total).Error
	return total, err
}

func (r *assetsRepository) CreateDepreciationPosting(ctx context.Context, p *domain.DepreciationPosting) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *assetsRepository) ListDepreciationPostings(ctx context.Context, period string) ([]domain.DepreciationPosting, error) {
	var out []domain.DepreciationPosting
	db := r.db.WithContext(ctx)
	if period != "" {
		db = db.Where("period = ?", period)
	}
	err := db.Order("period asc").Find(&out).Error
	return out, err
}
