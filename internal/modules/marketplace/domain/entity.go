package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Platform identifies a supported marketplace.
type Platform string

const (
	PlatformTikTokShop Platform = "tiktok_shop"
	PlatformShopee     Platform = "shopee"
	PlatformBlibli     Platform = "blibli"
	PlatformLazada     Platform = "lazada"
)

// ChannelLabel returns the human-readable label stored on the resulting
// Sales Order's Channel field (matches the existing sales module convention,
// e.g. "TikTok Shop").
func (p Platform) ChannelLabel() string {
	switch p {
	case PlatformTikTokShop:
		return "TikTok Shop"
	case PlatformShopee:
		return "Shopee"
	case PlatformBlibli:
		return "Blibli"
	case PlatformLazada:
		return "Lazada"
	default:
		return string(p)
	}
}

func (p Platform) Valid() bool {
	return p == PlatformTikTokShop || p == PlatformShopee || p == PlatformBlibli || p == PlatformLazada
}

// MarketplaceConnection stores one company's OAuth grant for a marketplace
// seller account. It lives in the tenant's own database (never the
// control-plane DB) since it is company-scoped credential data.
//
// NOTE: access_token/refresh_token are stored as plain text because no
// field-level encryption helper exists yet in this codebase (grepped
// foundation/ and pkg/ for "encrypt" - none found). This table has no API
// route that returns raw token values (GET /marketplace/connections masks
// them) and tokens are never logged. Adding column-level encryption (e.g.
// via pgcrypto or an application-level AES-GCM helper) is a recommended
// follow-up before handling production seller credentials at scale.
type MarketplaceConnection struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this connection to one business unit within the
	// company (see modules/workspace). Nil means it belongs to no specific
	// tenant - the default state for companies that never created more than
	// their seeded default Tenant.
	TenantID         *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Platform         Platform   `gorm:"type:varchar(30);not null;uniqueIndex:idx_marketplace_platform" json:"platform"`
	ShopID           string     `gorm:"type:varchar(100)" json:"shopId"`
	ShopName         string     `gorm:"type:varchar(255)" json:"shopName"`
	ShopCipher       string     `gorm:"type:varchar(255)" json:"-"` // TikTok Shop shop_cipher, required on every order API call
	AccessToken      string     `gorm:"type:text;not null" json:"-"`
	RefreshToken     string     `gorm:"type:text;not null" json:"-"`
	AccessExpiresAt  time.Time  `json:"accessExpiresAt"`
	RefreshExpiresAt time.Time  `json:"refreshExpiresAt"`
	Status           string     `gorm:"type:varchar(30);default:'connected'" json:"status"` // connected | disconnected
	ConnectedAt      time.Time  `json:"connectedAt"`
	LastSyncAt       *time.Time `json:"lastSyncAt,omitempty"`
	LastSyncError    string     `gorm:"type:text" json:"lastSyncError,omitempty"`

	// Blibli-specific credentials. Blibli has no OAuth redirect flow: each
	// tenant obtains these four values directly from their own Blibli Seller
	// Center account and types them into a form (see
	// application.ConnectBlibli). They don't fit the OAuth-shaped
	// token/shop_id columns above, so they get their own nullable columns.
	BusinessPartnerCode string `gorm:"type:varchar(100)" json:"businessPartnerCode,omitempty"`
	MtaUsername         string `gorm:"type:varchar(150)" json:"mtaUsername,omitempty"`
	ApiSellerKey        string `gorm:"type:text" json:"-"`
	SignatureKey        string `gorm:"type:text" json:"-"`
}

func (MarketplaceConnection) TableName() string {
	return "marketplace_connections"
}

// MarketplaceOrderSync records that a given marketplace order has already
// been synced into the Sales module, so re-running sync never duplicates a
// Sales Order. Enforced with a unique constraint per platform+order id.
type MarketplaceOrderSync struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this sync record to one business unit within the
	// company (see modules/workspace). Nil means it belongs to no specific
	// tenant - the default state for companies that never created more than
	// their seeded default Tenant.
	TenantID           *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Platform           Platform   `gorm:"type:varchar(30);not null;uniqueIndex:idx_marketplace_order_unique" json:"platform"`
	MarketplaceOrderID string     `gorm:"type:varchar(150);not null;uniqueIndex:idx_marketplace_order_unique" json:"marketplaceOrderId"`
	SalesOrderID       uuid.UUID  `gorm:"type:uuid;not null" json:"salesOrderId"`
	SyncedAt           time.Time  `json:"syncedAt"`
}

func (MarketplaceOrderSync) TableName() string {
	return "marketplace_order_syncs"
}

type MarketplaceRepository interface {
	UpsertConnection(ctx context.Context, conn *MarketplaceConnection) error
	GetConnection(ctx context.Context, platform Platform) (*MarketplaceConnection, error)
	ListConnections(ctx context.Context) ([]MarketplaceConnection, error)
	DeleteConnection(ctx context.Context, platform Platform) error

	IsOrderSynced(ctx context.Context, platform Platform, marketplaceOrderID string) (bool, error)
	RecordOrderSync(ctx context.Context, rec *MarketplaceOrderSync) error
	// GetOrderSyncBySalesOrderID looks up which marketplace order (platform
	// + its own order ID) produced a given Sales Order, so a "ready to
	// ship" request against that Sales Order can be routed to the right
	// marketplace API call. Returns (nil, nil) for a Sales Order that
	// wasn't created by a marketplace sync (manual entry, quotation
	// conversion, POS).
	GetOrderSyncBySalesOrderID(ctx context.Context, salesOrderID uuid.UUID) (*MarketplaceOrderSync, error)
}
