package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/pos/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type posRepository struct {
	db *gorm.DB
}

func NewPOSRepository(db *gorm.DB) domain.POSRepository {
	return &posRepository{db: db}
}

func (r *posRepository) Create(ctx context.Context, tx *domain.POSTransaction) error {
	tenantctx.SetTenantID(ctx, &tx.TenantID)
	return r.db.WithContext(ctx).Create(tx).Error
}

func (r *posRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.POSTransaction, error) {
	var tx domain.POSTransaction
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *posRepository) List(ctx context.Context, query types.PaginationQuery) ([]domain.POSTransaction, int64, error) {
	var transactions []domain.POSTransaction
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.POSTransaction{}))

	if query.Search != "" {
		searchPattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("order_no ILIKE ? OR outlet ILIKE ? OR customer ILIKE ?", searchPattern, searchPattern, searchPattern)
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
		Find(&transactions).Error

	return transactions, total, err
}

func (r *posRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.POSTransaction{}).Count(&total).Error
	return total, err
}

// Lines, refunds and settings

func (r *posRepository) CreateLines(ctx context.Context, lines []domain.POSTransactionLine) error {
	if len(lines) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&lines).Error
}

func (r *posRepository) ListLines(ctx context.Context, txID uuid.UUID) ([]domain.POSTransactionLine, error) {
	var out []domain.POSTransactionLine
	return out, r.db.WithContext(ctx).Where("transaction_id = ?", txID).Order("created_at asc").Find(&out).Error
}

func (r *posRepository) UpdateLine(ctx context.Context, l *domain.POSTransactionLine) error {
	return r.db.WithContext(ctx).Save(l).Error
}

func (r *posRepository) Update(ctx context.Context, tx *domain.POSTransaction) error {
	return r.db.WithContext(ctx).Omit("Lines").Save(tx).Error
}

func (r *posRepository) CreateRefund(ctx context.Context, rf *domain.POSRefund) error {
	tenantctx.SetTenantID(ctx, &rf.TenantID)
	return r.db.WithContext(ctx).Create(rf).Error
}

func (r *posRepository) ListRefunds(ctx context.Context, txID uuid.UUID) ([]domain.POSRefund, error) {
	var out []domain.POSRefund
	return out, r.db.WithContext(ctx).Where("transaction_id = ?", txID).Order("created_at asc").Find(&out).Error
}

// dayRange turns Jakarta-local dates into UTC instants [start, end).
func dayRange(from, to string) (time.Time, time.Time, error) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	start, err := time.ParseInLocation("2006-01-02", from, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := time.ParseInLocation("2006-01-02", to, loc)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, end.AddDate(0, 0, 1), nil
}

func (r *posRepository) ListInRange(ctx context.Context, from, to, outlet string) ([]domain.POSTransaction, error) {
	start, end, err := dayRange(from, to)
	if err != nil {
		return nil, err
	}
	var out []domain.POSTransaction
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.POSTransaction{})).Where("created_at >= ? AND created_at < ?", start, end)
	if outlet != "" {
		db = db.Where("outlet = ?", outlet)
	}
	return out, db.Preload("Lines").Order("created_at asc").Find(&out).Error
}

func (r *posRepository) ListRefundsInRange(ctx context.Context, from, to, outlet string) ([]domain.POSRefund, error) {
	start, end, err := dayRange(from, to)
	if err != nil {
		return nil, err
	}
	var out []domain.POSRefund
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.POSRefund{})).Where("pos_refunds.created_at >= ? AND pos_refunds.created_at < ?", start, end)
	if outlet != "" {
		db = db.Where("pos_refunds.transaction_id IN (SELECT id FROM pos_transactions WHERE outlet = ?)", outlet)
	}
	return out, db.Order("pos_refunds.created_at asc").Find(&out).Error
}

func (r *posRepository) GetSettings(ctx context.Context) (*domain.POSSettings, error) {
	var found []domain.POSSettings
	if err := r.db.WithContext(ctx).Order("created_at asc").Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		s := domain.DefaultSettings()
		return &s, nil
	}
	return &found[0], nil
}

func (r *posRepository) SaveSettings(ctx context.Context, s *domain.POSSettings) error {
	return r.db.WithContext(ctx).Save(s).Error
}
