package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type POSTransaction struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this transaction to one business unit within the company
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	OrderNo       string     `gorm:"type:varchar(50);not null;index" json:"orderNo"`
	Outlet        string     `gorm:"type:varchar(100);not null" json:"outlet"`
	Cashier       string     `gorm:"type:varchar(100);not null" json:"cashier"`
	Customer      string     `gorm:"type:varchar(100);default:'Pelanggan Umum'" json:"customer"`
	CustomerPhone string     `gorm:"type:varchar(30);index" json:"customerPhone,omitempty"`
	TotalItems    int        `gorm:"not null" json:"totalItems"`
	TotalAmount   float64    `gorm:"type:decimal(15,2);not null" json:"totalAmount"`
	PaymentMethod string     `gorm:"type:varchar(50);not null" json:"paymentMethod"`
	Status        string     `gorm:"type:varchar(50);default:'Completed'" json:"status"`
	// SalesOrderID/SalesOrderNumber link this cashier transaction to the
	// real Sales Order created for it (channel "POS") so revenue reporting,
	// AR, and the Sales Orders list all see POS sales too.
	SalesOrderID     *uuid.UUID `gorm:"type:uuid;index" json:"salesOrderId,omitempty"`
	SalesOrderNumber string     `gorm:"type:varchar(50)" json:"salesOrderNumber,omitempty"`
	// Loyalty fields are a snapshot of the point award made at checkout time
	// (see modules/loyalty) - kept here too so the receipt/history doesn't
	// need a second lookup just to show what the customer earned.
	LoyaltyMemberCode    string `gorm:"type:varchar(30)" json:"loyaltyMemberCode,omitempty"`
	LoyaltyPointsEarned  int    `gorm:"default:0" json:"loyaltyPointsEarned,omitempty"`
	LoyaltyPointsBalance int    `gorm:"default:0" json:"loyaltyPointsBalance,omitempty"`

	// Amount breakdown. TotalAmount = Subtotal - DiscountAmount + TaxAmount when
	// tax is added on top; with tax-inclusive pricing TaxAmount is the tax already
	// contained in the total. Legacy rows carry only TotalAmount.
	Subtotal       float64 `gorm:"type:decimal(15,2);default:0" json:"subtotal"`
	DiscountAmount float64 `gorm:"type:decimal(15,2);default:0" json:"discountAmount"`
	TaxAmount      float64 `gorm:"type:decimal(15,2);default:0" json:"taxAmount"`
	// RefundedAmount is the total handed back so far through refunds.
	RefundedAmount float64 `gorm:"type:decimal(15,2);default:0" json:"refundedAmount"`
	// AmountTendered/ChangeAmount record cash handed over and returned.
	AmountTendered float64 `gorm:"type:decimal(15,2);default:0" json:"amountTendered"`
	ChangeAmount   float64 `gorm:"type:decimal(15,2);default:0" json:"changeAmount"`
	// CashierEmail is the login that rang the sale up; a void or refund must be
	// approved by someone else.
	CashierEmail string     `gorm:"type:varchar(255);index" json:"cashierEmail,omitempty"`
	WarehouseID  *uuid.UUID `gorm:"type:uuid" json:"warehouseId,omitempty"`
	VoidReason   string     `gorm:"type:varchar(255)" json:"voidReason,omitempty"`
	VoidedBy     string     `gorm:"type:varchar(255)" json:"voidedBy,omitempty"`

	Lines []POSTransactionLine `gorm:"foreignKey:TransactionID" json:"lines,omitempty"`
}

func (POSTransaction) TableName() string {
	return "pos_transactions"
}

// POSTransactionLine is one product on a sale, with the name/SKU and unit cost
// as they were at the time (so later renames and cost changes never rewrite a
// receipt or its cost of goods sold).
type POSTransactionLine struct {
	types.BaseEntity
	TransactionID uuid.UUID `gorm:"type:uuid;not null;index" json:"transactionId"`
	ProductID     uuid.UUID `gorm:"type:uuid;not null" json:"productId"`
	SKU           string    `gorm:"type:varchar(50)" json:"sku"`
	Name          string    `gorm:"type:varchar(255)" json:"name"`
	Quantity      float64   `gorm:"type:decimal(15,2);not null" json:"quantity"`
	UnitPrice     float64   `gorm:"type:decimal(15,2);not null" json:"unitPrice"`
	UnitCost      float64   `gorm:"type:decimal(15,2);default:0" json:"unitCost"`
	Subtotal      float64   `gorm:"type:decimal(15,2);not null" json:"subtotal"`
	RefundedQty   float64   `gorm:"type:decimal(15,2);default:0" json:"refundedQty"`
}

func (POSTransactionLine) TableName() string { return "pos_transaction_lines" }

// POSRefund records one refund (or the full reversal of a void) against a sale.
type POSRefund struct {
	types.BaseEntity
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	TransactionID uuid.UUID  `gorm:"type:uuid;not null;index" json:"transactionId"`
	Kind          string     `gorm:"type:varchar(10);not null" json:"kind"` // refund | void
	Amount        float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	TaxAmount     float64    `gorm:"type:decimal(15,2);default:0" json:"taxAmount"`
	CostReturned  float64    `gorm:"type:decimal(15,2);default:0" json:"costReturned"`
	Reason        string     `gorm:"type:varchar(255);not null" json:"reason"`
	Restocked     bool       `gorm:"not null;default:false" json:"restocked"`
	ProcessedBy   string     `gorm:"type:varchar(255)" json:"processedBy,omitempty"`
}

func (POSRefund) TableName() string { return "pos_refunds" }

// POSSettings is the company's point-of-sale configuration (one row at most).
type POSSettings struct {
	types.BaseEntity
	// TaxPercent is the sales tax (PPN) rate; 0 disables tax.
	TaxPercent float64 `gorm:"type:decimal(5,2);default:0" json:"taxPercent"`
	// TaxInclusive means shelf prices already contain the tax.
	TaxInclusive bool `gorm:"default:true" json:"taxInclusive"`
	// RoundTo rounds the payable total (e.g. 100 or 500); 0 = no rounding.
	RoundTo       float64 `gorm:"type:decimal(10,2);default:0" json:"roundTo"`
	DefaultOutlet string  `gorm:"type:varchar(100)" json:"defaultOutlet"`
	ReceiptHeader string  `gorm:"type:varchar(500)" json:"receiptHeader"`
	ReceiptFooter string  `gorm:"type:varchar(500)" json:"receiptFooter"`
	// NonCashAccountCode is the ledger account card/QRIS/transfer sales settle
	// into (a bank or clearing asset account); empty means the cash account.
	NonCashAccountCode string `gorm:"type:varchar(50)" json:"nonCashAccountCode"`
}

func (POSSettings) TableName() string { return "pos_settings" }

// DefaultSettings applies until a company saves its own.
func DefaultSettings() POSSettings {
	return POSSettings{TaxInclusive: true, DefaultOutlet: "Outlet Utama"}
}

type POSRepository interface {
	// Lines, refunds and settings
	CreateLines(ctx context.Context, lines []POSTransactionLine) error
	ListLines(ctx context.Context, txID uuid.UUID) ([]POSTransactionLine, error)
	UpdateLine(ctx context.Context, l *POSTransactionLine) error
	Update(ctx context.Context, tx *POSTransaction) error
	CreateRefund(ctx context.Context, r *POSRefund) error
	ListRefunds(ctx context.Context, txID uuid.UUID) ([]POSRefund, error)
	// ListInRange returns transactions (with lines) created in [from, to]
	// (YYYY-MM-DD, Jakarta time), optionally for one outlet.
	ListInRange(ctx context.Context, from, to, outlet string) ([]POSTransaction, error)
	ListRefundsInRange(ctx context.Context, from, to, outlet string) ([]POSRefund, error)
	GetSettings(ctx context.Context) (*POSSettings, error)
	SaveSettings(ctx context.Context, s *POSSettings) error

	Create(ctx context.Context, tx *POSTransaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*POSTransaction, error)
	List(ctx context.Context, query types.PaginationQuery) ([]POSTransaction, int64, error)
	Count(ctx context.Context) (int64, error)
}
