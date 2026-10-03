package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type reimbursementRepository struct {
	db *gorm.DB
}

func NewReimbursementRepository(db *gorm.DB) domain.ReimbursementRepository {
	return &reimbursementRepository{db: db}
}

func (r *reimbursementRepository) CreateClaim(ctx context.Context, claim *domain.ReimbursementClaim) error {
	tenantctx.SetTenantID(ctx, &claim.TenantID)
	return r.db.WithContext(ctx).Create(claim).Error
}

func (r *reimbursementRepository) GetClaimByID(ctx context.Context, id uuid.UUID) (*domain.ReimbursementClaim, error) {
	var claim domain.ReimbursementClaim
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Where("id = ?", id)).First(&claim).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &claim, nil
}

func (r *reimbursementRepository) UpdateClaim(ctx context.Context, claim *domain.ReimbursementClaim) error {
	return r.db.WithContext(ctx).Save(claim).Error
}

func (r *reimbursementRepository) ListClaims(ctx context.Context, query types.PaginationQuery) ([]domain.ReimbursementClaim, int64, error) {
	var claims []domain.ReimbursementClaim
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ReimbursementClaim{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("employee_name ILIKE ? OR claim_no ILIKE ? OR department ILIKE ? OR category ILIKE ?", searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&claims).Error
	return claims, total, err
}

func (r *reimbursementRepository) CountClaims(ctx context.Context) (int64, error) {
	var total int64
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.ReimbursementClaim{})).Count(&total).Error
	return total, err
}
