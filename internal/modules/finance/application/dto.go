package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

// Accounts

type CreateAccountDTO struct {
	Code     string     `json:"code"`
	Name     string     `json:"name"`
	Type     string     `json:"type"`
	ParentID *uuid.UUID `json:"parentId,omitempty"`
	IsActive *bool      `json:"isActive,omitempty"`
}

type AccountResponseDTO struct {
	ID        uuid.UUID  `json:"id"`
	Code      string     `json:"code"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	ParentID  *uuid.UUID `json:"parentId,omitempty"`
	IsActive  bool       `json:"isActive"`
	Balance   float64    `json:"balance"`
	Currency  string     `json:"currency"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"createdAt"`
}

func ToAccountResponse(a *domain.Account, balance float64) *AccountResponseDTO {
	if a == nil {
		return nil
	}
	status := "inactive"
	if a.IsActive {
		status = "active"
	}
	return &AccountResponseDTO{
		ID:        a.ID,
		Code:      a.Code,
		Name:      a.Name,
		Type:      a.Type,
		ParentID:  a.ParentID,
		IsActive:  a.IsActive,
		Balance:   balance,
		Currency:  "IDR",
		Status:    status,
		CreatedAt: a.CreatedAt,
	}
}

// Journal entries

type JournalLineDTO struct {
	AccountID   uuid.UUID `json:"accountId"`
	Debit       float64   `json:"debit"`
	Credit      float64   `json:"credit"`
	Description string    `json:"description"`
}

type CreateJournalEntryDTO struct {
	EntryNumber string           `json:"entryNumber"`
	Date        string           `json:"date"`
	Memo        string           `json:"memo"`
	SourceDoc   string           `json:"sourceDoc"`
	Lines       []JournalLineDTO `json:"lines"`
}

type JournalLineResponseDTO struct {
	ID          uuid.UUID `json:"id"`
	AccountID   uuid.UUID `json:"accountId"`
	Debit       float64   `json:"debit"`
	Credit      float64   `json:"credit"`
	Description string    `json:"description"`
}

type JournalEntryResponseDTO struct {
	ID          uuid.UUID                `json:"id"`
	EntryNumber string                   `json:"entryNumber"`
	Date        string                   `json:"date"`
	Memo        string                   `json:"memo"`
	SourceDoc   string                   `json:"sourceDoc"`
	Status      string                   `json:"status"`
	TotalDebit  float64                  `json:"totalDebit"`
	TotalCredit float64                  `json:"totalCredit"`
	Lines       []JournalLineResponseDTO `json:"lines,omitempty"`
	CreatedAt   time.Time                `json:"createdAt"`
}

func ToJournalEntryResponse(je *domain.JournalEntry) *JournalEntryResponseDTO {
	if je == nil {
		return nil
	}
	var totalDebit, totalCredit float64
	lines := make([]JournalLineResponseDTO, len(je.Lines))
	for i, l := range je.Lines {
		totalDebit += l.Debit
		totalCredit += l.Credit
		lines[i] = JournalLineResponseDTO{
			ID:          l.ID,
			AccountID:   l.AccountID,
			Debit:       l.Debit,
			Credit:      l.Credit,
			Description: l.Description,
		}
	}
	return &JournalEntryResponseDTO{
		ID:          je.ID,
		EntryNumber: je.EntryNumber,
		Date:        je.Date,
		Memo:        je.Memo,
		SourceDoc:   je.SourceDoc,
		Status:      je.Status,
		TotalDebit:  totalDebit,
		TotalCredit: totalCredit,
		Lines:       lines,
		CreatedAt:   je.CreatedAt,
	}
}

func ToJournalEntryResponseList(entries []domain.JournalEntry) []JournalEntryResponseDTO {
	result := make([]JournalEntryResponseDTO, len(entries))
	for i, e := range entries {
		result[i] = *ToJournalEntryResponse(&e)
	}
	return result
}

// Payables

type CreatePayableDTO struct {
	SupplierID   *uuid.UUID `json:"supplierId,omitempty"`
	VendorName   string     `json:"vendorName"`
	InvoiceNo    string     `json:"invoiceNo"`
	IssueDate    string     `json:"issueDate"`
	DueDate      string     `json:"dueDate"`
	TotalInvoice float64    `json:"totalInvoice"`
	PaymentTerms string     `json:"paymentTerms"`
}

type RecordPaymentDTO struct {
	Amount float64 `json:"amount"`
}

type PayableResponseDTO struct {
	ID           uuid.UUID  `json:"id"`
	SupplierID   *uuid.UUID `json:"supplierId,omitempty"`
	VendorName   string     `json:"vendorName"`
	InvoiceNo    string     `json:"invoiceNo"`
	IssueDate    string     `json:"issueDate"`
	DueDate      string     `json:"dueDate"`
	TotalInvoice float64    `json:"totalInvoice"`
	PaidAmount   float64    `json:"paidAmount"`
	Outstanding  float64    `json:"outstanding"`
	PaymentTerms string     `json:"paymentTerms"`
	Status       string     `json:"status"`
	// SourceDoc is set when the payable was created from a purchase invoice.
	SourceDoc string    `json:"sourceDoc,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func ToPayableResponse(p *domain.Payable) *PayableResponseDTO {
	if p == nil {
		return nil
	}
	return &PayableResponseDTO{
		ID:           p.ID,
		SupplierID:   p.SupplierID,
		VendorName:   p.VendorName,
		InvoiceNo:    p.InvoiceNo,
		IssueDate:    p.IssueDate,
		DueDate:      p.DueDate,
		TotalInvoice: p.TotalInvoice,
		PaidAmount:   p.PaidAmount,
		Outstanding:  p.TotalInvoice - p.PaidAmount,
		PaymentTerms: p.PaymentTerms,
		Status:       p.Status,
		SourceDoc:    p.SourceDoc,
		CreatedAt:    p.CreatedAt,
	}
}

func ToPayableResponseList(payables []domain.Payable) []PayableResponseDTO {
	result := make([]PayableResponseDTO, len(payables))
	for i, p := range payables {
		result[i] = *ToPayableResponse(&p)
	}
	return result
}

// Receivables

type CreateReceivableDTO struct {
	CustomerID   *uuid.UUID `json:"customerId,omitempty"`
	CustomerName string     `json:"customerName"`
	InvoiceNo    string     `json:"invoiceNo"`
	IssueDate    string     `json:"issueDate"`
	DueDate      string     `json:"dueDate"`
	TotalInvoice float64    `json:"totalInvoice"`
}

type ReceivableResponseDTO struct {
	ID           uuid.UUID  `json:"id"`
	CustomerID   *uuid.UUID `json:"customerId,omitempty"`
	CustomerName string     `json:"customerName"`
	InvoiceNo    string     `json:"invoiceNo"`
	IssueDate    string     `json:"issueDate"`
	DueDate      string     `json:"dueDate"`
	TotalInvoice float64    `json:"totalInvoice"`
	PaidAmount   float64    `json:"paidAmount"`
	Outstanding  float64    `json:"outstanding"`
	Aging        string     `json:"aging"`
	Status       string     `json:"status"`
	// SourceDoc is set when the receivable was created from a sales invoice.
	SourceDoc string    `json:"sourceDoc,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func ToReceivableResponse(r *domain.Receivable, asOf time.Time) *ReceivableResponseDTO {
	if r == nil {
		return nil
	}
	return &ReceivableResponseDTO{
		ID:           r.ID,
		CustomerID:   r.CustomerID,
		CustomerName: r.CustomerName,
		InvoiceNo:    r.InvoiceNo,
		IssueDate:    r.IssueDate,
		DueDate:      r.DueDate,
		TotalInvoice: r.TotalInvoice,
		PaidAmount:   r.PaidAmount,
		Outstanding:  r.TotalInvoice - r.PaidAmount,
		Aging:        agingBracket(r.DueDate, asOf),
		Status:       r.Status,
		SourceDoc:    r.SourceDoc,
		CreatedAt:    r.CreatedAt,
	}
}

func ToReceivableResponseList(receivables []domain.Receivable, asOf time.Time) []ReceivableResponseDTO {
	result := make([]ReceivableResponseDTO, len(receivables))
	for i, r := range receivables {
		result[i] = *ToReceivableResponse(&r, asOf)
	}
	return result
}

func agingBracket(dueDate string, asOf time.Time) string {
	due, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return "Unknown"
	}
	days := int(asOf.Sub(due).Hours() / 24)
	switch {
	case days <= 0:
		return "Not Due"
	case days <= 30:
		return "Current (0-30 days)"
	case days <= 60:
		return "Past Due (31-60 days)"
	case days <= 90:
		return "Past Due (61-90 days)"
	default:
		return "Past Due (90+ days)"
	}
}

// Petty cash

type CreatePettyCashFundDTO struct {
	BranchName string  `json:"branchName"`
	Custodian  string  `json:"custodian"`
	MaxFloat   float64 `json:"maxFloat"`
}

type PettyCashTransactionDTO struct {
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type PettyCashFundResponseDTO struct {
	ID                  uuid.UUID `json:"id"`
	BranchName          string    `json:"branchName"`
	Custodian           string    `json:"custodian"`
	MaxFloat            float64   `json:"maxFloat"`
	CurrentBalance      float64   `json:"currentBalance"`
	TotalSpentThisMonth float64   `json:"totalSpentThisMonth"`
	LastReplenished     string    `json:"lastReplenished"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"createdAt"`
}

func ToPettyCashFundResponse(f *domain.PettyCashFund, spentThisMonth float64, lastReplenished string) *PettyCashFundResponseDTO {
	if f == nil {
		return nil
	}
	return &PettyCashFundResponseDTO{
		ID:                  f.ID,
		BranchName:          f.BranchName,
		Custodian:           f.Custodian,
		MaxFloat:            f.MaxFloat,
		CurrentBalance:      f.CurrentBalance,
		TotalSpentThisMonth: spentThisMonth,
		LastReplenished:     lastReplenished,
		Status:              f.Status,
		CreatedAt:           f.CreatedAt,
	}
}

// Budgets

type CreateBudgetDTO struct {
	Department      string     `json:"department"`
	AccountCategory string     `json:"accountCategory"`
	AccountID       *uuid.UUID `json:"accountId,omitempty"`
	Period          string     `json:"period"`
	AllocatedBudget float64    `json:"allocatedBudget"`
}

type BudgetResponseDTO struct {
	ID              uuid.UUID  `json:"id"`
	Department      string     `json:"department"`
	AccountCategory string     `json:"accountCategory"`
	AccountID       *uuid.UUID `json:"accountId,omitempty"`
	Period          string     `json:"period"`
	AllocatedBudget float64    `json:"allocatedBudget"`
	ActualSpent     float64    `json:"actualSpent"`
	Variance        float64    `json:"variance"`
	UtilizationRate float64    `json:"utilizationRate"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
}

func ToBudgetResponse(b *domain.Budget, actualSpent float64) *BudgetResponseDTO {
	if b == nil {
		return nil
	}
	variance := b.AllocatedBudget - actualSpent
	utilization := 0.0
	if b.AllocatedBudget > 0 {
		utilization = actualSpent / b.AllocatedBudget * 100
	}
	status := "Safe"
	if utilization > 100 {
		status = "Exceeded"
	} else if utilization > 85 {
		status = "Warning (Near Limit)"
	}
	return &BudgetResponseDTO{
		ID:              b.ID,
		Department:      b.Department,
		AccountCategory: b.AccountCategory,
		AccountID:       b.AccountID,
		Period:          b.Period,
		AllocatedBudget: b.AllocatedBudget,
		ActualSpent:     actualSpent,
		Variance:        variance,
		UtilizationRate: utilization,
		Status:          status,
		CreatedAt:       b.CreatedAt,
	}
}

// Reports

type TrialBalanceLineDTO struct {
	AccountID   uuid.UUID `json:"accountId"`
	AccountCode string    `json:"accountCode"`
	AccountName string    `json:"accountName"`
	Debit       float64   `json:"debit"`
	Credit      float64   `json:"credit"`
}

type TrialBalanceDTO struct {
	AsOf        string                `json:"asOf"`
	Lines       []TrialBalanceLineDTO `json:"lines"`
	TotalDebit  float64               `json:"totalDebit"`
	TotalCredit float64               `json:"totalCredit"`
}

type ProfitLossDTO struct {
	From             string  `json:"from"`
	To               string  `json:"to"`
	Revenue          float64 `json:"revenue"`
	CostOfGoodsSold  float64 `json:"costOfGoodsSold"`
	GrossProfit      float64 `json:"grossProfit"`
	OperatingExpense float64 `json:"operatingExpense"`
	NetProfit        float64 `json:"netProfit"`
}

type BalanceSheetDTO struct {
	AsOf        string  `json:"asOf"`
	Assets      float64 `json:"assets"`
	Liabilities float64 `json:"liabilities"`
	Equity      float64 `json:"equity"`
	// RetainedEarnings is cumulative revenue minus expenses up to AsOf.
	RetainedEarnings float64 `json:"retainedEarnings"`
	TotalLiabEq      float64 `json:"totalLiabilitiesAndEquity"`
}

type CashFlowDTO struct {
	From        string  `json:"from"`
	To          string  `json:"to"`
	CashInflow  float64 `json:"cashInflow"`
	CashOutflow float64 `json:"cashOutflow"`
	NetCashFlow float64 `json:"netCashFlow"`
}

type AgingBucketDTO struct {
	Bucket string  `json:"bucket"`
	Amount float64 `json:"amount"`
}

type AgingReportDTO struct {
	AsOf    string           `json:"asOf"`
	Buckets []AgingBucketDTO `json:"buckets"`
	Total   float64          `json:"total"`
}
