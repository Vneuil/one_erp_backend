package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/taxid"
)

// CoretaxHeaders is the column layout of the Faktur Pajak Keluaran export: the
// "Faktur" and "DetailFaktur" sheets of the Coretax import template flattened
// into one row per goods/services line. The layout follows the published
// template as best known and has NOT been validated against a live Coretax
// import; confirm it with a test upload before relying on it.
var CoretaxHeaders = []string{
	"Baris", "Tanggal Faktur Pajak", "Jenis Faktur Pajak", "Kode Transaksi", "Keterangan Tambahan", "Dokumen Pendukung", "Referensi", "Cap Fasilitas",
	"ID TKU Penjual", "NPWP/NIK Pembeli", "Jenis ID Pembeli", "Negara Pembeli", "Nomor Dokumen Pembeli", "Nama Pembeli", "Alamat Pembeli", "Email Pembeli", "ID TKU Pembeli",
	"Barang/Jasa", "Kode Barang Jasa", "Nama Barang/Jasa", "Nama Satuan Ukur", "Harga Satuan", "Jumlah Barang Jasa", "Total Diskon",
	"DPP", "DPP Nilai Lain", "Tarif PPN", "PPN", "Tarif PPnBM", "PPnBM",
}

const (
	coretaxNoGoodsCode = "000000"
	coretaxUnitPiece   = "UM.0021"
)

type CoretaxExportDTO struct {
	From     string     `json:"from"`
	To       string     `json:"to"`
	Headers  []string   `json:"headers"`
	Rows     [][]string `json:"rows"`
	Invoices int        `json:"invoices"`
	Warnings []string   `json:"warnings"`
}

func money(v float64) string { return fmt.Sprintf("%.2f", v) }

func coretaxDate(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return t.Format("02/01/2006")
}

// CoretaxRows turns issued Faktur Pajak Keluaran into export rows (one per line).
func CoretaxRows(sellerNPWP string, invoices []domain.TaxInvoice) [][]string {
	sellerTKU := taxid.NPWP16(sellerNPWP) + "000000"
	var rows [][]string
	for i, t := range invoices {
		kind := "Normal"
		if t.Revision > 0 {
			kind = "Pengganti"
		}
		buyerID, idType, buyerTKU := t.CounterpartyNIK, "National ID", "000000"
		if t.CounterpartyNPWP != "" {
			buyerID, idType, buyerTKU = taxid.NPWP16(t.CounterpartyNPWP), "TIN", taxid.NPWP16(t.CounterpartyNPWP)+"000000"
		}
		for _, l := range t.Lines {
			other := ""
			if t.TransactionCode == "04" || l.DPPOtherValue > 0 {
				other = money(l.DPPOtherValue)
			}
			rows = append(rows, []string{
				fmt.Sprint(i + 1), coretaxDate(t.Date), kind, t.TransactionCode, t.Notes, "", t.SourceReference, "",
				sellerTKU, buyerID, idType, "IDN", "", t.CounterpartyName, t.CounterpartyAddress, t.CounterpartyEmail, buyerTKU,
				"A", coretaxNoGoodsCode, l.Description, coretaxUnitPiece, money(l.UnitPrice), money(l.Quantity), "0.00",
				money(l.TaxBase), other, money(t.VATRate), money(l.VATAmount), "0.00", "0.00",
			})
		}
	}
	return rows
}

// CoretaxExport builds the Coretax CSV rows for issued Faktur Pajak Keluaran
// dated in the range (default: current month).
func (uc *taxUseCase) CoretaxExport(ctx context.Context, from, to string) (*CoretaxExportDTO, error) {
	if err := checkRange(from, to); err != nil {
		return nil, err
	}
	if from == "" && to == "" {
		from, to = monthStart(uc.now()), uc.today()
	}
	settings, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load tax settings")
	}
	if settings == nil || settings.NPWP == "" || !settings.IsPKP {
		return nil, apperrors.NewBadRequest("Set the company NPWP and PKP status in Tax Settings first")
	}
	invoices, err := uc.repo.ListTaxInvoices(ctx, domain.InvoiceFilter{Direction: domain.DirectionOutput, Status: domain.StatusIssued, From: from, To: to})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load Faktur Pajak")
	}
	out := &CoretaxExportDTO{From: from, To: to, Headers: CoretaxHeaders, Invoices: len(invoices), Warnings: []string{}}
	for _, t := range invoices {
		if t.TaxNumber == "" {
			out.Warnings = append(out.Warnings, t.Number+" ("+t.SourceReference+") has no Nomor Faktur Pajak yet")
		}
		if strings.TrimSpace(t.CounterpartyAddress) == "" {
			out.Warnings = append(out.Warnings, t.Number+" ("+t.SourceReference+"): the buyer has no address")
		}
	}
	out.Rows = CoretaxRows(settings.NPWP, invoices)
	if out.Rows == nil {
		out.Rows = [][]string{}
	}
	return out, nil
}
