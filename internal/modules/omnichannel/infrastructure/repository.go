package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/omnichannel/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type omnichannelRepository struct {
	db *gorm.DB
}

func NewOmnichannelRepository(db *gorm.DB) domain.Repository {
	return &omnichannelRepository{db: db}
}

func (r *omnichannelRepository) CreateConversation(ctx context.Context, conv *domain.Conversation) error {
	tenantctx.SetTenantID(ctx, &conv.TenantID)
	return r.db.WithContext(ctx).Create(conv).Error
}

func (r *omnichannelRepository) GetConversationByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	var conv domain.Conversation
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("id = ?", id).First(&conv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *omnichannelRepository) GetConversationByChannelAndContact(ctx context.Context, channel domain.Channel, externalContactID string) (*domain.Conversation, error) {
	var conv domain.Conversation
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("channel = ? AND external_contact_id = ?", channel, externalContactID).First(&conv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conv, nil
}

func (r *omnichannelRepository) ListConversations(ctx context.Context, channel *domain.Channel, query types.PaginationQuery) ([]domain.Conversation, int64, error) {
	var conversations []domain.Conversation
	var total int64

	db := tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Conversation{}))
	if channel != nil {
		db = db.Where("channel = ?", *channel)
	}
	if query.Search != "" {
		searchPattern := "%" + query.Search + "%"
		db = db.Where("external_contact_name ILIKE ? OR external_contact_id ILIKE ?", searchPattern, searchPattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("last_message_at desc nulls last, created_at desc").Offset(offset).Limit(query.PerPage).Find(&conversations).Error
	return conversations, total, err
}

func (r *omnichannelRepository) UpdateConversation(ctx context.Context, conv *domain.Conversation) error {
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	return db.Model(&domain.Conversation{}).Where("id = ?", conv.ID).Updates(map[string]any{
		"external_contact_name": conv.ExternalContactName,
		"last_message_preview":  conv.LastMessagePreview,
		"last_message_at":       conv.LastMessageAt,
		"unread_count":          conv.UnreadCount,
		"status":                conv.Status,
	}).Error
}

func (r *omnichannelRepository) MarkConversationRead(ctx context.Context, id uuid.UUID) error {
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	return db.Model(&domain.Conversation{}).Where("id = ?", id).Update("unread_count", 0).Error
}

func (r *omnichannelRepository) CreateMessage(ctx context.Context, msg *domain.Message) error {
	tenantctx.SetTenantID(ctx, &msg.TenantID)
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *omnichannelRepository) ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID) ([]domain.Message, error) {
	var messages []domain.Message
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("conversation_id = ?", conversationID).Order("sent_at asc").Find(&messages).Error
	return messages, err
}

// channelConnectionRepository persists domain.ChannelConnection, mirroring
// marketplace's marketplaceRepository.UpsertConnection exactly (find by
// unique key, Create if absent else Updates by ID).
type channelConnectionRepository struct {
	db *gorm.DB
}

func NewChannelConnectionRepository(db *gorm.DB) domain.ChannelConnectionRepository {
	return &channelConnectionRepository{db: db}
}

// UpsertConnection finds the existing row to overwrite using a scope-aware
// lookup: ConnectionScopeCompany connections are keyed by
// channel+tenant_id-IS-NULL (shared, never tied to whichever tenant happens
// to be active when reconnecting), ConnectionScopeTenant ones by
// channel+the active tenant (tenantctx.Scope - a no-op filter when no
// tenant is active, matching the company's single default tenant).
func (r *channelConnectionRepository) UpsertConnection(ctx context.Context, conn *domain.ChannelConnection) error {
	var existing domain.ChannelConnection
	var err error

	if conn.Scope == domain.ConnectionScopeCompany {
		conn.TenantID = nil // explicit shared connection, never auto-set from context
		err = r.db.WithContext(ctx).
			Where("channel = ? AND tenant_id IS NULL AND scope = ?", conn.Channel, domain.ConnectionScopeCompany).
			First(&existing).Error
	} else {
		conn.Scope = domain.ConnectionScopeTenant
		tenantctx.SetTenantID(ctx, &conn.TenantID)
		err = tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("channel = ?", conn.Channel).First(&existing).Error
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(conn).Error
		}
		return err
	}

	conn.ID = existing.ID
	return r.db.WithContext(ctx).Model(&domain.ChannelConnection{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"phone_number_id":     conn.PhoneNumberID,
		"access_token":        conn.AccessToken,
		"business_account_id": conn.BusinessAccountID,
		"display_name":        conn.DisplayName,
		"status":              conn.Status,
		"connected_at":        conn.ConnectedAt,
		"scope":               conn.Scope,
	}).Error
}

// GetActiveConnection tries this tenant's own connection first, then falls
// back to a company-wide (ConnectionScopeCompany) one. When no tenant is
// active, tenantctx.Scope adds no filter, so the first lookup already
// matches either kind - the fallback only matters when a specific tenant IS
// active but hasn't connected its own number while the company has a
// shared one.
func (r *channelConnectionRepository) GetActiveConnection(ctx context.Context, channel domain.Channel) (*domain.ChannelConnection, error) {
	var conn domain.ChannelConnection
	err := tenantctx.Scope(ctx, r.db.WithContext(ctx)).Where("channel = ?", channel).First(&conn).Error
	if err == nil {
		return &conn, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err = r.db.WithContext(ctx).
		Where("channel = ? AND tenant_id IS NULL AND scope = ?", channel, domain.ConnectionScopeCompany).
		First(&conn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conn, nil
}

// ListConnections returns this tenant's own connections plus any
// company-wide ones, so the UI can show both "your tenant" and "shared by
// company" connections together.
func (r *channelConnectionRepository) ListConnections(ctx context.Context) ([]domain.ChannelConnection, error) {
	var conns []domain.ChannelConnection
	db := r.db.WithContext(ctx)
	if tid := tenantctx.FromContext(ctx); tid != nil {
		db = db.Where("tenant_id = ? OR tenant_id IS NULL", *tid)
	}
	err := db.Order("created_at desc").Find(&conns).Error
	return conns, err
}

func (r *channelConnectionRepository) DeleteConnection(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.ChannelConnection{}).Error
}

func (r *omnichannelRepository) GetMessageByExternalID(ctx context.Context, externalMessageID string) (*domain.Message, error) {
	if externalMessageID == "" {
		return nil, nil
	}
	var msg domain.Message
	db := tenantctx.Scope(ctx, r.db.WithContext(ctx))
	err := db.Where("external_message_id = ?", externalMessageID).First(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &msg, nil
}
