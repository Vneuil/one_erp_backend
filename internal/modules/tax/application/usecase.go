package application

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/taxid"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type TaxUseCase interface {
	GetSettings(ctx context.Context) (*domain.TaxSettings, error)
	UpdateSettings(ctx context.Context, in SettingsInput) (*domain.TaxSettings, error)

	ListSerialRanges(ctx context.Context) ([]domain.SerialRange, error)
	CreateSerialRange(ctx context.Context, in SerialRangeInput) (*domain.SerialRange, error)
	SetSerialRangeActive(ctx context.Context, id uuid.UUID, active bool) (*domain.SerialRange, error)

	ListTaxInvoices(ctx context.Context, f domain.InvoiceFilter) ([]domain.TaxInvoice, error)
	GetTaxInvoice(ctx context.Context, id uuid.UUID) (*domain.TaxInvoice, error)
	CreateFromSalesInvoice(ctx context.Context, in FromSalesInput) (*domain.TaxInvoice, error)
	CreateFromPurchaseInvoice(ctx context.Context, in FromPurchaseInput) (*domain.TaxInvoice, error)
	UpdateDraft(ctx context.Context, id uuid.UUID, in UpdateInput) (*domain.TaxInvoice, error)
	Issue(ctx context.Context, id uuid.UUID) (*domain.TaxInvoice, error)
	Cancel(ctx context.Context, id uuid.UUID, reason string) (*domain.TaxInvoice, error)
	Replace(ctx context.Context, id uuid.UUID) (*domain.TaxInvoice, error)
	SetTaxNumber(ctx context.Context, id uuid.UUID, number string) (*domain.TaxInvoice, error)

	SalesBook(ctx context.Context, from, to string) (*SalesBookDTO, error)
	VATSummary(ctx context.Context, from, to string) (*VATSummaryDTO, error)
	CoretaxExport(ctx context.Context, from, to string) (*CoretaxExportDTO, error)
}

type taxUseCase struct {
	repo domain.TaxRepository
	src  domain.SourceReader
	now  func() time.Time
}

func NewTaxUseCase(repo domain.TaxRepository, src domain.SourceReader) TaxUseCase {
	return &taxUseCase{repo: repo, src: src, now: time.Now}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func (uc *taxUseCase) today() string { return uc.now().Format("2006-01-02") }

func validDate(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }

func validPeriod(s string) bool { _, err := time.Parse("2006-01", s); return err == nil }

func checkRange(from, to string) error {
	if from != "" && !validDate(from) {
		return apperrors.NewBadRequest("from must be YYYY-MM-DD")
	}
	if to != "" && !validDate(to) {
		return apperrors.NewBadRequest("to must be YYYY-MM-DD")
	}
	if from != "" && to != "" && from > to {
		return apperrors.NewBadRequest("from cannot be after to")
	}
	return nil
}

// Settings

type SettingsInput struct {
	TaxpayerName string `json:"taxpayerName"`
	NPWP         string `json:"npwp"`
	Address      string `json:"address"`
	IsPKP        bool   `json:"isPkp"`
}

func (uc *taxUseCase) GetSettings(ctx context.Context) (*domain.TaxSettings, error) {
	s, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load tax settings")
	}
	if s == nil {
		return &domain.TaxSettings{}, nil
	}
	return s, nil
}

func (uc *taxUseCase) UpdateSettings(ctx context.Context, in SettingsInput) (*domain.TaxSettings, error) {
	npwp, err := taxid.NPWP(in.NPWP)
	if err != nil {
		return nil, err
	}
	if in.IsPKP && npwp == "" {
		return nil, apperrors.NewBadRequest("A PKP needs an NPWP")
	}
	s, err := uc.repo.GetSettings(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load tax settings")
	}
	if s == nil {
		s = &domain.TaxSettings{}
		s.ID = uuid.New()
	}
	s.TaxpayerName, s.NPWP, s.Address, s.IsPKP = strings.TrimSpace(in.TaxpayerName), npwp, strings.TrimSpace(in.Address), in.IsPKP
	if err := uc.repo.SaveSettings(ctx, s); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save tax settings")
	}
	return s, nil
}

// Serial ranges

type SerialRangeInput struct {
	Prefix string `json:"prefix"`
	Start  int64  `json:"start"`
	End    int64  `json:"end"`
	Width  int    `json:"width"`
}

func (uc *taxUseCase) ListSerialRanges(ctx context.Context) ([]domain.SerialRange, error) {
	out, err := uc.repo.ListSerialRanges(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list serial ranges")
	}
	return out, nil
}

func (uc *taxUseCase) CreateSerialRange(ctx context.Context, in SerialRangeInput) (*domain.SerialRange, error) {
	width := in.Width
	if width == 0 {
		width = 8
	}
	if in.Start < 0 || in.End < in.Start || width < 1 || width > 20 {
		return nil, apperrors.NewBadRequest("A serial range needs 0 <= start <= end and a width of 1-20 digits")
	}
	if int64(len(fmt.Sprint(in.End))) > int64(width) {
		return nil, apperrors.NewBadRequest("The width is too small for the end number")
	}
	prefix := strings.TrimSpace(in.Prefix)
	existing, err := uc.repo.ListSerialRanges(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list serial ranges")
	}
	for _, e := range existing {
		if e.Prefix == prefix && e.Width == width && in.Start <= e.End && e.Start <= in.End {
			return nil, apperrors.NewConflict("This range overlaps an existing range with the same prefix")
		}
	}
	r := &domain.SerialRange{Prefix: prefix, Start: in.Start, End: in.End, Next: in.Start, Width: width, IsActive: true}
	r.ID = uuid.New()
	if err := uc.repo.CreateSerialRange(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save serial range")
	}
	return r, nil
}

func (uc *taxUseCase) SetSerialRangeActive(ctx context.Context, id uuid.UUID, active bool) (*domain.SerialRange, error) {
	r, err := uc.repo.GetSerialRange(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load serial range")
	}
	if r == nil {
		return nil, apperrors.NewNotFound("Serial range not found")
	}
	r.IsActive = active
	if err := uc.repo.UpdateSerialRange(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update serial range")
	}
	return r, nil
}

// Faktur Pajak

func (uc *taxUseCase) ListTaxInvoices(ctx context.Context, f domain.InvoiceFilter) ([]domain.TaxInvoice, error) {
	if err := checkRange(f.From, f.To); err != nil {
		return nil, err
	}
	out, err := uc.repo.ListTaxInvoices(ctx, f)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list Faktur Pajak")
	}
	return out, nil
}

func (uc *taxUseCase) GetTaxInvoice(ctx context.Context, id uuid.UUID) (*domain.TaxInvoice, error) {
	t, err := uc.repo.GetTaxInvoice(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load Faktur Pajak")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Faktur Pajak not found")
	}
	return t, nil
}

func (uc *taxUseCase) nextNumber(ctx context.Context, direction, date string) (string, error) {
	prefix := "FPK-"
	if direction == domain.DirectionInput {
		prefix = "FPM-"
	}
	prefix += strings.ReplaceAll(date[:7], "-", "") + "-"
	n, err := uc.repo.CountTaxInvoices(ctx, direction, prefix)
	if err != nil {
		return "", apperrors.NewInternal(err, "Failed to number the Faktur Pajak")
	}
	return fmt.Sprintf("%s%04d", prefix, n+1), nil
}

// allocateLines spreads an invoice's tax base, "nilai lain" base and PPN over
// goods lines in proportion to their subtotals; the last line absorbs rounding
// so the lines always add up to the header. UnitPrice is derived from the
// allocated base. With no source lines it returns one summary line.
func allocateLines(src []domain.SourceLine, fallbackDescription string, taxBase, otherValue, vat float64) []domain.TaxInvoiceLine {
	var sum float64
	usable := make([]domain.SourceLine, 0, len(src))
	for _, l := range src {
		if l.Quantity > 0 && l.Subtotal > 0 {
			usable = append(usable, l)
			sum += l.Subtotal
		}
	}
	if len(usable) == 0 {
		return []domain.TaxInvoiceLine{{Position: 1, Description: fallbackDescription, Unit: "Unit", Quantity: 1,
			UnitPrice: taxBase, TaxBase: taxBase, DPPOtherValue: otherValue, VATAmount: vat}}
	}
	out := make([]domain.TaxInvoiceLine, 0, len(usable))
	var usedBase, usedOther, usedVAT float64
	for i, l := range usable {
		base, other, v := round2(taxBase*l.Subtotal/sum), round2(otherValue*l.Subtotal/sum), round2(vat*l.Subtotal/sum)
		if i == len(usable)-1 {
			base, other, v = round2(taxBase-usedBase), round2(otherValue-usedOther), round2(vat-usedVAT)
		}
		usedBase, usedOther, usedVAT = usedBase+base, usedOther+other, usedVAT+v
		unit := l.Unit
		if unit == "" {
			unit = "Unit"
		}
		out = append(out, domain.TaxInvoiceLine{Position: i + 1, Description: l.Description, Unit: unit, Quantity: l.Quantity,
			UnitPrice: round2(base / l.Quantity), TaxBase: base, DPPOtherValue: other, VATAmount: v})
	}
	return out
}

func transactionCodeFor(otherValueBase bool) string {
	if otherValueBase {
		return "04" // DPP nilai lain
	}
	return "01"
}

type FromSalesInput struct {
	SalesInvoiceID  uuid.UUID `json:"salesInvoiceId"`
	TransactionCode string    `json:"transactionCode"`
	Notes           string    `json:"notes"`
}

// CreateFromSalesInvoice drafts a Faktur Pajak Keluaran from a sales invoice
// that carries PPN, snapshotting the buyer's identity from the customer master.
func (uc *taxUseCase) CreateFromSalesInvoice(ctx context.Context, in FromSalesInput) (*domain.TaxInvoice, error) {
	inv, err := uc.src.SalesInvoice(ctx, in.SalesInvoiceID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the sales invoice")
	}
	if inv == nil {
		return nil, apperrors.NewNotFound("Sales invoice not found")
	}
	if inv.VATAmt <= 0 {
		return nil, apperrors.NewBadRequest("This invoice has no PPN; create the invoice with PPN to raise a Faktur Pajak")
	}
	if open, err := uc.repo.FindOpenBySource(ctx, domain.SourceSalesInvoice, inv.ID); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing Faktur Pajak")
	} else if open != nil {
		return nil, apperrors.NewConflict("This invoice already has Faktur Pajak " + open.Number)
	}

	party := domain.Party{Name: inv.CustomerName}
	if customers, err := uc.src.CustomersByName(ctx); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load customers")
	} else if c, ok := customers[strings.ToLower(strings.TrimSpace(inv.CustomerName))]; ok {
		party = c
	}
	srcLines, err := uc.src.SalesInvoiceLines(ctx, inv.ID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load invoice lines")
	}

	date := inv.Date
	if !validDate(date) {
		date = uc.today()
	}
	code := strings.TrimSpace(in.TransactionCode)
	if code == "" {
		code = transactionCodeFor(inv.VATOtherValueBase)
	}
	number, err := uc.nextNumber(ctx, domain.DirectionOutput, date)
	if err != nil {
		return nil, err
	}
	t := &domain.TaxInvoice{
		Direction: domain.DirectionOutput, Number: number, Status: domain.StatusDraft, TransactionCode: code, Date: date, Period: date[:7],
		CounterpartyName: party.Name, CounterpartyNPWP: party.NPWP, CounterpartyNIK: party.NIK, CounterpartyAddress: party.Address, CounterpartyEmail: party.Email,
		SourceType: domain.SourceSalesInvoice, SourceID: inv.ID, SourceReference: inv.Number,
		TaxBase: inv.TaxBase, DPPOtherValue: inv.DPPOtherValue, VATRate: inv.VATRate, VATAmount: inv.VATAmt,
		Notes: strings.TrimSpace(in.Notes), CreatedBy: actor.EmailFrom(ctx),
		Lines: allocateLines(srcLines, "Penjualan sesuai invoice "+inv.Number, inv.TaxBase, inv.DPPOtherValue, inv.VATAmt),
	}
	return uc.save(ctx, t)
}

type FromPurchaseInput struct {
	PurchaseInvoiceID uuid.UUID `json:"purchaseInvoiceId"`
	// TaxNumber is the Nomor Faktur Pajak printed on the supplier's faktur.
	TaxNumber string `json:"taxNumber"`
	Date      string `json:"date"`
	// Period is the masa pajak in which the PPN Masukan is credited (YYYY-MM); defaults to the faktur month.
	Period string `json:"period"`
	Notes  string `json:"notes"`
}

// CreateFromPurchaseInvoice drafts a Faktur Pajak Masukan from a purchase
// invoice whose PPN was booked as creditable.
func (uc *taxUseCase) CreateFromPurchaseInvoice(ctx context.Context, in FromPurchaseInput) (*domain.TaxInvoice, error) {
	inv, err := uc.src.PurchaseInvoice(ctx, in.PurchaseInvoiceID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the purchase invoice")
	}
	if inv == nil {
		return nil, apperrors.NewNotFound("Purchase invoice not found")
	}
	if inv.VATAmt <= 0 {
		return nil, apperrors.NewBadRequest("This purchase invoice has no PPN")
	}
	if !inv.VATCreditable {
		return nil, apperrors.NewBadRequest("The PPN of this purchase invoice was booked as non-creditable (part of the cost)")
	}
	if open, err := uc.repo.FindOpenBySource(ctx, domain.SourcePurchaseInvoice, inv.ID); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check existing Faktur Pajak")
	} else if open != nil {
		return nil, apperrors.NewConflict("This invoice already has Faktur Pajak Masukan " + open.Number)
	}
	date := in.Date
	if date == "" {
		date = inv.Date
	}
	if !validDate(date) {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	period := in.Period
	if period == "" {
		period = date[:7]
	}
	if !validPeriod(period) {
		return nil, apperrors.NewBadRequest("period must be YYYY-MM")
	}
	party := domain.Party{Name: inv.SupplierName}
	if s, err := uc.src.Supplier(ctx, inv.SupplierID); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the supplier")
	} else if s != nil {
		party = *s
	}
	number, err := uc.nextNumber(ctx, domain.DirectionInput, date)
	if err != nil {
		return nil, err
	}
	t := &domain.TaxInvoice{
		Direction: domain.DirectionInput, Number: number, TaxNumber: strings.TrimSpace(in.TaxNumber), Status: domain.StatusDraft,
		TransactionCode: transactionCodeFor(inv.VATOtherValueBase), Date: date, Period: period,
		CounterpartyName: party.Name, CounterpartyNPWP: party.NPWP, CounterpartyNIK: party.NIK, CounterpartyAddress: party.Address, CounterpartyEmail: party.Email,
		SourceType: domain.SourcePurchaseInvoice, SourceID: inv.ID, SourceReference: inv.Number,
		TaxBase: inv.TaxBase, DPPOtherValue: inv.DPPOtherValue, VATRate: inv.VATRate, VATAmount: inv.VATAmt,
		Notes: strings.TrimSpace(in.Notes), CreatedBy: actor.EmailFrom(ctx),
		Lines: allocateLines(nil, "Pembelian sesuai invoice "+inv.Number, inv.TaxBase, inv.DPPOtherValue, inv.VATAmt),
	}
	return uc.save(ctx, t)
}

func (uc *taxUseCase) save(ctx context.Context, t *domain.TaxInvoice) (*domain.TaxInvoice, error) {
	t.ID = uuid.New()
	for i := range t.Lines {
		t.Lines[i].ID = uuid.New()
		t.Lines[i].TaxInvoiceID = t.ID
	}
	if err := uc.repo.CreateTaxInvoice(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save Faktur Pajak")
	}
	return t, nil
}

type UpdateInput struct {
	TransactionCode     *string `json:"transactionCode,omitempty"`
	Date                *string `json:"date,omitempty"`
	Period              *string `json:"period,omitempty"`
	CounterpartyName    *string `json:"counterpartyName,omitempty"`
	CounterpartyNPWP    *string `json:"counterpartyNpwp,omitempty"`
	CounterpartyNIK     *string `json:"counterpartyNik,omitempty"`
	CounterpartyAddress *string `json:"counterpartyAddress,omitempty"`
	CounterpartyEmail   *string `json:"counterpartyEmail,omitempty"`
	TaxNumber           *string `json:"taxNumber,omitempty"`
	Notes               *string `json:"notes,omitempty"`
}

// UpdateDraft edits the descriptive fields of a draft. Amounts and lines come
// from the source invoice and are never edited here.
func (uc *taxUseCase) UpdateDraft(ctx context.Context, id uuid.UUID, in UpdateInput) (*domain.TaxInvoice, error) {
	t, err := uc.GetTaxInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status != domain.StatusDraft {
		return nil, apperrors.NewBadRequest("Only a draft Faktur Pajak can be edited")
	}
	if in.TransactionCode != nil {
		t.TransactionCode = strings.TrimSpace(*in.TransactionCode)
	}
	if in.Date != nil {
		if !validDate(*in.Date) {
			return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
		}
		t.Date = *in.Date
		if t.Direction == domain.DirectionOutput {
			t.Period = t.Date[:7]
		}
	}
	if in.Period != nil && t.Direction == domain.DirectionInput {
		if !validPeriod(*in.Period) {
			return nil, apperrors.NewBadRequest("period must be YYYY-MM")
		}
		t.Period = *in.Period
	}
	if in.CounterpartyName != nil {
		t.CounterpartyName = strings.TrimSpace(*in.CounterpartyName)
	}
	if in.CounterpartyNPWP != nil {
		if t.CounterpartyNPWP, err = taxid.NPWP(*in.CounterpartyNPWP); err != nil {
			return nil, err
		}
	}
	if in.CounterpartyNIK != nil {
		if t.CounterpartyNIK, err = taxid.NIK(*in.CounterpartyNIK); err != nil {
			return nil, err
		}
	}
	if in.CounterpartyAddress != nil {
		t.CounterpartyAddress = strings.TrimSpace(*in.CounterpartyAddress)
	}
	if in.CounterpartyEmail != nil {
		t.CounterpartyEmail = strings.TrimSpace(*in.CounterpartyEmail)
	}
	if in.TaxNumber != nil && t.Direction == domain.DirectionInput {
		t.TaxNumber = strings.TrimSpace(*in.TaxNumber)
	}
	if in.Notes != nil {
		t.Notes = strings.TrimSpace(*in.Notes)
	}
	if err := uc.repo.UpdateTaxInvoice(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update Faktur Pajak")
	}
	return t, nil
}

// Issue finalises a draft. A Keluaran needs the seller to be a PKP with an NPWP
// and the buyer to be identified by NPWP or NIK; it takes the next number from
// the active serial range when there is one (otherwise the number is recorded
// later with SetTaxNumber). A Masukan needs the supplier's faktur number and NPWP.
func (uc *taxUseCase) Issue(ctx context.Context, id uuid.UUID) (*domain.TaxInvoice, error) {
	t, err := uc.GetTaxInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status != domain.StatusDraft {
		return nil, apperrors.NewBadRequest("Only a draft Faktur Pajak can be issued")
	}
	if t.VATAmount <= 0 || t.TaxBase <= 0 {
		return nil, apperrors.NewBadRequest("A Faktur Pajak needs a positive tax base and PPN")
	}
	switch t.Direction {
	case domain.DirectionOutput:
		s, err := uc.repo.GetSettings(ctx)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to load tax settings")
		}
		if s == nil || !s.IsPKP || s.NPWP == "" {
			return nil, apperrors.NewBadRequest("Set the company NPWP and PKP status in Tax Settings before issuing a Faktur Pajak")
		}
		if t.CounterpartyNPWP == "" && t.CounterpartyNIK == "" {
			return nil, apperrors.NewBadRequest("The buyer needs an NPWP or NIK; add it to the customer or edit the draft")
		}
		if t.TaxNumber == "" {
			number, err := uc.repo.ClaimNextSerial(ctx)
			if err != nil {
				return nil, apperrors.NewInternal(err, "Failed to claim a serial number")
			}
			t.TaxNumber = number
		}
	case domain.DirectionInput:
		if t.TaxNumber == "" {
			return nil, apperrors.NewBadRequest("Enter the supplier's Nomor Faktur Pajak first")
		}
		if t.CounterpartyNPWP == "" {
			return nil, apperrors.NewBadRequest("The supplier needs an NPWP; add it to the supplier or edit the draft")
		}
	}
	if t.TaxNumber != "" {
		taken, err := uc.repo.TaxNumberTaken(ctx, t.Direction, t.TaxNumber, t.ID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to check the Faktur Pajak number")
		}
		if taken {
			return nil, apperrors.NewConflict("Nomor Faktur Pajak " + t.TaxNumber + " is already used")
		}
	}
	now := uc.now()
	t.Status, t.IssuedAt = domain.StatusIssued, &now
	if err := uc.repo.UpdateTaxInvoice(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to issue Faktur Pajak")
	}
	if t.ReplacesID != nil {
		if old, err := uc.repo.GetTaxInvoice(ctx, *t.ReplacesID); err == nil && old != nil && old.Status == domain.StatusIssued {
			old.Status = domain.StatusReplaced
			if err := uc.repo.UpdateTaxInvoice(ctx, old); err != nil {
				return t, apperrors.NewInternal(err, "Issued, but the replaced Faktur Pajak could not be marked as replaced")
			}
		}
	}
	return t, nil
}

// Cancel voids a draft or issued Faktur Pajak. Its number stays on record.
func (uc *taxUseCase) Cancel(ctx context.Context, id uuid.UUID, reason string) (*domain.TaxInvoice, error) {
	t, err := uc.GetTaxInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status != domain.StatusDraft && t.Status != domain.StatusIssued {
		return nil, apperrors.NewBadRequest("Only a draft or issued Faktur Pajak can be cancelled")
	}
	reason = strings.TrimSpace(reason)
	if t.Status == domain.StatusIssued && reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required to cancel an issued Faktur Pajak")
	}
	t.Status, t.CancelReason = domain.StatusCancelled, reason
	if err := uc.repo.UpdateTaxInvoice(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to cancel Faktur Pajak")
	}
	return t, nil
}

// Replace drafts a replacement (pengganti) of an issued Faktur Pajak. The
// original is marked replaced when the replacement is issued.
func (uc *taxUseCase) Replace(ctx context.Context, id uuid.UUID) (*domain.TaxInvoice, error) {
	old, err := uc.GetTaxInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if old.Status != domain.StatusIssued {
		return nil, apperrors.NewBadRequest("Only an issued Faktur Pajak can be replaced")
	}
	number, err := uc.nextNumber(ctx, old.Direction, old.Date)
	if err != nil {
		return nil, err
	}
	t := *old
	t.BaseEntity = types.BaseEntity{}
	t.Number, t.Status, t.IssuedAt, t.CancelReason = number, domain.StatusDraft, nil, ""
	t.ReplacesID, t.Revision = &old.ID, old.Revision+1
	// An output replacement gets its own number on issue; an input one keeps the supplier's.
	if old.Direction == domain.DirectionOutput {
		t.TaxNumber = ""
	}
	t.CreatedBy = actor.EmailFrom(ctx)
	t.Lines = append([]domain.TaxInvoiceLine(nil), old.Lines...)
	for i := range t.Lines {
		t.Lines[i].BaseEntity = types.BaseEntity{}
	}
	return uc.save(ctx, &t)
}

// SetTaxNumber records the official Nomor Faktur Pajak, e.g. the number DJP
// returned after upload.
func (uc *taxUseCase) SetTaxNumber(ctx context.Context, id uuid.UUID, number string) (*domain.TaxInvoice, error) {
	t, err := uc.GetTaxInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status != domain.StatusDraft && t.Status != domain.StatusIssued {
		return nil, apperrors.NewBadRequest("The number of a cancelled or replaced Faktur Pajak cannot be changed")
	}
	number = strings.TrimSpace(number)
	if number == "" {
		return nil, apperrors.NewBadRequest("taxNumber is required")
	}
	taken, err := uc.repo.TaxNumberTaken(ctx, t.Direction, number, t.ID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check the Faktur Pajak number")
	}
	if taken {
		return nil, apperrors.NewConflict("Nomor Faktur Pajak " + number + " is already used")
	}
	t.TaxNumber = number
	if err := uc.repo.UpdateTaxInvoice(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save the number")
	}
	return t, nil
}
