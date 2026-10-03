package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type crmRepository struct {
	db *gorm.DB
}

func NewCRMRepository(db *gorm.DB) domain.CRMRepository {
	return &crmRepository{db: db}
}

func (r *crmRepository) Create(ctx context.Context, lead *domain.Lead) error {
	tenantctx.SetTenantID(ctx, &lead.TenantID)
	return r.db.WithContext(ctx).Create(lead).Error
}

func (r *crmRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Lead, error) {
	var lead domain.Lead
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&lead).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &lead, nil
}

func (r *crmRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.Lead, int64, error) {
	var leads []domain.Lead
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Lead{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("name ILIKE ? OR company ILIKE ? OR email ILIKE ?", searchPattern, searchPattern, searchPattern)
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
		Find(&leads).Error

	return leads, total, err
}

func (r *crmRepository) Update(ctx context.Context, lead *domain.Lead) error {
	return r.db.WithContext(ctx).Save(lead).Error
}

func (r *crmRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Lead{}, id).Error
}

func (r *crmRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Lead{}).Count(&total).Error
	return total, err
}

func (r *crmRepository) CreateDeal(ctx context.Context, deal *domain.Deal) error {
	tenantctx.SetTenantID(ctx, &deal.TenantID)
	return r.db.WithContext(ctx).Create(deal).Error
}

func (r *crmRepository) GetDealByID(ctx context.Context, id uuid.UUID) (*domain.Deal, error) {
	var deal domain.Deal
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&deal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &deal, nil
}

func (r *crmRepository) ListDeals(ctx context.Context, query types.PaginationQuery) ([]domain.Deal, int64, error) {
	var deals []domain.Deal
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Deal{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ? OR customer ILIKE ?", searchPattern, searchPattern)
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
		Find(&deals).Error

	return deals, total, err
}

func (r *crmRepository) UpdateDeal(ctx context.Context, deal *domain.Deal) error {
	return r.db.WithContext(ctx).Save(deal).Error
}

func (r *crmRepository) DeleteDeal(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Deal{}, id).Error
}

func (r *crmRepository) CountDeals(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Deal{}).Count(&total).Error
	return total, err
}
