package application

import (
	"fmt"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/pos/domain"
)

// Source-document key prefixes for POS postings.
const (
	SaleSourcePrefix   = "pos-sale:"
	COGSSourcePrefix   = "pos-cogs:"
	RefundSourcePrefix = "pos-refund:"
)

// PaymentAccount is the asset account a payment method settles into.
func PaymentAccount(method string, s domain.POSSettings) string {
	if method == "cash" || s.NonCashAccountCode == "" {
		return financeApp.AccountCash
	}
	return s.NonCashAccountCode
}

// SaleLedgerEntry is Dr cash-or-bank (total) / Cr Sales Revenue (net of tax) and
// Cr Sales Tax Payable. Revenue is derived as total - tax, so the entry always
// balances, including after rounding.
func SaleLedgerEntry(tx *domain.POSTransaction, s domain.POSSettings) financeApp.LedgerEntry {
	return financeApp.LedgerEntry{
		SourceDoc: SaleSourcePrefix + tx.ID.String(),
		Memo:      "POS sale " + tx.OrderNo,
		Lines: []financeApp.LedgerLine{
			{AccountCode: PaymentAccount(tx.PaymentMethod, s), Debit: tx.TotalAmount, Description: tx.OrderNo},
			{AccountCode: financeApp.AccountRevenue, Credit: tx.TotalAmount - tx.TaxAmount, Description: tx.OrderNo},
			{AccountCode: financeApp.AccountSalesTaxPayable, Credit: tx.TaxAmount, Description: tx.OrderNo},
		},
	}
}

// SaleCOGSEntry is Dr Cost of Goods Sold / Cr Inventory for the cost of what was
// sold. ok is false when no line carries a cost.
func SaleCOGSEntry(tx *domain.POSTransaction, lines []domain.POSTransactionLine) (financeApp.LedgerEntry, bool) {
	var cost float64
	for _, l := range lines {
		cost += l.UnitCost * l.Quantity
	}
	cost = round2(cost)
	if cost <= 0 {
		return financeApp.LedgerEntry{}, false
	}
	return financeApp.LedgerEntry{
		SourceDoc: COGSSourcePrefix + tx.ID.String(),
		Memo:      "POS cost of goods " + tx.OrderNo,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountCOGS, Debit: cost, Description: tx.OrderNo},
			{AccountCode: financeApp.AccountInventory, Credit: cost, Description: tx.OrderNo},
		},
	}, true
}

// RefundLedgerEntry reverses part (or all) of a sale: Dr Sales Revenue and Dr
// Sales Tax Payable / Cr cash-or-bank, plus Dr Inventory / Cr COGS for stock that
// went back on the shelf. n numbers the refund so each has its own key.
func RefundLedgerEntry(tx *domain.POSTransaction, rf *domain.POSRefund, s domain.POSSettings) financeApp.LedgerEntry {
	lines := []financeApp.LedgerLine{
		{AccountCode: financeApp.AccountRevenue, Debit: rf.Amount - rf.TaxAmount, Description: tx.OrderNo},
		{AccountCode: financeApp.AccountSalesTaxPayable, Debit: rf.TaxAmount, Description: tx.OrderNo},
		{AccountCode: PaymentAccount(tx.PaymentMethod, s), Credit: rf.Amount, Description: tx.OrderNo},
	}
	if rf.Restocked && rf.CostReturned > 0 {
		lines = append(lines,
			financeApp.LedgerLine{AccountCode: financeApp.AccountInventory, Debit: rf.CostReturned, Description: tx.OrderNo},
			financeApp.LedgerLine{AccountCode: financeApp.AccountCOGS, Credit: rf.CostReturned, Description: tx.OrderNo},
		)
	}
	return financeApp.LedgerEntry{
		SourceDoc: fmt.Sprintf("%s%s", RefundSourcePrefix, rf.ID.String()),
		Memo:      fmt.Sprintf("POS %s of %s", rf.Kind, tx.OrderNo),
		Lines:     lines,
	}
}
