package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/kpi/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type kpiRepository struct {
	db *gorm.DB
}

func NewKpiRepository(db *gorm.DB) domain.KpiRepository {
	return &kpiRepository{db: db}
}

func (r *kpiRepository) Create(ctx context.Context, review *domain.KpiReview) error {
	tenantctx.SetTenantID(ctx, &review.TenantID)
	return r.db.WithContext(ctx).Create(review).Error
}

func (r *kpiRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.KpiReview, error) {
	var review domain.KpiReview
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.KpiReview{})).Where("id = ?", id).First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &review, nil
}

func (r *kpiRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.KpiReview, int64, error) {
	var items []domain.KpiReview
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.KpiReview{}))

	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("employee_name ILIKE ? OR department ILIKE ? OR period ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *kpiRepository) Update(ctx context.Context, review *domain.KpiReview) error {
	return r.db.WithContext(ctx).Save(review).Error
}

func (r *kpiRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.KpiReview{})).Count(&total).Error
	return total, err
}
