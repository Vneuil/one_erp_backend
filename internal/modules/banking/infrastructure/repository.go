package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/banking/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type bankingRepository struct {
	db *gorm.DB
}

func NewBankingRepository(db *gorm.DB) domain.BankingRepository {
	return &bankingRepository{db: db}
}

// Bank accounts

func (r *bankingRepository) CreateBankAccount(ctx context.Context, a *domain.BankAccount) error {
	tenantctx.SetTenantID(ctx, &a.TenantID)
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *bankingRepository) GetBankAccountByID(ctx context.Context, id uuid.UUID) (*domain.BankAccount, error) {
	var a domain.BankAccount
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *bankingRepository) ListBankAccounts(ctx context.Context, query types.PaginationQuery) ([]domain.BankAccount, int64, error) {
	var accounts []domain.BankAccount
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.BankAccount{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("bank_name ILIKE ? OR account_number ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&accounts).Error
	return accounts, total, err
}

func (r *bankingRepository) UpdateBankAccount(ctx context.Context, a *domain.BankAccount) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *bankingRepository) CountBankAccounts(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.BankAccount{}).Count(&total).Error
	return total, err
}

// Statement lines

func (r *bankingRepository) CreateBankStatementLine(ctx context.Context, l *domain.BankStatementLine) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *bankingRepository) CreateBankStatementLines(ctx context.Context, lines []domain.BankStatementLine) error {
	if len(lines) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&lines).Error
}

func (r *bankingRepository) GetBankStatementLineByID(ctx context.Context, id uuid.UUID) (*domain.BankStatementLine, error) {
	var l domain.BankStatementLine
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&l).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &l, nil
}

func (r *bankingRepository) ListBankStatementLines(ctx context.Context, bankAccountID uuid.UUID, query types.PaginationQuery) ([]domain.BankStatementLine, int64, error) {
	var lines []domain.BankStatementLine
	var total int64

	db := r.db.WithContext(ctx).Model(&domain.BankStatementLine{}).Where("bank_account_id = ?", bankAccountID)
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("description ILIKE ?", pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("transaction_date desc").Offset(offset).Limit(query.PerPage).Find(&lines).Error
	return lines, total, err
}

func (r *bankingRepository) ListUnreconciledStatementLines(ctx context.Context, bankAccountID uuid.UUID) ([]domain.BankStatementLine, error) {
	var lines []domain.BankStatementLine
	err := r.db.WithContext(ctx).
		Where("bank_account_id = ? AND is_reconciled = ?", bankAccountID, false).
		Order("transaction_date asc").
		Find(&lines).Error
	return lines, err
}

func (r *bankingRepository) UpdateBankStatementLine(ctx context.Context, l *domain.BankStatementLine) error {
	return r.db.WithContext(ctx).Save(l).Error
}

func (r *bankingRepository) CountUnreconciledStatementLines(ctx context.Context, bankAccountID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.BankStatementLine{}).
		Where("bank_account_id = ? AND is_reconciled = ?", bankAccountID, false).
		Count(&total).Error
	return total, err
}

// Journal lines - loose cross-module read against finance tables (no FK, matches finance/procurement convention)

func (r *bankingRepository) FindCandidateJournalLines(ctx context.Context, glAccountID uuid.UUID) ([]domain.JournalLineCandidate, error) {
	var results []domain.JournalLineCandidate
	err := r.db.WithContext(ctx).
		Table("finance_journal_lines AS l").
		Select("l.id AS id, l.account_id AS account_id, e.date AS date, l.debit AS debit, l.credit AS credit").
		Joins("JOIN finance_journal_entries AS e ON e.id = l.journal_entry_id").
		Where("l.account_id = ?", glAccountID).
		Where("e.status = ?", "posted").
		Where("e.deleted_at IS NULL").
		Where("l.deleted_at IS NULL").
		Where("l.id NOT IN (SELECT matched_journal_line_id FROM banking_bank_statement_lines WHERE matched_journal_line_id IS NOT NULL AND deleted_at IS NULL)").
		Scan(&results).Error
	return results, err
}

func (r *bankingRepository) GetJournalLineByID(ctx context.Context, id uuid.UUID) (*domain.JournalLineCandidate, error) {
	var result domain.JournalLineCandidate
	err := r.db.WithContext(ctx).
		Table("finance_journal_lines AS l").
		Select("l.id AS id, l.account_id AS account_id, e.date AS date, l.debit AS debit, l.credit AS credit").
		Joins("JOIN finance_journal_entries AS e ON e.id = l.journal_entry_id").
		Where("l.id = ?", id).
		Where("l.deleted_at IS NULL").
		Scan(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if result.ID == uuid.Nil {
		return nil, nil
	}
	return &result, nil
}
