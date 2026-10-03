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
	Segment  string     `gorm:"type:varchar(100);default:'Enterprise B2B'" json:"segment"`
	Status   string     `gorm:"type:varchar(50);default:'Active'" json:"status"`
}

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
	// HasReferences reports whether this customer (matched by name, since
	// Sales Orders/Invoices/Quotations/Deals only carry a customerName
	// string, not a customer ID FK) is still referenced by any Sales Order,
	// Invoice, Quotation, or CRM Deal, so Delete can be blocked the same way
	// warehouse/product delete is blocked when dependents still exist.
	HasReferences(ctx context.Context, name string) (bool, error)
}
