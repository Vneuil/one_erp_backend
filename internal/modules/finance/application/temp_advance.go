package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

const (
	tempAdvanceIssuePrefix  = "kbs-issue:"
	tempAdvanceSettlePrefix = "kbs-settle:"
	tempAdvanceCancelPrefix = "kbs-cancel:"
)

type TempAdvanceInput struct {
	Date            string     `json:"date"`
	DueDate         string     `json:"dueDate"`
	RecipientName   string     `json:"recipientName"`
	EmployeeID      *uuid.UUID `json:"employeeId"`
	Purpose         string     `json:"purpose"`
	Amount          float64    `json:"amount"`
	CashAccountCode string     `json:"cashAccountCode"`
}

type TempAdvanceSettleInput struct {
	Date  string                 `json:"date"`
	Lines []CashVoucherLineInput `json:"lines"`
}

type OutstandingTempAdvanceDTO struct {
	ID            uuid.UUID `json:"id"`
	Number        string    `json:"number"`
	Date          string    `json:"date"`
	DueDate       string    `json:"dueDate,omitempty"`
	RecipientName string    `json:"recipientName"`
	Purpose       string    `json:"purpose"`
	Amount        float64   `json:"amount"`
	AgeDays       int       `json:"ageDays"`
	Overdue       bool      `json:"overdue"`
}

type OutstandingTempAdvancesDTO struct {
	AsOf         string                      `json:"asOf"`
	Rows         []OutstandingTempAdvanceDTO `json:"rows"`
	Total        float64                     `json:"total"`
	OverdueTotal float64                     `json:"overdueTotal"`
}

// IssueTempAdvance records a KBS and posts Dr Temporary Cash Advances / Cr cash-or-bank.
func (uc *financeUseCase) IssueTempAdvance(ctx context.Context, in TempAdvanceInput) (*domain.TempAdvance, error) {
	name := strings.TrimSpace(in.RecipientName)
	if name == "" {
		return nil, apperrors.NewBadRequest("recipientName is required")
	}
	if in.Amount <= 0 {
		return nil, apperrors.NewBadRequest("amount must be positive")
	}
	date, err := resolveDate(in.Date)
	if err != nil {
		return nil, err
	}
	if in.DueDate != "" {
		if !validDate(in.DueDate) {
			return nil, apperrors.NewBadRequest("dueDate must be YYYY-MM-DD")
		}
		if in.DueDate < date {
			return nil, apperrors.NewBadRequest("dueDate cannot be before the advance date")
		}
	}
	cashCode, err := uc.validCashAccount(ctx, in.CashAccountCode)
	if err != nil {
		return nil, err
	}
	numberPrefix := "KBS-" + strings.ReplaceAll(date[:7], "-", "") + "-"
	n, err := uc.repo.CountTempAdvances(ctx, numberPrefix)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to number the advance")
	}
	a := &domain.TempAdvance{
		Number: fmt.Sprintf("%s%04d", numberPrefix, n+1), Date: date, DueDate: in.DueDate, RecipientName: name,
		EmployeeID: in.EmployeeID, Purpose: strings.TrimSpace(in.Purpose), Amount: round2(in.Amount),
		CashAccountCode: cashCode, Status: "open", CreatedByEmail: actor.EmailFrom(ctx),
	}
	a.ID = uuid.New()
	if err := uc.repo.CreateTempAdvance(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save the advance")
	}
	desc := a.Number + " " + a.RecipientName
	err = NewLedgerPoster(uc.repo).PostEntryOn(ctx, a.Date, tempAdvanceIssuePrefix+a.ID.String(), "Temporary cash advance "+desc, []LedgerLine{
		{AccountCode: AccountTempAdvance, Debit: a.Amount, Description: desc},
		{AccountCode: a.CashAccountCode, Credit: a.Amount, Description: desc},
	})
	if err != nil {
		return a, apperrors.NewInternal(err, "Advance saved but could not be posted to the ledger")
	}
	a.Posted = true
	if err := uc.repo.SaveTempAdvanceState(ctx, a); err != nil {
		return a, apperrors.NewInternal(err, "Posted to the ledger but the advance could not be marked as posted")
	}
	return a, nil
}

func (uc *financeUseCase) validCashAccount(ctx context.Context, code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		code = AccountCash
	}
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return "", apperrors.NewInternal(err, "Failed to load accounts")
	}
	for _, a := range accounts {
		if a.Code == code {
			if !isCashAccount(a) || !a.IsActive {
				return "", apperrors.NewBadRequest("The cash/bank account must be an active cash or bank account")
			}
			return code, nil
		}
	}
	if code != AccountCash {
		return "", apperrors.NewBadRequest("Cash/bank account " + code + " does not exist")
	}
	return code, nil
}

// SettleTempAdvance accounts for a KBS (Pertanggungjawaban KBS). The spent
// lines are expensed; if less than the advance was spent the rest comes back
// to cash, if more the company pays the shortfall:
//
//	Dr expenses (spent) [+ Dr cash (returned)]  /  Cr Temporary Cash Advances (advance) [+ Cr cash (extra)]
func (uc *financeUseCase) SettleTempAdvance(ctx context.Context, id uuid.UUID, in TempAdvanceSettleInput) (*domain.TempAdvance, error) {
	a, err := uc.GetTempAdvance(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.Status != "open" {
		return nil, apperrors.NewBadRequest("Only an open advance can be settled")
	}
	date, err := resolveDate(in.Date)
	if err != nil {
		return nil, err
	}
	if date < a.Date {
		return nil, apperrors.NewBadRequest("Settlement date cannot be before the advance date")
	}
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load accounts")
	}
	byCode := make(map[string]domain.Account, len(accounts))
	for _, acc := range accounts {
		byCode[acc.Code] = acc
	}
	var spent float64
	lines := make([]domain.TempAdvanceLine, 0, len(in.Lines))
	for i, l := range in.Lines {
		code := strings.TrimSpace(l.AccountCode)
		acc, ok := byCode[code]
		if !ok || !acc.IsActive {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: account %q does not exist or is inactive", i+1, code))
		}
		if isCashAccount(acc) || code == AccountTempAdvance {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: choose an expense account, not cash or the advance account", i+1))
		}
		if l.Amount <= 0 {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: amount must be positive", i+1))
		}
		amt := round2(l.Amount)
		spent += amt
		line := domain.TempAdvanceLine{AdvanceID: a.ID, AccountCode: code, Description: strings.TrimSpace(l.Description), Amount: amt}
		line.ID = uuid.New()
		lines = append(lines, line)
	}
	spent = round2(spent)
	var returned, extra float64
	if spent < a.Amount {
		returned = round2(a.Amount - spent)
	} else {
		extra = round2(spent - a.Amount)
	}

	a.Status, a.SettledDate, a.SpentTotal, a.ReturnedAmount, a.ExtraPaid, a.Lines = "settled", date, spent, returned, extra, lines
	if err := uc.repo.SettleTempAdvance(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save the settlement")
	}

	desc := a.Number + " " + a.RecipientName
	entry := make([]LedgerLine, 0, len(lines)+3)
	for _, l := range lines {
		entry = append(entry, LedgerLine{AccountCode: l.AccountCode, Debit: l.Amount, Description: firstNonEmpty(l.Description, desc)})
	}
	if returned > 0 {
		entry = append(entry, LedgerLine{AccountCode: a.CashAccountCode, Debit: returned, Description: desc + " (sisa dikembalikan)"})
	}
	entry = append(entry, LedgerLine{AccountCode: AccountTempAdvance, Credit: a.Amount, Description: desc})
	if extra > 0 {
		entry = append(entry, LedgerLine{AccountCode: a.CashAccountCode, Credit: extra, Description: desc + " (kekurangan dibayar)"})
	}
	if err := NewLedgerPoster(uc.repo).PostEntryOn(ctx, date, tempAdvanceSettlePrefix+a.ID.String(), "KBS settlement "+desc, entry); err != nil {
		return a, apperrors.NewInternal(err, "Settlement saved but could not be posted to the ledger")
	}
	a.SettlementPosted = true
	if err := uc.repo.SaveTempAdvanceState(ctx, a); err != nil {
		return a, apperrors.NewInternal(err, "Posted to the ledger but the settlement could not be marked as posted")
	}
	return a, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// CancelTempAdvance voids an open advance that was never used, returning the cash.
func (uc *financeUseCase) CancelTempAdvance(ctx context.Context, id uuid.UUID) (*domain.TempAdvance, error) {
	a, err := uc.GetTempAdvance(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.Status != "open" {
		return nil, apperrors.NewBadRequest("Only an open advance can be cancelled")
	}
	today := time.Now().Format("2006-01-02")
	desc := a.Number + " " + a.RecipientName
	err = NewLedgerPoster(uc.repo).PostEntryOn(ctx, today, tempAdvanceCancelPrefix+a.ID.String(), "Cancel temporary cash advance "+desc, []LedgerLine{
		{AccountCode: a.CashAccountCode, Debit: a.Amount, Description: desc},
		{AccountCode: AccountTempAdvance, Credit: a.Amount, Description: desc},
	})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to reverse the advance in the ledger")
	}
	a.Status, a.CancelledDate = "cancelled", today
	if err := uc.repo.SaveTempAdvanceState(ctx, a); err != nil {
		return a, apperrors.NewInternal(err, "Reversed in the ledger but the advance could not be marked cancelled")
	}
	return a, nil
}

func (uc *financeUseCase) GetTempAdvance(ctx context.Context, id uuid.UUID) (*domain.TempAdvance, error) {
	a, err := uc.repo.GetTempAdvanceByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get the advance")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Cash advance not found")
	}
	return a, nil
}

func (uc *financeUseCase) ListTempAdvances(ctx context.Context, status, from, to string) ([]domain.TempAdvance, error) {
	if status != "" && status != "open" && status != "settled" && status != "cancelled" {
		return nil, apperrors.NewBadRequest("status must be open, settled or cancelled")
	}
	if err := checkDateRange(from, to); err != nil {
		return nil, err
	}
	out, err := uc.repo.ListTempAdvances(ctx, status, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list cash advances")
	}
	return out, nil
}

// OutstandingTempAdvances is the Laporan KBS yang Belum Selesai: advances
// issued on or before asOf that were not yet settled by then, oldest first.
func (uc *financeUseCase) OutstandingTempAdvances(ctx context.Context, asOf string) (*OutstandingTempAdvancesDTO, error) {
	today := time.Now().Format("2006-01-02")
	if asOf == "" {
		asOf = today
	} else if !validDate(asOf) {
		return nil, apperrors.NewBadRequest("asOf must be YYYY-MM-DD")
	}
	all, err := uc.repo.ListTempAdvances(ctx, "", "", asOf)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list cash advances")
	}
	ref, _ := time.Parse("2006-01-02", asOf)
	out := &OutstandingTempAdvancesDTO{AsOf: asOf, Rows: []OutstandingTempAdvanceDTO{}}
	for _, a := range all {
		open := a.Status == "open" ||
			(a.Status == "settled" && a.SettledDate > asOf) ||
			(a.Status == "cancelled" && a.CancelledDate > asOf)
		if !open {
			continue
		}
		issued, _ := time.Parse("2006-01-02", a.Date)
		row := OutstandingTempAdvanceDTO{
			ID: a.ID, Number: a.Number, Date: a.Date, DueDate: a.DueDate, RecipientName: a.RecipientName,
			Purpose: a.Purpose, Amount: a.Amount, AgeDays: int(ref.Sub(issued).Hours() / 24),
			Overdue: a.DueDate != "" && a.DueDate < asOf,
		}
		out.Total += a.Amount
		if row.Overdue {
			out.OverdueTotal += a.Amount
		}
		out.Rows = append(out.Rows, row)
	}
	// Oldest first so the longest-outstanding advances lead the report.
	for i := 1; i < len(out.Rows); i++ {
		for j := i; j > 0 && out.Rows[j].Date < out.Rows[j-1].Date; j-- {
			out.Rows[j], out.Rows[j-1] = out.Rows[j-1], out.Rows[j]
		}
	}
	out.Total, out.OverdueTotal = round2(out.Total), round2(out.OverdueTotal)
	return out, nil
}
