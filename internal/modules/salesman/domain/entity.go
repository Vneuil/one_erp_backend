package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type Salesman struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this salesman to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant.
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Code           string     `gorm:"type:varchar(50);not null;index" json:"code"`
	Name           string     `gorm:"type:varchar(255);not null" json:"name"`
	Email          string     `gorm:"type:varchar(255)" json:"email"`
	Phone          string     `gorm:"type:varchar(50)" json:"phone"`
	Territory      string     `gorm:"type:varchar(150)" json:"territory"`
	CommissionRate float64    `gorm:"type:decimal(5,2);default:0" json:"commissionRate"`
	Status         string     `gorm:"type:varchar(50);default:'Active'" json:"status"`
}

func (Salesman) TableName() string {
	return "salesmen"
}

type SalesmanRepository interface {
	Create(ctx context.Context, salesman *Salesman) error
	GetByID(ctx context.Context, id uuid.UUID) (*Salesman, error)
	GetByCode(ctx context.Context, code string) (*Salesman, error)
	List(ctx context.Context, query types.PaginationQuery) ([]Salesman, int64, error)
	Update(ctx context.Context, salesman *Salesman) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
}
