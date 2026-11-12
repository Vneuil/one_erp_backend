package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Stored statuses. "expired" and "not_yet_valid" are never stored: they are derived
// from the validity dates (EffectiveStatus), so a certificate switches itself off
// the day after ValidUntil without any job running.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusRevoked   = "revoked"

	EffectiveExpired     = "expired"
	EffectiveNotYetValid = "not_yet_valid"
)

// Certificate is one certification (USDA Organic, EU Organic, Halal, HACCP, ...) held by
// a supplier and/or covering a product. A supplier or product can hold many.
type Certificate struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`

	Scheme     string     `gorm:"type:varchar(100);not null;index" json:"scheme"` // e.g. "USDA Organic"
	Number     string     `gorm:"type:varchar(100);not null" json:"number"`
	Issuer     string     `gorm:"type:varchar(150)" json:"issuer,omitempty"`
	SupplierID *uuid.UUID `gorm:"type:uuid;index" json:"supplierId,omitempty"`
	ProductID  *uuid.UUID `gorm:"type:uuid;index" json:"productId,omitempty"`

	ValidFrom  string `gorm:"type:varchar(10)" json:"validFrom"`        // YYYY-MM-DD
	ValidUntil string `gorm:"type:varchar(10);index" json:"validUntil"` // YYYY-MM-DD

	Status        string     `gorm:"type:varchar(20);default:'active';index" json:"status"` // active | suspended | revoked
	SuspendReason string     `gorm:"type:varchar(500)" json:"suspendReason,omitempty"`
	SuspendedAt   *time.Time `json:"suspendedAt,omitempty"`
	SuspendedBy   string     `gorm:"type:varchar(255)" json:"suspendedBy,omitempty"`
	Attachment    string     `gorm:"type:text" json:"attachment,omitempty"`
	Notes         string     `gorm:"type:text" json:"notes,omitempty"`
}

func (Certificate) TableName() string { return "certifications" }

// EffectiveStatus is what the rest of the system should act on, evaluated on the given day.
func (c Certificate) EffectiveStatus(today string) string {
	switch c.Status {
	case StatusRevoked, StatusSuspended:
		return c.Status
	}
	if c.ValidFrom != "" && today < c.ValidFrom {
		return EffectiveNotYetValid
	}
	if c.ValidUntil != "" && today > c.ValidUntil {
		return EffectiveExpired
	}
	return StatusActive
}

type ListFilter struct {
	SupplierID *uuid.UUID
	ProductID  *uuid.UUID
	Scheme     string
}

type Repository interface {
	Create(ctx context.Context, c *Certificate) error
	GetByID(ctx context.Context, id uuid.UUID) (*Certificate, error)
	Update(ctx context.Context, c *Certificate) error
	List(ctx context.Context, f ListFilter) ([]Certificate, error)
}
