package application

import (
	"fmt"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
)

// Source-document key prefixes for sales postings.
const (
	SalesInvoiceSourcePrefix        = "sales-invoice:"
	SalesInvoicePaymentSourcePrefix = "sales-invoice-payment:"
)

// InvoiceLedgerEntry is Dr Accounts Receivable (final total) and Dr Sales
// Discounts / Cr Sales Revenue (gross amount) and Cr Additional Charges Income,
// with any rounding difference to the Rounding Adjustment account. Gross
// revenue is derived as total + discount - additional - rounding, which equals
// the subtotal for adjusted invoices and the plain total for legacy ones, so
// the entry always balances.
func InvoiceLedgerEntry(inv *domain.Invoice) financeApp.LedgerEntry {
	gross := inv.TotalAmount + inv.DiscountAmount - inv.AdditionalCost - inv.RoundingAmount
	lines := []financeApp.LedgerLine{
		{AccountCode: financeApp.AccountReceivble, Debit: inv.TotalAmount, Description: inv.CustomerName},
		{AccountCode: financeApp.AccountSalesDiscount, Debit: inv.DiscountAmount, Description: inv.InvoiceNumber},
		{AccountCode: financeApp.AccountRevenue, Credit: gross, Description: inv.InvoiceNumber},
		{AccountCode: financeApp.AccountAdditionalCharges, Credit: inv.AdditionalCost, Description: inv.InvoiceNumber},
	}
	// Rounding up means the customer pays more than gross-discount+charges: credit the difference.
	if inv.RoundingAmount > 0 {
		lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountRounding, Credit: inv.RoundingAmount, Description: inv.InvoiceNumber})
	} else if inv.RoundingAmount < 0 {
		lines = append(lines, financeApp.LedgerLine{AccountCode: financeApp.AccountRounding, Debit: -inv.RoundingAmount, Description: inv.InvoiceNumber})
	}
	return financeApp.LedgerEntry{
		SourceDoc: SalesInvoiceSourcePrefix + inv.ID.String(),
		Memo:      "Sales invoice " + inv.InvoiceNumber,
		Lines:     lines,
	}
}

// InvoicePaymentPrefix is the source-doc prefix shared by all payments of one invoice.
func InvoicePaymentPrefix(inv *domain.Invoice) string {
	return SalesInvoicePaymentSourcePrefix + inv.ID.String() + ":"
}

// InvoicePaymentLedgerEntry is Dr Cash / Cr Accounts Receivable. The
// cumulative paid amount in the key keeps it unique per payment while staying
// idempotent on retry.
func InvoicePaymentLedgerEntry(inv *domain.Invoice, amount float64) financeApp.LedgerEntry {
	return financeApp.LedgerEntry{
		SourceDoc: fmt.Sprintf("%s%.2f", InvoicePaymentPrefix(inv), inv.PaidAmount),
		Memo:      "Payment for invoice " + inv.InvoiceNumber,
		Lines: []financeApp.LedgerLine{
			{AccountCode: financeApp.AccountCash, Debit: amount, Description: inv.InvoiceNumber},
			{AccountCode: financeApp.AccountReceivble, Credit: amount, Description: inv.CustomerName},
		},
	}
}

// InvoiceSubledgerDoc mirrors a sales invoice into finance's receivables.
func InvoiceSubledgerDoc(inv *domain.Invoice) financeApp.SubledgerDoc {
	return financeApp.SubledgerDoc{
		SourceDoc: SalesInvoiceSourcePrefix + inv.ID.String(),
		PartyName: inv.CustomerName,
		DocNo:     inv.InvoiceNumber,
		IssueDate: inv.InvoiceDate,
		DueDate:   inv.DueDate,
		Total:     inv.TotalAmount,
		Paid:      inv.PaidAmount,
	}
}
