package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/commission/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type commissionRepository struct {
	db *gorm.DB
}

func NewCommissionRepository(db *gorm.DB) domain.CommissionRepository {
	return &commissionRepository{db: db}
}

func (r *commissionRepository) CreateRule(ctx context.Context, rule *domain.CommissionRule) error {
	tenantctx.SetTenantID(ctx, &rule.TenantID)
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *commissionRepository) GetRuleByID(ctx context.Context, id uuid.UUID) (*domain.CommissionRule, error) {
	var rule domain.CommissionRule
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *commissionRepository) ListRules(ctx context.Context, query types.PaginationQuery) ([]domain.CommissionRule, int64, error) {
	var rules []domain.CommissionRule
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.CommissionRule{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&rules).Error
	return rules, total, err
}

func (r *commissionRepository) CountRules(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.CommissionRule{}).Count(&total).Error
	return total, err
}

func (r *commissionRepository) CreateRecord(ctx context.Context, rec *domain.CommissionRecord) error {
	tenantctx.SetTenantID(ctx, &rec.TenantID)
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *commissionRepository) GetRecordByID(ctx context.Context, id uuid.UUID) (*domain.CommissionRecord, error) {
	var rec domain.CommissionRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *commissionRepository) ListRecords(ctx context.Context, query types.PaginationQuery) ([]domain.CommissionRecord, int64, error) {
	var records []domain.CommissionRecord
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.CommissionRecord{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("salesperson_name ILIKE ? OR sales_order_number ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&records).Error
	return records, total, err
}

func (r *commissionRepository) ListRecordsByPeriod(ctx context.Context, period string) ([]domain.CommissionRecord, error) {
	var records []domain.CommissionRecord
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("period = ?", period).Order("created_at desc").Find(&records).Error
	return records, err
}

func (r *commissionRepository) UpdateRecord(ctx context.Context, rec *domain.CommissionRecord) error {
	return r.db.WithContext(ctx).Save(rec).Error
}

func (r *commissionRepository) CountRecords(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.CommissionRecord{}).Count(&total).Error
	return total, err
}
