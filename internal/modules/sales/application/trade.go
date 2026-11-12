package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	productDomain "github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// DefaultTradeVATRate is the PPN applied to a PI/CI unless it is switched off (export).
const DefaultTradeVATRate = 11.0

type TradeLineDTO struct {
	ProductID   *uuid.UUID `json:"productId,omitempty"`
	Description string     `json:"description"`
	Unit        string     `json:"unit"`
	Quantity    float64    `json:"quantity"`
	UnitPrice   float64    `json:"unitPrice"`
}

// CreateTradeInvoiceDTO creates a PI/CI. Exactly one source may be given:
// a Sales Order, a Quotation, or (for a CI) the PI it is made from. Lines may be
// supplied to override the ones copied from the source.
type CreateTradeInvoiceDTO struct {
	Kind            string         `json:"kind"` // PI | CI
	SalesOrderID    *uuid.UUID     `json:"salesOrderId,omitempty"`
	QuotationID     *uuid.UUID     `json:"quotationId,omitempty"`
	ParentID        *uuid.UUID     `json:"parentId,omitempty"`
	CustomerName    string         `json:"customerName,omitempty"`
	BillingAddress  string         `json:"billingAddress,omitempty"`
	ShippingAddress string         `json:"shippingAddress,omitempty"`
	Currency        string         `json:"currency,omitempty"`
	PaymentTerm     string         `json:"paymentTerm,omitempty"`
	Incoterm        string         `json:"incoterm,omitempty"`
	Notes           string         `json:"notes,omitempty"`
	VATExempt       *bool          `json:"vatExempt,omitempty"`
	Lines           []TradeLineDTO `json:"lines,omitempty"`
}

type TradeInvoiceUseCase interface {
	Create(ctx context.Context, dto CreateTradeInvoiceDTO) (*domain.TradeInvoice, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error)
	List(ctx context.Context, kind string, q types.PaginationQuery) ([]domain.TradeInvoice, types.PaginationMeta, error)
	Issue(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error)
	Cancel(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error)
}

type tradeInvoiceUseCase struct {
	repo     domain.TradeInvoiceRepository
	sales    domain.SalesRepository
	products productDomain.ProductRepository // optional: fills line descriptions from the product master
}

func NewTradeInvoiceUseCase(repo domain.TradeInvoiceRepository, sales domain.SalesRepository, products productDomain.ProductRepository) TradeInvoiceUseCase {
	return &tradeInvoiceUseCase{repo: repo, sales: sales, products: products}
}

func (uc *tradeInvoiceUseCase) Create(ctx context.Context, dto CreateTradeInvoiceDTO) (*domain.TradeInvoice, error) {
	kind := strings.ToUpper(strings.TrimSpace(dto.Kind))
	if kind != domain.TradeKindPI && kind != domain.TradeKindCI {
		return nil, apperrors.NewBadRequest("kind must be PI (Proforma Invoice) or CI (Commercial Invoice)")
	}
	sources := 0
	for _, set := range []bool{dto.SalesOrderID != nil, dto.QuotationID != nil, dto.ParentID != nil} {
		if set {
			sources++
		}
	}
	if sources != 1 {
		return nil, apperrors.NewBadRequest("Provide exactly one source: salesOrderId, quotationId or parentId")
	}

	t := &domain.TradeInvoice{
		Kind: kind, Status: domain.TradeStatusDraft, IssueDate: time.Now().Format("2006-01-02"),
		CustomerName: strings.TrimSpace(dto.CustomerName), BillingAddr: dto.BillingAddress, ShippingAddr: dto.ShippingAddress,
		Currency: dto.Currency, PaymentTerm: dto.PaymentTerm, Incoterm: dto.Incoterm, Notes: dto.Notes,
		VATRate: DefaultTradeVATRate, CreatedBy: actor.EmailFrom(ctx),
		SalesOrderID: dto.SalesOrderID, QuotationID: dto.QuotationID, ParentID: dto.ParentID,
	}
	var lines []domain.TradeInvoiceLine

	switch {
	case dto.SalesOrderID != nil:
		o, err := uc.sales.GetOrderByID(ctx, *dto.SalesOrderID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get sales order")
		}
		if o == nil {
			return nil, apperrors.NewNotFound("Sales order not found")
		}
		if t.CustomerName == "" {
			t.CustomerName = o.CustomerName
		}
		if t.ShippingAddr == "" {
			t.ShippingAddr = o.ShippingAddress
		}
		if t.Currency == "" {
			t.Currency = o.Currency
		}
		t.VATExempt = o.VATExempt
		for _, l := range o.Lines {
			pid := l.ProductID
			desc, unit := uc.describe(ctx, pid)
			lines = append(lines, domain.TradeInvoiceLine{ProductID: &pid, Description: desc, Unit: unit, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
		}
	case dto.QuotationID != nil:
		q, err := uc.sales.GetQuotationByID(ctx, *dto.QuotationID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get quotation")
		}
		if q == nil {
			return nil, apperrors.NewNotFound("Quotation not found")
		}
		if t.CustomerName == "" {
			t.CustomerName = q.CustomerName
		}
		for _, l := range q.Lines {
			lines = append(lines, domain.TradeInvoiceLine{ProductID: l.ProductID, Description: l.Description, Unit: l.Unit, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
		}
	case dto.ParentID != nil:
		if kind != domain.TradeKindCI {
			return nil, apperrors.NewBadRequest("Only a Commercial Invoice can be created from a Proforma Invoice")
		}
		p, err := uc.repo.GetByID(ctx, *dto.ParentID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get proforma invoice")
		}
		if p == nil || p.Kind != domain.TradeKindPI {
			return nil, apperrors.NewNotFound("Proforma invoice not found")
		}
		if p.Status != domain.TradeStatusIssued {
			return nil, apperrors.NewConflict("Issue the proforma invoice before creating a commercial invoice from it")
		}
		t.SalesOrderID, t.QuotationID = p.SalesOrderID, p.QuotationID
		if t.CustomerName == "" {
			t.CustomerName = p.CustomerName
		}
		if t.BillingAddr == "" {
			t.BillingAddr = p.BillingAddr
		}
		if t.ShippingAddr == "" {
			t.ShippingAddr = p.ShippingAddr
		}
		if t.Currency == "" {
			t.Currency = p.Currency
		}
		if t.PaymentTerm == "" {
			t.PaymentTerm = p.PaymentTerm
		}
		if t.Incoterm == "" {
			t.Incoterm = p.Incoterm
		}
		t.VATExempt = p.VATExempt
		for _, l := range p.Lines {
			lines = append(lines, domain.TradeInvoiceLine{ProductID: l.ProductID, Description: l.Description, Unit: l.Unit, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
		}
	}
	if dto.VATExempt != nil {
		t.VATExempt = *dto.VATExempt
	}
	if len(dto.Lines) > 0 {
		lines = lines[:0]
		for _, l := range dto.Lines {
			lines = append(lines, domain.TradeInvoiceLine{ProductID: l.ProductID, Description: l.Description, Unit: l.Unit, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
		}
	}
	if t.CustomerName == "" {
		return nil, apperrors.NewBadRequest("Customer name is required")
	}
	if t.Currency == "" {
		t.Currency = "IDR"
	}
	if len(lines) == 0 {
		return nil, apperrors.NewBadRequest("At least one line item is required")
	}
	for i := range lines {
		if lines[i].Quantity <= 0 || lines[i].UnitPrice < 0 {
			return nil, apperrors.NewBadRequest("Each line needs a positive quantity and a non-negative price")
		}
		lines[i].Subtotal = round2(lines[i].Quantity * lines[i].UnitPrice)
	}
	t.Lines = lines
	computeTradeTotals(t)

	count, err := uc.repo.CountByKind(ctx, kind)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to number the invoice")
	}
	t.Number = fmt.Sprintf("%s-%s-%04d", kind, time.Now().Format("200601"), count+1)

	if err := uc.repo.Create(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create invoice")
	}
	return t, nil
}

func (uc *tradeInvoiceUseCase) describe(ctx context.Context, id uuid.UUID) (string, string) {
	if uc.products == nil {
		return "", ""
	}
	p, err := uc.products.GetByID(ctx, id)
	if err != nil || p == nil {
		return "", ""
	}
	return p.Name, p.Unit
}

// computeTradeTotals sets Subtotal, VATAmount and Total. PPN is zero when exempt.
func computeTradeTotals(t *domain.TradeInvoice) {
	sub := 0.0
	for _, l := range t.Lines {
		sub += l.Subtotal
	}
	t.Subtotal = round2(sub)
	t.VATAmount = 0
	if !t.VATExempt {
		t.VATAmount = round2(t.Subtotal * t.VATRate / 100)
	}
	t.Total = round2(t.Subtotal + t.VATAmount)
}

func (uc *tradeInvoiceUseCase) Get(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error) {
	t, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get invoice")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Invoice not found")
	}
	return t, nil
}

func (uc *tradeInvoiceUseCase) List(ctx context.Context, kind string, q types.PaginationQuery) ([]domain.TradeInvoice, types.PaginationMeta, error) {
	q.SetDefaults()
	kind = strings.ToUpper(strings.TrimSpace(kind))
	items, total, err := uc.repo.List(ctx, kind, q)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list invoices")
	}
	return items, types.NewPaginationMeta(total, q.Page, q.PerPage), nil
}

func (uc *tradeInvoiceUseCase) Issue(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error) {
	t, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status != domain.TradeStatusDraft {
		return nil, apperrors.NewConflict("Only a draft invoice can be issued")
	}
	now := time.Now()
	t.Status, t.IssuedAt, t.IssuedBy = domain.TradeStatusIssued, &now, actor.EmailFrom(ctx)
	if err := uc.repo.Update(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to issue invoice")
	}
	return t, nil
}

func (uc *tradeInvoiceUseCase) Cancel(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error) {
	t, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.Status == domain.TradeStatusCancelled {
		return t, nil
	}
	t.Status = domain.TradeStatusCancelled
	if err := uc.repo.Update(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to cancel invoice")
	}
	return t, nil
}
