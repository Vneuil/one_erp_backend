package application

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

var validConversationTypes = map[string]bool{
	"direct": true,
	"group":  true,
}

type ChatUseCase interface {
	CreateConversation(ctx context.Context, dto CreateConversationDTO) (*ConversationResponseDTO, error)
	ListConversations(ctx context.Context) ([]ConversationResponseDTO, error)
	SendMessage(ctx context.Context, conversationID uuid.UUID, dto SendMessageDTO) (*MessageResponseDTO, error)
	ListMessages(ctx context.Context, conversationID uuid.UUID) ([]MessageResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type chatUseCase struct {
	repo domain.ChatRepository
}

func NewChatUseCase(repo domain.ChatRepository) ChatUseCase {
	return &chatUseCase{repo: repo}
}

func (uc *chatUseCase) CreateConversation(ctx context.Context, dto CreateConversationDTO) (*ConversationResponseDTO, error) {
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("Conversation name is required")
	}
	if !validConversationTypes[dto.Type] {
		return nil, apperrors.NewBadRequest("Type must be one of direct, group")
	}
	if dto.ParticipantNames == "" {
		return nil, apperrors.NewBadRequest("Participant names is required")
	}

	c := &domain.Conversation{
		Name:             dto.Name,
		Type:             dto.Type,
		ParticipantNames: dto.ParticipantNames,
	}
	if err := uc.repo.CreateConversation(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create conversation")
	}
	return ToConversationResponse(c), nil
}

func (uc *chatUseCase) ListConversations(ctx context.Context) ([]ConversationResponseDTO, error) {
	items, err := uc.repo.ListConversations(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list conversations")
	}
	return ToConversationResponseList(items), nil
}

func (uc *chatUseCase) SendMessage(ctx context.Context, conversationID uuid.UUID, dto SendMessageDTO) (*MessageResponseDTO, error) {
	c, err := uc.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get conversation")
	}
	if c == nil {
		return nil, apperrors.NewNotFound("Conversation not found")
	}
	if dto.SenderName == "" {
		return nil, apperrors.NewBadRequest("Sender name is required")
	}
	if dto.Content == "" {
		return nil, apperrors.NewBadRequest("Message content is required")
	}

	m := &domain.Message{
		ConversationID: conversationID,
		SenderName:     dto.SenderName,
		Content:        dto.Content,
		AttachmentName: dto.AttachmentName,
		SentAt:         time.Now(),
	}
	if err := uc.repo.CreateMessage(ctx, m); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to send message")
	}
	return ToMessageResponse(m), nil
}

func (uc *chatUseCase) ListMessages(ctx context.Context, conversationID uuid.UUID) ([]MessageResponseDTO, error) {
	c, err := uc.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get conversation")
	}
	if c == nil {
		return nil, apperrors.NewNotFound("Conversation not found")
	}
	items, err := uc.repo.ListMessagesByConversation(ctx, conversationID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list messages")
	}
	return ToMessageResponseList(items), nil
}

// SeedInitialData populates a few sample conversations with messages on first boot
func (uc *chatUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountConversations(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	seeds := []struct {
		dto      CreateConversationDTO
		messages []SendMessageDTO
	}{
		{
			dto: CreateConversationDTO{
				Name:             "warehouse-ops-cikarang",
				Type:             "group",
				ParticipantNames: "Nicholas Tantra, Budi Santoso, Rizky Pratama",
			},
			messages: []SendMessageDTO{
				{SenderName: "Budi Santoso", Content: "Truk kontainer bahan baku sudah tiba di dock 2 Gudang Cikarang."},
				{SenderName: "Nicholas Tantra", Content: "Mantap, tolong tim QC lakukan sampling gramatur sebelum dibongkar."},
			},
		},
		{
			dto: CreateConversationDTO{
				Name:             "general-announcements",
				Type:             "group",
				ParticipantNames: "Nicholas Tantra, Dewi Lestari, Sarah Amelia",
			},
			messages: []SendMessageDTO{
				{SenderName: "Nicholas Tantra", Content: "Sistem ONE ERP 1.0 telah resmi aktif di seluruh cabang."},
			},
		},
		{
			dto: CreateConversationDTO{
				Name:             "Dewi Lestari",
				Type:             "direct",
				ParticipantNames: "Nicholas Tantra, Dewi Lestari",
			},
			messages: []SendMessageDTO{
				{SenderName: "Dewi Lestari", Content: "Laporan rekonsiliasi bank bulan ini sudah selesai, mohon direview."},
			},
		},
	}

	for _, seed := range seeds {
		res, err := uc.CreateConversation(ctx, seed.dto)
		if err != nil {
			continue
		}
		for _, m := range seed.messages {
			if _, err := uc.SendMessage(ctx, res.ID, m); err != nil {
				continue
			}
		}
	}
	return nil
}
