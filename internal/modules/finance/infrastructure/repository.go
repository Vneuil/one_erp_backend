package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type financeRepository struct {
	db *gorm.DB
}

func NewFinanceRepository(db *gorm.DB) domain.FinanceRepository {
	return &financeRepository{db: db}
}

// Accounts

func (r *financeRepository) CreateAccount(ctx context.Context, a *domain.Account) error {
	tenantctx.SetTenantID(ctx, &a.TenantID)
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *financeRepository) GetAccountByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	var a domain.Account
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *financeRepository) ListAccounts(ctx context.Context, query types.PaginationQuery) ([]domain.Account, int64, error) {
	var accounts []domain.Account
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Account{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("code ILIKE ? OR name ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("code asc").Offset(offset).Limit(query.PerPage).Find(&accounts).Error
	return accounts, total, err
}

func (r *financeRepository) ListAllAccounts(ctx context.Context) ([]domain.Account, error) {
	var accounts []domain.Account
	err := r.db.WithContext(ctx).Order("code asc").Find(&accounts).Error
	return accounts, err
}

func (r *financeRepository) UpdateAccount(ctx context.Context, a *domain.Account) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *financeRepository) CountAccounts(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Account{}).Count(&total).Error
	return total, err
}

// Journal entries

func (r *financeRepository) CreateJournalEntry(ctx context.Context, je *domain.JournalEntry) error {
	tenantctx.SetTenantID(ctx, &je.TenantID)
	return r.db.WithContext(ctx).Create(je).Error
}

func (r *financeRepository) GetJournalEntryByID(ctx context.Context, id uuid.UUID) (*domain.JournalEntry, error) {
	var je domain.JournalEntry
	err := r.db.WithContext(ctx).Preload("Lines").Where("id = ?", id).First(&je).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &je, nil
}

func (r *financeRepository) ListJournalEntries(ctx context.Context, query types.PaginationQuery) ([]domain.JournalEntry, int64, error) {
	var entries []domain.JournalEntry
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.JournalEntry{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("entry_number ILIKE ? OR memo ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Preload("Lines").Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&entries).Error
	return entries, total, err
}

func (r *financeRepository) ListPostedJournalEntries(ctx context.Context, from, to string) ([]domain.JournalEntry, error) {
	var entries []domain.JournalEntry
	db := r.db.WithContext(ctx).Preload("Lines").Where("status = ?", "posted")
	if from != "" {
		db = db.Where("date >= ?", from)
	}
	if to != "" {
		db = db.Where("date <= ?", to)
	}
	err := db.Order("date asc").Find(&entries).Error
	return entries, err
}

func (r *financeRepository) GetJournalEntryBySourceDoc(ctx context.Context, sourceDoc string) (*domain.JournalEntry, error) {
	// Find+Limit instead of First: a missing entry is the normal case here and
	// must not be logged as a "record not found" error.
	var found []domain.JournalEntry
	if err := r.db.WithContext(ctx).Where("source_doc = ?", sourceDoc).Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, nil
	}
	return &found[0], nil
}

func (r *financeRepository) SumDebitBySourceDocPrefix(ctx context.Context, prefix string) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).
		Table("finance_journal_lines AS l").
		Select("COALESCE(SUM(l.debit),0)").
		Joins("JOIN finance_journal_entries AS e ON e.id = l.journal_entry_id").
		Where("e.source_doc LIKE ?", prefix+"%").
		Where("e.deleted_at IS NULL").
		Scan(&total).Error
	return total, err
}

func (r *financeRepository) UpdateJournalEntryStatus(ctx context.Context, id uuid.UUID, status string) error {
	return r.db.WithContext(ctx).Model(&domain.JournalEntry{}).Where("id = ?", id).Update("status", status).Error
}

func (r *financeRepository) UpdateJournalEntry(ctx context.Context, je *domain.JournalEntry) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(je).Error
}

func (r *financeRepository) CountJournalEntries(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.JournalEntry{}).Count(&total).Error
	return total, err
}

// Payables

func (r *financeRepository) CreatePayable(ctx context.Context, p *domain.Payable) error {
	tenantctx.SetTenantID(ctx, &p.TenantID)
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *financeRepository) GetPayableByID(ctx context.Context, id uuid.UUID) (*domain.Payable, error) {
	var p domain.Payable
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *financeRepository) GetPayableBySourceDoc(ctx context.Context, sourceDoc string) (*domain.Payable, error) {
	var found []domain.Payable
	if err := r.db.WithContext(ctx).Where("source_doc = ?", sourceDoc).Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, nil
	}
	return &found[0], nil
}

func (r *financeRepository) ListPayables(ctx context.Context, query types.PaginationQuery) ([]domain.Payable, int64, error) {
	var payables []domain.Payable
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Payable{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("vendor_name ILIKE ? OR invoice_no ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("due_date asc").Offset(offset).Limit(query.PerPage).Find(&payables).Error
	return payables, total, err
}

func (r *financeRepository) ListAllPayables(ctx context.Context) ([]domain.Payable, error) {
	var payables []domain.Payable
	err := r.db.WithContext(ctx).Order("due_date asc").Find(&payables).Error
	return payables, err
}

func (r *financeRepository) UpdatePayable(ctx context.Context, p *domain.Payable) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *financeRepository) CountPayables(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Payable{}).Count(&total).Error
	return total, err
}

// Receivables

func (r *financeRepository) CreateReceivable(ctx context.Context, rec *domain.Receivable) error {
	tenantctx.SetTenantID(ctx, &rec.TenantID)
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *financeRepository) GetReceivableByID(ctx context.Context, id uuid.UUID) (*domain.Receivable, error) {
	var rec domain.Receivable
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (r *financeRepository) GetReceivableBySourceDoc(ctx context.Context, sourceDoc string) (*domain.Receivable, error) {
	var found []domain.Receivable
	if err := r.db.WithContext(ctx).Where("source_doc = ?", sourceDoc).Limit(1).Find(&found).Error; err != nil {
		return nil, err
	}
	if len(found) == 0 {
		return nil, nil
	}
	return &found[0], nil
}

func (r *financeRepository) ListReceivables(ctx context.Context, query types.PaginationQuery) ([]domain.Receivable, int64, error) {
	var receivables []domain.Receivable
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Receivable{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("customer_name ILIKE ? OR invoice_no ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("due_date asc").Offset(offset).Limit(query.PerPage).Find(&receivables).Error
	return receivables, total, err
}

func (r *financeRepository) ListAllReceivables(ctx context.Context) ([]domain.Receivable, error) {
	var receivables []domain.Receivable
	err := r.db.WithContext(ctx).Order("due_date asc").Find(&receivables).Error
	return receivables, err
}

func (r *financeRepository) UpdateReceivable(ctx context.Context, rec *domain.Receivable) error {
	return r.db.WithContext(ctx).Save(rec).Error
}

func (r *financeRepository) CountReceivables(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Receivable{}).Count(&total).Error
	return total, err
}

// Petty cash

func (r *financeRepository) CreatePettyCashFund(ctx context.Context, f *domain.PettyCashFund) error {
	tenantctx.SetTenantID(ctx, &f.TenantID)
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *financeRepository) GetPettyCashFundByID(ctx context.Context, id uuid.UUID) (*domain.PettyCashFund, error) {
	var f domain.PettyCashFund
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *financeRepository) ListPettyCashFunds(ctx context.Context, query types.PaginationQuery) ([]domain.PettyCashFund, int64, error) {
	var funds []domain.PettyCashFund
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.PettyCashFund{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("branch_name ILIKE ? OR custodian ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&funds).Error
	return funds, total, err
}

func (r *financeRepository) UpdatePettyCashFund(ctx context.Context, f *domain.PettyCashFund) error {
	return r.db.WithContext(ctx).Save(f).Error
}

func (r *financeRepository) CountPettyCashFunds(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.PettyCashFund{}).Count(&total).Error
	return total, err
}

func (r *financeRepository) CreatePettyCashTransaction(ctx context.Context, t *domain.PettyCashTransaction) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *financeRepository) ListPettyCashTransactions(ctx context.Context, fundID uuid.UUID) ([]domain.PettyCashTransaction, error) {
	var txs []domain.PettyCashTransaction
	err := r.db.WithContext(ctx).Where("fund_id = ?", fundID).Order("created_at desc").Find(&txs).Error
	return txs, err
}

func (r *financeRepository) GetPettyCashTransactionByID(ctx context.Context, id uuid.UUID) (*domain.PettyCashTransaction, error) {
	var t domain.PettyCashTransaction
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *financeRepository) UpdatePettyCashTransaction(ctx context.Context, t *domain.PettyCashTransaction) error {
	return r.db.WithContext(ctx).Save(t).Error
}

// Budgets

func (r *financeRepository) CreateBudget(ctx context.Context, b *domain.Budget) error {
	tenantctx.SetTenantID(ctx, &b.TenantID)
	return r.db.WithContext(ctx).Create(b).Error
}

func (r *financeRepository) ListBudgets(ctx context.Context, query types.PaginationQuery) ([]domain.Budget, int64, error) {
	var budgets []domain.Budget
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Budget{}))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("department ILIKE ? OR account_category ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&budgets).Error
	return budgets, total, err
}

func (r *financeRepository) GetBudgetByID(ctx context.Context, id uuid.UUID) (*domain.Budget, error) {
	var b domain.Budget
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&b).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &b, nil
}

func (r *financeRepository) UpdateBudget(ctx context.Context, b *domain.Budget) error {
	return r.db.WithContext(ctx).Save(b).Error
}

func (r *financeRepository) DeleteBudget(ctx context.Context, id uuid.UUID) (bool, error) {
	res := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).Delete(&domain.Budget{})
	return res.RowsAffected > 0, res.Error
}

func (r *financeRepository) CountBudgets(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&domain.Budget{}).Count(&total).Error
	return total, err
}

// Reports

func (r *financeRepository) SumPostedByAccount(ctx context.Context, from, to string) ([]domain.JournalActualByAccount, error) {
	var results []domain.JournalActualByAccount

	db := r.db.WithContext(ctx).
		Table("finance_journal_lines AS l").
		Select("l.account_id AS account_id, COALESCE(SUM(l.debit),0) AS total_debit, COALESCE(SUM(l.credit),0) AS total_credit").
		Joins("JOIN finance_journal_entries AS e ON e.id = l.journal_entry_id").
		Where("e.status = ?", "posted").
		Where("e.deleted_at IS NULL").
		Group("l.account_id")

	if from != "" {
		db = db.Where("e.date >= ?", from)
	}
	if to != "" {
		db = db.Where("e.date <= ?", to)
	}

	err := db.Scan(&results).Error
	return results, err
}

// Owner capital and other income

func (r *financeRepository) CreateCapitalTransaction(ctx context.Context, c *domain.CapitalTransaction) error {
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *financeRepository) UpdateCapitalTransaction(ctx context.Context, c *domain.CapitalTransaction) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *financeRepository) ListCapitalTransactions(ctx context.Context, typ, period string) ([]domain.CapitalTransaction, error) {
	var out []domain.CapitalTransaction
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.CapitalTransaction{}))
	if typ != "" {
		db = db.Where("type = ?", typ)
	}
	if period != "" {
		db = db.Where("date LIKE ?", period+"%")
	}
	return out, db.Order("date desc, created_at desc").Find(&out).Error
}

func (r *financeRepository) CreateOtherIncome(ctx context.Context, o *domain.OtherIncome) error {
	tenantctx.SetTenantID(ctx, &o.TenantID)
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *financeRepository) UpdateOtherIncome(ctx context.Context, o *domain.OtherIncome) error {
	return r.db.WithContext(ctx).Save(o).Error
}

func (r *financeRepository) ListOtherIncome(ctx context.Context, category, period string) ([]domain.OtherIncome, error) {
	var out []domain.OtherIncome
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.OtherIncome{}))
	if category != "" {
		db = db.Where("category = ?", category)
	}
	if period != "" {
		db = db.Where("date LIKE ?", period+"%")
	}
	return out, db.Order("date desc, created_at desc").Find(&out).Error
}

// Cash vouchers

func (r *financeRepository) CreateCashVoucher(ctx context.Context, v *domain.CashVoucher) error {
	tenantctx.SetTenantID(ctx, &v.TenantID)
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *financeRepository) UpdateCashVoucher(ctx context.Context, v *domain.CashVoucher) error {
	return r.db.WithContext(ctx).Model(&domain.CashVoucher{}).Where("id = ?", v.ID).
		Updates(map[string]any{"posted": v.Posted}).Error
}

func (r *financeRepository) GetCashVoucherByID(ctx context.Context, id uuid.UUID) (*domain.CashVoucher, error) {
	var v domain.CashVoucher
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Lines").Where("id = ?", id).First(&v).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &v, nil
}

func (r *financeRepository) ListCashVouchers(ctx context.Context, typ, from, to string) ([]domain.CashVoucher, error) {
	var out []domain.CashVoucher
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Preload("Lines")
	if typ != "" {
		db = db.Where("type = ?", typ)
	}
	if from != "" {
		db = db.Where("date >= ?", from)
	}
	if to != "" {
		db = db.Where("date <= ?", to)
	}
	err := db.Order("date desc, created_at desc").Find(&out).Error
	return out, err
}

func (r *financeRepository) CountCashVouchers(ctx context.Context, typ, numberPrefix string) (int64, error) {
	var n int64
	// Unscoped: soft-deleted vouchers still hold their number.
	err := r.db.WithContext(ctx).Unscoped().Model(&domain.CashVoucher{}).
		Where("type = ? AND number LIKE ?", typ, numberPrefix+"%").Count(&n).Error
	return n, err
}

func (r *financeRepository) ListPostedLines(ctx context.Context, from, to string, accountIDs []uuid.UUID) ([]domain.JournalLineDetail, error) {
	var out []domain.JournalLineDetail
	db := r.db.WithContext(ctx).
		Table("finance_journal_lines AS l").
		Select("l.account_id AS account_id, e.date AS date, e.entry_number AS entry_number, e.memo AS memo, e.source_doc AS source_doc, l.description AS description, l.debit AS debit, l.credit AS credit").
		Joins("JOIN finance_journal_entries AS e ON e.id = l.journal_entry_id").
		Where("e.status = ?", "posted").
		Where("e.deleted_at IS NULL")
	if from != "" {
		db = db.Where("e.date >= ?", from)
	}
	if to != "" {
		db = db.Where("e.date <= ?", to)
	}
	if accountIDs != nil {
		db = db.Where("l.account_id IN ?", accountIDs)
	}
	err := db.Order("e.date asc, e.created_at asc, l.created_at asc").Scan(&out).Error
	return out, err
}
