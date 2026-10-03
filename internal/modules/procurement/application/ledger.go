package application

import (
	"fmt"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
)

// Source-document key prefixes for procurement postings.
const (
	PurchaseInvoiceSourcePrefix        = "purchase-invoice:"
	PurchaseInvoicePaymentSourcePrefix = "purchase-invoice-payment:"
	PurchaseDownPaymentSourcePrefix    = "purchase-dp:"
)

// PurchaseInvoiceLedgerEntry is Dr Inventory / Cr Accounts Payable. Inventory is
// carried at cost: the supplier's discount lowers it and additional cost
// (freight, handling) raises it. Rounding goes to the Rounding Adjustment
// account. Inventory is derived as total - rounding (= subtotal - discount +
// additional cost), which equals the plain total for legacy invoices, so the
// entry always balances.
func PurchaseInvoiceLedgerEntry(inv *domain.PurchaseInvoice) financeApp.LedgerEntry {
	lines := []financeApp.LedgerLine{
		{AccountCode: financeApp.AccountInventory, Debit: inv.TotalAmount - inv.RoundingAmount, Description: inv.SupplierName},
		{AccountCode: financeApp.AccountPayable, Credit: inv.TotalAmount, Description: inv.InvoiceNumber},
	}
	// Rounding up raises the payable above the inventory cost: expense the difference.
	if inv.RoundingAmount > 0 {
		lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountRounding, Debit: inv.RoundingAmount, Description: inv.InvoiceNumber})
	} else if inv.RoundingAmount < 0 {
		lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountRounding, Credit: -inv.RoundingAmount, Description: inv.InvoiceNumber})
	}
	return financeApp.LedgerEntry{
		SourceDoc: PurchaseInvoiceSourcePrefix + inv.ID.String(),
		Memo:      "Purchase invoice " + inv.InvoiceNumber,
		Lines:     lines,
	}
}

// PurchaseInvoicePaymentPrefix is the source-doc prefix shared by all payments of one invoice.
func PurchaseInvoicePaymentPrefix(inv *domain.PurchaseInvoice) string {
	return PurchaseInvoicePaymentSourcePrefix + inv.ID.String() + ":"
}

// PurchaseInvoicePaymentLedgerEntry is Dr Accounts Payable / Cr Cash. The
// cumulative paid amount in the key keeps it unique per payment while staying
// idempotent on retry.
func PurchaseInvoicePaymentLedgerEntry(inv *domain.PurchaseInvoice, amount float64) financeApp.LedgerEntry {
	return financeApp.LedgerEntry{
		SourceDoc: fmt.Sprintf("%s%.2f", PurchaseInvoicePaymentPrefix(inv), inv.PaidAmount),
		Memo:      "Payment for purchase invoice " + inv.InvoiceNumber,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountPayable, Debit: amount, Description: inv.InvoiceNumber},
			{AccountCode: financeApp.AccountCash, Credit: amount, Description: inv.SupplierName},
		},
	}
}

// DownPaymentLedgerEntry is Dr Supplier Advances / Cr Cash.
func DownPaymentLedgerEntry(dp *domain.PurchaseDownPayment) financeApp.LedgerEntry {
	return financeApp.LedgerEntry{
		SourceDoc: PurchaseDownPaymentSourcePrefix + dp.ID.String(),
		Memo:      "Purchase down payment " + dp.DPNumber,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountSupplierAdvance, Debit: dp.Amount, Description: dp.SupplierName},
			{AccountCode: financeApp.AccountCash, Credit: dp.Amount, Description: dp.DPNumber},
		},
	}
}

// PurchaseInvoiceSubledgerDoc mirrors a purchase invoice into finance's payables.
func PurchaseInvoiceSubledgerDoc(inv *domain.PurchaseInvoice) financeApp.SubledgerDoc {
	supplierID := inv.SupplierID
	return financeApp.SubledgerDoc{
		SourceDoc: PurchaseInvoiceSourcePrefix + inv.ID.String(),
		PartyID:   &supplierID,
		PartyName: inv.SupplierName,
		DocNo:     inv.InvoiceNumber,
		IssueDate: inv.InvoiceDate,
		DueDate:   inv.DueDate,
		Total:     inv.TotalAmount,
		Paid:      inv.PaidAmount,
	}
}
