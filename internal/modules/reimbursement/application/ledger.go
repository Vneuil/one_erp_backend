package application

import (
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
)

// ClaimApprovalLedgerEntry is Dr Operating Expenses / Cr Reimbursements Payable.
func ClaimApprovalLedgerEntry(c *domain.ReimbursementClaim) financeApp.LedgerEntry {
	return financeApp.LedgerEntry{
		SourceDoc: "reimbursement-approval:" + c.ID.String(),
		Memo:      "Reimbursement " + c.ClaimNo,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountExpense, Debit: c.Amount, Description: c.EmployeeName + " - " + c.Category},
			{AccountCode: financeApp.AccountReimbursementPayable, Credit: c.Amount, Description: c.ClaimNo},
		},
	}
}

// ClaimPaymentLedgerEntry is Dr Reimbursements Payable / Cr Cash.
func ClaimPaymentLedgerEntry(c *domain.ReimbursementClaim) financeApp.LedgerEntry {
	return financeApp.LedgerEntry{
		SourceDoc: "reimbursement-payment:" + c.ID.String(),
		Memo:      "Reimbursement payment " + c.ClaimNo,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountReimbursementPayable, Debit: c.Amount, Description: c.ClaimNo},
			{AccountCode: financeApp.AccountCash, Credit: c.Amount, Description: c.EmployeeName},
		},
	}
}
