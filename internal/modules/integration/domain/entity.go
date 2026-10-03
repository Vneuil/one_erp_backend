package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// APIKey represents a programmatic access credential for external systems
type APIKey struct {
	types.BaseEntity
	CompanyID  *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	Name       string     `gorm:"type:varchar(255);not null" json:"name"`
	KeyValue   string     `gorm:"type:varchar(100);not null;uniqueIndex" json:"keyValue"`
	Scopes     string     `gorm:"type:varchar(500);not null" json:"scopes"`
	IsActive   bool       `gorm:"not null;default:true" json:"isActive"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
}

func (APIKey) TableName() string {
	return "integration_api_keys"
}

// WebhookSubscription represents a registered outbound webhook target
type WebhookSubscription struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	Name      string     `gorm:"type:varchar(255);not null" json:"name"`
	TargetURL string     `gorm:"type:varchar(500);not null" json:"targetUrl"`
	EventType string     `gorm:"type:varchar(100);not null" json:"eventType"`
	IsActive  bool       `gorm:"not null;default:true" json:"isActive"`
	Secret    string     `gorm:"type:varchar(100);not null" json:"secret"`
}

func (WebhookSubscription) TableName() string {
	return "integration_webhook_subscriptions"
}

// WebhookDelivery represents a simulated delivery attempt log entry
type WebhookDelivery struct {
	types.BaseEntity
	SubscriptionID      uuid.UUID `gorm:"type:uuid;not null;index" json:"subscriptionId"`
	EventType           string    `gorm:"type:varchar(100);not null" json:"eventType"`
	Payload             string    `gorm:"type:text" json:"payload"`
	Status              string    `gorm:"type:varchar(20);not null" json:"status"`
	ResponseCode        *int      `json:"responseCode,omitempty"`
	AttemptedAt         time.Time `gorm:"not null" json:"attemptedAt"`
	ResponseBodySnippet *string   `gorm:"type:text" json:"responseBodySnippet,omitempty"`
}

func (WebhookDelivery) TableName() string {
	return "integration_webhook_deliveries"
}

type APIKeyRepository interface {
	Create(ctx context.Context, k *APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*APIKey, error)
	ListAll(ctx context.Context) ([]APIKey, error)
	Update(ctx context.Context, k *APIKey) error
	Count(ctx context.Context) (int64, error)
}

type WebhookRepository interface {
	Create(ctx context.Context, w *WebhookSubscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*WebhookSubscription, error)
	ListAll(ctx context.Context) ([]WebhookSubscription, error)
	Update(ctx context.Context, w *WebhookSubscription) error
	Count(ctx context.Context) (int64, error)

	CreateDelivery(ctx context.Context, d *WebhookDelivery) error
	ListDeliveriesBySubscription(ctx context.Context, subscriptionID uuid.UUID) ([]WebhookDelivery, error)
}
