package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/salesman/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type salesmanRepository struct {
	db *gorm.DB
}

func NewSalesmanRepository(db *gorm.DB) domain.SalesmanRepository {
	return &salesmanRepository{db: db}
}

func (r *salesmanRepository) Create(ctx context.Context, salesman *domain.Salesman) error {
	tenantctx.SetTenantID(ctx, &salesman.TenantID)
	return r.db.WithContext(ctx).Create(salesman).Error
}

func (r *salesmanRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Salesman, error) {
	var salesman domain.Salesman
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&salesman).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &salesman, nil
}

func (r *salesmanRepository) GetByCode(ctx context.Context, code string) (*domain.Salesman, error) {
	var salesman domain.Salesman
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&salesman).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &salesman, nil
}

func (r *salesmanRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.Salesman, int64, error) {
	var salesmans []domain.Salesman
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Salesman{}))

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
		Find(&salesmans).Error

	return salesmans, total, err
}

func (r *salesmanRepository) Update(ctx context.Context, salesman *domain.Salesman) error {
	return r.db.WithContext(ctx).Save(salesman).Error
}

func (r *salesmanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Salesman{}, id).Error
}

func (r *salesmanRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Salesman{}).Count(&total).Error
	return total, err
}
