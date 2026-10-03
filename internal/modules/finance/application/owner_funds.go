package application

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// Source-document key prefixes for the entries posted here.
const (
	CapitalSourcePrefix     = "capital:"
	OtherIncomeSourcePrefix = "other-income:"
)

// CapitalLedgerEntry: an injection is Dr cash-or-bank / Cr Owner's Equity; a
// drawing is Dr Owner Drawings / Cr cash-or-bank.
func CapitalLedgerEntry(c *domain.CapitalTransaction) LedgerEntry {
	label := c.OwnerName
	if c.Type == "injection" {
		return LedgerEntry{
			SourceDoc: CapitalSourcePrefix + c.ID.String(),
			Memo:      "Capital injection by " + label,
			Lines: []LedgerLine{
				{AccountCode: c.PaymentAccountCode, Debit: c.Amount, Description: label},
				{AccountCode: AccountOwnerCapital, Credit: c.Amount, Description: label},
			},
		}
	}
	return LedgerEntry{
		SourceDoc: CapitalSourcePrefix + c.ID.String(),
		Memo:      "Owner drawing by " + label,
		Lines: []LedgerLine{
			{AccountCode: AccountOwnerDrawings, Debit: c.Amount, Description: label},
			{AccountCode: c.PaymentAccountCode, Credit: c.Amount, Description: label},
		},
	}
}

// OtherIncomeLedgerEntry is Dr cash-or-bank / Cr Other Income.
func OtherIncomeLedgerEntry(o *domain.OtherIncome) LedgerEntry {
	return LedgerEntry{
		SourceDoc: OtherIncomeSourcePrefix + o.ID.String(),
		Memo:      "Other income: " + o.Category,
		Lines: []LedgerLine{
			{AccountCode: o.PaymentAccountCode, Debit: o.Amount, Description: o.Category},
			{AccountCode: AccountOtherIncome, Credit: o.Amount, Description: o.Category},
		},
	}
}

type CapitalInput struct {
	OwnerName, Date, Description, PaymentAccountCode string
	Amount                                           float64
}

type OtherIncomeInput struct {
	Category, Date, Description, PaymentAccountCode string
	Amount                                          float64
}

func validDate(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }

// resolveDate defaults to today and refuses future dates, so a mistyped year
// cannot push a posting outside the period it belongs to.
func resolveDate(s string) (string, error) {
	today := time.Now().Format("2006-01-02")
	if s == "" {
		return today, nil
	}
	if !validDate(s) {
		return "", apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	if s > today {
		return "", apperrors.NewBadRequest("date cannot be in the future")
	}
	return s, nil
}

// paymentAccount validates the cash/bank account a movement goes through: the
// default cash account, or an existing active asset account.
func (uc *financeUseCase) paymentAccount(ctx context.Context, code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" || code == AccountCash {
		return AccountCash, nil
	}
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return "", apperrors.NewInternal(err, "Failed to load accounts")
	}
	for _, a := range accounts {
		if a.Code == code {
			if a.Type != "asset" || !a.IsActive {
				return "", apperrors.NewBadRequest("The payment account must be an active asset (cash/bank) account")
			}
			return code, nil
		}
	}
	return "", apperrors.NewBadRequest("Payment account " + code + " does not exist")
}

// RecordCapital records an injection or a drawing and posts it. The record is
// saved first and flagged Posted only once the journal entry exists, so a
// failed posting is visible (Posted=false) rather than silently lost.
func (uc *financeUseCase) RecordCapital(ctx context.Context, typ string, in CapitalInput) (*domain.CapitalTransaction, error) {
	if typ != "injection" && typ != "drawing" {
		return nil, apperrors.NewBadRequest("type must be injection or drawing")
	}
	owner := strings.TrimSpace(in.OwnerName)
	if owner == "" || in.Amount <= 0 {
		return nil, apperrors.NewBadRequest("ownerName and a positive amount are required")
	}
	date, err := resolveDate(in.Date)
	if err != nil {
		return nil, err
	}
	code, err := uc.paymentAccount(ctx, in.PaymentAccountCode)
	if err != nil {
		return nil, err
	}
	c := &domain.CapitalTransaction{Type: typ, OwnerName: owner, Amount: round2(in.Amount), Date: date,
		Description: strings.TrimSpace(in.Description), PaymentAccountCode: code, CreatedByEmail: actor.EmailFrom(ctx)}
	c.ID = uuid.New()
	if err := uc.repo.CreateCapitalTransaction(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save capital transaction")
	}
	e := CapitalLedgerEntry(c)
	if err := NewLedgerPoster(uc.repo).PostEntryOn(ctx, c.Date, e.SourceDoc, e.Memo, e.Lines); err != nil {
		return c, apperrors.NewInternal(err, "Capital transaction saved but could not be posted to the ledger")
	}
	c.Posted = true
	if err := uc.repo.UpdateCapitalTransaction(ctx, c); err != nil {
		return c, apperrors.NewInternal(err, "Posted to the ledger but the record could not be marked as posted")
	}
	return c, nil
}

func (uc *financeUseCase) ListCapital(ctx context.Context, typ, period string) ([]domain.CapitalTransaction, error) {
	out, err := uc.repo.ListCapitalTransactions(ctx, typ, period)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list capital transactions")
	}
	return out, nil
}

func (uc *financeUseCase) RecordOtherIncome(ctx context.Context, in OtherIncomeInput) (*domain.OtherIncome, error) {
	category := strings.TrimSpace(in.Category)
	if category == "" || in.Amount <= 0 {
		return nil, apperrors.NewBadRequest("category and a positive amount are required")
	}
	date, err := resolveDate(in.Date)
	if err != nil {
		return nil, err
	}
	code, err := uc.paymentAccount(ctx, in.PaymentAccountCode)
	if err != nil {
		return nil, err
	}
	o := &domain.OtherIncome{Category: category, Amount: round2(in.Amount), Date: date, Description: strings.TrimSpace(in.Description),
		PaymentAccountCode: code, CreatedByEmail: actor.EmailFrom(ctx)}
	o.ID = uuid.New()
	if err := uc.repo.CreateOtherIncome(ctx, o); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save other income")
	}
	e := OtherIncomeLedgerEntry(o)
	if err := NewLedgerPoster(uc.repo).PostEntryOn(ctx, o.Date, e.SourceDoc, e.Memo, e.Lines); err != nil {
		return o, apperrors.NewInternal(err, "Other income saved but could not be posted to the ledger")
	}
	o.Posted = true
	if err := uc.repo.UpdateOtherIncome(ctx, o); err != nil {
		return o, apperrors.NewInternal(err, "Posted to the ledger but the record could not be marked as posted")
	}
	return o, nil
}

func (uc *financeUseCase) ListOtherIncome(ctx context.Context, category, period string) ([]domain.OtherIncome, error) {
	out, err := uc.repo.ListOtherIncome(ctx, category, period)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list other income")
	}
	return out, nil
}
