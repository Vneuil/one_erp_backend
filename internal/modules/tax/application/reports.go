package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// invoiceTaxBase is the DPP of a sales invoice, falling back for invoices that
// predate the PPN columns.
func invoiceTaxBase(i domain.SalesInvoiceRef) float64 {
	switch {
	case i.TaxBase > 0:
		return i.TaxBase
	case i.Subtotal > 0:
		return round2(i.Subtotal - i.Discount + i.AdditionalCost)
	default:
		return round2(i.Total - i.Rounding)
	}
}

type SalesBookRow struct {
	InvoiceID      uuid.UUID  `json:"invoiceId"`
	Date           string     `json:"date"`
	InvoiceNumber  string     `json:"invoiceNumber"`
	CustomerName   string     `json:"customerName"`
	CustomerNPWP   string     `json:"customerNpwp"`
	CustomerNIK    string     `json:"customerNik"`
	Gross          float64    `json:"gross"`
	Discount       float64    `json:"discount"`
	TaxBase        float64    `json:"taxBase"`
	VATAmount      float64    `json:"vatAmount"`
	Total          float64    `json:"total"`
	TaxInvoiceID   *uuid.UUID `json:"taxInvoiceId,omitempty"`
	TaxInvoiceNo   string     `json:"taxInvoiceNumber,omitempty"`
	TaxNumber      string     `json:"taxNumber,omitempty"`
	TaxInvoiceStat string     `json:"taxInvoiceStatus,omitempty"`
}

type SalesBookDTO struct {
	From  string         `json:"from"`
	To    string         `json:"to"`
	Rows  []SalesBookRow `json:"rows"`
	Count int            `json:"count"`
	// Totals over all rows.
	Gross     float64 `json:"gross"`
	Discount  float64 `json:"discount"`
	TaxBase   float64 `json:"taxBase"`
	VATAmount float64 `json:"vatAmount"`
	Total     float64 `json:"total"`
	// PendingFaktur counts invoices that carry PPN but have no issued Faktur Pajak yet.
	PendingFaktur int `json:"pendingFaktur"`
}

func monthStart(now time.Time) string {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
}

// SalesBook (Laporan Buku Penjualan) lists every sales invoice in the range
// with its DPP, PPN and Faktur Pajak. Defaults to the current month.
func (uc *taxUseCase) SalesBook(ctx context.Context, from, to string) (*SalesBookDTO, error) {
	if err := checkRange(from, to); err != nil {
		return nil, err
	}
	if from == "" && to == "" {
		from, to = monthStart(uc.now()), uc.today()
	}
	if from == "" {
		from = "0000-01-01"
	}
	if to == "" {
		to = "9999-12-31"
	}
	invoices, err := uc.src.SalesInvoicesBetween(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load sales invoices")
	}
	ids := make([]uuid.UUID, len(invoices))
	for i, inv := range invoices {
		ids[i] = inv.ID
	}
	fakturs, err := uc.repo.ListBySources(ctx, domain.SourceSalesInvoice, ids)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load Faktur Pajak")
	}
	bySource := map[uuid.UUID]domain.TaxInvoice{}
	for _, f := range fakturs {
		if cur, ok := bySource[f.SourceID]; !ok || (cur.Status != domain.StatusIssued && f.Status == domain.StatusIssued) {
			bySource[f.SourceID] = f
		}
	}
	customers, err := uc.src.CustomersByName(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load customers")
	}

	out := &SalesBookDTO{From: from, To: to, Rows: []SalesBookRow{}}
	for _, inv := range invoices {
		base := invoiceTaxBase(inv)
		row := SalesBookRow{InvoiceID: inv.ID, Date: inv.Date, InvoiceNumber: inv.Number, CustomerName: inv.CustomerName,
			Gross: round2(base + inv.Discount - inv.AdditionalCost), Discount: inv.Discount, TaxBase: base, VATAmount: inv.VATAmt, Total: inv.Total}
		if c, ok := customers[strings.ToLower(strings.TrimSpace(inv.CustomerName))]; ok {
			row.CustomerNPWP, row.CustomerNIK = c.NPWP, c.NIK
		}
		if f, ok := bySource[inv.ID]; ok {
			id := f.ID
			row.TaxInvoiceID, row.TaxInvoiceNo, row.TaxNumber, row.TaxInvoiceStat = &id, f.Number, f.TaxNumber, f.Status
			if f.CounterpartyNPWP != "" {
				row.CustomerNPWP = f.CounterpartyNPWP
			}
		}
		if inv.VATAmt > 0 && row.TaxInvoiceStat != domain.StatusIssued {
			out.PendingFaktur++
		}
		out.Rows = append(out.Rows, row)
		out.Gross, out.Discount, out.TaxBase, out.VATAmount, out.Total = out.Gross+row.Gross, out.Discount+row.Discount, out.TaxBase+row.TaxBase, out.VATAmount+row.VATAmount, out.Total+row.Total
	}
	out.Count = len(out.Rows)
	out.Gross, out.Discount, out.TaxBase, out.VATAmount, out.Total = round2(out.Gross), round2(out.Discount), round2(out.TaxBase), round2(out.VATAmount), round2(out.Total)
	return out, nil
}

type VATPeriodRow struct {
	Period    string  `json:"period"`
	OutputVAT float64 `json:"outputVat"`
	InputVAT  float64 `json:"inputVat"`
	// Net is output minus input: positive = PPN kurang bayar, negative = lebih bayar.
	Net float64 `json:"net"`
	// OutputWithoutFaktur / InputWithoutFaktur are PPN booked on invoices that
	// have no issued Faktur Pajak yet, so they are not in the figures above.
	OutputWithoutFaktur float64 `json:"outputWithoutFaktur"`
	InputWithoutFaktur  float64 `json:"inputWithoutFaktur"`
}

type VATSummaryDTO struct {
	From    string         `json:"from"`
	To      string         `json:"to"`
	Periods []VATPeriodRow `json:"periods"`
	Total   VATPeriodRow   `json:"total"`
}

// VATSummary (ringkasan SPT Masa PPN) nets PPN Keluaran against creditable PPN
// Masukan per tax period, using issued Faktur Pajak. Defaults to January of
// the current year through today.
func (uc *taxUseCase) VATSummary(ctx context.Context, from, to string) (*VATSummaryDTO, error) {
	if err := checkRange(from, to); err != nil {
		return nil, err
	}
	now := uc.now()
	if from == "" {
		from = time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	}
	if to == "" {
		to = uc.today()
	}
	if from > to {
		return nil, apperrors.NewBadRequest("from cannot be after to")
	}
	fromP, toP := from[:7], to[:7]

	issued, err := uc.repo.ListTaxInvoices(ctx, domain.InvoiceFilter{Status: domain.StatusIssued})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load Faktur Pajak")
	}
	rows := map[string]*VATPeriodRow{}
	row := func(p string) *VATPeriodRow {
		if rows[p] == nil {
			rows[p] = &VATPeriodRow{Period: p}
		}
		return rows[p]
	}
	issuedSource := map[uuid.UUID]bool{}
	for _, f := range issued {
		issuedSource[f.SourceID] = true
		if f.Period < fromP || f.Period > toP {
			continue
		}
		if f.Direction == domain.DirectionOutput {
			row(f.Period).OutputVAT += f.VATAmount
		} else {
			row(f.Period).InputVAT += f.VATAmount
		}
	}

	sales, err := uc.src.SalesInvoicesBetween(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load sales invoices")
	}
	for _, inv := range sales {
		if inv.VATAmt > 0 && !issuedSource[inv.ID] && len(inv.Date) >= 7 {
			row(inv.Date[:7]).OutputWithoutFaktur += inv.VATAmt
		}
	}
	purchases, err := uc.src.PurchaseInvoicesBetween(ctx, from, to)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load purchase invoices")
	}
	for _, inv := range purchases {
		if inv.VATAmt > 0 && inv.VATCreditable && !issuedSource[inv.ID] && len(inv.Date) >= 7 {
			row(inv.Date[:7]).InputWithoutFaktur += inv.VATAmt
		}
	}

	out := &VATSummaryDTO{From: from, To: to, Periods: []VATPeriodRow{}, Total: VATPeriodRow{Period: "total"}}
	for _, r := range rows {
		r.OutputVAT, r.InputVAT = round2(r.OutputVAT), round2(r.InputVAT)
		r.OutputWithoutFaktur, r.InputWithoutFaktur = round2(r.OutputWithoutFaktur), round2(r.InputWithoutFaktur)
		r.Net = round2(r.OutputVAT - r.InputVAT)
		out.Periods = append(out.Periods, *r)
		out.Total.OutputVAT += r.OutputVAT
		out.Total.InputVAT += r.InputVAT
		out.Total.OutputWithoutFaktur += r.OutputWithoutFaktur
		out.Total.InputWithoutFaktur += r.InputWithoutFaktur
	}
	sort.Slice(out.Periods, func(i, j int) bool { return out.Periods[i].Period < out.Periods[j].Period })
	out.Total.OutputVAT, out.Total.InputVAT = round2(out.Total.OutputVAT), round2(out.Total.InputVAT)
	out.Total.OutputWithoutFaktur, out.Total.InputWithoutFaktur = round2(out.Total.OutputWithoutFaktur), round2(out.Total.InputWithoutFaktur)
	out.Total.Net = round2(out.Total.OutputVAT - out.Total.InputVAT)
	return out, nil
}
