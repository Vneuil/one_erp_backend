package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Tenant is a business unit (branch, subsidiary, brand) within a single
// Company. A Company owns exactly one tenant database (see
// foundation/tenant.Manager); Tenant is a further split *inside* that
// database - every Tenant of a Company shares the same database and is
// distinguished by the tenant_id column on business tables, not by a
// separate physical database. Every Company always has at least one Tenant
// (seeded automatically on provisioning - see Seed in module.go) so
// existing single-tenant companies keep working unchanged.
type Tenant struct {
	types.BaseEntity
	Name      string `gorm:"type:varchar(255);not null" json:"name"`
	Code      string `gorm:"type:varchar(50);not null;uniqueIndex" json:"code"`
	IsActive  bool   `gorm:"not null;default:true" json:"isActive"`
	IsDefault bool   `gorm:"not null;default:false" json:"isDefault"`
}

func (Tenant) TableName() string {
	return "tenants"
}

type TenantRepository interface {
	Create(ctx context.Context, t *Tenant) error
	List(ctx context.Context) ([]Tenant, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetDefault(ctx context.Context) (*Tenant, error)
}
