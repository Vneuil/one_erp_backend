package application

import (
	"context"
	"encoding/json"
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

func (l *TradeLineDTO) UnmarshalJSON(data []byte) error {
	type Alias TradeLineDTO
	aux := struct {
		ProductID *string `json:"productId,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(l),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ProductID != nil && strings.TrimSpace(*aux.ProductID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*aux.ProductID))
		if err != nil {
			return fmt.Errorf("invalid productId: %w", err)
		}
		l.ProductID = &id
	} else {
		l.ProductID = nil
	}
	return nil
}

// CreateTradeInvoiceDTO creates a PI/CI. At most one source may be given: a Sales
// Order, a Quotation, or (for a CI) the PI it is made from. With no source the
// document is raised directly from the customer's incoming PO (the usual PI flow:
// PO Masuk -> PI -> DP -> Sales Order), and customerName + lines are required.
// Lines, when supplied, override the ones copied from the source.
type CreateTradeInvoiceDTO struct {
	Kind                 string         `json:"kind"` // PI | CI
	SalesOrderID         *uuid.UUID     `json:"salesOrderId,omitempty"`
	QuotationID          *uuid.UUID     `json:"quotationId,omitempty"`
	QuotationNumber      string         `json:"quotationNumber,omitempty"`
	CustomerPOID         *uuid.UUID     `json:"customerPoId,omitempty"`
	ParentID             *uuid.UUID     `json:"parentId,omitempty"`
	CustomerName         string         `json:"customerName,omitempty"`
	CustomerPONumber     string         `json:"customerPoNumber,omitempty"`
	CustomerPOAttachment string         `json:"customerPoAttachment,omitempty"`
	DPPercent            float64        `json:"dpPercent,omitempty"`
	BillingAddress       string         `json:"billingAddress,omitempty"`
	ShippingAddress      string         `json:"shippingAddress,omitempty"`
	Currency             string         `json:"currency,omitempty"`
	PaymentTermID        *uuid.UUID     `json:"paymentTermId,omitempty"`
	PaymentTerm          string         `json:"paymentTerm,omitempty"`
	TaxRateID            *uuid.UUID     `json:"taxRateId,omitempty"`
	VATRate              *float64       `json:"vatRate,omitempty"`
	Incoterm             string         `json:"incoterm,omitempty"`
	Notes                string         `json:"notes,omitempty"`
	VATExempt            *bool          `json:"vatExempt,omitempty"`
	Lines                []TradeLineDTO `json:"lines,omitempty"`
}

func (dto *CreateTradeInvoiceDTO) UnmarshalJSON(data []byte) error {
	type Alias CreateTradeInvoiceDTO
	aux := struct {
		SalesOrderID  *string `json:"salesOrderId,omitempty"`
		QuotationID   *string `json:"quotationId,omitempty"`
		CustomerPOID  *string `json:"customerPoId,omitempty"`
		ParentID      *string `json:"parentId,omitempty"`
		PaymentTermID *string `json:"paymentTermId,omitempty"`
		TaxRateID     *string `json:"taxRateId,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(dto),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	parseOptionalUUID := func(s *string) (*uuid.UUID, error) {
		if s == nil || strings.TrimSpace(*s) == "" {
			return nil, nil
		}
		id, err := uuid.Parse(strings.TrimSpace(*s))
		if err != nil {
			return nil, err
		}
		return &id, nil
	}
	var err error
	if dto.SalesOrderID, err = parseOptionalUUID(aux.SalesOrderID); err != nil {
		return fmt.Errorf("invalid salesOrderId: %w", err)
	}
	if dto.QuotationID, err = parseOptionalUUID(aux.QuotationID); err != nil {
		return fmt.Errorf("invalid quotationId: %w", err)
	}
	if dto.CustomerPOID, err = parseOptionalUUID(aux.CustomerPOID); err != nil {
		return fmt.Errorf("invalid customerPoId: %w", err)
	}
	if dto.ParentID, err = parseOptionalUUID(aux.ParentID); err != nil {
		return fmt.Errorf("invalid parentId: %w", err)
	}
	if dto.PaymentTermID, err = parseOptionalUUID(aux.PaymentTermID); err != nil {
		return fmt.Errorf("invalid paymentTermId: %w", err)
	}
	if dto.TaxRateID, err = parseOptionalUUID(aux.TaxRateID); err != nil {
		return fmt.Errorf("invalid taxRateId: %w", err)
	}
	return nil
}

type TradeInvoiceUseCase interface {
	Create(ctx context.Context, dto CreateTradeInvoiceDTO) (*domain.TradeInvoice, error)
	Get(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error)
	List(ctx context.Context, kind string, q types.PaginationQuery) ([]domain.TradeInvoice, types.PaginationMeta, error)
	Issue(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error)
	Cancel(ctx context.Context, id uuid.UUID) (*domain.TradeInvoice, error)
	// LinkOrder records the Sales Order that was created from this PI.
	LinkOrder(ctx context.Context, id, orderID uuid.UUID) (*domain.TradeInvoice, error)
}

// OrderFromPIDTO are the choices made when a Sales Order is created from a PI.
type OrderFromPIDTO struct {
	WarehouseID     *uuid.UUID `json:"warehouseId,omitempty"`
	DPPaid          bool       `json:"dpPaid"` // false = "DP menyusul": production may start, DP is ticked later
	DeliveryDate    string     `json:"deliveryDate,omitempty"`
	ShippingCourier string     `json:"shippingCourier,omitempty"`
}

// OrderFromPI builds the Sales Order a PI turns into: same customer, items, prices,
// PO reference, PPN setting and DP terms. The PI must be issued, be a PI, not yet be
// linked to an order, and every line must point at a master product.
func OrderFromPI(pi *domain.TradeInvoice, dto OrderFromPIDTO) (CreateSalesOrderDTO, error) {
	if pi.Kind != domain.TradeKindPI {
		return CreateSalesOrderDTO{}, apperrors.NewBadRequest("Only a Proforma Invoice can be turned into a Sales Order")
	}
	if pi.Status != domain.TradeStatusIssued {
		return CreateSalesOrderDTO{}, apperrors.NewConflict("Issue the proforma invoice before creating a Sales Order from it")
	}
	if pi.SalesOrderID != nil {
		return CreateSalesOrderDTO{}, apperrors.NewConflict("This proforma invoice already has a Sales Order")
	}
	lines := make([]SalesOrderLineDTO, 0, len(pi.Lines))
	for _, l := range pi.Lines {
		if l.ProductID == nil {
			return CreateSalesOrderDTO{}, apperrors.NewBadRequest("Every PI line must reference a master product before a Sales Order can be created")
		}
		lines = append(lines, SalesOrderLineDTO{ProductID: *l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
	}
	// The PI itself is the outgoing document; keep its number as the reference when
	// the customer sent no PO number.
	poNumber := pi.CustomerPONumber
	if poNumber == "" {
		poNumber = pi.Number
	}
	return CreateSalesOrderDTO{
		CustomerName: pi.CustomerName, TotalAmount: pi.Total, Currency: pi.Currency,
		CustomerPONumber: poNumber, CustomerPOAttachment: pi.CustomerPOAttachment,
		PaymentTermID: pi.PaymentTermID, PaymentTerm: pi.PaymentTerm,
		TaxRateID: pi.TaxRateID, VATRate: pi.VATRate, VATAmount: pi.VATAmount,
		DPPaid: dto.DPPaid, DPPercent: pi.DPPercent, DPAmount: pi.DPAmount,
		ShippingAddress: pi.ShippingAddr, ShippingCourier: dto.ShippingCourier, DeliveryDate: dto.DeliveryDate,
		VATExempt: pi.VATExempt, WarehouseID: dto.WarehouseID, Lines: lines, Status: "draft",
	}, nil
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
	if sources > 1 {
		return nil, apperrors.NewBadRequest("Provide at most one source: salesOrderId, quotationId or parentId")
	}
	if dto.DPPercent < 0 || dto.DPPercent > 100 {
		return nil, apperrors.NewBadRequest("dpPercent must be between 0 and 100")
	}

	vatRate := DefaultTradeVATRate
	if dto.VATRate != nil && *dto.VATRate >= 0 {
		vatRate = *dto.VATRate
	}

	t := &domain.TradeInvoice{
		Kind: kind, Status: domain.TradeStatusDraft, IssueDate: time.Now().Format("2006-01-02"),
		CustomerName: strings.TrimSpace(dto.CustomerName), BillingAddr: dto.BillingAddress, ShippingAddr: dto.ShippingAddress,
		Currency: dto.Currency, PaymentTermID: dto.PaymentTermID, PaymentTerm: dto.PaymentTerm,
		TaxRateID: dto.TaxRateID, Incoterm: dto.Incoterm, Notes: dto.Notes,
		VATRate: vatRate, CreatedBy: actor.EmailFrom(ctx),
		CustomerPONumber: strings.TrimSpace(dto.CustomerPONumber), CustomerPOAttachment: strings.TrimSpace(dto.CustomerPOAttachment),
		DPPercent:    dto.DPPercent,
		SalesOrderID: dto.SalesOrderID, QuotationID: dto.QuotationID, QuotationNumber: strings.TrimSpace(dto.QuotationNumber),
		CustomerPOID: dto.CustomerPOID, ParentID: dto.ParentID,
	}
	var lines []domain.TradeInvoiceLine

	// If direct CustomerPO reference is passed
	if dto.CustomerPOID != nil {
		if po, err := uc.sales.GetCustomerPOByID(ctx, *dto.CustomerPOID); err == nil && po != nil {
			if t.CustomerPONumber == "" {
				t.CustomerPONumber = po.PoNumber
			}
			if t.CustomerPOAttachment == "" {
				t.CustomerPOAttachment = po.Attachment
			}
			if t.CustomerName == "" {
				t.CustomerName = po.CustomerName
			}
			if t.Currency == "" {
				t.Currency = po.Currency
			}
			if len(dto.Lines) == 0 && len(po.Lines) > 0 {
				for _, l := range po.Lines {
					lines = append(lines, domain.TradeInvoiceLine{
						ProductID:   l.ProductID,
						Description: l.ProductName,
						Unit:        l.Unit,
						Quantity:    l.Quantity,
						UnitPrice:   l.UnitPrice,
					})
				}
			}
		}
	}

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
		if t.CustomerPONumber == "" {
			t.CustomerPONumber = o.CustomerPONumber
		}
		if t.CustomerPOAttachment == "" {
			t.CustomerPOAttachment = o.CustomerPOAttachment
		}
		if t.DPPercent == 0 {
			t.DPPercent = o.DPPercent
		}
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
		t.QuotationID = dto.QuotationID
		t.QuotationNumber = q.QuotationNumber
		if q.CustomerPOID != nil {
			t.CustomerPOID = q.CustomerPOID
			t.CustomerPONumber = q.CustomerPONumber
		} else if q.CustomerPONumber != "" && t.CustomerPONumber == "" {
			t.CustomerPONumber = q.CustomerPONumber
		}
		if t.CustomerName == "" {
			t.CustomerName = q.CustomerName
		}
		if t.Currency == "" {
			t.Currency = q.Currency
		}
		for _, l := range q.Lines {
			lines = append(lines, domain.TradeInvoiceLine{ProductID: l.ProductID, Description: l.Description, Unit: l.Unit, Quantity: l.Quantity, UnitPrice: l.UnitPrice})
		}
		q.Status = "Converted"
		_ = uc.sales.UpdateQuotation(ctx, q)
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
		if t.CustomerPONumber == "" {
			t.CustomerPONumber = p.CustomerPONumber
		}
		if t.CustomerPOAttachment == "" {
			t.CustomerPOAttachment = p.CustomerPOAttachment
		}
		if t.DPPercent == 0 {
			t.DPPercent = p.DPPercent
		}
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
	t.DPAmount = round2(t.Total * t.DPPercent / 100)

	count, err := uc.repo.CountByKind(ctx, kind)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to number the invoice")
	}
	t.Number = fmt.Sprintf("%s-%s-%04d", kind, time.Now().Format("200601"), count+1)

	if err := uc.repo.Create(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create invoice")
	}

	// Synchronize with Customer PO: mark status as Agreed and store generated PI number & ID
	if t.CustomerPOID != nil {
		if po, err := uc.sales.GetCustomerPOByID(ctx, *t.CustomerPOID); err == nil && po != nil {
			po.Status = "Agreed"
			po.TradeInvoiceID = &t.ID
			po.TradeInvoiceNumber = t.Number
			if t.QuotationID != nil && po.QuotationID == nil {
				po.QuotationID = t.QuotationID
				po.QuotationNumber = t.QuotationNumber
			}
			_ = uc.sales.UpdateCustomerPO(ctx, po)
		}
	} else if t.CustomerPONumber != "" {
		pos, _, err := uc.sales.ListCustomerPOs(ctx, types.PaginationQuery{Search: t.CustomerPONumber, Page: 1, PerPage: 10})
		if err == nil {
			for _, po := range pos {
				if strings.EqualFold(strings.TrimSpace(po.PoNumber), t.CustomerPONumber) {
					po.Status = "Agreed"
					po.TradeInvoiceID = &t.ID
					po.TradeInvoiceNumber = t.Number
					if t.QuotationID != nil && po.QuotationID == nil {
						po.QuotationID = t.QuotationID
						po.QuotationNumber = t.QuotationNumber
					}
					_ = uc.sales.UpdateCustomerPO(ctx, &po)
					break
				}
			}
		}
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

func (uc *tradeInvoiceUseCase) LinkOrder(ctx context.Context, id, orderID uuid.UUID) (*domain.TradeInvoice, error) {
	t, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	t.SalesOrderID = &orderID
	if err := uc.repo.Update(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to link the Sales Order to the invoice")
	}
	return t, nil
}
