package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// BankAccount represents a company bank account used for treasury and reconciliation
type BankAccount struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this bank account to a branch/subsidiary Tenant within
	// the Company's database (nil for companies with only their default Tenant).
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	BankName          string     `gorm:"type:varchar(255);not null" json:"bankName"`
	AccountNumber     string     `gorm:"type:varchar(50);not null;index" json:"accountNumber"`
	AccountHolder     string     `gorm:"type:varchar(255)" json:"accountHolder"`
	Currency          string     `gorm:"type:varchar(10);default:'IDR'" json:"currency"`
	CurrentBalance    float64    `gorm:"type:decimal(15,2);default:0" json:"currentBalance"`
	Branch            string     `gorm:"type:varchar(255)" json:"branch"`
	LinkedGLAccountID *uuid.UUID `gorm:"type:uuid;index" json:"linkedGlAccountId,omitempty"`
	Status            string     `gorm:"type:varchar(20);default:'active'" json:"status"`
}

func (BankAccount) TableName() string {
	return "banking_bank_accounts"
}

// BankStatementLine is a single line from an imported bank statement
type BankStatementLine struct {
	types.BaseEntity
	BankAccountID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"bankAccountId"`
	TransactionDate      string     `gorm:"type:varchar(50);not null" json:"transactionDate"`
	Description          string     `gorm:"type:varchar(255)" json:"description"`
	Amount               float64    `gorm:"type:decimal(15,2);not null" json:"amount"` // positive = credit/inflow, negative = debit/outflow
	IsReconciled         bool       `gorm:"default:false" json:"isReconciled"`
	ReconciledAt         *string    `gorm:"type:varchar(50)" json:"reconciledAt,omitempty"`
	MatchedJournalLineID *uuid.UUID `gorm:"type:uuid;index" json:"matchedJournalLineId,omitempty"`
}

func (BankStatementLine) TableName() string {
	return "banking_bank_statement_lines"
}

type BankingRepository interface {
	// Bank accounts
	CreateBankAccount(ctx context.Context, a *BankAccount) error
	GetBankAccountByID(ctx context.Context, id uuid.UUID) (*BankAccount, error)
	ListBankAccounts(ctx context.Context, query types.PaginationQuery) ([]BankAccount, int64, error)
	UpdateBankAccount(ctx context.Context, a *BankAccount) error
	CountBankAccounts(ctx context.Context) (int64, error)

	// Statement lines
	CreateBankStatementLine(ctx context.Context, l *BankStatementLine) error
	CreateBankStatementLines(ctx context.Context, lines []BankStatementLine) error
	GetBankStatementLineByID(ctx context.Context, id uuid.UUID) (*BankStatementLine, error)
	ListBankStatementLines(ctx context.Context, bankAccountID uuid.UUID, query types.PaginationQuery) ([]BankStatementLine, int64, error)
	ListUnreconciledStatementLines(ctx context.Context, bankAccountID uuid.UUID) ([]BankStatementLine, error)
	UpdateBankStatementLine(ctx context.Context, l *BankStatementLine) error
	CountUnreconciledStatementLines(ctx context.Context, bankAccountID uuid.UUID) (int64, error)

	// Journal lines (loose read-only reference into finance module, no FK enforcement)
	FindCandidateJournalLines(ctx context.Context, glAccountID uuid.UUID) ([]JournalLineCandidate, error)
	GetJournalLineByID(ctx context.Context, id uuid.UUID) (*JournalLineCandidate, error)
}

// JournalLineCandidate is a minimal read projection of finance.JournalLine used for matching.
// It is populated via a loose cross-module query (no FK), matching the finance/procurement convention.
type JournalLineCandidate struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	Date      string
	Debit     float64
	Credit    float64
}
