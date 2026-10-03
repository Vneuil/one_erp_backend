package application

import (
	"fmt"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
)

// PayrollAccrualLedgerEntry is the entry posted when a payroll entry is
// approved: Dr Salary Expense (gross) / Cr Salaries, Tax and Cooperative
// Payable. ok is false when the entry cannot be posted (negative take-home).
func PayrollAccrualLedgerEntry(e *domain.PayrollEntry) (financeApp.LedgerEntry, bool) {
	if e.TakeHomePay < 0 {
		return financeApp.LedgerEntry{}, false
	}
	gross := e.BaseSalary + e.Allowance + e.OvertimePay - e.DeductionAbsence - e.DeductionLate
	label := fmt.Sprintf("%s %s", e.EmployeeName, e.Period)
	return financeApp.LedgerEntry{
		SourceDoc: "payroll-accrual:" + e.ID.String(),
		Memo:      "Payroll " + label,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountSalaryExpense, Debit: gross, Description: label},
			{AccountCode: financeApp.AccountSalaryPayable, Credit: e.TakeHomePay, Description: label},
			{AccountCode: financeApp.AccountTaxPayable, Credit: e.DeductionTax, Description: label},
			{AccountCode: financeApp.AccountCoopPayable, Credit: e.DeductionCoop, Description: label},
			// The installment repays the cash advance booked when it was approved.
			{AccountCode: financeApp.AccountEmployeeAdvance, Credit: e.DeductionAdvance, Description: label + " cash advance"},
			// Meals the employee paid for through salary are canteen income.
			{AccountCode: financeApp.AccountOtherIncome, Credit: e.DeductionCanteen, Description: label + " canteen"},
		},
	}, true
}

// PayrollPaymentLedgerEntry is Dr Salaries Payable / Cr Cash, posted when paid.
func PayrollPaymentLedgerEntry(e *domain.PayrollEntry) financeApp.LedgerEntry {
	label := fmt.Sprintf("%s %s", e.EmployeeName, e.Period)
	return financeApp.LedgerEntry{
		SourceDoc: "payroll-payment:" + e.ID.String(),
		Memo:      "Payroll payment " + label,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountSalaryPayable, Debit: e.TakeHomePay, Description: label},
			{AccountCode: financeApp.AccountCash, Credit: e.TakeHomePay, Description: label},
		},
	}
}
