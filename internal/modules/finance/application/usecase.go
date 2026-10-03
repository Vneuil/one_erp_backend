package application

import (
	"fmt"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"strings"
	"time"

	"context"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type FinanceUseCase interface {
	// Accounts
	CreateAccount(ctx context.Context, dto CreateAccountDTO) (*AccountResponseDTO, error)
	GetAccountByID(ctx context.Context, id uuid.UUID) (*AccountResponseDTO, error)
	ListAccounts(ctx context.Context, query types.PaginationQuery) ([]AccountResponseDTO, types.PaginationMeta, error)
	UpdateAccount(ctx context.Context, id uuid.UUID, dto CreateAccountDTO) (*AccountResponseDTO, error)

	// Journal entries
	CreateJournalEntry(ctx context.Context, dto CreateJournalEntryDTO) (*JournalEntryResponseDTO, error)
	GetJournalEntryByID(ctx context.Context, id uuid.UUID) (*JournalEntryResponseDTO, error)
	ListJournalEntries(ctx context.Context, query types.PaginationQuery) ([]JournalEntryResponseDTO, types.PaginationMeta, error)
	PostJournalEntry(ctx context.Context, id uuid.UUID) (*JournalEntryResponseDTO, error)
	ReverseJournalEntry(ctx context.Context, id uuid.UUID) (*JournalEntryResponseDTO, error)

	// Payables
	CreatePayable(ctx context.Context, dto CreatePayableDTO) (*PayableResponseDTO, error)
	GetPayableByID(ctx context.Context, id uuid.UUID) (*PayableResponseDTO, error)
	ListPayables(ctx context.Context, query types.PaginationQuery) ([]PayableResponseDTO, types.PaginationMeta, error)
	RecordPayablePayment(ctx context.Context, id uuid.UUID, dto RecordPaymentDTO) (*PayableResponseDTO, error)
	APAgingReport(ctx context.Context) (*AgingReportDTO, error)

	// Receivables
	CreateReceivable(ctx context.Context, dto CreateReceivableDTO) (*ReceivableResponseDTO, error)
	GetReceivableByID(ctx context.Context, id uuid.UUID) (*ReceivableResponseDTO, error)
	ListReceivables(ctx context.Context, query types.PaginationQuery) ([]ReceivableResponseDTO, types.PaginationMeta, error)
	RecordReceivablePayment(ctx context.Context, id uuid.UUID, dto RecordPaymentDTO) (*ReceivableResponseDTO, error)
	ARAgingReport(ctx context.Context) (*AgingReportDTO, error)

	// Petty cash
	CreatePettyCashFund(ctx context.Context, dto CreatePettyCashFundDTO) (*PettyCashFundResponseDTO, error)
	ListPettyCashFunds(ctx context.Context, query types.PaginationQuery) ([]PettyCashFundResponseDTO, types.PaginationMeta, error)
	CreatePettyCashTransaction(ctx context.Context, fundID uuid.UUID, dto PettyCashTransactionDTO) (*PettyCashFundResponseDTO, error)
	ListPettyCashTransactions(ctx context.Context, fundID uuid.UUID) ([]domain.PettyCashTransaction, error)
	ApprovePettyCashTransaction(ctx context.Context, fundID, txID uuid.UUID) (*PettyCashFundResponseDTO, error)
	RejectPettyCashTransaction(ctx context.Context, fundID, txID uuid.UUID) (*PettyCashFundResponseDTO, error)

	// Budgets
	CreateBudget(ctx context.Context, dto CreateBudgetDTO) (*BudgetResponseDTO, error)
	ListBudgets(ctx context.Context, query types.PaginationQuery) ([]BudgetResponseDTO, types.PaginationMeta, error)
	UpdateBudget(ctx context.Context, id uuid.UUID, dto UpdateBudgetDTO) (*BudgetResponseDTO, error)
	DeleteBudget(ctx context.Context, id uuid.UUID) error

	// Reports
	TrialBalance(ctx context.Context, asOf string) (*TrialBalanceDTO, error)
	Insights(ctx context.Context, from string) ([]Insight, error)
	ProfitAndLoss(ctx context.Context, from, to string) (*ProfitLossDTO, error)
	BalanceSheet(ctx context.Context, asOf string) (*BalanceSheetDTO, error)
	CashFlow(ctx context.Context, from, to string) (*CashFlowDTO, error)

	RecordCapital(ctx context.Context, typ string, in CapitalInput) (*domain.CapitalTransaction, error)
	ListCapital(ctx context.Context, typ, period string) ([]domain.CapitalTransaction, error)
	RecordOtherIncome(ctx context.Context, in OtherIncomeInput) (*domain.OtherIncome, error)
	ListOtherIncome(ctx context.Context, category, period string) ([]domain.OtherIncome, error)

	SeedInitialData(ctx context.Context) error
}

type financeUseCase struct {
	repo domain.FinanceRepository
}

func NewFinanceUseCase(repo domain.FinanceRepository) FinanceUseCase {
	return &financeUseCase{repo: repo}
}

func (uc *financeUseCase) accountBalance(ctx context.Context, accountID uuid.UUID, accountType string) (float64, error) {
	sums, err := uc.repo.SumPostedByAccount(ctx, "", "")
	if err != nil {
		return 0, err
	}
	var debit, credit float64
	for _, s := range sums {
		if s.AccountID == accountID {
			debit = s.TotalDebit
			credit = s.TotalCredit
			break
		}
	}
	switch accountType {
	case "asset", "expense":
		return debit - credit, nil
	default:
		return credit - debit, nil
	}
}

// Accounts

func (uc *financeUseCase) CreateAccount(ctx context.Context, dto CreateAccountDTO) (*AccountResponseDTO, error) {
	if dto.Code == "" || dto.Name == "" {
		return nil, apperrors.NewBadRequest("Account code and name are required")
	}
	validTypes := map[string]bool{"asset": true, "liability": true, "equity": true, "revenue": true, "expense": true}
	if !validTypes[dto.Type] {
		return nil, apperrors.NewBadRequest("Account type must be one of asset, liability, equity, revenue, expense")
	}

	isActive := true
	if dto.IsActive != nil {
		isActive = *dto.IsActive
	}

	a := &domain.Account{
		Code:     dto.Code,
		Name:     dto.Name,
		Type:     dto.Type,
		ParentID: dto.ParentID,
		IsActive: isActive,
	}

	if err := uc.repo.CreateAccount(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create account")
	}
	return ToAccountResponse(a, 0), nil
}

func (uc *financeUseCase) GetAccountByID(ctx context.Context, id uuid.UUID) (*AccountResponseDTO, error) {
	a, err := uc.repo.GetAccountByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get account")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Account not found")
	}
	balance, err := uc.accountBalance(ctx, a.ID, a.Type)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute account balance")
	}
	return ToAccountResponse(a, balance), nil
}

func (uc *financeUseCase) ListAccounts(ctx context.Context, query types.PaginationQuery) ([]AccountResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	accounts, total, err := uc.repo.ListAccounts(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list accounts")
	}

	sums, err := uc.repo.SumPostedByAccount(ctx, "", "")
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to compute account balances")
	}
	sumByID := make(map[uuid.UUID]domain.JournalActualByAccount, len(sums))
	for _, s := range sums {
		sumByID[s.AccountID] = s
	}

	result := make([]AccountResponseDTO, len(accounts))
	for i, a := range accounts {
		var balance float64
		if s, ok := sumByID[a.ID]; ok {
			if a.Type == "asset" || a.Type == "expense" {
				balance = s.TotalDebit - s.TotalCredit
			} else {
				balance = s.TotalCredit - s.TotalDebit
			}
		}
		result[i] = *ToAccountResponse(&a, balance)
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

func (uc *financeUseCase) UpdateAccount(ctx context.Context, id uuid.UUID, dto CreateAccountDTO) (*AccountResponseDTO, error) {
	a, err := uc.repo.GetAccountByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get account")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Account not found")
	}
	if dto.Name != "" {
		a.Name = dto.Name
	}
	if dto.Type != "" {
		a.Type = dto.Type
	}
	if dto.ParentID != nil {
		a.ParentID = dto.ParentID
	}
	if dto.IsActive != nil {
		a.IsActive = *dto.IsActive
	}
	if err := uc.repo.UpdateAccount(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update account")
	}
	balance, err := uc.accountBalance(ctx, a.ID, a.Type)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute account balance")
	}
	return ToAccountResponse(a, balance), nil
}

// Journal entries

func (uc *financeUseCase) CreateJournalEntry(ctx context.Context, dto CreateJournalEntryDTO) (*JournalEntryResponseDTO, error) {
	if len(dto.Lines) < 2 {
		return nil, apperrors.NewBadRequest("A journal entry requires at least two lines")
	}

	var totalDebit, totalCredit float64
	lines := make([]domain.JournalLine, len(dto.Lines))
	for i, l := range dto.Lines {
		if l.AccountID == uuid.Nil {
			return nil, apperrors.NewBadRequest("Each journal line requires an accountId")
		}
		if l.Debit < 0 || l.Credit < 0 {
			return nil, apperrors.NewBadRequest("Debit and credit amounts cannot be negative")
		}
		if l.Debit > 0 && l.Credit > 0 {
			return nil, apperrors.NewBadRequest("A journal line cannot have both debit and credit amounts")
		}
		totalDebit += l.Debit
		totalCredit += l.Credit
		lines[i] = domain.JournalLine{
			AccountID:   l.AccountID,
			Debit:       l.Debit,
			Credit:      l.Credit,
			Description: l.Description,
		}
	}

	if totalDebit != totalCredit {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Total debit (%.2f) must equal total credit (%.2f)", totalDebit, totalCredit))
	}

	entryNumber := dto.EntryNumber
	if entryNumber == "" {
		entryNumber = fmt.Sprintf("JE-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)
	}
	date := dto.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	je := &domain.JournalEntry{
		EntryNumber: entryNumber,
		Date:        date,
		Memo:        dto.Memo,
		SourceDoc:   dto.SourceDoc,
		Status:      "draft",
		Lines:       lines,
	}

	if err := uc.repo.CreateJournalEntry(ctx, je); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create journal entry")
	}
	return ToJournalEntryResponse(je), nil
}

func (uc *financeUseCase) GetJournalEntryByID(ctx context.Context, id uuid.UUID) (*JournalEntryResponseDTO, error) {
	je, err := uc.repo.GetJournalEntryByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get journal entry")
	}
	if je == nil {
		return nil, apperrors.NewNotFound("Journal entry not found")
	}
	return ToJournalEntryResponse(je), nil
}

func (uc *financeUseCase) ListJournalEntries(ctx context.Context, query types.PaginationQuery) ([]JournalEntryResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	entries, total, err := uc.repo.ListJournalEntries(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list journal entries")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToJournalEntryResponseList(entries), meta, nil
}

func (uc *financeUseCase) PostJournalEntry(ctx context.Context, id uuid.UUID) (*JournalEntryResponseDTO, error) {
	je, err := uc.repo.GetJournalEntryByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get journal entry")
	}
	if je == nil {
		return nil, apperrors.NewNotFound("Journal entry not found")
	}
	if je.Status == "posted" {
		return nil, apperrors.NewConflict("Journal entry is already posted")
	}

	var totalDebit, totalCredit float64
	for _, l := range je.Lines {
		totalDebit += l.Debit
		totalCredit += l.Credit
	}
	if totalDebit != totalCredit {
		return nil, apperrors.NewBadRequest("Cannot post an unbalanced journal entry")
	}

	if err := uc.repo.UpdateJournalEntryStatus(ctx, id, "posted"); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to post journal entry")
	}
	je.Status = "posted"
	return ToJournalEntryResponse(je), nil
}

// ReverseJournalEntry creates a new balancing journal entry that reverses a posted one.
// Posted entries are immutable; reversal is the only supported correction mechanism.
func (uc *financeUseCase) ReverseJournalEntry(ctx context.Context, id uuid.UUID) (*JournalEntryResponseDTO, error) {
	original, err := uc.repo.GetJournalEntryByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get journal entry")
	}
	if original == nil {
		return nil, apperrors.NewNotFound("Journal entry not found")
	}
	if original.Status != "posted" {
		return nil, apperrors.NewBadRequest("Only posted journal entries can be reversed")
	}

	lines := make([]domain.JournalLine, len(original.Lines))
	for i, l := range original.Lines {
		lines[i] = domain.JournalLine{
			AccountID:   l.AccountID,
			Debit:       l.Credit,
			Credit:      l.Debit,
			Description: fmt.Sprintf("Reversal of %s: %s", original.EntryNumber, l.Description),
		}
	}

	reversal := &domain.JournalEntry{
		EntryNumber: fmt.Sprintf("%s-REV", original.EntryNumber),
		Date:        time.Now().Format("2006-01-02"),
		Memo:        fmt.Sprintf("Reversal of %s", original.EntryNumber),
		SourceDoc:   original.EntryNumber,
		Status:      "posted",
		Lines:       lines,
	}

	if err := uc.repo.CreateJournalEntry(ctx, reversal); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create reversing journal entry")
	}
	return ToJournalEntryResponse(reversal), nil
}

// Payables

func (uc *financeUseCase) CreatePayable(ctx context.Context, dto CreatePayableDTO) (*PayableResponseDTO, error) {
	if dto.VendorName == "" || dto.TotalInvoice <= 0 {
		return nil, apperrors.NewBadRequest("Vendor name and a positive total invoice amount are required")
	}
	invoiceNo := dto.InvoiceNo
	if invoiceNo == "" {
		invoiceNo = fmt.Sprintf("BILL-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)
	}
	p := &domain.Payable{
		SupplierID:   dto.SupplierID,
		VendorName:   dto.VendorName,
		InvoiceNo:    invoiceNo,
		IssueDate:    dto.IssueDate,
		DueDate:      dto.DueDate,
		TotalInvoice: dto.TotalInvoice,
		PaymentTerms: dto.PaymentTerms,
		Status:       "pending",
	}
	if err := uc.repo.CreatePayable(ctx, p); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create payable")
	}
	return ToPayableResponse(p), nil
}

func (uc *financeUseCase) GetPayableByID(ctx context.Context, id uuid.UUID) (*PayableResponseDTO, error) {
	p, err := uc.repo.GetPayableByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get payable")
	}
	if p == nil {
		return nil, apperrors.NewNotFound("Payable not found")
	}
	return ToPayableResponse(p), nil
}

func (uc *financeUseCase) ListPayables(ctx context.Context, query types.PaginationQuery) ([]PayableResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListPayables(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list payables")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToPayableResponseList(items), meta, nil
}

func (uc *financeUseCase) RecordPayablePayment(ctx context.Context, id uuid.UUID, dto RecordPaymentDTO) (*PayableResponseDTO, error) {
	if dto.Amount <= 0 {
		return nil, apperrors.NewBadRequest("Payment amount must be positive")
	}
	p, err := uc.repo.GetPayableByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get payable")
	}
	if p == nil {
		return nil, apperrors.NewNotFound("Payable not found")
	}
	if p.SourceDoc != "" {
		return nil, apperrors.NewConflict("This payable comes from a purchase invoice; record the payment on that invoice instead")
	}
	if dto.Amount > p.TotalInvoice-p.PaidAmount {
		return nil, apperrors.NewBadRequest("Payment amount exceeds outstanding balance")
	}
	p.PaidAmount += dto.Amount
	p.Status = paymentStatus(p.TotalInvoice, p.PaidAmount, p.DueDate)
	if err := uc.repo.UpdatePayable(ctx, p); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record payment")
	}
	return ToPayableResponse(p), nil
}

func (uc *financeUseCase) APAgingReport(ctx context.Context) (*AgingReportDTO, error) {
	items, err := uc.repo.ListAllPayables(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to build AP aging report")
	}
	return buildAgingReport(items, func(p domain.Payable) (string, float64, string) {
		return p.InvoiceNo, p.TotalInvoice - p.PaidAmount, p.DueDate
	}), nil
}

// Receivables

func (uc *financeUseCase) CreateReceivable(ctx context.Context, dto CreateReceivableDTO) (*ReceivableResponseDTO, error) {
	if dto.CustomerName == "" || dto.TotalInvoice <= 0 {
		return nil, apperrors.NewBadRequest("Customer name and a positive total invoice amount are required")
	}
	invoiceNo := dto.InvoiceNo
	if invoiceNo == "" {
		invoiceNo = fmt.Sprintf("INV-%s-%04d", time.Now().Format("200601"), time.Now().Nanosecond()%10000)
	}
	r := &domain.Receivable{
		CustomerID:   dto.CustomerID,
		CustomerName: dto.CustomerName,
		InvoiceNo:    invoiceNo,
		IssueDate:    dto.IssueDate,
		DueDate:      dto.DueDate,
		TotalInvoice: dto.TotalInvoice,
		Status:       "pending",
	}
	if err := uc.repo.CreateReceivable(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create receivable")
	}
	return ToReceivableResponse(r, time.Now()), nil
}

func (uc *financeUseCase) GetReceivableByID(ctx context.Context, id uuid.UUID) (*ReceivableResponseDTO, error) {
	r, err := uc.repo.GetReceivableByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get receivable")
	}
	if r == nil {
		return nil, apperrors.NewNotFound("Receivable not found")
	}
	return ToReceivableResponse(r, time.Now()), nil
}

func (uc *financeUseCase) ListReceivables(ctx context.Context, query types.PaginationQuery) ([]ReceivableResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListReceivables(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list receivables")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToReceivableResponseList(items, time.Now()), meta, nil
}

func (uc *financeUseCase) RecordReceivablePayment(ctx context.Context, id uuid.UUID, dto RecordPaymentDTO) (*ReceivableResponseDTO, error) {
	if dto.Amount <= 0 {
		return nil, apperrors.NewBadRequest("Payment amount must be positive")
	}
	r, err := uc.repo.GetReceivableByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get receivable")
	}
	if r == nil {
		return nil, apperrors.NewNotFound("Receivable not found")
	}
	if r.SourceDoc != "" {
		return nil, apperrors.NewConflict("This receivable comes from a sales invoice; record the payment on that invoice instead")
	}
	if dto.Amount > r.TotalInvoice-r.PaidAmount {
		return nil, apperrors.NewBadRequest("Payment amount exceeds outstanding balance")
	}
	r.PaidAmount += dto.Amount
	r.Status = paymentStatus(r.TotalInvoice, r.PaidAmount, r.DueDate)
	if err := uc.repo.UpdateReceivable(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record payment")
	}
	return ToReceivableResponse(r, time.Now()), nil
}

func (uc *financeUseCase) ARAgingReport(ctx context.Context) (*AgingReportDTO, error) {
	items, err := uc.repo.ListAllReceivables(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to build AR aging report")
	}
	return buildAgingReport(items, func(r domain.Receivable) (string, float64, string) {
		return r.InvoiceNo, r.TotalInvoice - r.PaidAmount, r.DueDate
	}), nil
}

// paymentStatus derives unpaid/partial/paid/overdue from amounts and due date
func paymentStatus(total, paid float64, dueDate string) string {
	if paid >= total {
		return "paid"
	}
	if paid > 0 {
		if due, err := time.Parse("2006-01-02", dueDate); err == nil && time.Now().After(due) {
			return "overdue"
		}
		return "partial"
	}
	if due, err := time.Parse("2006-01-02", dueDate); err == nil && time.Now().After(due) {
		return "overdue"
	}
	return "pending"
}

func buildAgingReport[T any](items []T, extract func(T) (string, float64, string)) *AgingReportDTO {
	buckets := map[string]float64{
		"Current (0-30 days)":   0,
		"Past Due (31-60 days)": 0,
		"Past Due (61-90 days)": 0,
		"Past Due (90+ days)":   0,
	}
	var total float64
	now := time.Now()
	for _, item := range items {
		_, outstanding, dueDate := extract(item)
		if outstanding <= 0 {
			continue
		}
		total += outstanding
		due, err := time.Parse("2006-01-02", dueDate)
		days := 0
		if err == nil {
			days = int(now.Sub(due).Hours() / 24)
		}
		switch {
		case days <= 30:
			buckets["Current (0-30 days)"] += outstanding
		case days <= 60:
			buckets["Past Due (31-60 days)"] += outstanding
		case days <= 90:
			buckets["Past Due (61-90 days)"] += outstanding
		default:
			buckets["Past Due (90+ days)"] += outstanding
		}
	}

	order := []string{"Current (0-30 days)", "Past Due (31-60 days)", "Past Due (61-90 days)", "Past Due (90+ days)"}
	result := make([]AgingBucketDTO, len(order))
	for i, b := range order {
		result[i] = AgingBucketDTO{Bucket: b, Amount: buckets[b]}
	}
	return &AgingReportDTO{AsOf: now.Format("2006-01-02"), Buckets: result, Total: total}
}

// Petty cash

func (uc *financeUseCase) CreatePettyCashFund(ctx context.Context, dto CreatePettyCashFundDTO) (*PettyCashFundResponseDTO, error) {
	if dto.BranchName == "" || dto.MaxFloat <= 0 {
		return nil, apperrors.NewBadRequest("Branch name and a positive max float are required")
	}
	f := &domain.PettyCashFund{
		BranchName:     dto.BranchName,
		Custodian:      dto.Custodian,
		MaxFloat:       dto.MaxFloat,
		CurrentBalance: dto.MaxFloat,
		Status:         "Sufficient",
	}
	if err := uc.repo.CreatePettyCashFund(ctx, f); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create petty cash fund")
	}
	return ToPettyCashFundResponse(f, 0, ""), nil
}

func (uc *financeUseCase) ListPettyCashFunds(ctx context.Context, query types.PaginationQuery) ([]PettyCashFundResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	funds, total, err := uc.repo.ListPettyCashFunds(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list petty cash funds")
	}

	result := make([]PettyCashFundResponseDTO, len(funds))
	for i, f := range funds {
		txs, err := uc.repo.ListPettyCashTransactions(ctx, f.ID)
		if err != nil {
			return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list petty cash transactions")
		}
		spent, lastReplenished := summarizePettyCash(txs)
		result[i] = *ToPettyCashFundResponse(&f, spent, lastReplenished)
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

// summarizePettyCash only considers approved transactions - pending or
// rejected ones have not affected the fund balance yet (see
// CreatePettyCashTransaction/ApprovePettyCashTransaction) and should not be
// counted as spend or a replenishment.
func summarizePettyCash(txs []domain.PettyCashTransaction) (spentThisMonth float64, lastReplenished string) {
	now := time.Now()
	for _, t := range txs {
		if t.ApprovalStatus != "approved" {
			continue
		}
		txDate, err := time.Parse("2006-01-02", t.Date)
		if t.Type == "out" && err == nil && txDate.Year() == now.Year() && txDate.Month() == now.Month() {
			spentThisMonth += t.Amount
		}
		if t.Type == "in" && (lastReplenished == "" || t.Date > lastReplenished) {
			lastReplenished = t.Date
		}
	}
	return spentThisMonth, lastReplenished
}

// CreatePettyCashTransaction records a petty cash movement as pending -
// mirroring the Purchase Request approve/reject workflow (see
// modules/procurement/application/usecase.go), it does not touch the fund's
// balance until an authorized user explicitly approves it via
// ApprovePettyCashTransaction.
func (uc *financeUseCase) CreatePettyCashTransaction(ctx context.Context, fundID uuid.UUID, dto PettyCashTransactionDTO) (*PettyCashFundResponseDTO, error) {
	if dto.Type != "in" && dto.Type != "out" {
		return nil, apperrors.NewBadRequest("Transaction type must be 'in' or 'out'")
	}
	if dto.Amount <= 0 {
		return nil, apperrors.NewBadRequest("Transaction amount must be positive")
	}

	f, err := uc.repo.GetPettyCashFundByID(ctx, fundID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get petty cash fund")
	}
	if f == nil {
		return nil, apperrors.NewNotFound("Petty cash fund not found")
	}

	if dto.Type == "out" && dto.Amount > f.CurrentBalance {
		return nil, apperrors.NewBadRequest("Transaction amount exceeds current fund balance")
	}

	t := &domain.PettyCashTransaction{
		FundID:         fundID,
		Type:           dto.Type,
		Amount:         dto.Amount,
		Description:    dto.Description,
		Date:           time.Now().Format("2006-01-02"),
		ApprovalStatus: "pending",
		CreatedByEmail: actor.EmailFrom(ctx),
	}
	if err := uc.repo.CreatePettyCashTransaction(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record petty cash transaction")
	}

	txs, err := uc.repo.ListPettyCashTransactions(ctx, fundID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list petty cash transactions")
	}
	spent, lastReplenished := summarizePettyCash(txs)
	return ToPettyCashFundResponse(f, spent, lastReplenished), nil
}

// transitionPettyCashTransaction loads the fund and its pending transaction,
// applies `apply` (which mutates the fund's balance/status for an approval,
// or does nothing for a rejection), persists both, and returns the refreshed
// fund summary.
func (uc *financeUseCase) transitionPettyCashTransaction(ctx context.Context, fundID, txID uuid.UUID, newStatus string, apply func(f *domain.PettyCashFund, t *domain.PettyCashTransaction)) (*PettyCashFundResponseDTO, error) {
	f, err := uc.repo.GetPettyCashFundByID(ctx, fundID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get petty cash fund")
	}
	if f == nil {
		return nil, apperrors.NewNotFound("Petty cash fund not found")
	}

	t, err := uc.repo.GetPettyCashTransactionByID(ctx, txID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get petty cash transaction")
	}
	if t == nil || t.FundID != fundID {
		return nil, apperrors.NewNotFound("Petty cash transaction not found")
	}
	if t.ApprovalStatus != "pending" {
		return nil, apperrors.NewConflict("Only pending petty cash transactions can be " + newStatus)
	}

	if newStatus == "approved" {
		if err := sod.ForbidSelfApproval(ctx, t.CreatedByEmail); err != nil {
			return nil, err
		}
	}

	apply(f, t)
	t.ApprovalStatus = newStatus

	if err := uc.repo.UpdatePettyCashTransaction(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update petty cash transaction")
	}
	if err := uc.repo.UpdatePettyCashFund(ctx, f); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update petty cash fund balance")
	}

	txs, err := uc.repo.ListPettyCashTransactions(ctx, fundID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list petty cash transactions")
	}
	spent, lastReplenished := summarizePettyCash(txs)
	return ToPettyCashFundResponse(f, spent, lastReplenished), nil
}

// ApprovePettyCashTransaction approves a pending transaction and applies its
// effect on the fund balance for the first time.
func (uc *financeUseCase) ApprovePettyCashTransaction(ctx context.Context, fundID, txID uuid.UUID) (*PettyCashFundResponseDTO, error) {
	return uc.transitionPettyCashTransaction(ctx, fundID, txID, "approved", func(f *domain.PettyCashFund, t *domain.PettyCashTransaction) {
		if t.Type == "in" {
			f.CurrentBalance += t.Amount
		} else {
			f.CurrentBalance -= t.Amount
		}
		f.Status = "Sufficient"
		if f.MaxFloat > 0 && f.CurrentBalance < f.MaxFloat*0.2 {
			f.Status = "Low Balance"
		}
	})
}

// RejectPettyCashTransaction rejects a pending transaction. The fund balance
// was never touched at creation time, so rejecting it is a no-op on the
// fund itself.
func (uc *financeUseCase) RejectPettyCashTransaction(ctx context.Context, fundID, txID uuid.UUID) (*PettyCashFundResponseDTO, error) {
	return uc.transitionPettyCashTransaction(ctx, fundID, txID, "rejected", func(f *domain.PettyCashFund, t *domain.PettyCashTransaction) {})
}

func (uc *financeUseCase) ListPettyCashTransactions(ctx context.Context, fundID uuid.UUID) ([]domain.PettyCashTransaction, error) {
	f, err := uc.repo.GetPettyCashFundByID(ctx, fundID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get petty cash fund")
	}
	if f == nil {
		return nil, apperrors.NewNotFound("Petty cash fund not found")
	}
	txs, err := uc.repo.ListPettyCashTransactions(ctx, fundID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list petty cash transactions")
	}
	return txs, nil
}

// Budgets

func (uc *financeUseCase) CreateBudget(ctx context.Context, dto CreateBudgetDTO) (*BudgetResponseDTO, error) {
	if dto.Department == "" || dto.AccountCategory == "" || dto.Period == "" || dto.AllocatedBudget <= 0 {
		return nil, apperrors.NewBadRequest("Department, account category, period and a positive allocated budget are required")
	}
	if !ValidBudgetPeriod(dto.Period) {
		return nil, apperrors.NewBadRequest("Period must be YYYY-MM")
	}
	b := &domain.Budget{
		Department:      dto.Department,
		AccountCategory: dto.AccountCategory,
		AccountID:       dto.AccountID,
		Period:          dto.Period,
		AllocatedBudget: dto.AllocatedBudget,
	}
	if dup, err := uc.budgetExists(ctx, *b, uuid.Nil); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing budgets")
	} else if dup {
		return nil, apperrors.NewConflict("A budget for this department, account and month already exists")
	}
	if err := uc.repo.CreateBudget(ctx, b); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create budget")
	}
	actual, err := uc.budgetActual(ctx, b)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute budget actuals")
	}
	return ToBudgetResponse(b, actual), nil
}

func (uc *financeUseCase) budgetActual(ctx context.Context, b *domain.Budget) (float64, error) {
	if b.AccountID == nil {
		return 0, nil
	}
	from := b.Period + "-01"
	periodStart, err := time.Parse("2006-01-02", from)
	if err != nil {
		return 0, nil
	}
	to := periodStart.AddDate(0, 1, -1).Format("2006-01-02")

	sums, err := uc.repo.SumPostedByAccount(ctx, from, to)
	if err != nil {
		return 0, err
	}
	for _, s := range sums {
		if s.AccountID == *b.AccountID {
			return s.TotalDebit - s.TotalCredit, nil
		}
	}
	return 0, nil
}

func (uc *financeUseCase) ListBudgets(ctx context.Context, query types.PaginationQuery) ([]BudgetResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	budgets, total, err := uc.repo.ListBudgets(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list budgets")
	}

	result := make([]BudgetResponseDTO, len(budgets))
	for i, b := range budgets {
		actual, err := uc.budgetActual(ctx, &b)
		if err != nil {
			return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to compute budget actuals")
		}
		result[i] = *ToBudgetResponse(&b, actual)
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return result, meta, nil
}

// Reports

func (uc *financeUseCase) TrialBalance(ctx context.Context, asOf string) (*TrialBalanceDTO, error) {
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list accounts")
	}
	sums, err := uc.repo.SumPostedByAccount(ctx, "", asOf)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute posted totals")
	}
	sumByID := make(map[uuid.UUID]domain.JournalActualByAccount, len(sums))
	for _, s := range sums {
		sumByID[s.AccountID] = s
	}

	var lines []TrialBalanceLineDTO
	var totalDebit, totalCredit float64
	for _, a := range accounts {
		s, ok := sumByID[a.ID]
		if !ok {
			continue
		}
		var debit, credit float64
		net := s.TotalDebit - s.TotalCredit
		if a.Type == "asset" || a.Type == "expense" {
			if net >= 0 {
				debit = net
			} else {
				credit = -net
			}
		} else {
			if net <= 0 {
				credit = -net
			} else {
				debit = net
			}
		}
		if debit == 0 && credit == 0 {
			continue
		}
		totalDebit += debit
		totalCredit += credit
		lines = append(lines, TrialBalanceLineDTO{
			AccountID: a.ID, AccountCode: a.Code, AccountName: a.Name, Debit: debit, Credit: credit,
		})
	}

	if asOf == "" {
		asOf = time.Now().Format("2006-01-02")
	}
	return &TrialBalanceDTO{AsOf: asOf, Lines: lines, TotalDebit: totalDebit, TotalCredit: totalCredit}, nil
}

func (uc *financeUseCase) ProfitAndLoss(ctx context.Context, from, to string) (*ProfitLossDTO, error) {
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list accounts")
	}
	sums, err := uc.repo.SumPostedByAccount(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute posted totals")
	}
	sumByID := make(map[uuid.UUID]domain.JournalActualByAccount, len(sums))
	for _, s := range sums {
		sumByID[s.AccountID] = s
	}

	var revenue, expense float64
	for _, a := range accounts {
		s, ok := sumByID[a.ID]
		if !ok {
			continue
		}
		switch a.Type {
		case "revenue":
			revenue += s.TotalCredit - s.TotalDebit
		case "expense":
			expense += s.TotalDebit - s.TotalCredit
		}
	}

	return &ProfitLossDTO{
		From: from, To: to,
		Revenue: revenue, OperatingExpense: expense,
		GrossProfit: revenue, NetProfit: revenue - expense,
	}, nil
}

func (uc *financeUseCase) BalanceSheet(ctx context.Context, asOf string) (*BalanceSheetDTO, error) {
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list accounts")
	}
	sums, err := uc.repo.SumPostedByAccount(ctx, "", asOf)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute posted totals")
	}
	sumByID := make(map[uuid.UUID]domain.JournalActualByAccount, len(sums))
	for _, s := range sums {
		sumByID[s.AccountID] = s
	}

	var assets, liabilities, equity, earnings float64
	for _, a := range accounts {
		s, ok := sumByID[a.ID]
		if !ok {
			continue
		}
		switch a.Type {
		case "asset":
			assets += s.TotalDebit - s.TotalCredit
		case "liability":
			liabilities += s.TotalCredit - s.TotalDebit
		case "equity":
			equity += s.TotalCredit - s.TotalDebit
		// Revenue and expense accounts are not on the balance sheet themselves;
		// their cumulative net (retained earnings) is what keeps it in balance.
		case "revenue":
			earnings += s.TotalCredit - s.TotalDebit
		case "expense":
			earnings -= s.TotalDebit - s.TotalCredit
		}
	}

	if asOf == "" {
		asOf = time.Now().Format("2006-01-02")
	}
	return &BalanceSheetDTO{
		AsOf: asOf, Assets: assets, Liabilities: liabilities, Equity: equity, RetainedEarnings: earnings,
		TotalLiabEq: liabilities + equity + earnings,
	}, nil
}

// isCashAccount reports whether an account holds cash or bank balances.
func isCashAccount(a domain.Account) bool {
	return a.Type == "asset" && strings.HasPrefix(a.Code, "10")
}

func (uc *financeUseCase) CashFlow(ctx context.Context, from, to string) (*CashFlowDTO, error) {
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list accounts")
	}

	// Only cash and bank accounts (codes 1000-1099) count as cash. Treating every
	// asset as cash would report receivables and inventory movements as inflows.
	var cashAccountIDs []uuid.UUID
	for _, a := range accounts {
		if isCashAccount(a) {
			cashAccountIDs = append(cashAccountIDs, a.ID)
		}
	}

	sums, err := uc.repo.SumPostedByAccount(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to compute posted totals")
	}
	cashSet := make(map[uuid.UUID]bool, len(cashAccountIDs))
	for _, id := range cashAccountIDs {
		cashSet[id] = true
	}

	var cashIn, cashOut float64
	for _, s := range sums {
		if !cashSet[s.AccountID] {
			continue
		}
		cashIn += s.TotalDebit
		cashOut += s.TotalCredit
	}

	return &CashFlowDTO{
		From: from, To: to, CashInflow: cashIn, CashOutflow: cashOut, NetCashFlow: cashIn - cashOut,
	}, nil
}

// SeedInitialData populates a minimal chart of accounts on first boot
func (uc *financeUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountAccounts(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	accounts := []CreateAccountDTO{
		{Code: "1000", Name: "Cash and Cash Equivalents", Type: "asset"},
		{Code: "1100", Name: "Accounts Receivable", Type: "asset"},
		{Code: "1200", Name: "Inventory", Type: "asset"},
		{Code: "1300", Name: "Supplier Advances", Type: "asset"},
		{Code: "2000", Name: "Accounts Payable", Type: "liability"},
		{Code: "3000", Name: "Owner's Equity", Type: "equity"},
		{Code: "4000", Name: "Sales Revenue", Type: "revenue"},
		{Code: "5000", Name: "Operating Expenses", Type: "expense"},
	}
	for _, a := range accounts {
		_, _ = uc.CreateAccount(ctx, a)
	}
	return nil
}
