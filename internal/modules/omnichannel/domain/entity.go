package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Channel identifies a supported messaging channel.
type Channel string

const (
	ChannelWhatsApp          Channel = "whatsapp"
	ChannelInstagram         Channel = "instagram"
	ChannelMessenger         Channel = "messenger"
	ChannelMarketplaceShopee Channel = "marketplace_shopee"
	ChannelMarketplaceTikTok Channel = "marketplace_tiktok"
	ChannelManual            Channel = "manual"
)

// Label returns a human-readable label for the channel, used in error
// messages and the frontend channel filter tabs.
func (c Channel) Label() string {
	switch c {
	case ChannelWhatsApp:
		return "WhatsApp"
	case ChannelInstagram:
		return "Instagram"
	case ChannelMessenger:
		return "Messenger"
	case ChannelMarketplaceShopee:
		return "Shopee Chat"
	case ChannelMarketplaceTikTok:
		return "TikTok Shop Chat"
	case ChannelManual:
		return "Manual"
	default:
		return string(c)
	}
}

func (c Channel) Valid() bool {
	switch c {
	case ChannelWhatsApp, ChannelInstagram, ChannelMessenger, ChannelMarketplaceShopee, ChannelMarketplaceTikTok, ChannelManual:
		return true
	default:
		return false
	}
}

const (
	ConversationStatusOpen   = "open"
	ConversationStatusClosed = "closed"
)

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

// Conversation is one thread with a single external contact on a single
// channel, e.g. one WhatsApp phone number. It lives in the tenant's own
// database (never the control-plane DB), tenant-scoped exactly like every
// other business entity in this codebase (see internal/foundation/tenantctx).
type Conversation struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this conversation to one business unit within the
	// company (see modules/workspace). Nil means it belongs to no specific
	// tenant - the default state for companies that never created more than
	// their seeded default Tenant.
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`

	Channel Channel `gorm:"type:varchar(30);not null;index:idx_omni_conv_channel_contact" json:"channel"`
	// ExternalContactID is the platform's own identifier for the contact,
	// e.g. the WhatsApp wa_id (phone number without '+'). Combined with
	// Channel + TenantID this is the dedupe key used by webhook upserts
	// (see GetConversationByChannelAndContact).
	ExternalContactID   string     `gorm:"type:varchar(150);not null;index:idx_omni_conv_channel_contact" json:"externalContactId"`
	ExternalContactName string     `gorm:"type:varchar(255)" json:"externalContactName"`
	LastMessagePreview  string     `gorm:"type:text" json:"lastMessagePreview"`
	LastMessageAt       *time.Time `json:"lastMessageAt,omitempty"`
	UnreadCount         int        `gorm:"default:0" json:"unreadCount"`
	Status              string     `gorm:"type:varchar(20);default:'open'" json:"status"`
}

func (Conversation) TableName() string {
	return "omnichannel_conversations"
}

// Message is one message within a Conversation, either received from the
// external contact (inbound) or sent by a user of this ERP (outbound).
type Message struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`

	ConversationID uuid.UUID `gorm:"type:uuid;not null;index" json:"conversationId"`
	Direction      string    `gorm:"type:varchar(10);not null" json:"direction"` // inbound | outbound
	Content        string    `gorm:"type:text;not null" json:"content"`
	// ExternalMessageID is the platform's own message id (e.g. WhatsApp's
	// message `id` field), used to dedupe inbound webhook retries. Empty for
	// outbound messages we sent ourselves.
	ExternalMessageID string    `gorm:"type:varchar(150);index" json:"externalMessageId,omitempty"`
	SentAt            time.Time `gorm:"not null" json:"sentAt"`
	// SentBy is the ERP user's display name for outbound messages, empty for
	// inbound messages from the external contact.
	SentBy string `gorm:"type:varchar(255)" json:"sentBy,omitempty"`
}

func (Message) TableName() string {
	return "omnichannel_messages"
}

// ChannelConnection stores one tenant's own credentials for a messaging
// channel (currently WhatsApp Cloud API only). It lives in the tenant's own
// database (never the control-plane DB), tenant-scoped exactly like every
// other entity in this module, mirroring marketplace's MarketplaceConnection
// (see internal/modules/marketplace/domain/entity.go) - manual connect form
// + live API verification before being marked "connected", not OAuth.
//
// NOTE: AccessToken is stored as plain text, same precedent as
// MarketplaceConnection.AccessToken/RefreshToken - no field-level encryption
// helper exists yet in this codebase (grepped for "encrypt", none found).
// Never logged; never returned by any API response (see
// application.ConnectionResponseDTO masking).
// Connection scope: whether a connected channel number is shared by every
// Tenant in the Company (ConnectionScopeCompany, TenantID left nil) or
// exclusive to the Tenant that connected it (ConnectionScopeTenant, the
// default - TenantID set to whichever tenant was active at connect time).
// The client picks this explicitly in the connect form; it is not inferred.
const (
	ConnectionScopeTenant  = "tenant"
	ConnectionScopeCompany = "company"
)

type ChannelConnection struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this connection to one business unit within the
	// company (see modules/workspace), same convention as Conversation/Message
	// above. Nil means it belongs to no specific tenant - either because the
	// company never created more than its default tenant, or because Scope
	// is explicitly ConnectionScopeCompany (shared by every tenant).
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	// Scope is ConnectionScopeTenant or ConnectionScopeCompany - see the
	// constants above. Determines whether TenantID is set at connect time.
	Scope             string    `gorm:"type:varchar(20);not null;default:'tenant'" json:"scope"`
	Channel           Channel   `gorm:"type:varchar(30);not null;index" json:"channel"`
	PhoneNumberID     string    `gorm:"type:varchar(100)" json:"phoneNumberId,omitempty"`
	AccessToken       string    `gorm:"type:text" json:"-"`
	BusinessAccountID string    `gorm:"type:varchar(100)" json:"businessAccountId,omitempty"`
	DisplayName       string    `gorm:"type:varchar(150)" json:"displayName,omitempty"`
	Status            string    `gorm:"type:varchar(20);default:'connected'" json:"status"`
	ConnectedAt       time.Time `json:"connectedAt"`
}

func (ChannelConnection) TableName() string {
	return "omnichannel_channel_connections"
}

// ChannelConnectionRepository is the persistence port for tenant channel
// connections, kept separate from Repository (Conversation/Message) so the
// two concerns can be tested/composed independently, mirroring how
// marketplace keeps MarketplaceRepository as a single port but this module
// already splits Conversation/Message concerns across a shared Repository -
// a second small interface here avoids bloating that one further.
type ChannelConnectionRepository interface {
	// UpsertConnection creates the connection for a channel, or overwrites
	// the existing one on reconnect (one active connection per tenant per
	// channel), mirroring marketplace's UpsertConnection pattern exactly.
	UpsertConnection(ctx context.Context, conn *ChannelConnection) error
	// GetActiveConnection resolves the connection this tenant should use for
	// a channel: a tenant-specific connection if one exists, else the
	// company-wide (ConnectionScopeCompany) one if the company has one, else
	// nil. Used for sending and disconnecting.
	GetActiveConnection(ctx context.Context, channel Channel) (*ChannelConnection, error)
	// ListConnections returns every connection visible to the active tenant:
	// their own tenant-specific ones plus any company-wide ones.
	ListConnections(ctx context.Context) ([]ChannelConnection, error)
	DeleteConnection(ctx context.Context, id uuid.UUID) error
}

// Repository is the persistence port for the omnichannel module.
type Repository interface {
	CreateConversation(ctx context.Context, conv *Conversation) error
	GetConversationByID(ctx context.Context, id uuid.UUID) (*Conversation, error)
	// GetConversationByChannelAndContact finds the existing conversation for
	// a channel+external contact within the active tenant, used to
	// find-or-create on inbound webhook delivery. Returns (nil, nil) when
	// none exists yet.
	GetConversationByChannelAndContact(ctx context.Context, channel Channel, externalContactID string) (*Conversation, error)
	ListConversations(ctx context.Context, channel *Channel, query types.PaginationQuery) ([]Conversation, int64, error)
	UpdateConversation(ctx context.Context, conv *Conversation) error
	MarkConversationRead(ctx context.Context, id uuid.UUID) error

	CreateMessage(ctx context.Context, msg *Message) error
	ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID) ([]Message, error)
	// GetMessageByExternalID looks up an existing message by its platform
	// message id within the active tenant, used to dedupe webhook retries.
	GetMessageByExternalID(ctx context.Context, externalMessageID string) (*Message, error)
}
