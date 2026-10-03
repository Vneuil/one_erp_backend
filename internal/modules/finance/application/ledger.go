package application

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/google/uuid"
)

// Default chart-of-accounts codes used by automatic postings from other
// modules. They match the accounts created by SeedInitialData.
const (
	AccountCash                 = "1000"
	AccountReceivble            = "1100"
	AccountPayable              = "2000"
	AccountRevenue              = "4000"
	AccountExpense              = "5000"
	AccountInventory            = "1200"
	AccountSupplierAdvance      = "1300"
	AccountSalaryExpense        = "5100"
	AccountSalaryPayable        = "2100"
	AccountTaxPayable           = "2200"
	AccountCoopPayable          = "2300"
	AccountReimbursementPayable = "2400"
	AccountOwnerCapital         = "3000"
	AccountOwnerDrawings        = "3100"
	AccountSalesDiscount        = "4100"
	AccountAdditionalCharges    = "4200"
	AccountOtherIncome          = "4300"
	AccountRounding             = "5900"
	AccountSalesTaxPayable      = "2500"
	AccountCOGS                 = "5200"
	AccountDeliveryExpense      = "5300"
	AccountEmployeeAdvance      = "1400"
)

// defaultAccounts are created on demand when an automatic posting needs a
// standard account the tenant's chart of accounts does not have yet.
var defaultAccounts = map[string]struct{ Name, Type string }{
	AccountCash:                 {"Cash and Cash Equivalents", "asset"},
	AccountReceivble:            {"Accounts Receivable", "asset"},
	AccountInventory:            {"Inventory", "asset"},
	AccountSupplierAdvance:      {"Supplier Advances", "asset"},
	AccountPayable:              {"Accounts Payable", "liability"},
	AccountRevenue:              {"Sales Revenue", "revenue"},
	AccountExpense:              {"Operating Expenses", "expense"},
	AccountSalaryExpense:        {"Salary Expense", "expense"},
	AccountSalaryPayable:        {"Salaries Payable", "liability"},
	AccountTaxPayable:           {"Income Tax Payable", "liability"},
	AccountCoopPayable:          {"Cooperative Payable", "liability"},
	AccountReimbursementPayable: {"Reimbursements Payable", "liability"},
	AccountOwnerCapital:         {"Owner's Equity", "equity"},
	AccountOwnerDrawings:        {"Owner Drawings", "equity"},
	AccountSalesDiscount:        {"Sales Discounts", "revenue"},
	AccountAdditionalCharges:    {"Additional Charges Income", "revenue"},
	AccountOtherIncome:          {"Other Income", "revenue"},
	AccountRounding:             {"Rounding Adjustment", "expense"},
	AccountSalesTaxPayable:      {"Sales Tax Payable", "liability"},
	AccountCOGS:                 {"Cost of Goods Sold", "expense"},
	AccountDeliveryExpense:      {"Delivery Expense", "expense"},
	AccountEmployeeAdvance:      {"Employee Advances", "asset"},
}

// LedgerLine is one side of an automatic posting, addressed by account code so
// callers outside finance never need account UUIDs.
type LedgerLine struct {
	AccountCode string
	Debit       float64
	Credit      float64
	Description string
}

// LedgerPoster lets other modules (sales, procurement, payroll, ...) record a
// balanced, already-posted journal entry in the general ledger.
type LedgerPoster interface {
	// PostEntry is idempotent on sourceDoc: posting the same sourceDoc twice
	// is a no-op, so retries never double-count.
	PostEntry(ctx context.Context, sourceDoc, memo string, lines []LedgerLine) error
	// PostEntryOn is PostEntry with an explicit entry date (YYYY-MM-DD),
	// used when back-filling historical documents. Empty means today.
	PostEntryOn(ctx context.Context, date, sourceDoc, memo string, lines []LedgerLine) error
	// PostedDebitTotal sums the debit side of every entry whose source doc
	// starts with prefix, so a back-fill can post only what is still missing.
	PostedDebitTotal(ctx context.Context, sourceDocPrefix string) (float64, error)
	// UpsertReceivable / UpsertPayable mirror a sales / purchase invoice into
	// the AR / AP sub-ledger (creating it, or refreshing amounts and status),
	// so the aging reports include invoices raised in other modules.
	UpsertReceivable(ctx context.Context, d SubledgerDoc) error
	UpsertPayable(ctx context.Context, d SubledgerDoc) error
}

// SubledgerDoc describes an invoice to mirror into the AR/AP sub-ledger.
// SourceDoc is the stable key of the originating document.
type SubledgerDoc struct {
	SourceDoc string
	PartyID   *uuid.UUID
	PartyName string
	DocNo     string
	IssueDate string
	DueDate   string
	Total     float64
	Paid      float64
}

// LedgerEntry is a document-derived posting: builders in each module return
// one so live posting and back-fill share the same account mapping.
type LedgerEntry struct {
	SourceDoc string
	Memo      string
	Lines     []LedgerLine
}

type ledgerPoster struct {
	repo domain.FinanceRepository
}

func NewLedgerPoster(repo domain.FinanceRepository) LedgerPoster {
	return &ledgerPoster{repo: repo}
}

func (p *ledgerPoster) PostedDebitTotal(ctx context.Context, prefix string) (float64, error) {
	return p.repo.SumDebitBySourceDocPrefix(ctx, prefix)
}

func (p *ledgerPoster) PostEntry(ctx context.Context, sourceDoc, memo string, lines []LedgerLine) error {
	return p.PostEntryOn(ctx, "", sourceDoc, memo, lines)
}

func (p *ledgerPoster) PostEntryOn(ctx context.Context, date, sourceDoc, memo string, lines []LedgerLine) error {
	if sourceDoc == "" {
		return fmt.Errorf("ledger: sourceDoc is required")
	}
	// Zero-amount lines carry no information; drop them so callers can pass
	// optional components (e.g. tax, cooperative deduction) unconditionally.
	nonZero := make([]LedgerLine, 0, len(lines))
	for _, l := range lines {
		if l.Debit != 0 || l.Credit != 0 {
			nonZero = append(nonZero, l)
		}
	}
	lines = nonZero
	if len(lines) < 2 {
		return fmt.Errorf("ledger: at least two non-zero lines are required")
	}

	existing, err := p.repo.GetJournalEntryBySourceDoc(ctx, sourceDoc)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	accounts, err := p.repo.ListAllAccounts(ctx)
	if err != nil {
		return err
	}
	byCode := make(map[string]domain.Account, len(accounts))
	for _, a := range accounts {
		byCode[a.Code] = a
	}

	var debit, credit float64
	jl := make([]domain.JournalLine, 0, len(lines))
	for _, l := range lines {
		acc, ok := byCode[l.AccountCode]
		if !ok {
			def, known := defaultAccounts[l.AccountCode]
			if !known {
				return fmt.Errorf("ledger: account %s not found in chart of accounts", l.AccountCode)
			}
			created := domain.Account{Code: l.AccountCode, Name: def.Name, Type: def.Type, IsActive: true}
			if err := p.repo.CreateAccount(ctx, &created); err != nil {
				return err
			}
			acc = created
			byCode[l.AccountCode] = acc
		}
		debit += l.Debit
		credit += l.Credit
		jl = append(jl, domain.JournalLine{AccountID: acc.ID, Debit: l.Debit, Credit: l.Credit, Description: l.Description})
	}
	if math.Abs(debit-credit) > 0.005 {
		return fmt.Errorf("ledger: unbalanced entry (debit %.2f, credit %.2f)", debit, credit)
	}

	now := time.Now()
	if date == "" {
		date = now.Format("2006-01-02")
	}
	je := &domain.JournalEntry{
		EntryNumber: fmt.Sprintf("AUTO-%s-%06d", now.Format("200601"), now.Nanosecond()/1000%1000000),
		Date:        date,
		Memo:        memo,
		SourceDoc:   sourceDoc,
		Status:      "posted",
		Lines:       jl,
	}
	return p.repo.CreateJournalEntry(ctx, je)
}

func (p *ledgerPoster) UpsertReceivable(ctx context.Context, d SubledgerDoc) error {
	if d.SourceDoc == "" {
		return fmt.Errorf("subledger: sourceDoc is required")
	}
	rec, err := p.repo.GetReceivableBySourceDoc(ctx, d.SourceDoc)
	if err != nil {
		return err
	}
	if rec == nil {
		rec = &domain.Receivable{SourceDoc: d.SourceDoc}
	}
	rec.CustomerID, rec.CustomerName, rec.InvoiceNo = d.PartyID, d.PartyName, d.DocNo
	rec.IssueDate, rec.DueDate = d.IssueDate, d.DueDate
	rec.TotalInvoice, rec.PaidAmount = d.Total, d.Paid
	rec.Status = paymentStatus(d.Total, d.Paid, d.DueDate)
	if rec.ID == uuid.Nil {
		return p.repo.CreateReceivable(ctx, rec)
	}
	return p.repo.UpdateReceivable(ctx, rec)
}

func (p *ledgerPoster) UpsertPayable(ctx context.Context, d SubledgerDoc) error {
	if d.SourceDoc == "" {
		return fmt.Errorf("subledger: sourceDoc is required")
	}
	pay, err := p.repo.GetPayableBySourceDoc(ctx, d.SourceDoc)
	if err != nil {
		return err
	}
	if pay == nil {
		pay = &domain.Payable{SourceDoc: d.SourceDoc}
	}
	pay.SupplierID, pay.VendorName, pay.InvoiceNo = d.PartyID, d.PartyName, d.DocNo
	pay.IssueDate, pay.DueDate = d.IssueDate, d.DueDate
	pay.TotalInvoice, pay.PaidAmount = d.Total, d.Paid
	pay.Status = paymentStatus(d.Total, d.Paid, d.DueDate)
	if pay.ID == uuid.Nil {
		return p.repo.CreatePayable(ctx, pay)
	}
	return p.repo.UpdatePayable(ctx, pay)
}
