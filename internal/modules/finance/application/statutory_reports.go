package application

import (
	"context"
	"sort"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// Account categories used by the expense / non-operating reports.
const (
	CategoryMarketing    = "marketing"
	CategoryAdminGeneral = "admin_general"
	CategoryNonOperating = "non_operating"
	CategoryCOGS         = "cogs"
)

func validCategory(c string) (string, error) {
	switch c {
	case "", CategoryMarketing, CategoryAdminGeneral, CategoryNonOperating, CategoryCOGS:
		return c, nil
	}
	return "", apperrors.NewBadRequest("Category must be one of marketing, admin_general, non_operating, cogs")
}

// ClassifyAccount returns the report category of a revenue/expense account:
// the explicit Category when set, otherwise a default derived from the
// standard chart-of-accounts codes. Balance-sheet accounts return "".
func ClassifyAccount(a domain.Account) string {
	if a.Type != "expense" && a.Type != "revenue" {
		return ""
	}
	if a.Category != "" {
		return a.Category
	}
	switch a.Type {
	case "revenue":
		if a.Code == AccountOtherIncome {
			return CategoryNonOperating
		}
	case "expense":
		switch a.Code {
		case AccountCOGS:
			return CategoryCOGS
		case AccountDeliveryExpense:
			return CategoryMarketing
		case AccountRounding:
			return CategoryNonOperating
		default:
			return CategoryAdminGeneral
		}
	}
	return ""
}

func debitNormal(accountType string) bool { return accountType == "asset" || accountType == "expense" }

func checkDateRange(from, to string) error {
	if from != "" && !validDate(from) {
		return apperrors.NewBadRequest("from must be YYYY-MM-DD")
	}
	if to != "" && !validDate(to) {
		return apperrors.NewBadRequest("to must be YYYY-MM-DD")
	}
	if from != "" && to != "" && from > to {
		return apperrors.NewBadRequest("from cannot be after to")
	}
	return nil
}

// General ledger / cash book

type LedgerLineDTO struct {
	Date        string  `json:"date"`
	EntryNumber string  `json:"entryNumber"`
	Memo        string  `json:"memo"`
	Description string  `json:"description"`
	SourceDoc   string  `json:"sourceDoc"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
	Balance     float64 `json:"balance"`
}

type LedgerAccountDTO struct {
	AccountID   uuid.UUID       `json:"accountId"`
	Code        string          `json:"code"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Opening     float64         `json:"openingBalance"`
	Lines       []LedgerLineDTO `json:"lines"`
	TotalDebit  float64         `json:"totalDebit"`
	TotalCredit float64         `json:"totalCredit"`
	Closing     float64         `json:"closingBalance"`
}

// CashBookDayDTO summarises one day across the selected cash/bank accounts.
type CashBookDayDTO struct {
	Date    string  `json:"date"`
	Opening float64 `json:"opening"`
	In      float64 `json:"in"`
	Out     float64 `json:"out"`
	Closing float64 `json:"closing"`
}

type GeneralLedgerDTO struct {
	From     string             `json:"from"`
	To       string             `json:"to"`
	Accounts []LedgerAccountDTO `json:"accounts"`
	Days     []CashBookDayDTO   `json:"days,omitempty"`
}

// buildLedger assembles per-account ledgers (opening balance, running balance)
// for the given accounts. Accounts with neither opening balance nor activity
// in the range are left out.
func (uc *financeUseCase) buildLedger(ctx context.Context, accounts []domain.Account, from, to string) (*GeneralLedgerDTO, error) {
	ids := make([]uuid.UUID, 0, len(accounts))
	byID := make(map[uuid.UUID]domain.Account, len(accounts))
	for _, a := range accounts {
		ids = append(ids, a.ID)
		byID[a.ID] = a
	}

	opening := map[uuid.UUID]float64{}
	if from != "" {
		start, _ := time.Parse("2006-01-02", from)
		prior := start.AddDate(0, 0, -1).Format("2006-01-02")
		sums, err := uc.repo.SumPostedByAccount(ctx, "", prior)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to compute opening balances")
		}
		for _, s := range sums {
			a, ok := byID[s.AccountID]
			if !ok {
				continue
			}
			net := s.TotalDebit - s.TotalCredit
			if !debitNormal(a.Type) {
				net = -net
			}
			opening[s.AccountID] = net
		}
	}

	lines, err := uc.repo.ListPostedLines(ctx, from, to, ids)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load journal lines")
	}

	ledgers := map[uuid.UUID]*LedgerAccountDTO{}
	order := []uuid.UUID{}
	get := func(a domain.Account) *LedgerAccountDTO {
		if l, ok := ledgers[a.ID]; ok {
			return l
		}
		l := &LedgerAccountDTO{AccountID: a.ID, Code: a.Code, Name: a.Name, Type: a.Type,
			Opening: round2(opening[a.ID]), Closing: round2(opening[a.ID]), Lines: []LedgerLineDTO{}}
		ledgers[a.ID] = l
		order = append(order, a.ID)
		return l
	}
	for id, bal := range opening {
		if round2(bal) != 0 {
			get(byID[id])
		}
	}
	for _, ln := range lines {
		a, ok := byID[ln.AccountID]
		if !ok {
			continue
		}
		l := get(a)
		move := ln.Debit - ln.Credit
		if !debitNormal(a.Type) {
			move = -move
		}
		l.Closing = round2(l.Closing + move)
		l.TotalDebit = round2(l.TotalDebit + ln.Debit)
		l.TotalCredit = round2(l.TotalCredit + ln.Credit)
		l.Lines = append(l.Lines, LedgerLineDTO{Date: ln.Date, EntryNumber: ln.EntryNumber, Memo: ln.Memo,
			Description: ln.Description, SourceDoc: ln.SourceDoc, Debit: ln.Debit, Credit: ln.Credit, Balance: l.Closing})
	}

	out := &GeneralLedgerDTO{From: from, To: to, Accounts: make([]LedgerAccountDTO, 0, len(order))}
	for _, id := range order {
		out.Accounts = append(out.Accounts, *ledgers[id])
	}
	sort.Slice(out.Accounts, func(i, j int) bool { return out.Accounts[i].Code < out.Accounts[j].Code })
	return out, nil
}

// GeneralLedger (Buku Besar) lists each account's postings with a running
// balance. With no accountID it covers every account that has activity.
// With no date range at all it defaults to the current month.
func (uc *financeUseCase) GeneralLedger(ctx context.Context, accountID *uuid.UUID, from, to string) (*GeneralLedgerDTO, error) {
	if err := checkDateRange(from, to); err != nil {
		return nil, err
	}
	if from == "" && to == "" {
		now := time.Now()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		to = now.Format("2006-01-02")
	}
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list accounts")
	}
	if accountID != nil {
		var picked []domain.Account
		for _, a := range accounts {
			if a.ID == *accountID {
				picked = append(picked, a)
			}
		}
		if len(picked) == 0 {
			return nil, apperrors.NewNotFound("Account not found")
		}
		accounts = picked
	}
	return uc.buildLedger(ctx, accounts, from, to)
}

// CashBook (Laporan Kas/Bank Harian) is the ledger of the cash and bank
// accounts plus a per-day opening/in/out/closing summary. It defaults to today.
func (uc *financeUseCase) CashBook(ctx context.Context, accountCode, from, to string) (*GeneralLedgerDTO, error) {
	if err := checkDateRange(from, to); err != nil {
		return nil, err
	}
	if from == "" && to == "" {
		from = time.Now().Format("2006-01-02")
		to = from
	}
	if from == "" {
		from = to
	}
	if to == "" {
		to = from
	}
	all, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list accounts")
	}
	var cash []domain.Account
	for _, a := range all {
		if isCashAccount(a) && (accountCode == "" || a.Code == accountCode) {
			cash = append(cash, a)
		}
	}
	if accountCode != "" && len(cash) == 0 {
		return nil, apperrors.NewBadRequest("Account " + accountCode + " is not a cash or bank account")
	}
	out, err := uc.buildLedger(ctx, cash, from, to)
	if err != nil {
		return nil, err
	}

	var openingTotal float64
	type flow struct{ in, out float64 }
	byDay := map[string]*flow{}
	for _, acc := range out.Accounts {
		openingTotal += acc.Opening
		for _, l := range acc.Lines {
			f := byDay[l.Date]
			if f == nil {
				f = &flow{}
				byDay[l.Date] = f
			}
			f.in += l.Debit
			f.out += l.Credit
		}
	}
	days := make([]string, 0, len(byDay))
	for d := range byDay {
		days = append(days, d)
	}
	sort.Strings(days)
	running := openingTotal
	for _, d := range days {
		f := byDay[d]
		out.Days = append(out.Days, CashBookDayDTO{Date: d, Opening: round2(running), In: round2(f.in), Out: round2(f.out), Closing: round2(running + f.in - f.out)})
		running += f.in - f.out
	}
	return out, nil
}

// Expense / non-operating breakdowns

type AmountLineDTO struct {
	AccountCode string  `json:"accountCode"`
	AccountName string  `json:"accountName"`
	Amount      float64 `json:"amount"`
}

type ExpenseGroupDTO struct {
	Category string          `json:"category"`
	Label    string          `json:"label"`
	Lines    []AmountLineDTO `json:"lines"`
	Total    float64         `json:"total"`
}

type ExpenseBreakdownDTO struct {
	From   string            `json:"from"`
	To     string            `json:"to"`
	Groups []ExpenseGroupDTO `json:"groups"`
	Total  float64           `json:"total"`
}

type NonOperatingDTO struct {
	From          string          `json:"from"`
	To            string          `json:"to"`
	Income        []AmountLineDTO `json:"income"`
	Expenses      []AmountLineDTO `json:"expenses"`
	TotalIncome   float64         `json:"totalIncome"`
	TotalExpenses float64         `json:"totalExpenses"`
	Net           float64         `json:"net"`
}

// plAmounts returns the net P&L movement per account for the range: revenue
// is credit-minus-debit, expense is debit-minus-credit.
func (uc *financeUseCase) plAmounts(ctx context.Context, from, to string) ([]domain.Account, map[uuid.UUID]float64, error) {
	if err := checkDateRange(from, to); err != nil {
		return nil, nil, err
	}
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, nil, apperrors.NewInternal(err, "Failed to list accounts")
	}
	sums, err := uc.repo.SumPostedByAccount(ctx, from, to)
	if err != nil {
		return nil, nil, apperrors.NewInternal(err, "Failed to compute posted totals")
	}
	amounts := make(map[uuid.UUID]float64, len(sums))
	typeOf := make(map[uuid.UUID]string, len(accounts))
	for _, a := range accounts {
		typeOf[a.ID] = a.Type
	}
	for _, s := range sums {
		switch typeOf[s.AccountID] {
		case "revenue":
			amounts[s.AccountID] = s.TotalCredit - s.TotalDebit
		case "expense":
			amounts[s.AccountID] = s.TotalDebit - s.TotalCredit
		}
	}
	return accounts, amounts, nil
}

func sortedLines(m map[string]AmountLineDTO) []AmountLineDTO {
	out := make([]AmountLineDTO, 0, len(m))
	for _, l := range m {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AccountCode < out[j].AccountCode })
	return out
}

// ExpenseBreakdown (Rincian Biaya Usaha) splits operating expenses into
// marketing and administrative & general. Cost of goods sold and
// non-operating expenses are excluded.
func (uc *financeUseCase) ExpenseBreakdown(ctx context.Context, from, to string) (*ExpenseBreakdownDTO, error) {
	accounts, amounts, err := uc.plAmounts(ctx, from, to)
	if err != nil {
		return nil, err
	}
	groups := []struct{ key, label string }{
		{CategoryMarketing, "Biaya Pemasaran"},
		{CategoryAdminGeneral, "Biaya Administrasi & Umum"},
	}
	out := &ExpenseBreakdownDTO{From: from, To: to, Groups: []ExpenseGroupDTO{}}
	for _, g := range groups {
		lines := map[string]AmountLineDTO{}
		var total float64
		for _, a := range accounts {
			amt := round2(amounts[a.ID])
			if a.Type != "expense" || ClassifyAccount(a) != g.key || amt == 0 {
				continue
			}
			lines[a.Code] = AmountLineDTO{AccountCode: a.Code, AccountName: a.Name, Amount: amt}
			total += amt
		}
		out.Groups = append(out.Groups, ExpenseGroupDTO{Category: g.key, Label: g.label, Lines: sortedLines(lines), Total: round2(total)})
		out.Total += total
	}
	out.Total = round2(out.Total)
	return out, nil
}

// NonOperating (Rincian Pendapatan dan Biaya di Luar Usaha) lists income and
// expense accounts categorised as non_operating.
func (uc *financeUseCase) NonOperating(ctx context.Context, from, to string) (*NonOperatingDTO, error) {
	accounts, amounts, err := uc.plAmounts(ctx, from, to)
	if err != nil {
		return nil, err
	}
	income, expenses := map[string]AmountLineDTO{}, map[string]AmountLineDTO{}
	out := &NonOperatingDTO{From: from, To: to}
	for _, a := range accounts {
		amt := round2(amounts[a.ID])
		if amt == 0 || ClassifyAccount(a) != CategoryNonOperating {
			continue
		}
		line := AmountLineDTO{AccountCode: a.Code, AccountName: a.Name, Amount: amt}
		if a.Type == "revenue" {
			income[a.Code] = line
			out.TotalIncome += amt
		} else {
			expenses[a.Code] = line
			out.TotalExpenses += amt
		}
	}
	out.Income, out.Expenses = sortedLines(income), sortedLines(expenses)
	out.TotalIncome, out.TotalExpenses = round2(out.TotalIncome), round2(out.TotalExpenses)
	out.Net = round2(out.TotalIncome - out.TotalExpenses)
	return out, nil
}
