package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type Supplier struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this supplier to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant -
	// the default state for companies that never created more than their
	// seeded default Tenant.
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Code          string     `gorm:"type:varchar(50);not null;index" json:"code"`
	BatchCode     string     `gorm:"type:varchar(10);index" json:"batchCode,omitempty"`
	Name          string     `gorm:"type:varchar(255);not null" json:"name"`
	ContactPerson string     `gorm:"type:varchar(100)" json:"contactPerson"`
	Email         string     `gorm:"type:varchar(255)" json:"email"`
	Phone         string     `gorm:"type:varchar(50)" json:"phone"`
	Address       string     `gorm:"type:text" json:"address"`
	// VendorType is the 3-letter family the code is built from: PRD (product /
	// raw material), PCK (packaging), ATK (office supplies). Code = VD-<type>-<seq>.
	VendorType string `gorm:"type:varchar(10);index" json:"vendorType,omitempty"`
	City       string `gorm:"type:varchar(100);default:'Semarang'" json:"city"`
	Province   string `gorm:"type:varchar(100)" json:"province,omitempty"`
	PostalCode string `gorm:"type:varchar(10)" json:"postalCode,omitempty"`
	// PaymentTerm is the name of a master Payment Term (e.g. "Net 30").
	PaymentTerm string `gorm:"type:varchar(100)" json:"paymentTerm,omitempty"`
	// NPWP (15/16 digits) and NIK (16 digits) identify the seller on Faktur Pajak Masukan.
	NPWP     string `gorm:"type:varchar(20)" json:"npwp"`
	NIK      string `gorm:"type:varchar(20)" json:"nik"`
	Category string `gorm:"type:varchar(100);default:'Raw Materials'" json:"category"`
	Status   string `gorm:"type:varchar(50);default:'Active'" json:"status"`
}

func (Supplier) TableName() string {
	return "suppliers"
}

type SupplierRepository interface {
	Create(ctx context.Context, supplier *Supplier) error
	GetByID(ctx context.Context, id uuid.UUID) (*Supplier, error)
	GetByCode(ctx context.Context, code string) (*Supplier, error)
	List(ctx context.Context, query types.PaginationQuery) ([]Supplier, int64, error)
	Update(ctx context.Context, supplier *Supplier) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
	// CodesWithPrefix returns every supplier code starting with prefix, so the
	// next sequence number for a vendor type can be derived.
	CodesWithPrefix(ctx context.Context, prefix string) ([]string, error)
	// HasReferences reports whether this supplier is still referenced by any
	// Purchase Order or Purchase Invoice (both carry a real supplier_id FK),
	// so Delete can be blocked the same way warehouse/product delete is
	// blocked when dependents still exist.
	HasReferences(ctx context.Context, id uuid.UUID) (bool, error)
}
