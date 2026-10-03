package infrastructure

import (
	"context"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/activitylog/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"gorm.io/gorm"
)

type activityLogRepository struct {
	db *gorm.DB
}

func NewActivityLogRepository(db *gorm.DB) domain.ActivityLogRepository {
	return &activityLogRepository{db: db}
}

func (r *activityLogRepository) Create(ctx context.Context, log *domain.ActivityLog) error {
	tenantctx.SetTenantID(ctx, &log.TenantID)
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *activityLogRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.ActivityLog, int64, error) {
	var items []domain.ActivityLog
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ActivityLog{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("user_email ILIKE ? OR path ILIKE ? OR module ILIKE ?", pattern, pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}
