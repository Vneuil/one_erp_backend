package application

import (
	"context"
	"sort"
	"strings"
	"time"

	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	procApp "github.com/divinecoid/one-backend/internal/modules/procurement/application"
	salesApp "github.com/divinecoid/one-backend/internal/modules/sales/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// Party ledgers: Kartu Piutang / Kartu Hutang and Saldo Akhir Piutang / Hutang.
//
// Invoices come from the AR/AP sub-ledger. Dated payments come from the posted
// journal entries of invoice payments. Payments that the sub-ledger knows about
// but the journal has no dated entry for (manual receivables/payables, or a
// posting that failed) are shown as one approximate payment.

type partyDoc struct {
	SourceDoc string
	DocNo     string
	Party     string
	IssueDate string
	DueDate   string
	Total     float64
	Paid      float64
	Created   time.Time
	Updated   time.Time
}

type partyPayment struct {
	Date      string
	SourceDoc string
	Amount    float64
}

// partyKind says how one side (AR or AP) names its source documents.
type partyKind struct {
	invoicePrefix string
	paymentPrefix string
}

type ledgerEntry struct {
	Date        string
	Type        string // invoice | payment
	Reference   string
	Description string
	DueDate     string
	Charge      float64
	Payment     float64
	Approximate bool
	docKey      string
}

func docDate(d partyDoc) string {
	if validDate(d.IssueDate) {
		return d.IssueDate
	}
	return d.Created.In(wib).Format("2006-01-02")
}

// partyEntries returns each party's invoice and payment entries in date order,
// keyed by the lower-cased party name.
func partyEntries(docs []partyDoc, pays []partyPayment, kind partyKind) map[string][]ledgerEntry {
	byKey := map[string]partyDoc{}
	for _, d := range docs {
		if d.SourceDoc != "" {
			byKey[d.SourceDoc] = d
		}
	}
	journalPaid := map[string]float64{}
	out := map[string][]ledgerEntry{}
	pk := func(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

	for _, p := range pays {
		id := strings.TrimPrefix(p.SourceDoc, kind.paymentPrefix)
		if i := strings.Index(id, ":"); i >= 0 {
			id = id[:i]
		}
		doc, ok := byKey[kind.invoicePrefix+id]
		if !ok {
			continue
		}
		journalPaid[doc.SourceDoc] += p.Amount
		out[pk(doc.Party)] = append(out[pk(doc.Party)], ledgerEntry{Date: p.Date, Type: "payment", Reference: doc.DocNo,
			Description: "Pembayaran " + doc.DocNo, Payment: p.Amount, docKey: doc.SourceDoc})
	}
	for _, d := range docs {
		issue := docDate(d)
		key := pk(d.Party)
		out[key] = append(out[key], ledgerEntry{Date: issue, Type: "invoice", Reference: d.DocNo, Description: "Invoice " + d.DocNo,
			DueDate: d.DueDate, Charge: d.Total, docKey: d.SourceDoc})
		if rest := round2(d.Paid - journalPaid[d.SourceDoc]); rest > 0.005 {
			date := d.Updated.In(wib).Format("2006-01-02")
			if date < issue {
				date = issue
			}
			out[key] = append(out[key], ledgerEntry{Date: date, Type: "payment", Reference: d.DocNo,
				Description: "Pembayaran " + d.DocNo + " (tanggal tidak tercatat)", Payment: rest, Approximate: true, docKey: d.SourceDoc})
		}
	}
	for k := range out {
		es := out[k]
		sort.SliceStable(es, func(i, j int) bool {
			if es[i].Date != es[j].Date {
				return es[i].Date < es[j].Date
			}
			return es[i].Type == "invoice" && es[j].Type != "invoice"
		})
	}
	return out
}

type PartyCardEntry struct {
	Date        string  `json:"date"`
	Type        string  `json:"type"`
	Reference   string  `json:"reference"`
	Description string  `json:"description"`
	DueDate     string  `json:"dueDate,omitempty"`
	Charge      float64 `json:"charge"`
	Payment     float64 `json:"payment"`
	Balance     float64 `json:"balance"`
	Approximate bool    `json:"approximate,omitempty"`
}

type PartyCardDTO struct {
	Party         string           `json:"party"`
	From          string           `json:"from"`
	To            string           `json:"to"`
	Opening       float64          `json:"openingBalance"`
	Entries       []PartyCardEntry `json:"entries"`
	TotalCharges  float64          `json:"totalCharges"`
	TotalPayments float64          `json:"totalPayments"`
	Closing       float64          `json:"closingBalance"`
}

// buildPartyCard assembles one party's card. Charge increases the balance
// (an invoice), Payment decreases it.
func buildPartyCard(party string, entries []ledgerEntry, from, to string) *PartyCardDTO {
	out := &PartyCardDTO{Party: party, From: from, To: to, Entries: []PartyCardEntry{}}
	balance := 0.0
	for _, e := range entries {
		if e.Date > to {
			continue
		}
		if e.Date < from {
			balance += e.Charge - e.Payment
			continue
		}
		if len(out.Entries) == 0 {
			out.Opening = round2(balance)
		}
		balance += e.Charge - e.Payment
		out.TotalCharges += e.Charge
		out.TotalPayments += e.Payment
		out.Entries = append(out.Entries, PartyCardEntry{Date: e.Date, Type: e.Type, Reference: e.Reference, Description: e.Description,
			DueDate: e.DueDate, Charge: e.Charge, Payment: e.Payment, Balance: round2(balance), Approximate: e.Approximate})
	}
	if len(out.Entries) == 0 {
		out.Opening = round2(balance)
	}
	out.TotalCharges, out.TotalPayments, out.Closing = round2(out.TotalCharges), round2(out.TotalPayments), round2(balance)
	return out
}

type PartyBalanceRow struct {
	Party         string  `json:"party"`
	Invoiced      float64 `json:"invoiced"`
	Paid          float64 `json:"paid"`
	Balance       float64 `json:"balance"`
	OpenDocuments int     `json:"openDocuments"`
	OldestDue     string  `json:"oldestDue,omitempty"`
}

type PartyBalancesDTO struct {
	AsOf          string            `json:"asOf"`
	Rows          []PartyBalanceRow `json:"rows"`
	TotalInvoiced float64           `json:"totalInvoiced"`
	TotalPaid     float64           `json:"totalPaid"`
	TotalBalance  float64           `json:"totalBalance"`
}

// buildPartyBalances is Saldo Akhir: per party, everything invoiced and paid up
// to asOf and what is still open. Parties with a zero balance are left out
// unless includeZero is set.
func buildPartyBalances(names map[string]string, entries map[string][]ledgerEntry, asOf string, includeZero bool) *PartyBalancesDTO {
	out := &PartyBalancesDTO{AsOf: asOf, Rows: []PartyBalanceRow{}}
	for key, es := range entries {
		row := PartyBalanceRow{Party: names[key]}
		open := map[string]float64{}
		due := map[string]string{}
		for _, e := range es {
			if e.Date > asOf {
				continue
			}
			row.Invoiced += e.Charge
			row.Paid += e.Payment
			open[e.docKey] += e.Charge - e.Payment
			if e.Type == "invoice" {
				due[e.docKey] = e.DueDate
			}
		}
		for dk, rem := range open {
			if rem > 0.005 {
				row.OpenDocuments++
				if d := due[dk]; d != "" && (row.OldestDue == "" || d < row.OldestDue) {
					row.OldestDue = d
				}
			}
		}
		row.Invoiced, row.Paid = round2(row.Invoiced), round2(row.Paid)
		row.Balance = round2(row.Invoiced - row.Paid)
		if row.Balance == 0 && !includeZero {
			continue
		}
		out.Rows = append(out.Rows, row)
		out.TotalInvoiced += row.Invoiced
		out.TotalPaid += row.Paid
	}
	sort.Slice(out.Rows, func(i, j int) bool {
		if out.Rows[i].Balance != out.Rows[j].Balance {
			return out.Rows[i].Balance > out.Rows[j].Balance
		}
		return out.Rows[i].Party < out.Rows[j].Party
	})
	out.TotalInvoiced, out.TotalPaid = round2(out.TotalInvoiced), round2(out.TotalPaid)
	out.TotalBalance = round2(out.TotalInvoiced - out.TotalPaid)
	return out
}

// loadParty gathers one side's documents, dated payments and display names.
func (q *ReportQuery) loadParty(ctx context.Context, receivable bool, to string) ([]partyDoc, []partyPayment, partyKind, map[string]string, error) {
	repo := financeInfra.NewFinanceRepository(q.db)
	names := map[string]string{}
	var docs []partyDoc
	var kind partyKind
	accountCode := financeApp.AccountReceivble
	if receivable {
		kind = partyKind{invoicePrefix: salesApp.SalesInvoiceSourcePrefix, paymentPrefix: salesApp.SalesInvoicePaymentSourcePrefix}
		items, err := repo.ListAllReceivables(ctx)
		if err != nil {
			return nil, nil, kind, nil, apperrors.NewInternal(err, "Failed to load receivables")
		}
		for _, r := range items {
			docs = append(docs, partyDoc{SourceDoc: r.SourceDoc, DocNo: r.InvoiceNo, Party: r.CustomerName, IssueDate: r.IssueDate, DueDate: r.DueDate,
				Total: r.TotalInvoice, Paid: r.PaidAmount, Created: r.CreatedAt, Updated: r.UpdatedAt})
		}
	} else {
		accountCode = financeApp.AccountPayable
		kind = partyKind{invoicePrefix: procApp.PurchaseInvoiceSourcePrefix, paymentPrefix: procApp.PurchaseInvoicePaymentSourcePrefix}
		items, err := repo.ListAllPayables(ctx)
		if err != nil {
			return nil, nil, kind, nil, apperrors.NewInternal(err, "Failed to load payables")
		}
		for _, p := range items {
			docs = append(docs, partyDoc{SourceDoc: p.SourceDoc, DocNo: p.InvoiceNo, Party: p.VendorName, IssueDate: p.IssueDate, DueDate: p.DueDate,
				Total: p.TotalInvoice, Paid: p.PaidAmount, Created: p.CreatedAt, Updated: p.UpdatedAt})
		}
	}
	for _, d := range docs {
		k := strings.ToLower(strings.TrimSpace(d.Party))
		if _, ok := names[k]; !ok {
			names[k] = strings.TrimSpace(d.Party)
		}
	}

	accounts, err := repo.ListAllAccounts(ctx)
	if err != nil {
		return nil, nil, kind, nil, apperrors.NewInternal(err, "Failed to list accounts")
	}
	var accountID *uuid.UUID
	for _, a := range accounts {
		if a.Code == accountCode {
			id := a.ID
			accountID = &id
		}
	}
	var pays []partyPayment
	if accountID != nil {
		lines, err := repo.ListPostedLines(ctx, "", to, []uuid.UUID{*accountID})
		if err != nil {
			return nil, nil, kind, nil, apperrors.NewInternal(err, "Failed to load payment postings")
		}
		for _, l := range lines {
			if !strings.HasPrefix(l.SourceDoc, kind.paymentPrefix) {
				continue
			}
			amount := l.Credit // a receipt credits Accounts Receivable
			if !receivable {
				amount = l.Debit // a payment debits Accounts Payable
			}
			if amount > 0 {
				pays = append(pays, partyPayment{Date: l.Date, SourceDoc: l.SourceDoc, Amount: amount})
			}
		}
	}
	return docs, pays, kind, names, nil
}

func (q *ReportQuery) PartyBalances(ctx context.Context, receivable, includeZero bool, asOf string) (*PartyBalancesDTO, error) {
	if asOf == "" {
		asOf = today()
	}
	if !validDate(asOf) {
		return nil, apperrors.NewBadRequest("asOf must be YYYY-MM-DD")
	}
	docs, pays, kind, names, err := q.loadParty(ctx, receivable, asOf)
	if err != nil {
		return nil, err
	}
	return buildPartyBalances(names, partyEntries(docs, pays, kind), asOf, includeZero), nil
}

func (q *ReportQuery) PartyCard(ctx context.Context, receivable bool, party, from, to string) (*PartyCardDTO, error) {
	if strings.TrimSpace(party) == "" {
		return nil, apperrors.NewBadRequest("party is required")
	}
	from, to, err := resolveRange(from, to)
	if err != nil {
		return nil, err
	}
	docs, pays, kind, names, err := q.loadParty(ctx, receivable, to)
	if err != nil {
		return nil, err
	}
	key := strings.ToLower(strings.TrimSpace(party))
	name, ok := names[key]
	if !ok {
		return nil, apperrors.NewNotFound("No invoices found for " + strings.TrimSpace(party))
	}
	return buildPartyCard(name, partyEntries(docs, pays, kind)[key], from, to), nil
}
