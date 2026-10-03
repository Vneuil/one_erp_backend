package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/collaboration/domain"
	"github.com/google/uuid"
)

type CreateConversationDTO struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	ParticipantNames string `json:"participantNames"`
}

type SendMessageDTO struct {
	SenderName     string  `json:"senderName"`
	Content        string  `json:"content"`
	AttachmentName *string `json:"attachmentName,omitempty"`
}

type ConversationResponseDTO struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	ParticipantNames string    `json:"participantNames"`
	CreatedAt        time.Time `json:"createdAt"`
}

type MessageResponseDTO struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversationId"`
	SenderName     string    `json:"senderName"`
	Content        string    `json:"content"`
	AttachmentName *string   `json:"attachmentName,omitempty"`
	SentAt         time.Time `json:"sentAt"`
}

func ToConversationResponse(c *domain.Conversation) *ConversationResponseDTO {
	if c == nil {
		return nil
	}
	return &ConversationResponseDTO{
		ID:               c.ID,
		Name:             c.Name,
		Type:             c.Type,
		ParticipantNames: c.ParticipantNames,
		CreatedAt:        c.CreatedAt,
	}
}

func ToConversationResponseList(items []domain.Conversation) []ConversationResponseDTO {
	result := make([]ConversationResponseDTO, len(items))
	for i, c := range items {
		result[i] = *ToConversationResponse(&c)
	}
	return result
}

func ToMessageResponse(m *domain.Message) *MessageResponseDTO {
	if m == nil {
		return nil
	}
	return &MessageResponseDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderName:     m.SenderName,
		Content:        m.Content,
		AttachmentName: m.AttachmentName,
		SentAt:         m.SentAt,
	}
}

func ToMessageResponseList(items []domain.Message) []MessageResponseDTO {
	result := make([]MessageResponseDTO, len(items))
	for i, m := range items {
		result[i] = *ToMessageResponse(&m)
	}
	return result
}
