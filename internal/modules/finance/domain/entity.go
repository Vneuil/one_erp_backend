package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Account represents a Chart of Accounts (COA) node
type Account struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this account to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant.
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Code     string     `gorm:"type:varchar(50);not null;index" json:"code"`
	Name     string     `gorm:"type:varchar(255);not null" json:"name"`
	Type     string     `gorm:"type:varchar(50);not null" json:"type"`
	ParentID *uuid.UUID `gorm:"type:uuid;index" json:"parentId,omitempty"`
	IsActive bool       `gorm:"default:true" json:"isActive"`
	// Category sub-classifies revenue/expense accounts for the statutory
	// expense reports: marketing, admin_general or non_operating. Empty falls
	// back to the default mapping in application.ClassifyAccount.
	Category string `gorm:"type:varchar(30)" json:"category,omitempty"`
}

func (Account) TableName() string {
	return "finance_accounts"
}

// JournalEntry is the double-entry journal voucher header
type JournalEntry struct {
	types.BaseEntity
	CompanyID   *uuid.UUID    `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID    *uuid.UUID    `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	EntryNumber string        `gorm:"type:varchar(50);not null;index" json:"entryNumber"`
	Date        string        `gorm:"type:varchar(50);not null" json:"date"`
	Memo        string        `gorm:"type:text" json:"memo"`
	SourceDoc   string        `gorm:"type:varchar(100)" json:"sourceDoc"`
	Status      string        `gorm:"type:varchar(50);default:'draft'" json:"status"`
	Lines       []JournalLine `gorm:"foreignKey:JournalEntryID" json:"lines,omitempty"`
}

func (JournalEntry) TableName() string {
	return "finance_journal_entries"
}

// JournalLine is a single debit/credit line of a journal entry
type JournalLine struct {
	types.BaseEntity
	JournalEntryID uuid.UUID `gorm:"type:uuid;not null;index" json:"journalEntryId"`
	AccountID      uuid.UUID `gorm:"type:uuid;not null;index" json:"accountId"`
	Debit          float64   `gorm:"type:decimal(15,2);default:0" json:"debit"`
	Credit         float64   `gorm:"type:decimal(15,2);default:0" json:"credit"`
	Description    string    `gorm:"type:varchar(255)" json:"description"`
}

func (JournalLine) TableName() string {
	return "finance_journal_lines"
}

// Payable is a vendor bill (Accounts Payable)
type Payable struct {
	types.BaseEntity
	CompanyID    *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	SupplierID   *uuid.UUID `gorm:"type:uuid;index" json:"supplierId,omitempty"`
	VendorName   string     `gorm:"type:varchar(255);not null" json:"vendorName"`
	InvoiceNo    string     `gorm:"type:varchar(50);not null;index" json:"invoiceNo"`
	IssueDate    string     `gorm:"type:varchar(50)" json:"issueDate"`
	DueDate      string     `gorm:"type:varchar(50)" json:"dueDate"`
	TotalInvoice float64    `gorm:"type:decimal(15,2);not null" json:"totalInvoice"`
	PaidAmount   float64    `gorm:"type:decimal(15,2);default:0" json:"paidAmount"`
	PaymentTerms string     `gorm:"type:varchar(50)" json:"paymentTerms"`
	Status       string     `gorm:"type:varchar(50);default:'pending'" json:"status"`
	// SourceDoc links a bill created automatically from a procurement purchase
	// invoice; empty for manually entered bills. Linked bills are paid via the source invoice.
	SourceDoc string `gorm:"type:varchar(100);index" json:"sourceDoc,omitempty"`
}

func (Payable) TableName() string {
	return "finance_payables"
}

// Receivable is a customer invoice (Accounts Receivable)
type Receivable struct {
	types.BaseEntity
	CompanyID    *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	CustomerID   *uuid.UUID `gorm:"type:uuid;index" json:"customerId,omitempty"`
	CustomerName string     `gorm:"type:varchar(255);not null" json:"customerName"`
	InvoiceNo    string     `gorm:"type:varchar(50);not null;index" json:"invoiceNo"`
	IssueDate    string     `gorm:"type:varchar(50)" json:"issueDate"`
	DueDate      string     `gorm:"type:varchar(50)" json:"dueDate"`
	TotalInvoice float64    `gorm:"type:decimal(15,2);not null" json:"totalInvoice"`
	PaidAmount   float64    `gorm:"type:decimal(15,2);default:0" json:"paidAmount"`
	Status       string     `gorm:"type:varchar(50);default:'pending'" json:"status"`
	// SourceDoc links a receivable created automatically from a sales invoice;
	// empty for manually entered receivables. Linked receivables are paid via the source invoice.
	SourceDoc string `gorm:"type:varchar(100);index" json:"sourceDoc,omitempty"`
}

func (Receivable) TableName() string {
	return "finance_receivables"
}

// PettyCashFund is an imprest cash fund per branch/warehouse
type PettyCashFund struct {
	types.BaseEntity
	CompanyID      *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	BranchName     string     `gorm:"type:varchar(255);not null" json:"branchName"`
	Custodian      string     `gorm:"type:varchar(150)" json:"custodian"`
	MaxFloat       float64    `gorm:"type:decimal(15,2);not null" json:"maxFloat"`
	CurrentBalance float64    `gorm:"type:decimal(15,2);not null" json:"currentBalance"`
	Status         string     `gorm:"type:varchar(50);default:'Sufficient'" json:"status"`
}

func (PettyCashFund) TableName() string {
	return "finance_petty_cash_funds"
}

// PettyCashTransaction is an in/out movement against a petty cash fund
type PettyCashTransaction struct {
	types.BaseEntity
	FundID         uuid.UUID `gorm:"type:uuid;not null;index" json:"fundId"`
	Type           string    `gorm:"type:varchar(20);not null" json:"type"` // in | out
	Amount         float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
	Description    string    `gorm:"type:varchar(255)" json:"description"`
	Date           string    `gorm:"type:varchar(50)" json:"date"`
	ApprovalStatus string    `gorm:"type:varchar(50);default:'approved'" json:"approvalStatus"`
	CreatedByEmail string    `gorm:"type:varchar(255);index" json:"createdByEmail,omitempty"`
}

func (PettyCashTransaction) TableName() string {
	return "finance_petty_cash_transactions"
}

// Budget is a spending limit per department/account for a period
type Budget struct {
	types.BaseEntity
	CompanyID       *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Department      string     `gorm:"type:varchar(150);not null" json:"department"`
	AccountCategory string     `gorm:"type:varchar(255);not null" json:"accountCategory"`
	AccountID       *uuid.UUID `gorm:"type:uuid;index" json:"accountId,omitempty"`
	Period          string     `gorm:"type:varchar(20);not null" json:"period"`
	AllocatedBudget float64    `gorm:"type:decimal(15,2);not null" json:"allocatedBudget"`
}

func (Budget) TableName() string {
	return "finance_budgets"
}

// CapitalTransaction is an owner putting money into the business (injection) or
// taking it out (drawing). Both post to the general ledger.
type CapitalTransaction struct {
	types.BaseEntity
	CompanyID          *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID           *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Type               string     `gorm:"type:varchar(20);not null;index" json:"type"` // injection | drawing
	OwnerName          string     `gorm:"type:varchar(255);not null" json:"ownerName"`
	Amount             float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	Date               string     `gorm:"type:varchar(10);not null;index" json:"date"`
	Description        string     `gorm:"type:varchar(500)" json:"description"`
	PaymentAccountCode string     `gorm:"type:varchar(50);not null;default:'1000'" json:"paymentAccountCode"`
	Posted             bool       `gorm:"not null;default:false" json:"posted"`
	CreatedByEmail     string     `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
}

func (CapitalTransaction) TableName() string { return "finance_capital_transactions" }

// OtherIncome is income outside sales (interest, rent received, grants, asset
// disposal gains...) recorded directly against cash/bank.
type OtherIncome struct {
	types.BaseEntity
	CompanyID          *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID           *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Category           string     `gorm:"type:varchar(100);not null;index" json:"category"`
	Amount             float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	Date               string     `gorm:"type:varchar(10);not null;index" json:"date"`
	Description        string     `gorm:"type:varchar(500)" json:"description"`
	PaymentAccountCode string     `gorm:"type:varchar(50);not null;default:'1000'" json:"paymentAccountCode"`
	Posted             bool       `gorm:"not null;default:false" json:"posted"`
	CreatedByEmail     string     `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
}

func (OtherIncome) TableName() string { return "finance_other_income" }

// CashVoucher is a cash/bank receipt (Bukti Kas/Bank Masuk) or payment (Bukti
// Kas/Bank Keluar) against one cash or bank account, spread over one or more
// counter accounts. It posts to the general ledger when saved.
type CashVoucher struct {
	types.BaseEntity
	CompanyID       *uuid.UUID        `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID        *uuid.UUID        `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Number          string            `gorm:"type:varchar(50);not null;index" json:"number"`
	Type            string            `gorm:"type:varchar(20);not null;index" json:"type"` // receipt | payment
	Date            string            `gorm:"type:varchar(10);not null;index" json:"date"`
	CashAccountCode string            `gorm:"type:varchar(50);not null" json:"cashAccountCode"`
	Counterparty    string            `gorm:"type:varchar(255)" json:"counterparty"`
	Description     string            `gorm:"type:varchar(500)" json:"description"`
	Total           float64           `gorm:"type:decimal(15,2);not null" json:"total"`
	Posted          bool              `gorm:"not null;default:false" json:"posted"`
	CreatedByEmail  string            `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
	Lines           []CashVoucherLine `gorm:"foreignKey:VoucherID" json:"lines,omitempty"`
}

func (CashVoucher) TableName() string { return "finance_cash_vouchers" }

// CashVoucherLine is one counter-account allocation of a cash voucher.
type CashVoucherLine struct {
	types.BaseEntity
	VoucherID   uuid.UUID `gorm:"type:uuid;not null;index" json:"voucherId"`
	AccountCode string    `gorm:"type:varchar(50);not null" json:"accountCode"`
	Description string    `gorm:"type:varchar(255)" json:"description"`
	Amount      float64   `gorm:"type:decimal(15,2);not null" json:"amount"`
}

func (CashVoucherLine) TableName() string { return "finance_cash_voucher_lines" }

// JournalLineDetail is a posted journal line joined with its entry header,
// used by the general ledger and cash book reports.
type JournalLineDetail struct {
	AccountID   uuid.UUID
	Date        string
	EntryNumber string
	Memo        string
	SourceDoc   string
	Description string
	Debit       float64
	Credit      float64
}

type FinanceRepository interface {
	// Cash vouchers
	CreateCashVoucher(ctx context.Context, v *CashVoucher) error
	UpdateCashVoucher(ctx context.Context, v *CashVoucher) error
	GetCashVoucherByID(ctx context.Context, id uuid.UUID) (*CashVoucher, error)
	ListCashVouchers(ctx context.Context, typ, from, to string) ([]CashVoucher, error)
	CountCashVouchers(ctx context.Context, typ, numberPrefix string) (int64, error)

	// ListPostedLines returns posted journal lines in date order. A nil
	// accountIDs filter means every account.
	ListPostedLines(ctx context.Context, from, to string, accountIDs []uuid.UUID) ([]JournalLineDetail, error)

	// Owner capital and other income
	CreateCapitalTransaction(ctx context.Context, c *CapitalTransaction) error
	UpdateCapitalTransaction(ctx context.Context, c *CapitalTransaction) error
	ListCapitalTransactions(ctx context.Context, typ, period string) ([]CapitalTransaction, error)
	CreateOtherIncome(ctx context.Context, o *OtherIncome) error
	UpdateOtherIncome(ctx context.Context, o *OtherIncome) error
	ListOtherIncome(ctx context.Context, category, period string) ([]OtherIncome, error)

	// Accounts
	CreateAccount(ctx context.Context, a *Account) error
	GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error)
	ListAccounts(ctx context.Context, query types.PaginationQuery) ([]Account, int64, error)
	ListAllAccounts(ctx context.Context) ([]Account, error)
	UpdateAccount(ctx context.Context, a *Account) error
	CountAccounts(ctx context.Context) (int64, error)

	// Journal entries
	CreateJournalEntry(ctx context.Context, je *JournalEntry) error
	GetJournalEntryByID(ctx context.Context, id uuid.UUID) (*JournalEntry, error)
	GetJournalEntryBySourceDoc(ctx context.Context, sourceDoc string) (*JournalEntry, error)
	SumDebitBySourceDocPrefix(ctx context.Context, prefix string) (float64, error)
	ListJournalEntries(ctx context.Context, query types.PaginationQuery) ([]JournalEntry, int64, error)
	ListPostedJournalEntries(ctx context.Context, from, to string) ([]JournalEntry, error)
	UpdateJournalEntryStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateJournalEntry(ctx context.Context, je *JournalEntry) error
	CountJournalEntries(ctx context.Context) (int64, error)

	// Payables
	CreatePayable(ctx context.Context, p *Payable) error
	GetPayableByID(ctx context.Context, id uuid.UUID) (*Payable, error)
	GetPayableBySourceDoc(ctx context.Context, sourceDoc string) (*Payable, error)
	ListPayables(ctx context.Context, query types.PaginationQuery) ([]Payable, int64, error)
	ListAllPayables(ctx context.Context) ([]Payable, error)
	UpdatePayable(ctx context.Context, p *Payable) error
	CountPayables(ctx context.Context) (int64, error)

	// Receivables
	CreateReceivable(ctx context.Context, r *Receivable) error
	GetReceivableByID(ctx context.Context, id uuid.UUID) (*Receivable, error)
	GetReceivableBySourceDoc(ctx context.Context, sourceDoc string) (*Receivable, error)
	ListReceivables(ctx context.Context, query types.PaginationQuery) ([]Receivable, int64, error)
	ListAllReceivables(ctx context.Context) ([]Receivable, error)
	UpdateReceivable(ctx context.Context, r *Receivable) error
	CountReceivables(ctx context.Context) (int64, error)

	// Petty cash
	CreatePettyCashFund(ctx context.Context, f *PettyCashFund) error
	GetPettyCashFundByID(ctx context.Context, id uuid.UUID) (*PettyCashFund, error)
	ListPettyCashFunds(ctx context.Context, query types.PaginationQuery) ([]PettyCashFund, int64, error)
	UpdatePettyCashFund(ctx context.Context, f *PettyCashFund) error
	CountPettyCashFunds(ctx context.Context) (int64, error)
	CreatePettyCashTransaction(ctx context.Context, t *PettyCashTransaction) error
	ListPettyCashTransactions(ctx context.Context, fundID uuid.UUID) ([]PettyCashTransaction, error)
	GetPettyCashTransactionByID(ctx context.Context, id uuid.UUID) (*PettyCashTransaction, error)
	UpdatePettyCashTransaction(ctx context.Context, t *PettyCashTransaction) error

	// Budgets
	CreateBudget(ctx context.Context, b *Budget) error
	GetBudgetByID(ctx context.Context, id uuid.UUID) (*Budget, error)
	ListBudgets(ctx context.Context, query types.PaginationQuery) ([]Budget, int64, error)
	UpdateBudget(ctx context.Context, b *Budget) error
	DeleteBudget(ctx context.Context, id uuid.UUID) (bool, error)
	CountBudgets(ctx context.Context) (int64, error)

	// Reports
	SumPostedByAccount(ctx context.Context, from, to string) ([]JournalActualByAccount, error)
}

// JournalActualByAccount aggregates posted journal debit/credit totals per account
type JournalActualByAccount struct {
	AccountID   uuid.UUID
	TotalDebit  float64
	TotalCredit float64
}
