package application

import (
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
)

func TestBuildPayslip(t *testing.T) {
	e := &domain.PayrollEntry{Period: "2026-09", EmployeeName: "Sari", NIP: "S1", BaseSalary: 17_300_000, Allowance: 500_000,
		OvertimePay: 250_000, DeductionAbsence: 1_572_727, DeductionLate: 50_000, DeductionTax: 100_000, DeductionCanteen: 50_000, DeductionAdvance: 400_000,
		UnpaidDays: 2, LateCount: 1, BankAccount: "1234567890", PaymentBank: "BCA"}
	e.TakeHomePay = e.BaseSalary + e.Allowance + e.OvertimePay - e.DeductionAbsence - e.DeductionLate - e.DeductionTax - e.DeductionCanteen - e.DeductionAdvance
	p := BuildPayslip(e, []string{"2026-09-03", "2026-09-04"}, []int{90, 30})

	if p.Gross != 18_050_000 {
		t.Errorf("gross = %v", p.Gross)
	}
	// Earnings less deductions is exactly the take-home pay on the slip.
	if p.Gross-p.TotalDeduction != p.TakeHomePay {
		t.Errorf("gross %v - deductions %v != take-home %v", p.Gross, p.TotalDeduction, p.TakeHomePay)
	}
	if p.AccountMasked != "••••••7890" {
		t.Errorf("account = %q", p.AccountMasked)
	}
	if len(p.Overtime) != 2 || p.OvertimeHours != 2 {
		t.Errorf("overtime = %+v hours=%v", p.Overtime, p.OvertimeHours)
	}
	// 90 min = 1h at 1.5x + 0.5h at 2x of 100,000/h; 30 min = 0.5h at 1.5x.
	if p.Overtime[0].Amount != 250_000 || p.Overtime[1].Amount != 75_000 {
		t.Errorf("overtime amounts = %+v", p.Overtime)
	}
	var sawAbsence bool
	for _, d := range p.Deductions {
		if d.Label == "Potongan cuti tanpa gaji" && d.Note == "2 hari cuti tanpa gaji" {
			sawAbsence = true
		}
		if d.Amount == 0 {
			t.Errorf("zero deduction listed: %+v", d)
		}
	}
	if !sawAbsence {
		t.Errorf("absence note missing: %+v", p.Deductions)
	}
}
