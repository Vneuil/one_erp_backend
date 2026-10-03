package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// PayslipLine is one earning or deduction on a payslip.
type PayslipLine struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
	Note   string  `json:"note,omitempty"`
}

// PayslipOvertime is one approved overtime day and what it paid.
type PayslipOvertime struct {
	Date    string  `json:"date"`
	Minutes int     `json:"minutes"`
	Amount  float64 `json:"amount"`
}

// Payslip is the employee-facing breakdown of one payroll entry.
type Payslip struct {
	EntryID        uuid.UUID         `json:"entryId"`
	Period         string            `json:"period"`
	EmployeeName   string            `json:"employeeName"`
	NIP            string            `json:"nip"`
	Department     string            `json:"department"`
	Status         string            `json:"status"`
	Bank           string            `json:"bank,omitempty"`
	AccountMasked  string            `json:"accountMasked,omitempty"`
	Earnings       []PayslipLine     `json:"earnings"`
	Gross          float64           `json:"gross"`
	Deductions     []PayslipLine     `json:"deductions"`
	TotalDeduction float64           `json:"totalDeductions"`
	Overtime       []PayslipOvertime `json:"overtime"`
	OvertimeHours  float64           `json:"overtimeHours"`
	TakeHomePay    float64           `json:"takeHomePay"`
}

// maskAccount keeps only the last four digits of a bank account number.
func maskAccount(acc string) string {
	acc = strings.TrimSpace(acc)
	if len(acc) <= 4 {
		return acc
	}
	return strings.Repeat("•", len(acc)-4) + acc[len(acc)-4:]
}

// BuildPayslip lays an entry out as earnings and deductions. dates and minutes
// are the approved overtime records behind the overtime pay, priced one by one
// the same way payroll priced them. It is pure so it can be tested directly.
func BuildPayslip(e *domain.PayrollEntry, dates []string, minutes []int) Payslip {
	p := Payslip{EntryID: e.ID, Period: e.Period, EmployeeName: e.EmployeeName, NIP: e.NIP, Department: e.Department, Status: e.Status,
		Bank: e.PaymentBank, AccountMasked: maskAccount(e.BankAccount), TakeHomePay: e.TakeHomePay,
		Earnings: []PayslipLine{}, Deductions: []PayslipLine{}, Overtime: []PayslipOvertime{}}

	add := func(lines *[]PayslipLine, label string, amount float64, note string) {
		if amount != 0 {
			*lines = append(*lines, PayslipLine{Label: label, Amount: round2(amount), Note: note})
		}
	}
	p.Earnings = append(p.Earnings, PayslipLine{Label: "Gaji pokok", Amount: round2(e.BaseSalary)})
	add(&p.Earnings, "Tunjangan", e.Allowance, "")

	var totalMin int
	var otSum float64
	for i, m := range minutes {
		d := ""
		if i < len(dates) {
			d = dates[i]
		}
		amt := OvertimePayFor(e.BaseSalary, []int{m})
		p.Overtime = append(p.Overtime, PayslipOvertime{Date: d, Minutes: m, Amount: amt})
		totalMin += m
		otSum += amt
	}
	p.OvertimeHours = round2(float64(totalMin) / 60)
	note := ""
	if len(p.Overtime) > 0 {
		note = "lihat rincian lembur"
	}
	add(&p.Earnings, "Lembur", e.OvertimePay, note)

	for _, l := range p.Earnings {
		p.Gross += l.Amount
	}
	p.Gross = round2(p.Gross)

	absNote, lateNote := "", ""
	if e.UnpaidDays > 0 {
		absNote = itoa(e.UnpaidDays) + " hari cuti tanpa gaji"
	}
	if e.LateCount > 0 {
		lateNote = itoa(e.LateCount) + " kali terlambat"
	}
	add(&p.Deductions, "Potongan cuti tanpa gaji", e.DeductionAbsence, absNote)
	add(&p.Deductions, "Denda keterlambatan", e.DeductionLate, lateNote)
	add(&p.Deductions, "PPh 21", e.DeductionTax, "")
	add(&p.Deductions, "Koperasi", e.DeductionCoop, "")
	add(&p.Deductions, "Kantin", e.DeductionCanteen, "")
	add(&p.Deductions, "Cicilan kasbon", e.DeductionAdvance, "")
	for _, l := range p.Deductions {
		p.TotalDeduction += l.Amount
	}
	p.TotalDeduction = round2(p.TotalDeduction)
	return p
}

func itoa(n int) string {
	const digits = "0123456789"
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{digits[n%10]}, b...)
		n /= 10
	}
	return string(b)
}

// OvertimeDetailSource lists the approved overtime records behind a month's
// overtime pay (dates and minutes, in matching order).
type OvertimeDetailSource interface {
	ApprovedOvertimeDays(ctx context.Context, nip, period string) (dates []string, minutes []int, err error)
}

// WithOvertimeDetail adds the per-day overtime table to payslips.
func WithOvertimeDetail(o OvertimeDetailSource) Option {
	return func(uc *payrollUseCase) { uc.overtimeDetail = o }
}

// PayslipSummary is one row of an employee's payslip list.
type PayslipSummary struct {
	EntryID     uuid.UUID `json:"entryId"`
	Period      string    `json:"period"`
	Status      string    `json:"status"`
	TakeHomePay float64   `json:"takeHomePay"`
}

// payslipVisible: employees see a payslip once it is approved; drafts are still being worked on.
func payslipVisible(status string) bool { return status == "approved" || status == "paid" }

func (uc *payrollUseCase) buildPayslip(ctx context.Context, e *domain.PayrollEntry) (*Payslip, error) {
	var dates []string
	var mins []int
	if uc.overtimeDetail != nil && e.NIP != "" && e.OvertimePay > 0 {
		var err error
		dates, mins, err = uc.overtimeDetail.ApprovedOvertimeDays(ctx, e.NIP, e.Period)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to load overtime records")
		}
	}
	p := BuildPayslip(e, dates, mins)
	return &p, nil
}

// Payslip returns the payslip of an entry. Unless the caller is HR it must be
// the caller's own entry, and approved or paid.
func (uc *payrollUseCase) Payslip(ctx context.Context, id uuid.UUID, callerEmail string, privileged bool) (*Payslip, error) {
	e, err := uc.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load payroll entry")
	}
	if e == nil {
		return nil, apperrors.NewNotFound("Payslip not found")
	}
	if !privileged {
		own, err := uc.ownEmployeeID(ctx, callerEmail)
		if err != nil {
			return nil, err
		}
		// A foreign entry looks the same as a missing one.
		if own == uuid.Nil || e.EmployeeID == nil || *e.EmployeeID != own || !payslipVisible(e.Status) {
			return nil, apperrors.NewNotFound("Payslip not found")
		}
	}
	return uc.buildPayslip(ctx, e)
}

// MyPayslips lists the caller's own approved and paid payslips, newest first.
func (uc *payrollUseCase) MyPayslips(ctx context.Context, callerEmail string) ([]PayslipSummary, error) {
	own, err := uc.ownEmployeeID(ctx, callerEmail)
	if err != nil {
		return nil, err
	}
	out := []PayslipSummary{}
	if own == uuid.Nil {
		return out, nil
	}
	list, err := uc.repo.ListEntriesByEmployee(ctx, own)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list payslips")
	}
	for _, e := range list {
		if payslipVisible(e.Status) {
			out = append(out, PayslipSummary{EntryID: e.ID, Period: e.Period, Status: e.Status, TakeHomePay: e.TakeHomePay})
		}
	}
	return out, nil
}

func (uc *payrollUseCase) ownEmployeeID(ctx context.Context, email string) (uuid.UUID, error) {
	if uc.hrmRepo == nil {
		return uuid.Nil, nil
	}
	emp, err := uc.hrmRepo.FindEmployeeByEmail(ctx, email)
	if err != nil {
		return uuid.Nil, apperrors.NewInternal(err, "Failed to look up your employee record")
	}
	if emp == nil {
		return uuid.Nil, nil
	}
	return emp.ID, nil
}
