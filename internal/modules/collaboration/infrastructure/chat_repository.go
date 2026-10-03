package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/foundation/companyctx"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) domain.ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) CreateConversation(ctx context.Context, c *domain.Conversation) error {
	companyctx.SetCompanyID(ctx, &c.CompanyID)
	tenantctx.SetTenantID(ctx, &c.TenantID)
	return r.db.WithContext(ctx).Create(c).Error
}

// GetConversationByID is scoped by company: a conversation ID from another
// company must never resolve, even if guessed/enumerated.
func (r *chatRepository) GetConversationByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	var c domain.Conversation
	err := companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *chatRepository) ListConversations(ctx context.Context) ([]domain.Conversation, error) {
	var items []domain.Conversation
	db := companyctx.Scope(ctx, r.db.WithContext(ctx))
	err := tenantctx.Scope(ctx, db).Order("created_at desc").Find(&items).Error
	return items, err
}

func (r *chatRepository) CountConversations(ctx context.Context) (int64, error) {
	var total int64
	err := companyctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Conversation{})).Count(&total).Error
	return total, err
}

func (r *chatRepository) CreateMessage(ctx context.Context, m *domain.Message) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// ListMessagesByConversation trusts conversationID has already been
// resolved through GetConversationByID (company-scoped) by the caller -
// see application.ChatUseCase.ListMessages.
func (r *chatRepository) ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID) ([]domain.Message, error) {
	var items []domain.Message
	err := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("sent_at asc").Find(&items).Error
	return items, err
}
