package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/integration/domain"
	"github.com/google/uuid"
)

type CreateAPIKeyDTO struct {
	Name      string     `json:"name"`
	Scopes    string     `json:"scopes"`
	CompanyID *uuid.UUID `json:"companyId,omitempty"`
}

type APIKeyResponseDTO struct {
	ID         uuid.UUID  `json:"id"`
	CompanyID  *uuid.UUID `json:"companyId,omitempty"`
	Name       string     `json:"name"`
	KeyValue   string     `json:"keyValue,omitempty"`
	MaskedKey  string     `json:"maskedKey"`
	Scopes     string     `json:"scopes"`
	IsActive   bool       `json:"isActive"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func maskKey(key string) string {
	if len(key) <= 10 {
		return key
	}
	return key[:8] + "..." + key[len(key)-4:]
}

// ToAPIKeyResponse builds the response DTO. The full plaintext key_value is only
// included when reveal=true (immediately after creation) — every other path
// (list, revoke) must only expose the masked form.
func ToAPIKeyResponse(k *domain.APIKey, reveal bool) *APIKeyResponseDTO {
	if k == nil {
		return nil
	}
	dto := &APIKeyResponseDTO{
		ID:         k.ID,
		CompanyID:  k.CompanyID,
		Name:       k.Name,
		MaskedKey:  maskKey(k.KeyValue),
		Scopes:     k.Scopes,
		IsActive:   k.IsActive,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
	}
	if reveal {
		dto.KeyValue = k.KeyValue
	}
	return dto
}

func ToAPIKeyResponseList(keys []domain.APIKey) []APIKeyResponseDTO {
	result := make([]APIKeyResponseDTO, len(keys))
	for i, k := range keys {
		result[i] = *ToAPIKeyResponse(&k, false)
	}
	return result
}

type CreateWebhookDTO struct {
	Name      string     `json:"name"`
	TargetURL string     `json:"targetUrl"`
	EventType string     `json:"eventType"`
	CompanyID *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateWebhookDTO struct {
	Name      string `json:"name"`
	TargetURL string `json:"targetUrl"`
	EventType string `json:"eventType"`
	IsActive  *bool  `json:"isActive,omitempty"`
}

type WebhookResponseDTO struct {
	ID        uuid.UUID  `json:"id"`
	CompanyID *uuid.UUID `json:"companyId,omitempty"`
	Name      string     `json:"name"`
	TargetURL string     `json:"targetUrl"`
	EventType string     `json:"eventType"`
	IsActive  bool       `json:"isActive"`
	Secret    string     `json:"secret"`
	CreatedAt time.Time  `json:"createdAt"`
}

func ToWebhookResponse(w *domain.WebhookSubscription) *WebhookResponseDTO {
	if w == nil {
		return nil
	}
	return &WebhookResponseDTO{
		ID:        w.ID,
		CompanyID: w.CompanyID,
		Name:      w.Name,
		TargetURL: w.TargetURL,
		EventType: w.EventType,
		IsActive:  w.IsActive,
		Secret:    w.Secret,
		CreatedAt: w.CreatedAt,
	}
}

func ToWebhookResponseList(webhooks []domain.WebhookSubscription) []WebhookResponseDTO {
	result := make([]WebhookResponseDTO, len(webhooks))
	for i, w := range webhooks {
		result[i] = *ToWebhookResponse(&w)
	}
	return result
}

type WebhookDeliveryResponseDTO struct {
	ID                  uuid.UUID `json:"id"`
	SubscriptionID      uuid.UUID `json:"subscriptionId"`
	EventType           string    `json:"eventType"`
	Payload             string    `json:"payload"`
	Status              string    `json:"status"`
	ResponseCode        *int      `json:"responseCode,omitempty"`
	AttemptedAt         time.Time `json:"attemptedAt"`
	ResponseBodySnippet *string   `json:"responseBodySnippet,omitempty"`
}

func ToWebhookDeliveryResponse(d *domain.WebhookDelivery) *WebhookDeliveryResponseDTO {
	if d == nil {
		return nil
	}
	return &WebhookDeliveryResponseDTO{
		ID:                  d.ID,
		SubscriptionID:      d.SubscriptionID,
		EventType:           d.EventType,
		Payload:             d.Payload,
		Status:              d.Status,
		ResponseCode:        d.ResponseCode,
		AttemptedAt:         d.AttemptedAt,
		ResponseBodySnippet: d.ResponseBodySnippet,
	}
}

func ToWebhookDeliveryResponseList(deliveries []domain.WebhookDelivery) []WebhookDeliveryResponseDTO {
	result := make([]WebhookDeliveryResponseDTO, len(deliveries))
	for i, d := range deliveries {
		result[i] = *ToWebhookDeliveryResponse(&d)
	}
	return result
}
