package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/google/uuid"
)

type SalesOrderLineDTO struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
}

type CreateSalesOrderDTO struct {
	OrderNumber   string  `json:"orderNumber"`
	CustomerName  string  `json:"customerName"`
	TotalAmount   float64 `json:"totalAmount"`
	Currency      string  `json:"currency"`
	Channel       string  `json:"channel"`
	PaymentStatus string  `json:"paymentStatus"`
	Status        string  `json:"status"`
	// WarehouseID + Lines are optional. When both are set, stock for each
	// line is deducted from this warehouse - see usecase.CreateOrder.
	WarehouseID *uuid.UUID          `json:"warehouseId,omitempty"`
	Lines       []SalesOrderLineDTO `json:"lines,omitempty"`
	// MarketplacePlatform/MarketplaceOrderID are set only by the marketplace
	// sync use case (see internal/modules/marketplace) to tag orders created
	// from a synced marketplace order. Left empty for normal orders.
	MarketplacePlatform string `json:"marketplacePlatform,omitempty"`
	MarketplaceOrderID  string `json:"marketplaceOrderId,omitempty"`
}

// UpdateSalesOrderDTO carries the editable fields of an existing Sales
// Order: customer name and line items. Stock is not re-adjusted when lines
// change - see usecase.UpdateOrder for why - so this is meant for orders
// whose stock effect either never happened (quotation conversions, which
// carry no warehouse) or is corrected separately.
type UpdateSalesOrderDTO struct {
	CustomerName string              `json:"customerName"`
	Lines        []SalesOrderLineDTO `json:"lines"`
}

// SalesApprovalActionDTO carries the acting user's identity when approving
// or rejecting a Sales Order that may be routed through modules/approval,
// mirroring procurement's ApprovalActionDTO.
type SalesApprovalActionDTO struct {
	ActorName string `json:"actorName"`
	ActorRole string `json:"actorRole"`
	Comments  string `json:"comments"`
}

type SalesOrderLineResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
	Subtotal  float64   `json:"subtotal"`
}

type SalesOrderResponseDTO struct {
	ID             uuid.UUID                   `json:"id"`
	OrderNumber    string                      `json:"orderNumber"`
	CustomerName   string                      `json:"customerName"`
	TotalAmount    float64                     `json:"totalAmount"`
	Currency       string                      `json:"currency"`
	BaseAmount     float64                     `json:"baseAmount"`
	Status         string                      `json:"status"`
	CreatedByEmail string                      `json:"createdByEmail,omitempty"`
	PaymentStatus  string                      `json:"paymentStatus"`
	Channel        string                      `json:"channel"`
	OrderDate      string                      `json:"orderDate"`
	WarehouseID    *uuid.UUID                  `json:"warehouseId,omitempty"`
	Lines          []SalesOrderLineResponseDTO `json:"lines,omitempty"`
	CreatedAt      time.Time                   `json:"createdAt"`
}

type QuotationLineDTO struct {
	ProductID   *uuid.UUID `json:"productId"`
	Description string     `json:"description"`
	Unit        string     `json:"unit"`
	Quantity    float64    `json:"quantity"`
	UnitPrice   float64    `json:"unitPrice"`
}

type CreateQuotationDTO struct {
	LeadID          *uuid.UUID         `json:"leadId"`
	Lines           []QuotationLineDTO `json:"lines"`
	QuotationNumber string             `json:"quotationNumber"`
	CustomerName    string             `json:"customerName"`
	TotalAmount     float64            `json:"totalAmount"`
	ValidUntil      string             `json:"validUntil"`
}

// UpdateQuotationDTO carries the editable fields of an existing, not-yet-converted
// Quotation. LeadID is intentionally omitted - the linked CRM lead cannot be
// changed after creation, only the customer's items and validity.
type UpdateQuotationDTO struct {
	Lines        []QuotationLineDTO `json:"lines"`
	CustomerName string             `json:"customerName"`
	ValidUntil   string             `json:"validUntil"`
}

type QuotationResponseDTO struct {
	LeadID          *uuid.UUID             `json:"leadId,omitempty"`
	Lines           []domain.QuotationLine `json:"lines"`
	ID              uuid.UUID              `json:"id"`
	QuotationNumber string                 `json:"quotationNumber"`
	CustomerName    string                 `json:"customerName"`
	TotalAmount     float64                `json:"totalAmount"`
	Status          string                 `json:"status"`
	ValidUntil      string                 `json:"validUntil"`
	CreatedAt       time.Time              `json:"createdAt"`
}

func ToSalesOrderResponse(o *domain.SalesOrder) *SalesOrderResponseDTO {
	if o == nil {
		return nil
	}
	lines := make([]SalesOrderLineResponseDTO, len(o.Lines))
	for i, l := range o.Lines {
		lines[i] = SalesOrderLineResponseDTO{ID: l.ID, ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice, Subtotal: l.Subtotal}
	}
	return &SalesOrderResponseDTO{
		ID:             o.ID,
		OrderNumber:    o.OrderNumber,
		CustomerName:   o.CustomerName,
		TotalAmount:    o.TotalAmount,
		Currency:       o.Currency,
		BaseAmount:     o.BaseAmount,
		Status:         o.Status,
		CreatedByEmail: o.CreatedByEmail,
		PaymentStatus:  o.PaymentStatus,
		Channel:        o.Channel,
		OrderDate:      o.OrderDate,
		WarehouseID:    o.WarehouseID,
		Lines:          lines,
		CreatedAt:      o.CreatedAt,
	}
}

func ToSalesOrderResponseList(orders []domain.SalesOrder) []SalesOrderResponseDTO {
	result := make([]SalesOrderResponseDTO, len(orders))
	for i, o := range orders {
		result[i] = *ToSalesOrderResponse(&o)
	}
	return result
}

func ToQuotationResponse(q *domain.Quotation) *QuotationResponseDTO {
	if q == nil {
		return nil
	}
	return &QuotationResponseDTO{
		ID:              q.ID,
		QuotationNumber: q.QuotationNumber,
		LeadID:          q.LeadID,
		Lines:           q.Lines,
		CustomerName:    q.CustomerName,
		TotalAmount:     q.TotalAmount,
		Status:          q.Status,
		ValidUntil:      q.ValidUntil,
		CreatedAt:       q.CreatedAt,
	}
}

func ToQuotationResponseList(quotations []domain.Quotation) []QuotationResponseDTO {
	result := make([]QuotationResponseDTO, len(quotations))
	for i, q := range quotations {
		result[i] = *ToQuotationResponse(&q)
	}
	return result
}

type CreateInvoiceDTO struct {
	InvoiceNumber string `json:"invoiceNumber"`
	CustomerName  string `json:"customerName"`
	// TotalAmount is the invoice amount before adjustments. With none of the
	// fields below it is also the final total (the original behaviour).
	TotalAmount     float64 `json:"totalAmount"`
	DiscountPercent float64 `json:"discountPercent"`
	DiscountAmount  float64 `json:"discountAmount"`
	AdditionalCost  float64 `json:"additionalCost"`
	// RoundTo rounds the final total to this step (e.g. 100 or 1000); 0 = off.
	RoundTo float64 `json:"roundTo"`
	DueDate string  `json:"dueDate"`
	// SalesOrderID is optional - when supplied it must reference a real
	// Sales Order, but an invoice can still be created without one so
	// existing callers/invoices keep working unchanged.
	SalesOrderID *uuid.UUID `json:"salesOrderId,omitempty"`
}

type RecordSalesPaymentDTO struct {
	Amount float64 `json:"amount"`
}

type InvoiceResponseDTO struct {
	ID             uuid.UUID  `json:"id"`
	InvoiceNumber  string     `json:"invoiceNumber"`
	CustomerName   string     `json:"customerName"`
	InvoiceDate    string     `json:"invoiceDate"`
	DueDate        string     `json:"dueDate"`
	TotalAmount    float64    `json:"totalAmount"`
	Subtotal       float64    `json:"subtotal"`
	DiscountAmount float64    `json:"discountAmount"`
	AdditionalCost float64    `json:"additionalCost"`
	RoundingAmount float64    `json:"roundingAmount"`
	PaidAmount     float64    `json:"paidAmount"`
	Outstanding    float64    `json:"outstanding"`
	Status         string     `json:"status"`
	SalesOrderID   *uuid.UUID `json:"salesOrderId,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func ToInvoiceResponse(inv *domain.Invoice) *InvoiceResponseDTO {
	if inv == nil {
		return nil
	}
	return &InvoiceResponseDTO{
		ID: inv.ID, InvoiceNumber: inv.InvoiceNumber, CustomerName: inv.CustomerName,
		InvoiceDate: inv.InvoiceDate, DueDate: inv.DueDate, TotalAmount: inv.TotalAmount,
		Subtotal: inv.Subtotal, DiscountAmount: inv.DiscountAmount, AdditionalCost: inv.AdditionalCost, RoundingAmount: inv.RoundingAmount,
		PaidAmount: inv.PaidAmount, Outstanding: inv.TotalAmount - inv.PaidAmount,
		Status: inv.Status, SalesOrderID: inv.SalesOrderID, CreatedAt: inv.CreatedAt,
	}
}

func ToInvoiceResponseList(invoices []domain.Invoice) []InvoiceResponseDTO {
	result := make([]InvoiceResponseDTO, len(invoices))
	for i, inv := range invoices {
		result[i] = *ToInvoiceResponse(&inv)
	}
	return result
}

type DeliveryLineDTO struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
}

type CreateDeliveryDTO struct {
	// SalesOrderID is the real Sales Order this delivery fulfills, picked
	// from a dropdown on the frontend (see soNumber below for legacy
	// free-text support).
	SalesOrderID uuid.UUID `json:"salesOrderId"`
	// SONumber is retained for backwards compatibility with any caller
	// still posting the legacy free-text field, but it is ignored whenever
	// SalesOrderID is supplied - the order's real number is used instead.
	SONumber     string `json:"soNumber"`
	CustomerName string `json:"customerName"`
	Carrier      string `json:"carrier"`
	DeliveryDate string `json:"deliveryDate"`
	// Lines is how much of each of the order's own products this delivery
	// ships. Required - see usecase.CreateDelivery, which rejects any line
	// whose product isn't on the order or whose quantity would push that
	// product's total shipped-to-date over what the order itself has.
	Lines []DeliveryLineDTO `json:"lines"`
}

type DeliveryResponseDTO struct {
	ID             uuid.UUID             `json:"id"`
	DeliveryNumber string                `json:"deliveryNumber"`
	SalesOrderID   *uuid.UUID            `json:"salesOrderId,omitempty"`
	SONumber       string                `json:"soNumber"`
	CustomerName   string                `json:"customerName"`
	DeliveryDate   string                `json:"deliveryDate"`
	Carrier        string                `json:"carrier"`
	Status         string                `json:"status"`
	TrackingNumber string                `json:"trackingNumber,omitempty"`
	ShippingCost   float64               `json:"shippingCost"`
	ShippingPaid   bool                  `json:"shippingPaid"`
	Lines          []domain.DeliveryLine `json:"lines,omitempty"`
	CreatedAt      time.Time             `json:"createdAt"`
}

func ToDeliveryResponse(d *domain.Delivery) *DeliveryResponseDTO {
	if d == nil {
		return nil
	}
	return &DeliveryResponseDTO{
		ID: d.ID, DeliveryNumber: d.DeliveryNumber, SalesOrderID: d.SalesOrderID, SONumber: d.SONumber, CustomerName: d.CustomerName,
		DeliveryDate: d.DeliveryDate, Carrier: d.Carrier, Status: d.Status, TrackingNumber: d.TrackingNumber,
		ShippingCost: d.ShippingCost, ShippingPaid: d.ShippingPaid, Lines: d.Lines, CreatedAt: d.CreatedAt,
	}
}

func ToDeliveryResponseList(deliveries []domain.Delivery) []DeliveryResponseDTO {
	result := make([]DeliveryResponseDTO, len(deliveries))
	for i, d := range deliveries {
		result[i] = *ToDeliveryResponse(&d)
	}
	return result
}

// Sales down payments

type CreateSalesDownPaymentDTO struct {
	SalesOrderID  uuid.UUID `json:"salesOrderId"`
	CustomerName  string    `json:"customerName"`
	Amount        float64   `json:"amount"`
	PaymentDate   string    `json:"paymentDate"`
	PaymentMethod string    `json:"paymentMethod"`
	Notes         string    `json:"notes"`
}

type SalesDownPaymentResponseDTO struct {
	ID               uuid.UUID `json:"id"`
	DPNumber         string    `json:"dpNumber"`
	SalesOrderID     uuid.UUID `json:"salesOrderId"`
	SalesOrderNumber string    `json:"salesOrderNumber,omitempty"`
	CustomerName     string    `json:"customerName"`
	Amount           float64   `json:"amount"`
	AppliedAmount    float64   `json:"appliedAmount"`
	RemainingAmount  float64   `json:"remainingAmount"`
	PaymentDate      string    `json:"paymentDate"`
	PaymentMethod    string    `json:"paymentMethod"`
	Status           string    `json:"status"`
	Notes            string    `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

func ToSalesDownPaymentResponse(dp *domain.SalesDownPayment, soNumber string) *SalesDownPaymentResponseDTO {
	if dp == nil {
		return nil
	}
	return &SalesDownPaymentResponseDTO{
		ID: dp.ID, DPNumber: dp.DPNumber, SalesOrderID: dp.SalesOrderID, SalesOrderNumber: soNumber,
		CustomerName: dp.CustomerName, Amount: dp.Amount, AppliedAmount: dp.AppliedAmount,
		RemainingAmount: dp.Amount - dp.AppliedAmount, PaymentDate: dp.PaymentDate, PaymentMethod: dp.PaymentMethod,
		Status: dp.Status, Notes: dp.Notes, CreatedAt: dp.CreatedAt,
	}
}

// ApplySalesDownPaymentDTO applies a down payment to a target invoice,
// reducing its outstanding balance by the down payment's amount.
type ApplySalesDownPaymentDTO struct {
	InvoiceID uuid.UUID `json:"invoiceId"`
}

func ToSalesDownPaymentResponseList(items []domain.SalesDownPayment, soNumbers map[uuid.UUID]string) []SalesDownPaymentResponseDTO {
	result := make([]SalesDownPaymentResponseDTO, len(items))
	for i, dp := range items {
		result[i] = *ToSalesDownPaymentResponse(&dp, soNumbers[dp.SalesOrderID])
	}
	return result
}

// Sales returns

type SalesReturnLineDTO struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
}

type CreateSalesReturnDTO struct {
	DeliveryID uuid.UUID            `json:"deliveryId"`
	ReturnDate string               `json:"returnDate"`
	Reason     string               `json:"reason"`
	Lines      []SalesReturnLineDTO `json:"lines"`
}

type SalesReturnLineResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"productId"`
	Quantity  float64   `json:"quantity"`
	UnitPrice float64   `json:"unitPrice"`
	Subtotal  float64   `json:"subtotal"`
}

type SalesReturnResponseDTO struct {
	ID           uuid.UUID                    `json:"id"`
	ReturnNo     string                       `json:"returnNo"`
	DeliveryID   uuid.UUID                    `json:"deliveryId"`
	CustomerName string                       `json:"customerName"`
	ReturnDate   string                       `json:"returnDate"`
	Reason       string                       `json:"reason"`
	Status       string                       `json:"status"`
	TotalAmount  float64                      `json:"totalAmount"`
	Lines        []SalesReturnLineResponseDTO `json:"lines,omitempty"`
	CreatedAt    time.Time                    `json:"createdAt"`
}

func ToSalesReturnResponse(r *domain.SalesReturn) *SalesReturnResponseDTO {
	if r == nil {
		return nil
	}
	lines := make([]SalesReturnLineResponseDTO, len(r.Lines))
	for i, l := range r.Lines {
		lines[i] = SalesReturnLineResponseDTO{
			ID: l.ID, ProductID: l.ProductID, Quantity: l.Quantity, UnitPrice: l.UnitPrice, Subtotal: l.Subtotal,
		}
	}
	return &SalesReturnResponseDTO{
		ID: r.ID, ReturnNo: r.ReturnNo, DeliveryID: r.DeliveryID, CustomerName: r.CustomerName,
		ReturnDate: r.ReturnDate, Reason: r.Reason, Status: r.Status, TotalAmount: r.TotalAmount,
		Lines: lines, CreatedAt: r.CreatedAt,
	}
}

func ToSalesReturnResponseList(items []domain.SalesReturn) []SalesReturnResponseDTO {
	result := make([]SalesReturnResponseDTO, len(items))
	for i, r := range items {
		result[i] = *ToSalesReturnResponse(&r)
	}
	return result
}

// ShippingCostDTO records what the carrier charged for a delivery.
type ShippingCostDTO struct {
	Amount         float64 `json:"amount"`
	Carrier        string  `json:"carrier"`
	TrackingNumber string  `json:"trackingNumber"`
	// PaidNow books the cost against cash; otherwise it is owed to the carrier.
	PaidNow bool `json:"paidNow"`
}
