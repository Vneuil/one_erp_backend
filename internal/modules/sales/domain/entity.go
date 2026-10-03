package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type SalesOrder struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this order to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant -
	// the default state for companies that never created more than their
	// seeded default Tenant.
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	OrderNumber  string     `gorm:"type:varchar(50);not null;index" json:"orderNumber"`
	CustomerName string     `gorm:"type:varchar(255);not null" json:"customerName"`
	TotalAmount  float64    `gorm:"type:decimal(15,2);not null" json:"totalAmount"`
	// Currency/BaseAmount support multi-currency orders (see
	// modules/currency): Currency is the code TotalAmount is denominated
	// in, BaseAmount is TotalAmount converted to the company's base
	// currency using the exchange rate in effect on OrderDate - computed
	// once at creation time so later rate changes never shift a past
	// order's reported value. A company that never uses another currency
	// simply has Currency == base and BaseAmount == TotalAmount always.
	Currency       string  `gorm:"type:varchar(10);default:'IDR'" json:"currency"`
	BaseAmount     float64 `gorm:"type:decimal(15,2);not null;default:0" json:"baseAmount"`
	CreatedByEmail string  `gorm:"type:varchar(255);index" json:"createdByEmail,omitempty"`
	Status         string  `gorm:"type:varchar(50);default:'Confirmed'" json:"status"`
	PaymentStatus  string  `gorm:"type:varchar(50);default:'Unpaid'" json:"paymentStatus"`
	Channel        string  `gorm:"type:varchar(50);default:'Direct B2B'" json:"channel"`
	OrderDate      string  `gorm:"type:varchar(50)" json:"orderDate"`

	// MarketplacePlatform/MarketplaceOrderID identify the source marketplace
	// order when this Sales Order was created by the marketplace sync (see
	// internal/modules/marketplace). Real dedupe (a DB-level unique
	// constraint on platform+marketplace order id) is enforced on the
	// marketplace module's own MarketplaceOrderSync table; these columns are
	// kept here for reporting/traceability on the order itself.
	MarketplacePlatform string `gorm:"type:varchar(30);index" json:"marketplacePlatform,omitempty"`
	MarketplaceOrderID  string `gorm:"type:varchar(150);index" json:"marketplaceOrderId,omitempty"`

	// WarehouseID + Lines are optional: when both are present, CreateOrder
	// deducts stock for each line from this warehouse (see modules/
	// inventory). Nil/empty means "no stock effect" - the behavior every
	// caller had before this field existed (marketplace sync, quick-total
	// order creation, etc. all still work unchanged).
	WarehouseID *uuid.UUID       `gorm:"type:uuid;index" json:"warehouseId,omitempty"`
	Lines       []SalesOrderLine `gorm:"foreignKey:SalesOrderID" json:"lines,omitempty"`
}

func (SalesOrder) TableName() string {
	return "sales_orders"
}

type SalesOrderLine struct {
	types.BaseEntity
	SalesOrderID uuid.UUID `gorm:"type:uuid;not null;index" json:"salesOrderId"`
	ProductID    uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	Quantity     float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
	UnitPrice    float64   `gorm:"type:decimal(15,2);not null" json:"unitPrice"`
	Subtotal     float64   `gorm:"type:decimal(15,2);not null" json:"subtotal"`
}

func (SalesOrderLine) TableName() string {
	return "sales_order_lines"
}

type QuotationLine struct {
	ProductID *uuid.UUID `gorm:"type:uuid;index" json:"productId,omitempty"`
	types.BaseEntity
	QuotationID uuid.UUID `gorm:"type:uuid;not null;index" json:"quotationId"`
	Position    int       `gorm:"not null" json:"position"`
	Description string    `gorm:"type:text;not null" json:"description"`
	Unit        string    `gorm:"type:varchar(50);not null" json:"unit"`
	Quantity    float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
	UnitPrice   float64   `gorm:"type:decimal(15,2);not null" json:"unitPrice"`
	Subtotal    float64   `gorm:"type:decimal(15,2);not null" json:"subtotal"`
}

func (QuotationLine) TableName() string { return "quotation_lines" }

type Quotation struct {
	LeadID *uuid.UUID      `gorm:"type:uuid;index" json:"leadId,omitempty"`
	Lines  []QuotationLine `gorm:"foreignKey:QuotationID" json:"lines"`
	types.BaseEntity
	CompanyID       *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	QuotationNumber string     `gorm:"type:varchar(50);not null;index" json:"quotationNumber"`
	CustomerName    string     `gorm:"type:varchar(255);not null" json:"customerName"`
	TotalAmount     float64    `gorm:"type:decimal(15,2);not null" json:"totalAmount"`
	Status          string     `gorm:"type:varchar(50);default:'Sent'" json:"status"`
	ValidUntil      string     `gorm:"type:varchar(50)" json:"validUntil"`
}

func (Quotation) TableName() string {
	return "quotations"
}

type Invoice struct {
	types.BaseEntity
	CompanyID     *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	InvoiceNumber string     `gorm:"type:varchar(50);not null;index" json:"invoiceNumber"`
	CustomerName  string     `gorm:"type:varchar(255);not null" json:"customerName"`
	InvoiceDate   string     `gorm:"type:varchar(50)" json:"invoiceDate"`
	DueDate       string     `gorm:"type:varchar(50)" json:"dueDate"`
	TotalAmount   float64    `gorm:"type:decimal(15,2);not null" json:"totalAmount"`
	// Subtotal is the amount before adjustments; TotalAmount is the final
	// billed total = Subtotal - DiscountAmount + AdditionalCost + RoundingAmount.
	// Legacy rows have all four adjustment columns at 0 and TotalAmount alone.
	Subtotal       float64 `gorm:"type:decimal(15,2);default:0" json:"subtotal"`
	DiscountAmount float64 `gorm:"type:decimal(15,2);default:0" json:"discountAmount"`
	AdditionalCost float64 `gorm:"type:decimal(15,2);default:0" json:"additionalCost"`
	RoundingAmount float64 `gorm:"type:decimal(15,2);default:0" json:"roundingAmount"`
	PaidAmount     float64 `gorm:"type:decimal(15,2);not null;default:0" json:"paidAmount"`
	Status         string  `gorm:"type:varchar(50);default:'pending'" json:"status"`
	// SalesOrderID optionally links this invoice back to the Sales Order it
	// was billed against. Nil for invoices that predate this field or that
	// were never tied to a specific order - both remain fully valid.
	SalesOrderID *uuid.UUID `gorm:"type:uuid;index" json:"salesOrderId,omitempty"`
}

func (Invoice) TableName() string {
	return "sales_invoices"
}

type Delivery struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	// SalesOrderID is the validated FK to the originating Sales Order -
	// picked from a real dropdown on the frontend, never typed free-text.
	// SONumber is kept in sync from the order for display/backwards
	// compatibility with older deliveries that predate this field. One
	// Sales Order may have many Deliveries (partial shipments) - see
	// usecase.CreateDelivery, which validates each line's quantity against
	// what the order still has left to ship across every prior Delivery.
	SalesOrderID   *uuid.UUID `gorm:"type:uuid;index" json:"salesOrderId,omitempty"`
	DeliveryNumber string     `gorm:"type:varchar(50);not null;index" json:"deliveryNumber"`
	SONumber       string     `gorm:"type:varchar(50)" json:"soNumber"`
	CustomerName   string     `gorm:"type:varchar(255);not null" json:"customerName"`
	DeliveryDate   string     `gorm:"type:varchar(50)" json:"deliveryDate"`
	Carrier        string     `gorm:"type:varchar(255)" json:"carrier"`
	Status         string     `gorm:"type:varchar(50);default:'processing'" json:"status"`
	TrackingNumber string     `gorm:"type:varchar(100)" json:"trackingNumber,omitempty"`
	// ShippingCost is what the carrier charged the company for this delivery. It
	// is set once and booked as a delivery expense; ShippingPaid says whether it
	// was settled in cash (true) or left owing to the carrier (false).
	ShippingCost float64        `gorm:"type:decimal(15,2);default:0" json:"shippingCost"`
	ShippingPaid bool           `gorm:"default:false" json:"shippingPaid"`
	Lines        []DeliveryLine `gorm:"foreignKey:DeliveryID" json:"lines,omitempty"`
}

func (Delivery) TableName() string {
	return "sales_deliveries"
}

// DeliveryLine records how much of one Sales Order line this particular
// Delivery ships. A Sales Order line's total shipped quantity is the sum of
// its DeliveryLine rows across every Delivery for that order.
type DeliveryLine struct {
	types.BaseEntity
	DeliveryID uuid.UUID `gorm:"type:uuid;not null;index" json:"deliveryId"`
	ProductID  uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	Quantity   float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
}

func (DeliveryLine) TableName() string {
	return "sales_delivery_lines"
}

// SalesDownPayment is an advance payment collected from a customer against a
// Sales Order, recorded before the final Sales Invoice. Mirrors
// procurement's PurchaseDownPayment for symmetry between the two flows.
type SalesDownPayment struct {
	types.BaseEntity
	CompanyID     *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DPNumber      string     `gorm:"type:varchar(50);not null;index" json:"dpNumber"`
	SalesOrderID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"salesOrderId"`
	CustomerName  string     `gorm:"type:varchar(255);not null" json:"customerName"`
	Amount        float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	AppliedAmount float64    `gorm:"type:decimal(15,2);default:0" json:"appliedAmount"`
	PaymentDate   string     `gorm:"type:varchar(50)" json:"paymentDate"`
	PaymentMethod string     `gorm:"type:varchar(50)" json:"paymentMethod"`
	Status        string     `gorm:"type:varchar(50);default:'unapplied'" json:"status"` // unapplied | partially_applied | applied
	Notes         string     `gorm:"type:varchar(255)" json:"notes,omitempty"`
}

func (SalesDownPayment) TableName() string {
	return "sales_down_payments"
}

// SalesReturn is a return of previously delivered goods (see Delivery) back
// from the customer - damaged, wrong items, etc.
type SalesReturn struct {
	types.BaseEntity
	CompanyID    *uuid.UUID        `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID     *uuid.UUID        `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ReturnNo     string            `gorm:"type:varchar(50);not null;index" json:"returnNo"`
	DeliveryID   uuid.UUID         `gorm:"type:uuid;not null;index" json:"deliveryId"`
	CustomerName string            `gorm:"type:varchar(255);not null" json:"customerName"`
	ReturnDate   string            `gorm:"type:varchar(50)" json:"returnDate"`
	Reason       string            `gorm:"type:varchar(255)" json:"reason"`
	Status       string            `gorm:"type:varchar(50);default:'draft'" json:"status"` // draft | completed
	TotalAmount  float64           `gorm:"type:decimal(15,2);default:0" json:"totalAmount"`
	Lines        []SalesReturnLine `gorm:"foreignKey:SalesReturnID" json:"lines,omitempty"`
}

func (SalesReturn) TableName() string {
	return "sales_returns"
}

type SalesReturnLine struct {
	types.BaseEntity
	SalesReturnID uuid.UUID `gorm:"type:uuid;not null;index" json:"salesReturnId"`
	ProductID     uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	Quantity      float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
	UnitPrice     float64   `gorm:"type:decimal(15,2);default:0" json:"unitPrice"`
	Subtotal      float64   `gorm:"type:decimal(15,2);default:0" json:"subtotal"`
}

func (SalesReturnLine) TableName() string {
	return "sales_return_lines"
}

// BillingTerm is one instalment of a Sales Order's billing schedule (termin):
// a share of the order to be invoiced by a due date. Invoicing a term creates a
// sub-invoice linked to the order and records it here.
type BillingTerm struct {
	types.BaseEntity
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	SalesOrderID uuid.UUID  `gorm:"type:uuid;not null;index" json:"salesOrderId"`
	Seq          int        `gorm:"not null" json:"seq"`
	Label        string     `gorm:"type:varchar(100);not null" json:"label"`
	Percent      float64    `gorm:"type:decimal(6,3);not null" json:"percent"`
	Amount       float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	DueDate      string     `gorm:"type:varchar(10)" json:"dueDate"`
	InvoiceID    *uuid.UUID `gorm:"type:uuid;index" json:"invoiceId,omitempty"`
	Status       string     `gorm:"type:varchar(20);not null;default:'scheduled'" json:"status"` // scheduled | invoiced
}

func (BillingTerm) TableName() string { return "sales_billing_terms" }

type SalesRepository interface {
	// Billing schedule (termin)
	ReplaceBillingTerms(ctx context.Context, orderID uuid.UUID, terms []BillingTerm) error
	ListBillingTerms(ctx context.Context, orderID uuid.UUID) ([]BillingTerm, error)
	GetBillingTerm(ctx context.Context, id uuid.UUID) (*BillingTerm, error)
	UpdateBillingTerm(ctx context.Context, t *BillingTerm) error

	CreateOrder(ctx context.Context, order *SalesOrder) error
	GetOrderByID(ctx context.Context, id uuid.UUID) (*SalesOrder, error)
	UpdateOrder(ctx context.Context, order *SalesOrder) error
	// UpdateOrderWithLines is UpdateOrder plus a full replace of the order's
	// line items (delete then recreate), for editing an order's contents -
	// UpdateOrder alone leaves stale/removed line rows behind since a plain
	// Save never deletes association rows missing from the slice.
	UpdateOrderWithLines(ctx context.Context, order *SalesOrder) error
	ListOrders(ctx context.Context, query types.PaginationQuery) ([]SalesOrder, int64, error)
	CountOrders(ctx context.Context) (int64, error)

	CreateQuotation(ctx context.Context, quo *Quotation) error
	GetQuotationByID(ctx context.Context, id uuid.UUID) (*Quotation, error)
	UpdateQuotation(ctx context.Context, quo *Quotation) error
	ListQuotations(ctx context.Context, query types.PaginationQuery) ([]Quotation, int64, error)

	CreateInvoice(ctx context.Context, inv *Invoice) error
	GetInvoiceByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
	ListInvoices(ctx context.Context, query types.PaginationQuery) ([]Invoice, int64, error)
	UpdateInvoice(ctx context.Context, inv *Invoice) error

	CreateDelivery(ctx context.Context, d *Delivery) error
	GetDeliveryByID(ctx context.Context, id uuid.UUID) (*Delivery, error)
	UpdateDelivery(ctx context.Context, d *Delivery) error
	ListDeliveries(ctx context.Context, query types.PaginationQuery) ([]Delivery, int64, error)
	// ListDeliveriesByOrderID returns every Delivery (with lines) already
	// recorded against one Sales Order, so CreateDelivery can sum what has
	// already shipped per product before accepting a new one.
	ListDeliveriesByOrderID(ctx context.Context, orderID uuid.UUID) ([]Delivery, error)

	// Sales down payments
	CreateDownPayment(ctx context.Context, dp *SalesDownPayment) error
	GetDownPaymentByID(ctx context.Context, id uuid.UUID) (*SalesDownPayment, error)
	UpdateDownPayment(ctx context.Context, dp *SalesDownPayment) error
	ListDownPayments(ctx context.Context, query types.PaginationQuery) ([]SalesDownPayment, int64, error)

	// Sales returns
	CreateSalesReturn(ctx context.Context, r *SalesReturn) error
	GetSalesReturnByID(ctx context.Context, id uuid.UUID) (*SalesReturn, error)
	ListSalesReturns(ctx context.Context, query types.PaginationQuery) ([]SalesReturn, int64, error)
}
