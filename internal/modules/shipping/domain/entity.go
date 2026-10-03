package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type ShippingMethod struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this shippingMethod to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant.
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Code          string     `gorm:"type:varchar(50);not null;index" json:"code"`
	Name          string     `gorm:"type:varchar(255);not null" json:"name"`
	Carrier       string     `gorm:"type:varchar(150)" json:"carrier"`
	EstimatedDays int        `gorm:"default:0" json:"estimatedDays"`
	Cost          float64    `gorm:"type:decimal(15,2);default:0" json:"cost"`
	Status        string     `gorm:"type:varchar(50);default:'Active'" json:"status"`
}

func (ShippingMethod) TableName() string {
	return "shipping_methods"
}

type ShippingMethodRepository interface {
	Create(ctx context.Context, shippingMethod *ShippingMethod) error
	GetByID(ctx context.Context, id uuid.UUID) (*ShippingMethod, error)
	GetByCode(ctx context.Context, code string) (*ShippingMethod, error)
	List(ctx context.Context, query types.PaginationQuery) ([]ShippingMethod, int64, error)
	Update(ctx context.Context, shippingMethod *ShippingMethod) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
}
