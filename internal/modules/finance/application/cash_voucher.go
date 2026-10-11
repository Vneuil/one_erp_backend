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

const CashVoucherSourcePrefix = "cash-voucher:"

type CashVoucherLineInput struct {
	AccountCode string  `json:"accountCode"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

type CashVoucherInput struct {
	Type            string                 `json:"type"` // receipt | payment
	Date            string                 `json:"date"`
	CashAccountCode string                 `json:"cashAccountCode"`
	Counterparty    string                 `json:"counterparty"`
	Description     string                 `json:"description"`
	Lines           []CashVoucherLineInput `json:"lines"`
}

// CashVoucherLedgerEntry: a receipt is Dr cash-or-bank / Cr each counter
// account; a payment is Dr each counter account / Cr cash-or-bank.
func CashVoucherLedgerEntry(v *domain.CashVoucher) LedgerEntry {
	who := v.Counterparty
	if who == "" {
		who = v.Description
	}
	memo := "Cash/bank receipt " + v.Number
	if v.Type == "payment" {
		memo = "Cash/bank payment " + v.Number
	}
	if who != "" {
		memo += " - " + who
	}
	lines := make([]LedgerLine, 0, len(v.Lines)+1)
	if v.Type == "receipt" {
		lines = append(lines, LedgerLine{AccountCode: v.CashAccountCode, Debit: v.Total, Description: v.Number})
		for _, l := range v.Lines {
			lines = append(lines, LedgerLine{AccountCode: l.AccountCode, Credit: l.Amount, Description: l.Description})
		}
	} else {
		for _, l := range v.Lines {
			lines = append(lines, LedgerLine{AccountCode: l.AccountCode, Debit: l.Amount, Description: l.Description})
		}
		lines = append(lines, LedgerLine{AccountCode: v.CashAccountCode, Credit: v.Total, Description: v.Number})
	}
	return LedgerEntry{SourceDoc: CashVoucherSourcePrefix + v.ID.String(), Memo: memo, Lines: lines}
}

// CreateCashVoucher validates, numbers and saves a voucher, then posts it. As
// with owner funds, Posted is set only once the journal entry exists.
func (uc *financeUseCase) CreateCashVoucher(ctx context.Context, in CashVoucherInput) (*domain.CashVoucher, error) {
	if in.Type != "receipt" && in.Type != "payment" {
		return nil, apperrors.NewBadRequest("type must be receipt or payment")
	}
	if len(in.Lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line is required")
	}
	date, err := resolveDate(in.Date)
	if err != nil {
		return nil, err
	}
	accounts, err := uc.repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load accounts")
	}
	byCode := make(map[string]domain.Account, len(accounts))
	for _, a := range accounts {
		byCode[a.Code] = a
	}

	cashCode := strings.TrimSpace(in.CashAccountCode)
	if cashCode == "" {
		cashCode = AccountCash
	}
	if a, ok := byCode[cashCode]; ok {
		if !isCashAccount(a) || !a.IsActive {
			return nil, apperrors.NewBadRequest("The cash/bank account must be an active cash or bank account")
		}
	} else if cashCode != AccountCash {
		return nil, apperrors.NewBadRequest("Cash/bank account " + cashCode + " does not exist")
	}

	var total float64
	lines := make([]domain.CashVoucherLine, 0, len(in.Lines))
	for i, l := range in.Lines {
		code := strings.TrimSpace(l.AccountCode)
		acc, ok := byCode[code]
		if !ok || !acc.IsActive {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: account %q does not exist or is inactive", i+1, code))
		}
		if code == cashCode {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: the counter account cannot be the cash/bank account itself", i+1))
		}
		if l.Amount <= 0 {
			return nil, apperrors.NewBadRequest(fmt.Sprintf("Line %d: amount must be positive", i+1))
		}
		amt := round2(l.Amount)
		total += amt
		line := domain.CashVoucherLine{AccountCode: code, Description: strings.TrimSpace(l.Description), Amount: amt}
		line.ID = uuid.New()
		lines = append(lines, line)
	}

	prefix := "BKM"
	if in.Type == "payment" {
		prefix = "BKK"
	}
	numberPrefix := fmt.Sprintf("%s-%s-", prefix, strings.ReplaceAll(date[:7], "-", ""))
	n, err := uc.repo.CountCashVouchers(ctx, in.Type, numberPrefix)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to number the voucher")
	}

	v := &domain.CashVoucher{
		Number: fmt.Sprintf("%s%04d", numberPrefix, n+1), Type: in.Type, Date: date, CashAccountCode: cashCode,
		Counterparty: strings.TrimSpace(in.Counterparty), Description: strings.TrimSpace(in.Description),
		Total: round2(total), CreatedByEmail: actor.EmailFrom(ctx), Lines: lines,
	}
	v.ID = uuid.New()
	for i := range v.Lines {
		v.Lines[i].VoucherID = v.ID
	}
	if err := uc.repo.CreateCashVoucher(ctx, v); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save cash voucher")
	}
	e := CashVoucherLedgerEntry(v)
	if err := NewLedgerPoster(uc.repo).PostEntryOn(ctx, v.Date, e.SourceDoc, e.Memo, e.Lines); err != nil {
		return v, apperrors.NewInternal(err, "Voucher saved but could not be posted to the ledger")
	}
	v.Posted = true
	if err := uc.repo.UpdateCashVoucher(ctx, v); err != nil {
		return v, apperrors.NewInternal(err, "Posted to the ledger but the voucher could not be marked as posted")
	}
	return v, nil
}

func (uc *financeUseCase) GetCashVoucher(ctx context.Context, id uuid.UUID) (*domain.CashVoucher, error) {
	v, err := uc.repo.GetCashVoucherByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get cash voucher")
	}
	if v == nil {
		return nil, apperrors.NewNotFound("Cash voucher not found")
	}
	return v, nil
}

func (uc *financeUseCase) ListCashVouchers(ctx context.Context, typ, from, to string) ([]domain.CashVoucher, error) {
	if err := checkDateRange(from, to); err != nil {
		return nil, err
	}
	out, err := uc.repo.ListCashVouchers(ctx, typ, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list cash vouchers")
	}
	return out, nil
}

const CashVoucherVoidSourcePrefix = "cash-voucher-void:"

// CashVoucherVoidEntry is the voucher's journal with debit and credit swapped.
func CashVoucherVoidEntry(v *domain.CashVoucher) LedgerEntry {
	e := CashVoucherLedgerEntry(v)
	lines := make([]LedgerLine, len(e.Lines))
	for i, l := range e.Lines {
		lines[i] = LedgerLine{AccountCode: l.AccountCode, Debit: l.Credit, Credit: l.Debit, Description: "Batal " + l.Description}
	}
	return LedgerEntry{SourceDoc: CashVoucherVoidSourcePrefix + v.ID.String(), Memo: "Pembatalan " + e.Memo, Lines: lines}
}

// VoidCashVoucher cancels a voucher: its journal entry is reversed by an
// entry dated today (closed periods stay untouched) and the voucher stays on
// record marked void. To correct a voucher, void it and record the right one.
func (uc *financeUseCase) VoidCashVoucher(ctx context.Context, id uuid.UUID, reason string) (*domain.CashVoucher, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required to void a voucher")
	}
	v, err := uc.GetCashVoucher(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.Voided {
		return nil, apperrors.NewConflict("This voucher is already voided")
	}
	if v.Posted {
		e := CashVoucherVoidEntry(v)
		if err := NewLedgerPoster(uc.repo).PostEntryOn(ctx, time.Now().Format("2006-01-02"), e.SourceDoc, e.Memo, e.Lines); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to reverse the voucher in the ledger; nothing was changed")
		}
	}
	now := time.Now()
	v.Voided, v.VoidedAt, v.VoidedBy, v.VoidReason = true, &now, actor.EmailFrom(ctx), reason
	if err := uc.repo.VoidCashVoucher(ctx, v); err != nil {
		return v, apperrors.NewInternal(err, "The journal was reversed but the voucher could not be marked void; void it again to finish")
	}
	return v, nil
}
