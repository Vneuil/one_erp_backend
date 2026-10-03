package application

import (
	"context"
	"math"
	"strings"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"github.com/google/uuid"
)

const maxAdvanceInstallments = 12

// WithLedger books the cash paid out on approval (Dr Employee Advances / Cr Cash).
func (s *Service) WithLedger(l financeApp.LedgerPoster) *Service {
	s.ledger = l
	return s
}

// AdvanceView is an advance plus where it stands as of the current month.
type AdvanceView struct {
	domain.CashAdvance
	Monthly     float64 `json:"monthly"`     // the regular installment
	Outstanding float64 `json:"outstanding"` // balance after this month's installment
	Settled     bool    `json:"settled"`
}

func (s *Service) advanceView(a domain.CashAdvance) AdvanceView {
	period := s.now().In(zone).Format("2006-01")
	v := AdvanceView{CashAdvance: a}
	if a.Installments > 0 {
		v.Monthly = math.Floor(a.Amount/float64(a.Installments)*100) / 100
	}
	if a.Status == "approved" {
		v.Outstanding = a.OutstandingAfter(period)
		v.Settled = v.Outstanding == 0
	}
	return v
}

func validPeriod(p string) bool {
	_, err := time.Parse("2006-01", p)
	return err == nil && len(p) == 7
}

func (s *Service) RequestAdvance(ctx context.Context, c Caller, nip string, amount float64, installments int, startPeriod, reason string) (*AdvanceView, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 {
		return nil, apperrors.NewBadRequest("amount must be positive")
	}
	if installments < 1 || installments > maxAdvanceInstallments {
		return nil, apperrors.NewBadRequest("installments must be between 1 and 12")
	}
	if !validPeriod(startPeriod) {
		return nil, apperrors.NewBadRequest("startPeriod must be YYYY-MM")
	}
	if len(reason) > 500 {
		return nil, apperrors.NewBadRequest("reason is too long")
	}
	emp, err := s.resolveEmployee(ctx, c, nip)
	if err != nil {
		return nil, err
	}
	// One open advance at a time: a new one waits until the last is repaid or decided.
	existing, err := s.repo.ListAdvances(ctx, emp.NIP, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing advances")
	}
	for _, e := range existing {
		v := s.advanceView(e)
		if e.Status == "pending" || (e.Status == "approved" && !v.Settled) {
			return nil, apperrors.NewConflict("This employee still has an open cash advance")
		}
	}
	a := &domain.CashAdvance{NIP: emp.NIP, EmployeeName: emp.Name, EmployeeEmail: emp.Email, Amount: math.Round(amount*100) / 100,
		Installments: installments, StartPeriod: startPeriod, Reason: strings.TrimSpace(reason), Status: "pending", RequestedByEmail: c.Email}
	if err := s.repo.CreateAdvance(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save cash advance")
	}
	s.notifyManagerOf(ctx, emp, "Kasbon baru", emp.Name+" mengajukan kasbon")
	v := s.advanceView(*a)
	return &v, nil
}

func (s *Service) ListAdvances(ctx context.Context, nip, status string) ([]AdvanceView, error) {
	list, err := s.repo.ListAdvances(ctx, nip, status)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list cash advances")
	}
	out := make([]AdvanceView, len(list))
	for i, a := range list {
		out[i] = s.advanceView(a)
	}
	return out, nil
}

func (s *Service) DecideAdvance(ctx context.Context, id uuid.UUID, approve bool) (*AdvanceView, error) {
	a, err := s.repo.GetAdvance(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load cash advance")
	}
	if a == nil {
		return nil, apperrors.NewNotFound("Cash advance not found")
	}
	if err := decided(a.Status); err != nil {
		return nil, err
	}
	if approve {
		// Neither the borrower nor whoever filed it on their behalf may approve it.
		if err := sod.ForbidSelfApproval(ctx, a.EmployeeEmail); err != nil {
			return nil, err
		}
		if err := sod.ForbidSelfApproval(ctx, a.RequestedByEmail); err != nil {
			return nil, err
		}
		a.Status = "approved"
	} else {
		a.Status = "rejected"
	}
	a.DecidedBy = actor.EmailFrom(ctx)
	if err := s.repo.UpdateAdvance(ctx, a); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update cash advance")
	}
	if approve && s.ledger != nil {
		// Best effort, like payroll: a ledger failure must not undo the decision.
		_ = s.ledger.PostEntry(ctx, "cash-advance:"+a.ID.String(), "Cash advance "+a.EmployeeName, []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountEmployeeAdvance, Debit: a.Amount, Description: a.EmployeeName},
			{AccountCode: financeApp.AccountCash, Credit: a.Amount, Description: a.EmployeeName},
		})
	}
	s.notifyOutcome(ctx, a.NIP, a.RequestedByEmail, "Kasbon", approve)
	v := s.advanceView(*a)
	return &v, nil
}
