package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// SalesOrderRevision is the snapshot of a Sales Order *before* a revision was
// applied, so every earlier version stays available for later analysis.
type SalesOrderRevision struct {
	types.BaseEntity
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	SalesOrderID uuid.UUID  `gorm:"type:uuid;not null;index" json:"salesOrderId"`
	Revision     int        `gorm:"not null" json:"revision"` // the revision number this snapshot was REPLACED by
	ChangedBy    string     `gorm:"type:varchar(255)" json:"changedBy"`
	Reason       string     `gorm:"type:varchar(500)" json:"reason"`
	TotalAmount  float64    `gorm:"type:decimal(15,2)" json:"totalAmount"`
	LinesJSON    string     `gorm:"type:text" json:"linesJson"` // []SalesOrderLine as JSON
}

func (SalesOrderRevision) TableName() string { return "sales_order_revisions" }

// Trade invoice kinds: Proforma Invoice (sent before shipment / payment) and
// Commercial Invoice (the export billing document).
const (
	TradeKindPI = "PI"
	TradeKindCI = "CI"
)

const (
	TradeStatusDraft     = "draft"
	TradeStatusIssued    = "issued"
	TradeStatusCancelled = "cancelled"
)

// TradeInvoice is a PI or CI. It is created from a Quotation / Sales Order (a CI
// may also be created from its PI) and is managed by superadmin only.
type TradeInvoice struct {
	types.BaseEntity
	CompanyID    *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Kind         string     `gorm:"type:varchar(2);not null;index" json:"kind"`
	Number       string     `gorm:"type:varchar(50);not null;index" json:"number"`
	SalesOrderID *uuid.UUID `gorm:"type:uuid;index" json:"salesOrderId,omitempty"`
	QuotationID  *uuid.UUID `gorm:"type:uuid;index" json:"quotationId,omitempty"`
	ParentID     *uuid.UUID `gorm:"type:uuid;index" json:"parentId,omitempty"` // PI this CI was made from
	CustomerName string     `gorm:"type:varchar(255);not null" json:"customerName"`
	BillingAddr  string     `gorm:"type:text" json:"billingAddress,omitempty"`
	ShippingAddr string     `gorm:"type:text" json:"shippingAddress,omitempty"`
	Currency     string     `gorm:"type:varchar(10);default:'IDR'" json:"currency"`
	IssueDate    string     `gorm:"type:varchar(10)" json:"issueDate"`
	PaymentTerm  string     `gorm:"type:varchar(100)" json:"paymentTerm,omitempty"`
	Incoterm     string     `gorm:"type:varchar(20)" json:"incoterm,omitempty"`
	Notes        string     `gorm:"type:text" json:"notes,omitempty"`
	// VATExempt turns PPN off (export). When false, VATRate (default 11) applies.
	VATExempt bool               `gorm:"default:false" json:"vatExempt"`
	VATRate   float64            `gorm:"type:decimal(5,2);default:11" json:"vatRate"`
	Subtotal  float64            `gorm:"type:decimal(15,2);default:0" json:"subtotal"`
	VATAmount float64            `gorm:"type:decimal(15,2);default:0" json:"vatAmount"`
	Total     float64            `gorm:"type:decimal(15,2);default:0" json:"total"`
	Status    string             `gorm:"type:varchar(20);default:'draft';index" json:"status"`
	IssuedBy  string             `gorm:"type:varchar(255)" json:"issuedBy,omitempty"`
	IssuedAt  *time.Time         `json:"issuedAt,omitempty"`
	CreatedBy string             `gorm:"type:varchar(255)" json:"createdBy,omitempty"`
	Lines     []TradeInvoiceLine `gorm:"foreignKey:TradeInvoiceID" json:"lines,omitempty"`
}

func (TradeInvoice) TableName() string { return "sales_trade_invoices" }

type TradeInvoiceLine struct {
	types.BaseEntity
	TradeInvoiceID uuid.UUID  `gorm:"type:uuid;not null;index" json:"tradeInvoiceId"`
	ProductID      *uuid.UUID `gorm:"type:uuid;index" json:"productId,omitempty"`
	Description    string     `gorm:"type:text" json:"description"`
	Unit           string     `gorm:"type:varchar(50)" json:"unit"`
	Quantity       float64    `gorm:"type:decimal(15,2);not null" json:"quantity"`
	UnitPrice      float64    `gorm:"type:decimal(15,2);not null" json:"unitPrice"`
	Subtotal       float64    `gorm:"type:decimal(15,2);not null" json:"subtotal"`
}

func (TradeInvoiceLine) TableName() string { return "sales_trade_invoice_lines" }

type TradeInvoiceRepository interface {
	Create(ctx context.Context, t *TradeInvoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*TradeInvoice, error)
	Update(ctx context.Context, t *TradeInvoice) error
	List(ctx context.Context, kind string, query types.PaginationQuery) ([]TradeInvoice, int64, error)
	CountByKind(ctx context.Context, kind string) (int64, error)
}
