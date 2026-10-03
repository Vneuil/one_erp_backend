package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/loyalty/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type loyaltyRepository struct {
	db *gorm.DB
}

func NewLoyaltyRepository(db *gorm.DB) domain.LoyaltyRepository {
	return &loyaltyRepository{db: db}
}

func (r *loyaltyRepository) GetConfig(ctx context.Context) (*domain.LoyaltyConfig, error) {
	var cfg domain.LoyaltyConfig
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Order("created_at asc").First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *loyaltyRepository) UpsertConfig(ctx context.Context, cfg *domain.LoyaltyConfig) error {
	tenantctx.SetTenantID(ctx, &cfg.TenantID)
	if cfg.ID == uuid.Nil {
		return r.db.WithContext(ctx).Create(cfg).Error
	}
	return r.db.WithContext(ctx).Save(cfg).Error
}

func (r *loyaltyRepository) GetMemberByPhone(ctx context.Context, phone string) (*domain.LoyaltyMember, error) {
	var m domain.LoyaltyMember
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("phone_number = ?", phone).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *loyaltyRepository) GetMemberByID(ctx context.Context, id uuid.UUID) (*domain.LoyaltyMember, error) {
	var m domain.LoyaltyMember
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *loyaltyRepository) CreateMember(ctx context.Context, m *domain.LoyaltyMember) error {
	tenantctx.SetTenantID(ctx, &m.TenantID)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *loyaltyRepository) UpdateMemberBalance(ctx context.Context, id uuid.UUID, newBalance int) error {
	return r.db.WithContext(ctx).Model(&domain.LoyaltyMember{}).
		Where("id = ?", id).
		Update("points_balance", newBalance).Error
}

func (r *loyaltyRepository) ListMembers(ctx context.Context, query types.PaginationQuery) ([]domain.LoyaltyMember, int64, error) {
	var members []domain.LoyaltyMember
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.LoyaltyMember{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("phone_number ILIKE ? OR name ILIKE ? OR member_code ILIKE ?", searchPattern, searchPattern, searchPattern)
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
		Find(&members).Error

	return members, total, err
}

func (r *loyaltyRepository) CountMembersWithCodePrefix(ctx context.Context, prefix string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.LoyaltyMember{}).
		Where("member_code LIKE ?", prefix+"%").
		Count(&count).Error
	return count, err
}

func (r *loyaltyRepository) CreateTransaction(ctx context.Context, tx *domain.LoyaltyTransaction) error {
	tenantctx.SetTenantID(ctx, &tx.TenantID)
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *loyaltyRepository) ListTransactionsByMember(ctx context.Context, memberID uuid.UUID) ([]domain.LoyaltyTransaction, error) {
	var txs []domain.LoyaltyTransaction
	err := r.db.WithContext(ctx).Where("member_id = ?", memberID).Order("created_at desc").Find(&txs).Error
	return txs, err
}
