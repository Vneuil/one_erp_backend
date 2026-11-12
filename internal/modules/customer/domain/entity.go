package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type Customer struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this customer to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant.
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Code     string     `gorm:"type:varchar(50);not null;index" json:"code"`
	Name     string     `gorm:"type:varchar(255);not null" json:"name"`
	Email    string     `gorm:"type:varchar(255)" json:"email"`
	Phone    string     `gorm:"type:varchar(50)" json:"phone"`
	Address  string     `gorm:"type:text" json:"address"`
	// NPWP (15/16 digits) and NIK (16 digits) identify the buyer on Faktur Pajak.
	NPWP    string `gorm:"type:varchar(20)" json:"npwp"`
	NIK     string `gorm:"type:varchar(20)" json:"nik"`
	Segment string `gorm:"type:varchar(100);default:'Enterprise B2B'" json:"segment"`
	Status  string `gorm:"type:varchar(50);default:'Active'" json:"status"`

	// CustomerType is B2B (company) or B2C (individual); the code is built from it:
	// B2B-0001 / B2C-0001.
	CustomerType string `gorm:"type:varchar(10);index" json:"customerType,omitempty"`
	// Contact person block - required for B2B customers.
	ContactPerson string `gorm:"type:varchar(150)" json:"contactPerson,omitempty"`
	ContactPhone  string `gorm:"type:varchar(50)" json:"contactPhone,omitempty"`
	ContactMobile string `gorm:"type:varchar(50)" json:"contactMobile,omitempty"`
	ContactWA     string `gorm:"type:varchar(50)" json:"contactWhatsapp,omitempty"`
	// Office (billing) address parts; Address keeps the street line.
	City       string `gorm:"type:varchar(100);default:'Semarang'" json:"city"`
	Province   string `gorm:"type:varchar(100)" json:"province,omitempty"`
	PostalCode string `gorm:"type:varchar(10)" json:"postalCode,omitempty"`
	// ShippingAddresses are delivery addresses that may differ from the office
	// address (warehouse / production site). Sales Orders pick from this list.
	ShippingAddresses []CustomerAddress `gorm:"foreignKey:CustomerID" json:"shippingAddresses,omitempty"`
}

// CustomerAddress is one delivery address of a customer.
type CustomerAddress struct {
	types.BaseEntity
	CustomerID    uuid.UUID `gorm:"type:uuid;not null;index" json:"customerId"`
	Label         string    `gorm:"type:varchar(100)" json:"label"` // e.g. "Gudang", "Pabrik"
	RecipientName string    `gorm:"type:varchar(150)" json:"recipientName,omitempty"`
	Phone         string    `gorm:"type:varchar(50)" json:"phone,omitempty"`
	Address       string    `gorm:"type:text;not null" json:"address"`
	City          string    `gorm:"type:varchar(100)" json:"city,omitempty"`
	Province      string    `gorm:"type:varchar(100)" json:"province,omitempty"`
	PostalCode    string    `gorm:"type:varchar(10)" json:"postalCode,omitempty"`
	IsDefault     bool      `gorm:"default:false" json:"isDefault"`
}

func (CustomerAddress) TableName() string { return "customer_addresses" }

func (Customer) TableName() string {
	return "customers"
}

type CustomerRepository interface {
	Create(ctx context.Context, customer *Customer) error
	GetByID(ctx context.Context, id uuid.UUID) (*Customer, error)
	GetByCode(ctx context.Context, code string) (*Customer, error)
	List(ctx context.Context, query types.PaginationQuery) ([]Customer, int64, error)
	Update(ctx context.Context, customer *Customer) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
	// CodesWithPrefix returns existing codes starting with prefix (B2B-/B2C-).
	CodesWithPrefix(ctx context.Context, prefix string) ([]string, error)
	// ReplaceShippingAddresses swaps the customer's delivery addresses wholesale.
	ReplaceShippingAddresses(ctx context.Context, customerID uuid.UUID, addrs []CustomerAddress) error
	// HasReferences reports whether this customer (matched by name, since
	// Sales Orders/Invoices/Quotations/Deals only carry a customerName
	// string, not a customer ID FK) is still referenced by any Sales Order,
	// Invoice, Quotation, or CRM Deal, so Delete can be blocked the same way
	// warehouse/product delete is blocked when dependents still exist.
	HasReferences(ctx context.Context, name string) (bool, error)
}
