package application

import (
	"context"
	"github.com/divinecoid/one-backend/internal/shared/inbox"
	"log/slog"
	"math"
	"strings"

	coopdomain "github.com/divinecoid/one-backend/internal/modules/cooperative/domain"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	leavedomain "github.com/divinecoid/one-backend/internal/modules/leave/domain"
	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// statusOrder defines the valid forward progression of a payroll entry's
// status. A transition is only allowed to the very next status in this
// list (e.g. draft -> calculated), never skipping ahead.
var statusOrder = []string{"draft", "calculated", "approved", "paid"}

func statusIndex(status string) int {
	for i, s := range statusOrder {
		if s == status {
			return i
		}
	}
	return -1
}

type PayrollUseCase interface {
	CreateEntry(ctx context.Context, dto CreatePayrollEntryDTO) (*PayrollEntryResponseDTO, error)
	ListEntries(ctx context.Context, query types.PaginationQuery, period string) ([]PayrollEntryResponseDTO, types.PaginationMeta, error)
	CalculatePeriod(ctx context.Context, period string) ([]PayrollEntryResponseDTO, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*PayrollEntryResponseDTO, error)
	GetPolicy(ctx context.Context) (*domain.PayrollPolicy, error)
	UpdatePolicy(ctx context.Context, dto UpdatePolicyDTO) (*domain.PayrollPolicy, error)
	Payslip(ctx context.Context, id uuid.UUID, callerEmail string, privileged bool) (*Payslip, error)
	MyPayslips(ctx context.Context, callerEmail string) ([]PayslipSummary, error)

	SeedInitialData(ctx context.Context) error
}

type payrollUseCase struct {
	repo           domain.PayrollRepository
	hrmRepo        hrmdomain.HRMRepository
	coopRepo       coopdomain.CooperativeRepository
	overtime       OvertimeSource              // optional; prices approved overtime records
	inbox          inbox.Sender                // optional; tells the employee their payslip is ready
	overtimeDetail OvertimeDetailSource        // optional; per-day overtime on payslips
	advances       AdvanceSource               // optional; withholds the month's cash advance installment
	canteen        CanteenSource               // optional; withholds the month's meal orders
	leaveRepo      leavedomain.LeaveRepository // optional; enables unpaid-leave withholding
	ledger         financeApp.LedgerPoster     // optional; nil disables auto-posting to the general ledger
}

// Option customises optional collaborators of the payroll use case.
type Option func(*payrollUseCase)

// CanteenSource totals an employee's non-cancelled canteen orders for a period (YYYY-MM).
type CanteenSource interface {
	CanteenAmountFor(ctx context.Context, nip, period string) (float64, error)
}

// AdvanceSource totals the cash advance installments due for an employee in a period (YYYY-MM).
type AdvanceSource interface {
	AdvanceAmountFor(ctx context.Context, nip, period string) (float64, error)
}

// WithInbox notifies employees when their payslip is approved.
func WithInbox(s inbox.Sender) Option { return func(uc *payrollUseCase) { uc.inbox = s } }

// WithAdvances withholds cash advance installments when calculating.
func WithAdvances(a AdvanceSource) Option { return func(uc *payrollUseCase) { uc.advances = a } }

// WithCanteen deducts the month's canteen orders when calculating.
func WithCanteen(c CanteenSource) Option { return func(uc *payrollUseCase) { uc.canteen = c } }

// WithOvertime fills OvertimePay from approved overtime records when calculating.
func WithOvertime(o OvertimeSource) Option {
	return func(uc *payrollUseCase) { uc.overtime = o }
}

// WithLeave enables withholding pay for approved unpaid leave.
func WithLeave(l leavedomain.LeaveRepository) Option {
	return func(uc *payrollUseCase) { uc.leaveRepo = l }
}

// WithLedger enables automatic journal posting when payroll is approved and paid.
func WithLedger(l financeApp.LedgerPoster) Option {
	return func(uc *payrollUseCase) { uc.ledger = l }
}

// NewPayrollUseCase wires the payroll repository together with the HRM and
// Cooperative repositories, so payroll entries can source BaseSalary from
// the employee's HRM record and DeductionCoop from the employee's active
// cooperative loans, and so paying an entry can pay down those loans (see
// applyCoopDeduction). hrmRepo/coopRepo may be nil, in which case payroll
// entries fall back to fully manual entry.
func NewPayrollUseCase(repo domain.PayrollRepository, hrmRepo hrmdomain.HRMRepository, coopRepo coopdomain.CooperativeRepository, opts ...Option) PayrollUseCase {
	uc := &payrollUseCase{repo: repo, hrmRepo: hrmRepo, coopRepo: coopRepo}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

func computeTakeHomePay(entry *domain.PayrollEntry) {
	entry.TakeHomePay = entry.BaseSalary + entry.Allowance + entry.OvertimePay -
		entry.DeductionAbsence - entry.DeductionLate - entry.DeductionTax - entry.DeductionCoop - entry.DeductionCanteen - entry.DeductionAdvance
}

// taxableGross is the income actually earned in the period: pay components less
// what was withheld for unpaid leave and lateness.
func taxableGross(e *domain.PayrollEntry) float64 {
	return e.BaseSalary + e.Allowance + e.OvertimePay - e.DeductionAbsence - e.DeductionLate
}

// applyAttendanceAdjustments derives the unpaid-leave withholding and late
// penalty for entries linked to an employee, from approved leave and attendance
// records in the entry's period, under the company's payroll policy.
func (uc *payrollUseCase) applyAttendanceAdjustments(ctx context.Context, entry *domain.PayrollEntry) error {
	entry.DeductionAbsence, entry.DeductionLate, entry.UnpaidDays, entry.LateCount = 0, 0, 0, 0
	if entry.EmployeeID == nil {
		return nil
	}
	policy, err := uc.repo.GetPolicy(ctx)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load payroll policy")
	}
	if uc.leaveRepo != nil && policy.DeductUnpaidLeave && len(entry.Period) == 7 {
		start, end := entry.Period+"-01", entry.Period+"-31"
		leaves, err := uc.leaveRepo.FindOverlapping(ctx, *entry.EmployeeID, start, end)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load leave records")
		}
		entry.UnpaidDays = UnpaidLeaveDays(leaves, entry.Period)
	}
	if uc.hrmRepo != nil && policy.LatePenaltyPerIncident > 0 && entry.NIP != "" {
		recs, err := uc.hrmRepo.ListAttendanceByNIPAndPeriod(ctx, entry.NIP, entry.Period)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load attendance records")
		}
		entry.LateCount = LateCount(recs)
	}
	if uc.overtime != nil && entry.NIP != "" {
		mins, err := uc.overtime.ApprovedOvertimeMinutes(ctx, entry.NIP, entry.Period)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load overtime records")
		}
		// Approved overtime records are the source of truth: with none, the
		// amount typed into the entry is left alone.
		if len(mins) > 0 {
			entry.OvertimePay = OvertimePayFor(entry.BaseSalary, mins)
		}
	}
	entry.DeductionCanteen = 0
	if uc.canteen != nil && entry.NIP != "" && len(entry.Period) == 7 {
		amt, err := uc.canteen.CanteenAmountFor(ctx, entry.NIP, entry.Period)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load canteen orders")
		}
		// Never withhold more than what is left after the other deductions' base pay.
		entry.DeductionCanteen = math.Min(round2(amt), entry.BaseSalary+entry.Allowance+entry.OvertimePay)
	}
	entry.DeductionAdvance = 0
	if uc.advances != nil && entry.NIP != "" && len(entry.Period) == 7 {
		amt, err := uc.advances.AdvanceAmountFor(ctx, entry.NIP, entry.Period)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load cash advances")
		}
		room := entry.BaseSalary + entry.Allowance + entry.OvertimePay - entry.DeductionCanteen
		entry.DeductionAdvance = math.Max(0, math.Min(round2(amt), room))
	}
	entry.DeductionAbsence, entry.DeductionLate = AttendanceAdjustments(entry.BaseSalary, *policy, entry.UnpaidDays, entry.LateCount)
	return nil
}

// applyAutoTax refreshes DeductionTax with an estimated PPh 21 for entries
// using the "auto" tax method; manual entries are left untouched. The PTKP
// status and NPWP flag come from the linked HRM employee (TK/0 with NPWP when
// the entry has no employee link).
func (uc *payrollUseCase) applyAutoTax(ctx context.Context, entry *domain.PayrollEntry) error {
	if entry.TaxMethod != "auto" {
		return nil
	}
	status, hasNPWP := DefaultPTKPStatus, true
	if entry.EmployeeID != nil && uc.hrmRepo != nil {
		emp, err := uc.hrmRepo.GetEmployeeByID(ctx, *entry.EmployeeID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to get employee for tax calculation")
		}
		if emp != nil {
			if s, ok := NormalizePTKPStatus(emp.PTKPStatus); ok {
				status = s
			}
			hasNPWP = !emp.NoNPWP
		}
	}
	entry.PTKPStatus = status
	entry.DeductionTax = EstimateMonthlyPPh21(taxableGross(entry), status, hasNPWP)
	return nil
}

// activeLoanDeductionTotal sums the MonthlyDeduction of an employee's active
// cooperative loans (see modules/cooperative). Used to auto-populate
// DeductionCoop when creating a payroll entry.
func (uc *payrollUseCase) activeLoanDeductionTotal(ctx context.Context, employeeID uuid.UUID) (float64, error) {
	if uc.coopRepo == nil {
		return 0, nil
	}
	loans, err := uc.coopRepo.ListActiveLoansByEmployeeID(ctx, employeeID)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, loan := range loans {
		total += loan.MonthlyDeduction
	}
	return total, nil
}

func (uc *payrollUseCase) CreateEntry(ctx context.Context, dto CreatePayrollEntryDTO) (*PayrollEntryResponseDTO, error) {
	period := strings.TrimSpace(dto.Period)
	if period == "" {
		return nil, apperrors.NewBadRequest("Period is required")
	}

	nip := strings.TrimSpace(dto.NIP)
	name := strings.TrimSpace(dto.EmployeeName)
	department := dto.Department
	baseSalary := dto.BaseSalary
	deductionCoop := dto.DeductionCoop

	if dto.EmployeeID != nil && uc.hrmRepo != nil {
		emp, err := uc.hrmRepo.GetEmployeeByID(ctx, *dto.EmployeeID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get employee")
		}
		if emp == nil {
			return nil, apperrors.NewNotFound("Employee not found")
		}

		nip = emp.NIP
		name = emp.Name
		department = emp.Department

		// BaseSalary is auto-filled from the employee's HRM record but can
		// still be overridden by passing an explicit non-zero value.
		if baseSalary == 0 {
			baseSalary = emp.BaseSalary
		}

		// DeductionCoop is auto-filled from the sum of the employee's
		// active cooperative loan MonthlyDeduction, also overridable.
		if deductionCoop == 0 {
			coopTotal, err := uc.activeLoanDeductionTotal(ctx, emp.ID)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to get cooperative loans")
			}
			deductionCoop = coopTotal
		}
	}

	if nip == "" || name == "" {
		return nil, apperrors.NewBadRequest("Period, NIP and Employee Name are required")
	}
	taxMethod := strings.ToLower(strings.TrimSpace(dto.TaxMethod))
	switch taxMethod {
	case "":
		taxMethod = "manual"
	case "manual", "auto":
	default:
		return nil, apperrors.NewBadRequest(`taxMethod must be "manual" or "auto"`)
	}
	if baseSalary <= 0 {
		return nil, apperrors.NewBadRequest("Base salary is required — update the employee's salary in HRM (e.g. after a Recruitment hire, which does not set one) or provide a value directly")
	}

	entry := &domain.PayrollEntry{
		Period:        period,
		EmployeeID:    dto.EmployeeID,
		NIP:           nip,
		EmployeeName:  name,
		Department:    department,
		BaseSalary:    baseSalary,
		Allowance:     dto.Allowance,
		OvertimePay:   dto.OvertimePay,
		DeductionTax:  dto.DeductionTax,
		DeductionCoop: deductionCoop,
		PaymentBank:   dto.PaymentBank,
		BankAccount:   dto.BankAccount,
		TaxMethod:     taxMethod,
		Status:        "draft",
	}
	if err := uc.applyAutoTax(ctx, entry); err != nil {
		return nil, err
	}
	computeTakeHomePay(entry)

	if err := uc.repo.CreateEntry(ctx, entry); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create payroll entry")
	}

	return ToPayrollEntryResponse(entry), nil
}

func (uc *payrollUseCase) ListEntries(ctx context.Context, query types.PaginationQuery, period string) ([]PayrollEntryResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	entries, total, err := uc.repo.ListEntries(ctx, query, period)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list payroll entries")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToPayrollEntryResponseList(entries), meta, nil
}

func (uc *payrollUseCase) CalculatePeriod(ctx context.Context, period string) ([]PayrollEntryResponseDTO, error) {
	period = strings.TrimSpace(period)
	if period == "" {
		return nil, apperrors.NewBadRequest("Period is required")
	}

	entries, err := uc.repo.ListEntriesByPeriodAndStatus(ctx, period, "draft")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load payroll entries for period")
	}

	updated := make([]PayrollEntryResponseDTO, 0, len(entries))
	for i := range entries {
		entry := entries[i]
		if err := uc.applyAttendanceAdjustments(ctx, &entry); err != nil {
			return nil, err
		}
		if err := uc.applyAutoTax(ctx, &entry); err != nil {
			return nil, err
		}
		computeTakeHomePay(&entry)
		entry.Status = "calculated"
		if err := uc.repo.UpdateEntry(ctx, &entry); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to update payroll entry")
		}
		updated = append(updated, *ToPayrollEntryResponse(&entry))
	}

	return updated, nil
}

func (uc *payrollUseCase) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*PayrollEntryResponseDTO, error) {
	entry, err := uc.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get payroll entry")
	}
	if entry == nil {
		return nil, apperrors.NewNotFound("Payroll entry not found")
	}

	currentIdx := statusIndex(entry.Status)
	newIdx := statusIndex(status)
	if newIdx == -1 {
		return nil, apperrors.NewBadRequest("Invalid status value")
	}
	if newIdx != currentIdx+1 {
		return nil, apperrors.NewBadRequest("Invalid status transition from " + entry.Status + " to " + status)
	}

	// Nobody approves or pays their own salary.
	if (status == "approved" || status == "paid") && entry.EmployeeID != nil && uc.hrmRepo != nil {
		emp, err := uc.hrmRepo.GetEmployeeByID(ctx, *entry.EmployeeID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get employee")
		}
		if emp != nil {
			if err := sod.ForbidSelfApproval(ctx, emp.Email); err != nil {
				return nil, err
			}
		}
	}

	entry.Status = status
	computeTakeHomePay(entry)
	if err := uc.repo.UpdateEntry(ctx, entry); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update payroll entry status")
	}

	uc.postStatusLedger(ctx, entry)
	uc.notifyStatus(ctx, entry)

	// On successful transition to "paid", pay down the employee's active
	// cooperative loans by the deducted amount (see applyCoopDeduction).
	if status == "paid" && entry.EmployeeID != nil {
		if err := uc.applyCoopDeduction(ctx, *entry.EmployeeID); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to apply cooperative loan deduction")
		}
	}

	return ToPayrollEntryResponse(entry), nil
}

// postStatusLedger records the accounting for a payroll entry reaching
// "approved" (accrual) or "paid" (cash out). A ledger failure must not roll
// back the payroll transition, so it is logged for later reconciliation.
func (uc *payrollUseCase) postStatusLedger(ctx context.Context, e *domain.PayrollEntry) {
	if uc.ledger == nil {
		return
	}
	var entry financeApp.LedgerEntry
	switch e.Status {
	case "approved":
		var ok bool
		if entry, ok = PayrollAccrualLedgerEntry(e); !ok {
			slog.Warn("payroll: negative take-home pay, skipping ledger posting", "entry", e.ID)
			return
		}
	case "paid":
		entry = PayrollPaymentLedgerEntry(e)
	default:
		return
	}
	if err := uc.ledger.PostEntry(ctx, entry.SourceDoc, entry.Memo, entry.Lines); err != nil {
		slog.Warn("payroll: failed to post ledger entry", "sourceDoc", entry.SourceDoc, "error", err)
	}
}

// applyCoopDeduction decrements each of the employee's active cooperative
// loans by that loan's MonthlyDeduction (capped at RemainingBalance), and
// marks a loan completed once its remaining balance reaches zero. This
// connects payroll processing back to the cooperative module (audit gap
// #2), instead of MonthlyDeduction being calculated but never applied.
func (uc *payrollUseCase) applyCoopDeduction(ctx context.Context, employeeID uuid.UUID) error {
	if uc.coopRepo == nil {
		return nil
	}
	loans, err := uc.coopRepo.ListActiveLoansByEmployeeID(ctx, employeeID)
	if err != nil {
		return err
	}
	for i := range loans {
		loan := loans[i]
		deduction := loan.MonthlyDeduction
		if deduction > loan.RemainingBalance {
			deduction = loan.RemainingBalance
		}
		loan.RemainingBalance -= deduction
		loan.MonthsPaid++
		if loan.RemainingBalance <= 0 {
			loan.RemainingBalance = 0
			loan.Status = "completed"
		}
		if err := uc.coopRepo.UpdateLoan(ctx, &loan); err != nil {
			return err
		}
	}
	return nil
}

func (uc *payrollUseCase) SeedInitialData(ctx context.Context) error {
	return nil
}

func (uc *payrollUseCase) GetPolicy(ctx context.Context) (*domain.PayrollPolicy, error) {
	p, err := uc.repo.GetPolicy(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load payroll policy")
	}
	return p, nil
}

func (uc *payrollUseCase) UpdatePolicy(ctx context.Context, dto UpdatePolicyDTO) (*domain.PayrollPolicy, error) {
	if dto.WorkDaysPerMonth < 1 || dto.WorkDaysPerMonth > 31 {
		return nil, apperrors.NewBadRequest("workDaysPerMonth must be between 1 and 31")
	}
	if dto.LatePenaltyPerIncident < 0 {
		return nil, apperrors.NewBadRequest("latePenaltyPerIncident cannot be negative")
	}
	p, err := uc.repo.GetPolicy(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load payroll policy")
	}
	p.WorkDaysPerMonth = dto.WorkDaysPerMonth
	p.LatePenaltyPerIncident = dto.LatePenaltyPerIncident
	p.DeductUnpaidLeave = dto.DeductUnpaidLeave
	if err := uc.repo.SavePolicy(ctx, p); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save payroll policy")
	}
	return p, nil
}

// notifyStatus tells the employee when their payslip becomes available or the
// salary has been paid. Best effort.
func (uc *payrollUseCase) notifyStatus(ctx context.Context, e *domain.PayrollEntry) {
	if uc.inbox == nil || uc.hrmRepo == nil || e.EmployeeID == nil || (e.Status != "approved" && e.Status != "paid") {
		return
	}
	emp, err := uc.hrmRepo.GetEmployeeByID(ctx, *e.EmployeeID)
	if err != nil || emp == nil {
		return
	}
	if e.Status == "approved" {
		uc.inbox(ctx, emp.Email, "Slip gaji tersedia", "Slip gaji periode "+e.Period+" sudah dapat dilihat.", "/hrm/my-payslips")
		return
	}
	uc.inbox(ctx, emp.Email, "Gaji dibayarkan", "Gaji periode "+e.Period+" telah dibayarkan.", "/hrm/my-payslips")
}
