package application

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	"github.com/google/uuid"
)

type memRepo struct {
	settings *domain.TaxSettings
	ranges   []*domain.SerialRange
	invoices []*domain.TaxInvoice
}

func (r *memRepo) GetSettings(context.Context) (*domain.TaxSettings, error) { return r.settings, nil }
func (r *memRepo) SaveSettings(_ context.Context, s *domain.TaxSettings) error {
	r.settings = s
	return nil
}
func (r *memRepo) CreateSerialRange(_ context.Context, sr *domain.SerialRange) error {
	r.ranges = append(r.ranges, sr)
	return nil
}
func (r *memRepo) GetSerialRange(_ context.Context, id uuid.UUID) (*domain.SerialRange, error) {
	for _, s := range r.ranges {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}
func (r *memRepo) UpdateSerialRange(context.Context, *domain.SerialRange) error { return nil }
func (r *memRepo) ListSerialRanges(context.Context) ([]domain.SerialRange, error) {
	var out []domain.SerialRange
	for _, s := range r.ranges {
		out = append(out, *s)
	}
	return out, nil
}
func (r *memRepo) ClaimNextSerial(context.Context) (string, error) {
	for _, s := range r.ranges {
		if s.IsActive && s.Next <= s.End {
			n := fmt.Sprintf("%s%0*d", s.Prefix, s.Width, s.Next)
			s.Next++
			return n, nil
		}
	}
	return "", nil
}
func (r *memRepo) CreateTaxInvoice(_ context.Context, t *domain.TaxInvoice) error {
	r.invoices = append(r.invoices, t)
	return nil
}
func (r *memRepo) GetTaxInvoice(_ context.Context, id uuid.UUID) (*domain.TaxInvoice, error) {
	for _, t := range r.invoices {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, nil
}
func (r *memRepo) UpdateTaxInvoice(context.Context, *domain.TaxInvoice) error { return nil }
func (r *memRepo) ListTaxInvoices(_ context.Context, f domain.InvoiceFilter) ([]domain.TaxInvoice, error) {
	var out []domain.TaxInvoice
	for _, t := range r.invoices {
		if (f.Direction != "" && t.Direction != f.Direction) || (f.Status != "" && t.Status != f.Status) ||
			(f.From != "" && t.Date < f.From) || (f.To != "" && t.Date > f.To) {
			continue
		}
		out = append(out, *t)
	}
	return out, nil
}
func (r *memRepo) FindOpenBySource(_ context.Context, st string, id uuid.UUID) (*domain.TaxInvoice, error) {
	for _, t := range r.invoices {
		if t.SourceType == st && t.SourceID == id && (t.Status == domain.StatusDraft || t.Status == domain.StatusIssued) {
			return t, nil
		}
	}
	return nil, nil
}
func (r *memRepo) ListBySources(_ context.Context, st string, ids []uuid.UUID) ([]domain.TaxInvoice, error) {
	var out []domain.TaxInvoice
	for _, t := range r.invoices {
		for _, id := range ids {
			if t.SourceType == st && t.SourceID == id && (t.Status == domain.StatusDraft || t.Status == domain.StatusIssued) {
				out = append(out, *t)
			}
		}
	}
	return out, nil
}
func (r *memRepo) TaxNumberTaken(_ context.Context, dir, number string, except uuid.UUID) (bool, error) {
	for _, t := range r.invoices {
		if t.Direction == dir && t.TaxNumber == number && t.ID != except && t.Status != domain.StatusCancelled {
			return true, nil
		}
	}
	return false, nil
}
func (r *memRepo) CountTaxInvoices(_ context.Context, dir, prefix string) (int64, error) {
	var n int64
	for _, t := range r.invoices {
		if t.Direction == dir && strings.HasPrefix(t.Number, prefix) {
			n++
		}
	}
	return n, nil
}

type memSource struct {
	sales     map[uuid.UUID]domain.SalesInvoiceRef
	purchases map[uuid.UUID]domain.PurchaseInvoiceRef
	lines     map[uuid.UUID][]domain.SourceLine
	customers map[string]domain.Party
	suppliers map[uuid.UUID]domain.Party
}

func (s *memSource) SalesInvoice(_ context.Context, id uuid.UUID) (*domain.SalesInvoiceRef, error) {
	if v, ok := s.sales[id]; ok {
		return &v, nil
	}
	return nil, nil
}
func (s *memSource) SalesInvoicesBetween(_ context.Context, from, to string) ([]domain.SalesInvoiceRef, error) {
	var out []domain.SalesInvoiceRef
	for _, v := range s.sales {
		if v.Date >= from && v.Date <= to {
			out = append(out, v)
		}
	}
	return out, nil
}
func (s *memSource) SalesInvoiceLines(_ context.Context, id uuid.UUID) ([]domain.SourceLine, error) {
	return s.lines[id], nil
}
func (s *memSource) PurchaseInvoice(_ context.Context, id uuid.UUID) (*domain.PurchaseInvoiceRef, error) {
	if v, ok := s.purchases[id]; ok {
		return &v, nil
	}
	return nil, nil
}
func (s *memSource) PurchaseInvoicesBetween(_ context.Context, from, to string) ([]domain.PurchaseInvoiceRef, error) {
	var out []domain.PurchaseInvoiceRef
	for _, v := range s.purchases {
		if v.Date >= from && v.Date <= to {
			out = append(out, v)
		}
	}
	return out, nil
}
func (s *memSource) CustomersByName(context.Context) (map[string]domain.Party, error) {
	return s.customers, nil
}
func (s *memSource) Supplier(_ context.Context, id uuid.UUID) (*domain.Party, error) {
	if v, ok := s.suppliers[id]; ok {
		return &v, nil
	}
	return nil, nil
}

type fixture struct {
	uc   TaxUseCase
	repo *memRepo
	src  *memSource
	inv  domain.SalesInvoiceRef
}

// newFixture has a PKP seller, one customer with an NPWP, and one 1.000.000
// sales invoice with PPN (DPP nilai lain: 12% x 11/12 => 110.000) billed over two goods lines.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	repo := &memRepo{settings: &domain.TaxSettings{TaxpayerName: "PT Contoh", NPWP: "0123456789012000", IsPKP: true}}
	inv := domain.SalesInvoiceRef{ID: uuid.New(), Number: "INV-1", CustomerName: "PT Pembeli", Date: "2026-03-10",
		Subtotal: 1_000_000, Total: 1_110_000, TaxBase: 1_000_000, DPPOtherValue: 916_666.67, VATRate: 12, VATAmt: 110_000, VATOtherValueBase: true}
	src := &memSource{
		sales:     map[uuid.UUID]domain.SalesInvoiceRef{inv.ID: inv},
		purchases: map[uuid.UUID]domain.PurchaseInvoiceRef{},
		lines: map[uuid.UUID][]domain.SourceLine{inv.ID: {
			{Description: "Barang A", Unit: "Pcs", Quantity: 3, Subtotal: 600_000}, {Description: "Barang B", Quantity: 2, Subtotal: 400_000}}},
		customers: map[string]domain.Party{"pt pembeli": {Name: "PT Pembeli", NPWP: "0987654321098000", Address: "Jl. Mawar 1", Email: "ap@pembeli.id"}},
		suppliers: map[uuid.UUID]domain.Party{},
	}
	uc := NewTaxUseCase(repo, src).(*taxUseCase)
	uc.now = func() time.Time { return time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC) }
	return &fixture{uc: uc, repo: repo, src: src, inv: inv}
}

func TestFakturFromSalesInvoiceSnapshotsBuyerAndAllocatesLines(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	d, err := f.uc.CreateFromSalesInvoice(ctx, FromSalesInput{SalesInvoiceID: f.inv.ID})
	if err != nil {
		t.Fatal(err)
	}
	if d.Direction != "output" || d.Status != "draft" || d.Number != "FPK-202603-0001" || d.TransactionCode != "04" || d.Period != "2026-03" {
		t.Fatalf("unexpected draft: %+v", d)
	}
	if d.CounterpartyNPWP != "0987654321098000" || d.CounterpartyAddress != "Jl. Mawar 1" || d.SourceReference != "INV-1" {
		t.Fatalf("buyer not snapshotted: %+v", d)
	}
	if len(d.Lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(d.Lines))
	}
	var base, other, vat float64
	for _, l := range d.Lines {
		base, other, vat = base+l.TaxBase, other+l.DPPOtherValue, vat+l.VATAmount
	}
	if round2(base) != 1_000_000 || round2(other) != 916_666.67 || round2(vat) != 110_000 {
		t.Fatalf("lines do not add up to header: base=%v other=%v vat=%v", base, other, vat)
	}
	if d.Lines[0].TaxBase != 600_000 || d.Lines[0].UnitPrice != 200_000 || d.Lines[1].Unit != "Unit" {
		t.Fatalf("bad allocation: %+v", d.Lines)
	}

	if _, err := f.uc.CreateFromSalesInvoice(ctx, FromSalesInput{SalesInvoiceID: f.inv.ID}); err == nil {
		t.Fatal("a second open Faktur for the same invoice must be rejected")
	}
	noVAT := domain.SalesInvoiceRef{ID: uuid.New(), Number: "INV-2", CustomerName: "X", Date: "2026-03-11", Total: 100}
	f.src.sales[noVAT.ID] = noVAT
	if _, err := f.uc.CreateFromSalesInvoice(ctx, FromSalesInput{SalesInvoiceID: noVAT.ID}); err == nil {
		t.Fatal("an invoice without PPN must be rejected")
	}
}

func TestIssueRequiresPKPBuyerIdentityAndClaimsSerial(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	d, _ := f.uc.CreateFromSalesInvoice(ctx, FromSalesInput{SalesInvoiceID: f.inv.ID})

	f.repo.settings.IsPKP = false
	if _, err := f.uc.Issue(ctx, d.ID); err == nil {
		t.Fatal("a non-PKP must not issue")
	}
	f.repo.settings.IsPKP = true

	npwp := d.CounterpartyNPWP
	d.CounterpartyNPWP = ""
	if _, err := f.uc.Issue(ctx, d.ID); err == nil {
		t.Fatal("buyer without NPWP or NIK must be rejected")
	}
	d.CounterpartyNPWP = npwp

	if _, err := f.uc.CreateSerialRange(ctx, SerialRangeInput{Prefix: "040-26.", Start: 1, End: 2, Width: 8}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.CreateSerialRange(ctx, SerialRangeInput{Prefix: "040-26.", Start: 2, End: 5, Width: 8}); err == nil {
		t.Fatal("overlapping range must be rejected")
	}
	issued, err := f.uc.Issue(ctx, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if issued.Status != "issued" || issued.TaxNumber != "040-26.00000001" || issued.IssuedAt == nil {
		t.Fatalf("not issued properly: %+v", issued)
	}
	if _, err := f.uc.Issue(ctx, d.ID); err == nil {
		t.Fatal("issuing twice must fail")
	}
	if _, err := f.uc.UpdateDraft(ctx, d.ID, UpdateInput{}); err == nil {
		t.Fatal("an issued Faktur is no longer editable")
	}
}

func TestReplaceAndCancelFlow(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.uc.CreateSerialRange(ctx, SerialRangeInput{Prefix: "S", Start: 1, End: 9, Width: 3})
	d, _ := f.uc.CreateFromSalesInvoice(ctx, FromSalesInput{SalesInvoiceID: f.inv.ID})
	orig, err := f.uc.Issue(ctx, d.ID)
	if err != nil {
		t.Fatal(err)
	}

	rep, err := f.uc.Replace(ctx, orig.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Status != "draft" || rep.Revision != 1 || rep.ReplacesID == nil || *rep.ReplacesID != orig.ID || rep.TaxNumber != "" || len(rep.Lines) != 2 {
		t.Fatalf("bad replacement: %+v", rep)
	}
	if rep.ID == orig.ID || rep.Lines[0].ID == orig.Lines[0].ID {
		t.Fatal("replacement must be a new record with new lines")
	}
	if _, err := f.uc.Issue(ctx, rep.ID); err != nil {
		t.Fatal(err)
	}
	if orig.Status != "replaced" || rep.TaxNumber == orig.TaxNumber || rep.TaxNumber != "S002" {
		t.Fatalf("original should be replaced and the replacement renumbered: orig=%s rep=%s", orig.Status, rep.TaxNumber)
	}

	if _, err := f.uc.Cancel(ctx, rep.ID, ""); err == nil {
		t.Fatal("cancelling an issued Faktur needs a reason")
	}
	c, err := f.uc.Cancel(ctx, rep.ID, "salah buyer")
	if err != nil || c.Status != "cancelled" {
		t.Fatalf("cancel: %+v err=%v", c, err)
	}
	// Once cancelled, the invoice can be billed again.
	if _, err := f.uc.CreateFromSalesInvoice(ctx, FromSalesInput{SalesInvoiceID: f.inv.ID}); err != nil {
		t.Fatalf("a cancelled Faktur must free the invoice: %v", err)
	}
}

func TestPurchaseFakturMasukanRules(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	supplier := uuid.New()
	f.src.suppliers[supplier] = domain.Party{Name: "PT Pemasok", NPWP: "0111222233334000"}
	creditable := domain.PurchaseInvoiceRef{ID: uuid.New(), SupplierID: supplier, Number: "PINV-1", SupplierName: "PT Pemasok", Date: "2026-03-05",
		Total: 2_220_000, TaxBase: 2_000_000, DPPOtherValue: 1_833_333.33, VATRate: 12, VATAmt: 220_000, VATOtherValueBase: true, VATCreditable: true}
	costed := creditable
	costed.ID, costed.VATCreditable = uuid.New(), false
	f.src.purchases[creditable.ID], f.src.purchases[costed.ID] = creditable, costed

	if _, err := f.uc.CreateFromPurchaseInvoice(ctx, FromPurchaseInput{PurchaseInvoiceID: costed.ID}); err == nil {
		t.Fatal("non-creditable PPN must not get a Faktur Masukan")
	}
	d, err := f.uc.CreateFromPurchaseInvoice(ctx, FromPurchaseInput{PurchaseInvoiceID: creditable.ID, Period: "2026-04"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Direction != "input" || d.Number != "FPM-202603-0001" || d.Period != "2026-04" || d.CounterpartyNPWP != "0111222233334000" {
		t.Fatalf("unexpected: %+v", d)
	}
	if _, err := f.uc.Issue(ctx, d.ID); err == nil {
		t.Fatal("a Faktur Masukan needs the supplier's faktur number")
	}
	if _, err := f.uc.SetTaxNumber(ctx, d.ID, "010.000-26.12345678"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Issue(ctx, d.ID); err != nil {
		t.Fatal(err)
	}
	// The supplier's number cannot be registered twice (here the second invoice is made creditable).
	second := costed
	second.VATCreditable = true
	f.src.purchases[costed.ID] = second
	d2, err := f.uc.CreateFromPurchaseInvoice(ctx, FromPurchaseInput{PurchaseInvoiceID: costed.ID, TaxNumber: "010.000-26.12345678"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Issue(ctx, d2.ID); err == nil {
		t.Fatal("duplicate supplier faktur number must be rejected")
	}
}

func TestSalesBookAndVATSummary(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	supplier := uuid.New()
	f.src.suppliers[supplier] = domain.Party{Name: "PT Pemasok", NPWP: "0111222233334000"}
	legacy := domain.SalesInvoiceRef{ID: uuid.New(), Number: "INV-OLD", CustomerName: "Walk-in", Date: "2026-03-12", Total: 500_000}
	f.src.sales[legacy.ID] = legacy
	pi := domain.PurchaseInvoiceRef{ID: uuid.New(), SupplierID: supplier, Number: "PINV-1", SupplierName: "PT Pemasok", Date: "2026-03-05",
		Total: 1_110_000, TaxBase: 1_000_000, VATRate: 12, VATAmt: 110_000, VATCreditable: true}
	unmatched := pi
	unmatched.ID, unmatched.Number, unmatched.VATAmt = uuid.New(), "PINV-2", 55_000
	f.src.purchases[pi.ID], f.src.purchases[unmatched.ID] = pi, unmatched

	book, err := f.uc.SalesBook(ctx, "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if book.Count != 2 || book.TaxBase != 1_500_000 || book.VATAmount != 110_000 || book.PendingFaktur != 1 {
		t.Fatalf("sales book before faktur: %+v", book)
	}

	f.uc.CreateSerialRange(ctx, SerialRangeInput{Start: 1, End: 9, Width: 3})
	d, _ := f.uc.CreateFromSalesInvoice(ctx, FromSalesInput{SalesInvoiceID: f.inv.ID})
	f.uc.Issue(ctx, d.ID)
	in, _ := f.uc.CreateFromPurchaseInvoice(ctx, FromPurchaseInput{PurchaseInvoiceID: pi.ID, TaxNumber: "010-26-1"})
	f.uc.Issue(ctx, in.ID)

	book, _ = f.uc.SalesBook(ctx, "2026-03-01", "2026-03-31")
	if book.PendingFaktur != 0 {
		t.Fatalf("issued invoice should no longer be pending: %+v", book)
	}
	var row *SalesBookRow
	for i := range book.Rows {
		if book.Rows[i].InvoiceNumber == "INV-1" {
			row = &book.Rows[i]
		}
	}
	if row == nil || row.TaxNumber != "001" || row.CustomerNPWP != "0987654321098000" || row.Gross != 1_000_000 {
		t.Fatalf("book row: %+v", row)
	}

	sum, err := f.uc.VATSummary(ctx, "2026-01-01", "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Periods) != 1 {
		t.Fatalf("periods: %+v", sum.Periods)
	}
	p := sum.Periods[0]
	if p.Period != "2026-03" || p.OutputVAT != 110_000 || p.InputVAT != 110_000 || p.Net != 0 || p.InputWithoutFaktur != 55_000 || p.OutputWithoutFaktur != 0 {
		t.Fatalf("summary: %+v", p)
	}
}

func TestCoretaxExportRows(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	f.uc.CreateSerialRange(ctx, SerialRangeInput{Start: 1, End: 9, Width: 3})
	d, _ := f.uc.CreateFromSalesInvoice(ctx, FromSalesInput{SalesInvoiceID: f.inv.ID})
	if _, err := f.uc.Issue(ctx, d.ID); err != nil {
		t.Fatal(err)
	}
	ex, err := f.uc.CoretaxExport(ctx, "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatal(err)
	}
	if ex.Invoices != 1 || len(ex.Rows) != 2 || len(ex.Rows[0]) != len(CoretaxHeaders) {
		t.Fatalf("shape: invoices=%d rows=%d cols=%d", ex.Invoices, len(ex.Rows), len(ex.Rows[0]))
	}
	r := ex.Rows[0]
	col := func(name string) string {
		for i, h := range CoretaxHeaders {
			if h == name {
				return r[i]
			}
		}
		t.Fatalf("no column %q", name)
		return ""
	}
	if col("Tanggal Faktur Pajak") != "10/03/2026" || col("Jenis Faktur Pajak") != "Normal" || col("Kode Transaksi") != "04" ||
		col("ID TKU Penjual") != "0123456789012000000000" || col("NPWP/NIK Pembeli") != "0987654321098000" || col("Jenis ID Pembeli") != "TIN" ||
		col("Nama Barang/Jasa") != "Barang A" || col("DPP") != "600000.00" || col("Tarif PPN") != "12.00" {
		t.Fatalf("row: %v", r)
	}
	f.repo.settings.IsPKP = false
	if _, err := f.uc.CoretaxExport(ctx, "", ""); err == nil {
		t.Fatal("export without PKP settings must be refused")
	}
}
