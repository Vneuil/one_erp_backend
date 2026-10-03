// Package domain holds the CheckoutOrder entity for the public pricing
// page's self-serve subscription checkout. This is a control-plane
// concept (a prospective customer paying before they have a company/tenant
// at all), so - like company/user/tenant - it lives in the central
// database, not a per-tenant one.
package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type CheckoutStatus string

const (
	CheckoutStatusPending    CheckoutStatus = "pending"    // created, Xendit invoice not yet confirmed created
	CheckoutStatusProcessing CheckoutStatus = "processing" // Xendit invoice created, awaiting payment
	CheckoutStatusPaid       CheckoutStatus = "paid"
	CheckoutStatusExpired    CheckoutStatus = "expired"
	CheckoutStatusFailed     CheckoutStatus = "failed"
)

// CheckoutOrder is one self-serve subscription purchase attempt from the
// public pricing/checkout pages. PlanCode/BillingCycle/AmountIDR are all
// validated and priced server-side (see application.planPrice) - never
// trust a client-supplied amount for something that triggers a real charge.
type CheckoutOrder struct {
	types.BaseEntity
	PlanCode     string         `gorm:"type:varchar(30);not null" json:"planCode"`
	BillingCycle string         `gorm:"type:varchar(20);not null" json:"billingCycle"`
	AmountIDR    float64        `gorm:"type:decimal(15,2);not null" json:"amountIdr"`
	CompanyName  string         `gorm:"type:varchar(255);not null" json:"companyName"`
	ContactName  string         `gorm:"type:varchar(255);not null" json:"contactName"`
	ContactEmail string         `gorm:"type:varchar(255);not null;index" json:"contactEmail"`
	ContactPhone string         `gorm:"type:varchar(50)" json:"contactPhone"`
	Status       CheckoutStatus `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	// ProviderRef is the Xendit invoice ID - how CheckoutOrder is matched
	// against an incoming webhook payload.
	ProviderRef string     `gorm:"type:varchar(150);index" json:"providerRef,omitempty"`
	CheckoutURL string     `gorm:"type:text" json:"-"`
	PaidAt      *time.Time `json:"paidAt,omitempty"`
}

func (CheckoutOrder) TableName() string {
	return "checkout_orders"
}

type CheckoutOrderRepository interface {
	Create(ctx context.Context, o *CheckoutOrder) error
	GetByID(ctx context.Context, id uuid.UUID) (*CheckoutOrder, error)
	GetByProviderRef(ctx context.Context, providerRef string) (*CheckoutOrder, error)
	Update(ctx context.Context, o *CheckoutOrder) error
}
