package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/modules/company/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type companyRepository struct {
	db *gorm.DB
}

func NewCompanyRepository(db *gorm.DB) domain.CompanyRepository {
	return &companyRepository{db: db}
}

func (r *companyRepository) Create(ctx context.Context, company *domain.Company) error {
	return r.db.WithContext(ctx).Create(company).Error
}

func (r *companyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	var company domain.Company
	err := r.db.WithContext(ctx).First(&company, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &company, nil
}

func (r *companyRepository) GetByCode(ctx context.Context, code string) (*domain.Company, error) {
	var company domain.Company
	err := r.db.WithContext(ctx).First(&company, "code = ?", code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &company, nil
}

func (r *companyRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.Company, int64, error) {
	var companies []domain.Company
	var total int64

	dbQuery := r.db.WithContext(ctx).Model(&domain.Company{})

	if query.Search != "" {
		searchParam := "%" + query.Search + "%"
		dbQuery = dbQuery.Where("code ILIKE ? OR name ILIKE ?", searchParam, searchParam)
	}

	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := "created_at"
	if query.SortBy != "" {
		// Basic sanitization of sort column
		allowedSorts := map[string]bool{
			"code":       true,
			"name":       true,
			"created_at": true,
			"updated_at": true,
		}
		if allowedSorts[query.SortBy] {
			sortBy = query.SortBy
		}
	}

	orderClause := fmt.Sprintf("%s %s", sortBy, query.SortDir)

	err := dbQuery.Order(orderClause).
		Offset(query.Offset()).
		Limit(query.PerPage).
		Find(&companies).Error

	if err != nil {
		return nil, 0, err
	}

	return companies, total, nil
}

func (r *companyRepository) Update(ctx context.Context, company *domain.Company) error {
	return r.db.WithContext(ctx).Save(company).Error
}

func (r *companyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Company{}, "id = ?", id).Error
}
