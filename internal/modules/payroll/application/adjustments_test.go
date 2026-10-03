package application

import (
	"context"
	"testing"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	leavedomain "github.com/divinecoid/one-backend/internal/modules/leave/domain"
	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	"github.com/google/uuid"
)

func TestUnpaidLeaveDaysCountsWeekdaysInPeriodOnly(t *testing.T) {
	leaves := []leavedomain.LeaveRequest{
		// Mon 2026-09-28 .. Fri 2026-10-02: only 28,29,30 Sep fall in 2026-09.
		{Type: "Cuti Tidak Dibayar", Status: "approved", StartDate: "2026-09-28", EndDate: "2026-10-02"},
		// Paid leave and pending requests never count.
		{Type: "Annual Leave", Status: "approved", StartDate: "2026-09-01", EndDate: "2026-09-05"},
		{Type: "Unpaid Leave", Status: "pending", StartDate: "2026-09-07", EndDate: "2026-09-08"},
		// Weekend-only range.
		{Type: "Unpaid Leave", Status: "approved", StartDate: "2026-09-05", EndDate: "2026-09-06"},
	}
	if got := UnpaidLeaveDays(leaves, "2026-09"); got != 3 {
		t.Fatalf("unpaid days = %d, want 3", got)
	}
}

func TestAttendanceAdjustments(t *testing.T) {
	p := domain.PayrollPolicy{WorkDaysPerMonth: 20, LatePenaltyPerIncident: 25_000, DeductUnpaidLeave: true}
	abs, late := AttendanceAdjustments(10_000_000, p, 2, 3)
	if abs != 1_000_000 || late != 75_000 {
		t.Fatalf("got %v/%v", abs, late)
	}
	// Never withhold more than the salary.
	if abs, _ := AttendanceAdjustments(1_000_000, p, 40, 0); abs != 1_000_000 {
		t.Fatalf("absence must be capped at base salary, got %v", abs)
	}
	// Switched off.
	p.DeductUnpaidLeave = false
	p.LatePenaltyPerIncident = 0
	if abs, late := AttendanceAdjustments(10_000_000, p, 2, 3); abs != 0 || late != 0 {
		t.Fatalf("policy off must not deduct: %v/%v", abs, late)
	}
}

type fakeLeaveRepo struct {
	leavedomain.LeaveRepository
	leaves []leavedomain.LeaveRequest
}

func (f *fakeLeaveRepo) FindOverlapping(context.Context, uuid.UUID, string, string) ([]leavedomain.LeaveRequest, error) {
	return f.leaves, nil
}

type fakeAttendanceHRM struct {
	fakeHRM
	recs []hrmdomain.Attendance
}

func (f *fakeAttendanceHRM) ListAttendanceByNIPAndPeriod(context.Context, string, string) ([]hrmdomain.Attendance, error) {
	return f.recs, nil
}

type policyRepo struct {
	*fakePayrollRepo
	policy domain.PayrollPolicy
}

func (p *policyRepo) GetPolicy(context.Context) (*domain.PayrollPolicy, error) {
	c := p.policy
	return &c, nil
}

func TestCalculatePeriodAppliesUnpaidLeaveAndLatePenalty(t *testing.T) {
	emp := newEmployee("TK/0", false)
	repo := &policyRepo{fakePayrollRepo: &fakePayrollRepo{}, policy: domain.PayrollPolicy{WorkDaysPerMonth: 20, LatePenaltyPerIncident: 50_000, DeductUnpaidLeave: true}}
	hrm := &fakeAttendanceHRM{fakeHRM: fakeHRM{emp: emp}, recs: []hrmdomain.Attendance{{Status: "Late"}, {Status: "On Time"}, {Status: "Late"}}}
	leave := &fakeLeaveRepo{leaves: []leavedomain.LeaveRequest{{Type: "Unpaid Leave", Status: "approved", StartDate: "2026-09-07", EndDate: "2026-09-08"}}}
	uc := NewPayrollUseCase(repo, hrm, nil, WithLeave(leave))

	e := &domain.PayrollEntry{Period: "2026-09", EmployeeID: &emp.ID, NIP: emp.NIP, BaseSalary: 10_000_000, Status: "draft", TaxMethod: "manual"}
	e.ID = uuid.New()
	repo.entries = append(repo.entries, e)

	if _, err := uc.CalculatePeriod(context.Background(), "2026-09"); err != nil {
		t.Fatal(err)
	}
	got := repo.entries[0]
	if got.UnpaidDays != 2 || got.LateCount != 2 {
		t.Fatalf("unpaid=%d late=%d", got.UnpaidDays, got.LateCount)
	}
	if got.DeductionAbsence != 1_000_000 || got.DeductionLate != 100_000 {
		t.Fatalf("absence=%v late=%v", got.DeductionAbsence, got.DeductionLate)
	}
	if got.TakeHomePay != 10_000_000-1_000_000-100_000 {
		t.Fatalf("take-home = %v", got.TakeHomePay)
	}
	le, ok := PayrollAccrualLedgerEntry(got)
	if !ok {
		t.Fatal("ledger entry expected")
	}
	var dr, cr float64
	for _, l := range le.Lines {
		dr += l.Debit
		cr += l.Credit
	}
	if dr != cr {
		t.Fatalf("accrual journal unbalanced: dr %v cr %v", dr, cr)
	}
}

func TestOvertimePayFor(t *testing.T) {
	// base 17.3m -> hourly 100,000. 90 min = 1h*1.5 + 0.5h*2 = 250,000.
	if got := OvertimePayFor(17_300_000, []int{90}); got != 250_000 {
		t.Fatalf("90 min = %v", got)
	}
	// Two records are priced per record (1.5x applies to each record's first hour).
	if got := OvertimePayFor(17_300_000, []int{60, 60}); got != 300_000 {
		t.Fatalf("2x60 = %v", got)
	}
	if OvertimePayFor(17_300_000, nil) != 0 || OvertimePayFor(17_300_000, []int{0, -5}) != 0 {
		t.Fatal("no overtime must be zero")
	}
}

type fakeOT struct{ mins []int }

func (f fakeOT) ApprovedOvertimeMinutes(context.Context, string, string) ([]int, error) {
	return f.mins, nil
}

func TestCalculatePeriodPricesApprovedOvertimeAndKeepsManualWhenNone(t *testing.T) {
	emp := newEmployee("TK/0", false)
	repo := &policyRepo{fakePayrollRepo: &fakePayrollRepo{}, policy: domain.DefaultPayrollPolicy()}
	mk := func(ot OvertimeSource) *domain.PayrollEntry {
		uc := NewPayrollUseCase(repo, &fakeAttendanceHRM{fakeHRM: fakeHRM{emp: emp}}, nil, WithOvertime(ot))
		e := &domain.PayrollEntry{Period: "2026-09", EmployeeID: &emp.ID, NIP: emp.NIP, BaseSalary: 17_300_000, OvertimePay: 12345, Status: "draft", TaxMethod: "manual"}
		e.ID = uuid.New()
		repo.entries = []*domain.PayrollEntry{e}
		if _, err := uc.CalculatePeriod(context.Background(), "2026-09"); err != nil {
			t.Fatal(err)
		}
		return repo.entries[0]
	}
	if got := mk(fakeOT{mins: []int{90}}); got.OvertimePay != 250_000 {
		t.Fatalf("approved overtime should replace the typed amount, got %v", got.OvertimePay)
	}
	if got := mk(fakeOT{}); got.OvertimePay != 12345 {
		t.Fatalf("with no approved overtime the typed amount must stay, got %v", got.OvertimePay)
	}
}

type fakeCanteen struct{ amount float64 }

func (f fakeCanteen) CanteenAmountFor(context.Context, string, string) (float64, error) {
	return f.amount, nil
}

func TestCanteenIsDeductedAndJournalStaysBalanced(t *testing.T) {
	emp := newEmployee("TK/0", false)
	repo := &policyRepo{fakePayrollRepo: &fakePayrollRepo{}, policy: domain.DefaultPayrollPolicy()}
	run := func(amount float64) *domain.PayrollEntry {
		uc := NewPayrollUseCase(repo, &fakeAttendanceHRM{fakeHRM: fakeHRM{emp: emp}}, nil, WithCanteen(fakeCanteen{amount}))
		e := &domain.PayrollEntry{Period: "2026-09", EmployeeID: &emp.ID, NIP: emp.NIP, BaseSalary: 5_000_000, Allowance: 500_000, Status: "draft", TaxMethod: "manual"}
		e.ID = uuid.New()
		repo.entries = []*domain.PayrollEntry{e}
		if _, err := uc.CalculatePeriod(context.Background(), "2026-09"); err != nil {
			t.Fatal(err)
		}
		return repo.entries[0]
	}
	got := run(275_500.456)
	if got.DeductionCanteen != 275_500.46 || got.TakeHomePay != 5_500_000-275_500.46 {
		t.Fatalf("canteen deduction: %+v", got)
	}
	le, ok := PayrollAccrualLedgerEntry(got)
	if !ok {
		t.Fatal("entry expected")
	}
	var dr, cr float64
	for _, l := range le.Lines {
		dr += l.Debit
		cr += l.Credit
	}
	if dr != cr {
		t.Fatalf("accrual with canteen unbalanced: dr %v cr %v", dr, cr)
	}
	// A huge tab can never push pay below zero.
	if capped := run(99_000_000); capped.DeductionCanteen != 5_500_000 || capped.TakeHomePay != 0 {
		t.Fatalf("cap: %+v", capped)
	}
	// Recalculating never stacks deductions.
	if again := run(0); again.DeductionCanteen != 0 {
		t.Fatalf("no orders, no deduction: %+v", again)
	}
}
