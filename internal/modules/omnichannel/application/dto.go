package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/omnichannel/domain"
	"github.com/google/uuid"
)

// ChannelConnectionResponseDTO is the connection shape returned to clients.
// The access token is intentionally never included, mirroring marketplace's
// ConnectionResponseDTO.
type ChannelConnectionResponseDTO struct {
	Channel           domain.Channel `json:"channel"`
	ChannelLabel      string         `json:"channelLabel"`
	Scope             string         `json:"scope"`
	PhoneNumberID     string         `json:"phoneNumberId,omitempty"`
	BusinessAccountID string         `json:"businessAccountId,omitempty"`
	DisplayName       string         `json:"displayName,omitempty"`
	Status            string         `json:"status"`
	ConnectedAt       time.Time      `json:"connectedAt"`
}

func ToChannelConnectionResponse(c *domain.ChannelConnection) ChannelConnectionResponseDTO {
	return ChannelConnectionResponseDTO{
		Channel:           c.Channel,
		ChannelLabel:      c.Channel.Label(),
		Scope:             c.Scope,
		PhoneNumberID:     c.PhoneNumberID,
		BusinessAccountID: c.BusinessAccountID,
		DisplayName:       c.DisplayName,
		Status:            c.Status,
		ConnectedAt:       c.ConnectedAt,
	}
}

func ToChannelConnectionResponseList(cs []domain.ChannelConnection) []ChannelConnectionResponseDTO {
	out := make([]ChannelConnectionResponseDTO, 0, len(cs))
	for i := range cs {
		out = append(out, ToChannelConnectionResponse(&cs[i]))
	}
	return out
}

// ConversationDTO is the API shape for a conversation list row.
type ConversationDTO struct {
	ID                  uuid.UUID  `json:"id"`
	Channel             string     `json:"channel"`
	ChannelLabel        string     `json:"channelLabel"`
	ExternalContactID   string     `json:"externalContactId"`
	ExternalContactName string     `json:"externalContactName"`
	LastMessagePreview  string     `json:"lastMessagePreview"`
	LastMessageAt       *time.Time `json:"lastMessageAt,omitempty"`
	UnreadCount         int        `json:"unreadCount"`
	Status              string     `json:"status"`
}

// MessageDTO is the API shape for one message in a thread.
type MessageDTO struct {
	ID        uuid.UUID `json:"id"`
	Direction string    `json:"direction"`
	Content   string    `json:"content"`
	SentAt    time.Time `json:"sentAt"`
	SentBy    string    `json:"sentBy,omitempty"`
}

// ThreadDTO is the conversation + its full message history.
type ThreadDTO struct {
	Conversation ConversationDTO `json:"conversation"`
	Messages     []MessageDTO    `json:"messages"`
}

// WhatsAppWebhookPayload mirrors the shape of a Meta WhatsApp Cloud API
// "messages" webhook notification:
//
//	{
//	  "entry": [{
//	    "changes": [{
//	      "value": {
//	        "contacts": [{"profile": {"name": "..."}, "wa_id": "..."}],
//	        "messages": [{"from": "...", "id": "...", "timestamp": "...", "type": "text", "text": {"body": "..."}}]
//	      },
//	      "field": "messages"
//	    }]
//	  }]
//	}
//
// This shape is well-documented and stable in the WhatsApp Cloud API, but
// has NOT been verified against a live Meta webhook delivery in this pass -
// confirm against https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks
// (or an actual captured payload) before relying on it in production.
type WhatsAppWebhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Contacts []struct {
					Profile struct {
						Name string `json:"name"`
					} `json:"profile"`
					WaID string `json:"wa_id"`
				} `json:"contacts"`
				Messages []struct {
					From      string `json:"from"`
					ID        string `json:"id"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
			Field string `json:"field"`
		} `json:"changes"`
	} `json:"entry"`
}
