package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/cooperative/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type cooperativeRepository struct {
	db *gorm.DB
}

func NewCooperativeRepository(db *gorm.DB) domain.CooperativeRepository {
	return &cooperativeRepository{db: db}
}

func (r *cooperativeRepository) CreateLoan(ctx context.Context, loan *domain.CooperativeLoan) error {
	tenantctx.SetTenantID(ctx, &loan.TenantID)
	return r.db.WithContext(ctx).Create(loan).Error
}

func (r *cooperativeRepository) GetLoanByID(ctx context.Context, id uuid.UUID) (*domain.CooperativeLoan, error) {
	var loan domain.CooperativeLoan
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Where("id = ?", id)).First(&loan).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &loan, nil
}

func (r *cooperativeRepository) UpdateLoan(ctx context.Context, loan *domain.CooperativeLoan) error {
	return r.db.WithContext(ctx).Save(loan).Error
}

func (r *cooperativeRepository) ListLoans(ctx context.Context, query types.PaginationQuery) ([]domain.CooperativeLoan, int64, error) {
	var loans []domain.CooperativeLoan
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.CooperativeLoan{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("employee_name ILIKE ? OR loan_no ILIKE ? OR department ILIKE ?", searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&loans).Error
	return loans, total, err
}

func (r *cooperativeRepository) CountLoans(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.CooperativeLoan{})).Count(&total).Error
	return total, err
}

func (r *cooperativeRepository) ListActiveLoansByEmployeeID(ctx context.Context, employeeID uuid.UUID) ([]domain.CooperativeLoan, error) {
	var loans []domain.CooperativeLoan
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.CooperativeLoan{}))
	err := db.Where("employee_id = ? AND status = ?", employeeID, "active").Order("created_at asc").Find(&loans).Error
	return loans, err
}
