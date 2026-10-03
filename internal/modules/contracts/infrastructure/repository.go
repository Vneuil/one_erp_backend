package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/contracts/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contractsRepository struct {
	db *gorm.DB
}

func NewContractsRepository(db *gorm.DB) domain.ContractsRepository {
	return &contractsRepository{db: db}
}

func (r *contractsRepository) CreateContract(ctx context.Context, c *domain.Contract) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *contractsRepository) GetContractByID(ctx context.Context, id uuid.UUID) (*domain.Contract, error) {
	var c domain.Contract
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *contractsRepository) ListContracts(ctx context.Context, query types.PaginationQuery) ([]domain.Contract, int64, error) {
	var contracts []domain.Contract
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Contract{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("contract_number ILIKE ? OR title ILIKE ? OR party_name ILIKE ?", pattern, pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&contracts).Error
	return contracts, total, err
}

func (r *contractsRepository) ListAllContracts(ctx context.Context) ([]domain.Contract, error) {
	var contracts []domain.Contract
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Order("created_at desc").Find(&contracts).Error
	return contracts, err
}

func (r *contractsRepository) UpdateContract(ctx context.Context, c *domain.Contract) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *contractsRepository) CountContracts(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Contract{}).Count(&total).Error
	return total, err
}
