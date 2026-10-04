package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Faktur Pajak directions and statuses.
const (
	DirectionOutput = "output" // Faktur Pajak Keluaran (sales)
	DirectionInput  = "input"  // Faktur Pajak Masukan (purchases)

	StatusDraft     = "draft"
	StatusIssued    = "issued"
	StatusCancelled = "cancelled"
	StatusReplaced  = "replaced"

	SourceSalesInvoice    = "sales_invoice"
	SourcePurchaseInvoice = "purchase_invoice"
)

// TaxSettings holds the taxpayer (seller) identity printed on every Faktur
// Pajak Keluaran. One row per company database.
type TaxSettings struct {
	types.BaseEntity
	TaxpayerName string `gorm:"type:varchar(255)" json:"taxpayerName"`
	NPWP         string `gorm:"type:varchar(20)" json:"npwp"`
	Address      string `gorm:"type:text" json:"address"`
	// IsPKP marks a Pengusaha Kena Pajak: only PKP may issue Faktur Pajak.
	IsPKP bool `gorm:"default:false" json:"isPkp"`
}

func (TaxSettings) TableName() string { return "tax_settings" }

// SerialRange is a block of Faktur Pajak numbers: Prefix + Next..End, each
// number zero-padded to Width digits. Numbers are claimed one at a time when a
// Faktur Pajak Keluaran is issued.
type SerialRange struct {
	types.BaseEntity
	Prefix   string `gorm:"type:varchar(30)" json:"prefix"`
	Start    int64  `gorm:"column:first_number;not null" json:"start"`
	End      int64  `gorm:"column:last_number;not null" json:"end"`
	Next     int64  `gorm:"column:next_number;not null" json:"next"`
	Width    int    `gorm:"not null;default:8" json:"width"`
	IsActive bool   `gorm:"default:true" json:"isActive"`
}

func (SerialRange) TableName() string { return "tax_serial_ranges" }

// Remaining is how many numbers are still unclaimed.
func (r SerialRange) Remaining() int64 {
	if r.Next > r.End {
		return 0
	}
	return r.End - r.Next + 1
}

// TaxInvoice is a Faktur Pajak: Keluaran (issued to a customer, source = sales
// invoice) or Masukan (received from a supplier, source = purchase invoice).
// Counterparty fields are a snapshot taken when it was created.
type TaxInvoice struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`

	Direction string `gorm:"type:varchar(10);not null;index" json:"direction"`
	// Number is the internal document number; TaxNumber is the official Nomor
	// Faktur Pajak (from a serial range, from DJP, or the supplier's).
	Number          string `gorm:"type:varchar(50);not null;index" json:"number"`
	TaxNumber       string `gorm:"type:varchar(50);index" json:"taxNumber"`
	Status          string `gorm:"type:varchar(20);not null;default:'draft';index" json:"status"`
	TransactionCode string `gorm:"type:varchar(5)" json:"transactionCode"`
	Date            string `gorm:"type:varchar(10);not null;index" json:"date"`
	// Period is the tax period (masa pajak) as YYYY-MM.
	Period string `gorm:"type:varchar(7);not null;index" json:"period"`

	CounterpartyName    string `gorm:"type:varchar(255)" json:"counterpartyName"`
	CounterpartyNPWP    string `gorm:"type:varchar(20)" json:"counterpartyNpwp"`
	CounterpartyNIK     string `gorm:"type:varchar(20)" json:"counterpartyNik"`
	CounterpartyAddress string `gorm:"type:text" json:"counterpartyAddress"`
	CounterpartyEmail   string `gorm:"type:varchar(255)" json:"counterpartyEmail"`

	SourceType      string    `gorm:"type:varchar(30);index" json:"sourceType"`
	SourceID        uuid.UUID `gorm:"type:uuid;index" json:"sourceId"`
	SourceReference string    `gorm:"type:varchar(100)" json:"sourceReference"`

	TaxBase       float64 `gorm:"type:decimal(15,2);not null" json:"taxBase"`
	DPPOtherValue float64 `gorm:"type:decimal(15,2);default:0" json:"dppOtherValue"`
	VATRate       float64 `gorm:"type:decimal(5,2);not null" json:"vatRate"`
	VATAmount     float64 `gorm:"type:decimal(15,2);not null" json:"vatAmount"`

	ReplacesID   *uuid.UUID `gorm:"type:uuid;index" json:"replacesId,omitempty"`
	Revision     int        `gorm:"default:0" json:"revision"`
	CancelReason string     `gorm:"type:varchar(500)" json:"cancelReason,omitempty"`
	Notes        string     `gorm:"type:varchar(500)" json:"notes,omitempty"`
	IssuedAt     *time.Time `json:"issuedAt,omitempty"`
	CreatedBy    string     `gorm:"type:varchar(255)" json:"createdBy,omitempty"`

	Lines []TaxInvoiceLine `gorm:"foreignKey:TaxInvoiceID" json:"lines,omitempty"`
}

func (TaxInvoice) TableName() string { return "tax_invoices" }

// TaxInvoiceLine is one goods/services line. TaxBase = Quantity x UnitPrice.
type TaxInvoiceLine struct {
	types.BaseEntity
	TaxInvoiceID  uuid.UUID `gorm:"type:uuid;not null;index" json:"taxInvoiceId"`
	Position      int       `gorm:"not null" json:"position"`
	Description   string    `gorm:"type:varchar(255);not null" json:"description"`
	Unit          string    `gorm:"type:varchar(30)" json:"unit"`
	Quantity      float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
	UnitPrice     float64   `gorm:"type:decimal(15,2);not null" json:"unitPrice"`
	TaxBase       float64   `gorm:"type:decimal(15,2);not null" json:"taxBase"`
	DPPOtherValue float64   `gorm:"type:decimal(15,2);default:0" json:"dppOtherValue"`
	VATAmount     float64   `gorm:"type:decimal(15,2);not null" json:"vatAmount"`
}

func (TaxInvoiceLine) TableName() string { return "tax_invoice_lines" }

// InvoiceFilter narrows a Faktur Pajak listing; empty fields are ignored.
type InvoiceFilter struct {
	Direction, Status, From, To, Search string
}

type TaxRepository interface {
	GetSettings(ctx context.Context) (*TaxSettings, error)
	SaveSettings(ctx context.Context, s *TaxSettings) error

	CreateSerialRange(ctx context.Context, r *SerialRange) error
	GetSerialRange(ctx context.Context, id uuid.UUID) (*SerialRange, error)
	UpdateSerialRange(ctx context.Context, r *SerialRange) error
	ListSerialRanges(ctx context.Context) ([]SerialRange, error)
	// ClaimNextSerial atomically takes the next number from the oldest active
	// range that still has some. It returns "" when none is available.
	ClaimNextSerial(ctx context.Context) (string, error)

	CreateTaxInvoice(ctx context.Context, t *TaxInvoice) error
	GetTaxInvoice(ctx context.Context, id uuid.UUID) (*TaxInvoice, error)
	UpdateTaxInvoice(ctx context.Context, t *TaxInvoice) error
	ListTaxInvoices(ctx context.Context, f InvoiceFilter) ([]TaxInvoice, error)
	// FindOpenBySource returns the draft or issued Faktur Pajak of a source document, if any.
	FindOpenBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) (*TaxInvoice, error)
	// ListBySources returns the non-cancelled, non-replaced Faktur Pajak of the given source documents.
	ListBySources(ctx context.Context, sourceType string, ids []uuid.UUID) ([]TaxInvoice, error)
	// TaxNumberTaken reports whether another live Faktur Pajak of the direction already uses the number.
	TaxNumberTaken(ctx context.Context, direction, taxNumber string, exceptID uuid.UUID) (bool, error)
	CountTaxInvoices(ctx context.Context, direction, numberPrefix string) (int64, error)
}

// Party is a customer or supplier as seen on a Faktur Pajak.
type Party struct {
	Name    string
	NPWP    string
	NIK     string
	Address string
	Email   string
}

// SourceLine is one goods line of a source document.
type SourceLine struct {
	Description string
	Unit        string
	Quantity    float64
	Subtotal    float64
}

// SalesInvoiceRef and PurchaseInvoiceRef are the parts of an invoice a Faktur
// Pajak needs, decoupled from the sales and procurement modules' entities.
type SalesInvoiceRef struct {
	ID                                      uuid.UUID
	Number, CustomerName, Date              string
	Subtotal, Discount, AdditionalCost      float64
	Rounding, Total                         float64
	TaxBase, DPPOtherValue, VATRate, VATAmt float64
	VATOtherValueBase                       bool
}

type PurchaseInvoiceRef struct {
	ID                                      uuid.UUID
	SupplierID                              uuid.UUID
	Number, SupplierName, Date              string
	Total                                   float64
	TaxBase, DPPOtherValue, VATRate, VATAmt float64
	VATOtherValueBase, VATCreditable        bool
}

// SourceReader reads the documents Faktur Pajak are raised from. Implemented
// against the sales, procurement, customer and supplier tables.
type SourceReader interface {
	SalesInvoice(ctx context.Context, id uuid.UUID) (*SalesInvoiceRef, error)
	SalesInvoicesBetween(ctx context.Context, from, to string) ([]SalesInvoiceRef, error)
	// SalesInvoiceLines returns the goods lines of the order an invoice was
	// billed against, or nil when it has none.
	SalesInvoiceLines(ctx context.Context, id uuid.UUID) ([]SourceLine, error)
	PurchaseInvoice(ctx context.Context, id uuid.UUID) (*PurchaseInvoiceRef, error)
	PurchaseInvoicesBetween(ctx context.Context, from, to string) ([]PurchaseInvoiceRef, error)
	// CustomersByName returns every customer keyed by lower-cased name.
	CustomersByName(ctx context.Context) (map[string]Party, error)
	Supplier(ctx context.Context, id uuid.UUID) (*Party, error)
}
