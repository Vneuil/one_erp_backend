package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// PurchaseRequest is an internal request to procure goods
type PurchaseRequest struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this request to one business unit within the company
	TenantID     *uuid.UUID            `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	RequestNo    string                `gorm:"type:varchar(50);not null;index" json:"requestNo"`
	RequestedBy  string                `gorm:"type:varchar(150);not null" json:"requestedBy"`
	Department   string                `gorm:"type:varchar(150)" json:"department"`
	NeededByDate string                `gorm:"type:varchar(50)" json:"neededByDate"`
	Status       string                `gorm:"type:varchar(50);default:'draft'" json:"status"`
	Lines        []PurchaseRequestLine `gorm:"foreignKey:PurchaseRequestID" json:"lines,omitempty"`
}

func (PurchaseRequest) TableName() string {
	return "procurement_purchase_requests"
}

type PurchaseRequestLine struct {
	types.BaseEntity
	PurchaseRequestID uuid.UUID `gorm:"type:uuid;not null;index" json:"purchaseRequestId"`
	ProductID         uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	Quantity          float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
	Notes             string    `gorm:"type:varchar(255)" json:"notes"`
}

func (PurchaseRequestLine) TableName() string {
	return "procurement_purchase_request_lines"
}

// PurchaseOrder is issued to a supplier, optionally originating from a PR
type PurchaseOrder struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this order to one business unit within the company
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	OrderNo           string     `gorm:"type:varchar(50);not null;index" json:"orderNo"`
	SupplierID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"supplierId"`
	SupplierName      string     `gorm:"type:varchar(255)" json:"supplierName"`
	PurchaseRequestID *uuid.UUID `gorm:"type:uuid;index" json:"purchaseRequestId,omitempty"`
	OrderDate         string     `gorm:"type:varchar(50)" json:"orderDate"`
	ExpectedDate      string     `gorm:"type:varchar(50)" json:"expectedDate"`
	Status            string     `gorm:"type:varchar(50);default:'draft'" json:"status"`
	TotalAmount       float64    `gorm:"type:decimal(15,2);default:0" json:"totalAmount"`
	// Currency/BaseAmount support multi-currency purchase orders (see
	// modules/currency): Currency is the code TotalAmount is denominated
	// in, BaseAmount is TotalAmount converted to the company's base
	// currency (IDR) using the exchange rate as of OrderDate. A PO that
	// never sets a currency simply has Currency == base and BaseAmount ==
	// TotalAmount always.
	// CreatedByEmail is the login email of whoever raised the order; used to stop
	// them approving it themselves.
	CreatedByEmail string              `gorm:"type:varchar(255);index" json:"createdByEmail,omitempty"`
	Currency       string              `gorm:"type:varchar(10);default:'IDR'" json:"currency"`
	BaseAmount     float64             `gorm:"type:decimal(15,2);not null;default:0" json:"baseAmount"`
	Lines          []PurchaseOrderLine `gorm:"foreignKey:PurchaseOrderID" json:"lines,omitempty"`
}

func (PurchaseOrder) TableName() string {
	return "procurement_purchase_orders"
}

type PurchaseOrderLine struct {
	types.BaseEntity
	PurchaseOrderID  uuid.UUID `gorm:"type:uuid;not null;index" json:"purchaseOrderId"`
	ProductID        uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	Quantity         float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
	UnitPrice        float64   `gorm:"type:decimal(15,2);not null" json:"unitPrice"`
	Subtotal         float64   `gorm:"type:decimal(15,2);not null" json:"subtotal"`
	QuantityReceived float64   `gorm:"type:decimal(15,2);default:0" json:"quantityReceived"`
}

func (PurchaseOrderLine) TableName() string {
	return "procurement_purchase_order_lines"
}

// GoodsReceipt records inbound goods against a purchase order
type GoodsReceipt struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this receipt to one business unit within the company
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ReceiptNo       string     `gorm:"type:varchar(50);not null;index" json:"receiptNo"`
	PurchaseOrderID uuid.UUID  `gorm:"type:uuid;not null;index" json:"purchaseOrderId"`
	ReceivedDate    string     `gorm:"type:varchar(50)" json:"receivedDate"`
	WarehouseID     *uuid.UUID `gorm:"type:uuid;index" json:"warehouseId,omitempty"`
	// ReceivedBy is always set server-side from the authenticated caller
	// (see handler.CreateGoodsReceipt) - never trust a client-supplied
	// value for who received a shipment.
	ReceivedBy string             `gorm:"type:varchar(255)" json:"receivedBy"`
	Status     string             `gorm:"type:varchar(50);default:'draft'" json:"status"`
	Lines      []GoodsReceiptLine `gorm:"foreignKey:GoodsReceiptID" json:"lines,omitempty"`
}

func (GoodsReceipt) TableName() string {
	return "procurement_goods_receipts"
}

type GoodsReceiptLine struct {
	types.BaseEntity
	GoodsReceiptID     uuid.UUID `gorm:"type:uuid;not null;index" json:"goodsReceiptId"`
	ProductID          uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	QuantityReceived   float64   `gorm:"type:decimal(15,2);not null" json:"quantityReceived"`
	QuantityOrderedRef float64   `gorm:"type:decimal(15,2);default:0" json:"quantityOrderedReference"`
}

func (GoodsReceiptLine) TableName() string {
	return "procurement_goods_receipt_lines"
}

// PurchaseInvoice is a vendor/purchase invoice for a supplier, optionally tied to a PO
type PurchaseInvoice struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this invoice to one business unit within the company
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	SupplierID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"supplierId"`
	SupplierName    string     `gorm:"type:varchar(255)" json:"supplierName"`
	PurchaseOrderID *uuid.UUID `gorm:"type:uuid;index" json:"purchaseOrderId,omitempty"`
	InvoiceNumber   string     `gorm:"type:varchar(50);not null;index" json:"invoiceNumber"`
	InvoiceDate     string     `gorm:"type:varchar(50)" json:"invoiceDate"`
	DueDate         string     `gorm:"type:varchar(50)" json:"dueDate"`
	TotalAmount     float64    `gorm:"type:decimal(15,2);not null" json:"totalAmount"`
	// Subtotal is the amount before adjustments; TotalAmount is the final
	// billed total = Subtotal - DiscountAmount + AdditionalCost + RoundingAmount.
	// Legacy rows have all four adjustment columns at 0 and TotalAmount alone.
	Subtotal       float64 `gorm:"type:decimal(15,2);default:0" json:"subtotal"`
	DiscountAmount float64 `gorm:"type:decimal(15,2);default:0" json:"discountAmount"`
	AdditionalCost float64 `gorm:"type:decimal(15,2);default:0" json:"additionalCost"`
	RoundingAmount float64 `gorm:"type:decimal(15,2);default:0" json:"roundingAmount"`
	PaidAmount     float64 `gorm:"type:decimal(15,2);default:0" json:"paidAmount"`
	Status         string  `gorm:"type:varchar(50);default:'unpaid'" json:"status"`
}

func (PurchaseInvoice) TableName() string {
	return "procurement_purchase_invoices"
}

// PurchaseDownPayment is an advance payment made to a supplier against a
// Purchase Order, recorded before (or independent of) the final Purchase
// Invoice. AppliedAmount tracks how much of it has been offset against an
// invoice so far; RemainingAmount = Amount - AppliedAmount is computed in
// the response DTO, not stored.
type PurchaseDownPayment struct {
	types.BaseEntity
	CompanyID       *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DPNumber        string     `gorm:"type:varchar(50);not null;index" json:"dpNumber"`
	PurchaseOrderID uuid.UUID  `gorm:"type:uuid;not null;index" json:"purchaseOrderId"`
	SupplierID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"supplierId"`
	SupplierName    string     `gorm:"type:varchar(255)" json:"supplierName"`
	Amount          float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	AppliedAmount   float64    `gorm:"type:decimal(15,2);default:0" json:"appliedAmount"`
	PaymentDate     string     `gorm:"type:varchar(50)" json:"paymentDate"`
	PaymentMethod   string     `gorm:"type:varchar(50)" json:"paymentMethod"`
	Status          string     `gorm:"type:varchar(50);default:'unapplied'" json:"status"` // unapplied | partially_applied | applied
	Notes           string     `gorm:"type:varchar(255)" json:"notes,omitempty"`
}

func (PurchaseDownPayment) TableName() string {
	return "procurement_purchase_down_payments"
}

// PurchaseReturn is a return of previously received goods (see GoodsReceipt)
// back to the supplier - e.g. damaged or wrong items.
type PurchaseReturn struct {
	types.BaseEntity
	CompanyID      *uuid.UUID           `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID       *uuid.UUID           `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ReturnNo       string               `gorm:"type:varchar(50);not null;index" json:"returnNo"`
	GoodsReceiptID uuid.UUID            `gorm:"type:uuid;not null;index" json:"goodsReceiptId"`
	SupplierID     uuid.UUID            `gorm:"type:uuid;not null;index" json:"supplierId"`
	SupplierName   string               `gorm:"type:varchar(255)" json:"supplierName"`
	ReturnDate     string               `gorm:"type:varchar(50)" json:"returnDate"`
	Reason         string               `gorm:"type:varchar(255)" json:"reason"`
	Status         string               `gorm:"type:varchar(50);default:'draft'" json:"status"` // draft | completed
	TotalAmount    float64              `gorm:"type:decimal(15,2);default:0" json:"totalAmount"`
	Lines          []PurchaseReturnLine `gorm:"foreignKey:PurchaseReturnID" json:"lines,omitempty"`
}

func (PurchaseReturn) TableName() string {
	return "procurement_purchase_returns"
}

type PurchaseReturnLine struct {
	types.BaseEntity
	PurchaseReturnID uuid.UUID `gorm:"type:uuid;not null;index" json:"purchaseReturnId"`
	ProductID        uuid.UUID `gorm:"type:uuid;not null;index" json:"productId"`
	Quantity         float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
	UnitPrice        float64   `gorm:"type:decimal(15,2);default:0" json:"unitPrice"`
	Subtotal         float64   `gorm:"type:decimal(15,2);default:0" json:"subtotal"`
}

func (PurchaseReturnLine) TableName() string {
	return "procurement_purchase_return_lines"
}

type ProcurementRepository interface {
	// Purchase requests
	CreatePurchaseRequest(ctx context.Context, pr *PurchaseRequest) error
	GetPurchaseRequestByID(ctx context.Context, id uuid.UUID) (*PurchaseRequest, error)
	ListPurchaseRequests(ctx context.Context, query types.PaginationQuery) ([]PurchaseRequest, int64, error)
	UpdatePurchaseRequest(ctx context.Context, pr *PurchaseRequest) error
	CountPurchaseRequests(ctx context.Context) (int64, error)

	// Purchase orders
	CreatePurchaseOrder(ctx context.Context, po *PurchaseOrder) error
	GetPurchaseOrderByID(ctx context.Context, id uuid.UUID) (*PurchaseOrder, error)
	ListPurchaseOrders(ctx context.Context, query types.PaginationQuery) ([]PurchaseOrder, int64, error)
	UpdatePurchaseOrder(ctx context.Context, po *PurchaseOrder) error
	CountPurchaseOrders(ctx context.Context) (int64, error)

	// Goods receipts
	CreateGoodsReceipt(ctx context.Context, gr *GoodsReceipt) error
	GetGoodsReceiptByID(ctx context.Context, id uuid.UUID) (*GoodsReceipt, error)
	ListGoodsReceipts(ctx context.Context, query types.PaginationQuery) ([]GoodsReceipt, int64, error)
	CountGoodsReceipts(ctx context.Context) (int64, error)

	// Purchase invoices
	CreatePurchaseInvoice(ctx context.Context, inv *PurchaseInvoice) error
	GetPurchaseInvoiceByID(ctx context.Context, id uuid.UUID) (*PurchaseInvoice, error)
	ListPurchaseInvoices(ctx context.Context, query types.PaginationQuery) ([]PurchaseInvoice, int64, error)
	UpdatePurchaseInvoice(ctx context.Context, inv *PurchaseInvoice) error
	CountPurchaseInvoices(ctx context.Context) (int64, error)

	// Purchase down payments
	CreateDownPayment(ctx context.Context, dp *PurchaseDownPayment) error
	GetDownPaymentByID(ctx context.Context, id uuid.UUID) (*PurchaseDownPayment, error)
	ListDownPayments(ctx context.Context, query types.PaginationQuery) ([]PurchaseDownPayment, int64, error)
	UpdateDownPayment(ctx context.Context, dp *PurchaseDownPayment) error
	CountDownPayments(ctx context.Context) (int64, error)

	// Purchase returns
	CreatePurchaseReturn(ctx context.Context, r *PurchaseReturn) error
	GetPurchaseReturnByID(ctx context.Context, id uuid.UUID) (*PurchaseReturn, error)
	ListPurchaseReturns(ctx context.Context, query types.PaginationQuery) ([]PurchaseReturn, int64, error)
	CountPurchaseReturns(ctx context.Context) (int64, error)
}
