package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mathrand "math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/integration/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

var deliveryRand = mathrand.New(mathrand.NewSource(time.Now().UnixNano()))

var validEventTypes = map[string]bool{
	"sales.order.created":          true,
	"inventory.stock.low":          true,
	"procurement.po.approved":      true,
	"finance.journal.posted":       true,
	"hrm.employee.hired":           true,
	"support.ticket.created":       true,
	"manufacturing.bom.updated":    true,
	"warehouse.pickwave.completed": true,
}

func ValidEventTypes() []string {
	types := make([]string, 0, len(validEventTypes))
	for t := range validEventTypes {
		types = append(types, t)
	}
	return types
}

func generateToken(prefix string, byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return prefix + fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return prefix + hex.EncodeToString(b)
}

func isValidURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

type IntegrationUseCase interface {
	CreateAPIKey(ctx context.Context, dto CreateAPIKeyDTO) (*APIKeyResponseDTO, error)
	ListAPIKeys(ctx context.Context) ([]APIKeyResponseDTO, error)
	RevokeAPIKey(ctx context.Context, id uuid.UUID) (*APIKeyResponseDTO, error)

	CreateWebhook(ctx context.Context, dto CreateWebhookDTO) (*WebhookResponseDTO, error)
	ListWebhooks(ctx context.Context) ([]WebhookResponseDTO, error)
	UpdateWebhook(ctx context.Context, id uuid.UUID, dto UpdateWebhookDTO) (*WebhookResponseDTO, error)
	TestWebhook(ctx context.Context, id uuid.UUID) (*WebhookDeliveryResponseDTO, error)
	ListDeliveries(ctx context.Context, subscriptionID uuid.UUID) ([]WebhookDeliveryResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type integrationUseCase struct {
	apiKeyRepo  domain.APIKeyRepository
	webhookRepo domain.WebhookRepository
}

func NewIntegrationUseCase(apiKeyRepo domain.APIKeyRepository, webhookRepo domain.WebhookRepository) IntegrationUseCase {
	return &integrationUseCase{apiKeyRepo: apiKeyRepo, webhookRepo: webhookRepo}
}

func (uc *integrationUseCase) CreateAPIKey(ctx context.Context, dto CreateAPIKeyDTO) (*APIKeyResponseDTO, error) {
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("API key name is required")
	}
	scopes := strings.TrimSpace(dto.Scopes)
	if scopes == "" {
		scopes = "read:all"
	}

	k := &domain.APIKey{
		CompanyID: dto.CompanyID,
		Name:      dto.Name,
		KeyValue:  generateToken("sk_live_", 16),
		Scopes:    scopes,
		IsActive:  true,
	}
	if err := uc.apiKeyRepo.Create(ctx, k); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create API key")
	}
	return ToAPIKeyResponse(k, true), nil
}

func (uc *integrationUseCase) ListAPIKeys(ctx context.Context) ([]APIKeyResponseDTO, error) {
	keys, err := uc.apiKeyRepo.ListAll(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list API keys")
	}
	return ToAPIKeyResponseList(keys), nil
}

func (uc *integrationUseCase) RevokeAPIKey(ctx context.Context, id uuid.UUID) (*APIKeyResponseDTO, error) {
	k, err := uc.apiKeyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get API key")
	}
	if k == nil {
		return nil, apperrors.NewNotFound("API key not found")
	}
	k.IsActive = false
	if err := uc.apiKeyRepo.Update(ctx, k); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to revoke API key")
	}
	return ToAPIKeyResponse(k, false), nil
}

func (uc *integrationUseCase) CreateWebhook(ctx context.Context, dto CreateWebhookDTO) (*WebhookResponseDTO, error) {
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("Webhook name is required")
	}
	if !isValidURL(dto.TargetURL) {
		return nil, apperrors.NewBadRequest("Target URL must be a valid http(s) URL")
	}
	if !validEventTypes[dto.EventType] {
		return nil, apperrors.NewBadRequest("Event type must be one of the supported event types")
	}

	w := &domain.WebhookSubscription{
		CompanyID: dto.CompanyID,
		Name:      dto.Name,
		TargetURL: dto.TargetURL,
		EventType: dto.EventType,
		IsActive:  true,
		Secret:    generateToken("whsec_", 16),
	}
	if err := uc.webhookRepo.Create(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create webhook subscription")
	}
	return ToWebhookResponse(w), nil
}

func (uc *integrationUseCase) ListWebhooks(ctx context.Context) ([]WebhookResponseDTO, error) {
	webhooks, err := uc.webhookRepo.ListAll(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list webhook subscriptions")
	}
	return ToWebhookResponseList(webhooks), nil
}

func (uc *integrationUseCase) UpdateWebhook(ctx context.Context, id uuid.UUID, dto UpdateWebhookDTO) (*WebhookResponseDTO, error) {
	w, err := uc.webhookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get webhook subscription")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Webhook subscription not found")
	}
	if dto.Name != "" {
		w.Name = dto.Name
	}
	if dto.TargetURL != "" {
		if !isValidURL(dto.TargetURL) {
			return nil, apperrors.NewBadRequest("Target URL must be a valid http(s) URL")
		}
		w.TargetURL = dto.TargetURL
	}
	if dto.EventType != "" {
		if !validEventTypes[dto.EventType] {
			return nil, apperrors.NewBadRequest("Event type must be one of the supported event types")
		}
		w.EventType = dto.EventType
	}
	if dto.IsActive != nil {
		w.IsActive = *dto.IsActive
	}

	if err := uc.webhookRepo.Update(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update webhook subscription")
	}
	return ToWebhookResponse(w), nil
}

// TestWebhook simulates a delivery attempt. There is no real outbound HTTP
// infrastructure in this scope, so the attempt is faked: it succeeds with a
// 200 roughly 85% of the time and otherwise fails with a 500. This avoids
// making real HTTP requests to user-provided URLs (an SSRF-shaped risk).
func (uc *integrationUseCase) TestWebhook(ctx context.Context, id uuid.UUID) (*WebhookDeliveryResponseDTO, error) {
	w, err := uc.webhookRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get webhook subscription")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Webhook subscription not found")
	}

	payload := fmt.Sprintf(`{"event":"%s","webhookId":"%s","simulated":true,"timestamp":"%s"}`, w.EventType, w.ID, time.Now().Format(time.RFC3339))

	success := deliveryRand.Float64() < 0.85
	d := &domain.WebhookDelivery{
		SubscriptionID: w.ID,
		EventType:      w.EventType,
		Payload:        payload,
		AttemptedAt:    time.Now(),
	}
	if success {
		code := 200
		snippet := "OK"
		d.Status = "delivered"
		d.ResponseCode = &code
		d.ResponseBodySnippet = &snippet
	} else {
		code := 500
		snippet := "Internal Server Error (simulated)"
		d.Status = "failed"
		d.ResponseCode = &code
		d.ResponseBodySnippet = &snippet
	}

	if err := uc.webhookRepo.CreateDelivery(ctx, d); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to log webhook delivery")
	}
	return ToWebhookDeliveryResponse(d), nil
}

func (uc *integrationUseCase) ListDeliveries(ctx context.Context, subscriptionID uuid.UUID) ([]WebhookDeliveryResponseDTO, error) {
	w, err := uc.webhookRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get webhook subscription")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Webhook subscription not found")
	}
	deliveries, err := uc.webhookRepo.ListDeliveriesBySubscription(ctx, subscriptionID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list webhook deliveries")
	}
	return ToWebhookDeliveryResponseList(deliveries), nil
}

// SeedInitialData populates a few sample API keys, webhook subscriptions,
// and delivery log entries on first boot.
func (uc *integrationUseCase) SeedInitialData(ctx context.Context) error {
	keyCount, err := uc.apiKeyRepo.Count(ctx)
	if err != nil {
		return err
	}
	if keyCount == 0 {
		keySeeds := []CreateAPIKeyDTO{
			{Name: "ERP Integration - Production", Scopes: "read:all,write:sales,write:inventory"},
			{Name: "Reporting Dashboard", Scopes: "read:finance,read:sales"},
			{Name: "Mobile App Client", Scopes: "read:all"},
		}
		for _, seed := range keySeeds {
			if _, err := uc.CreateAPIKey(ctx, seed); err != nil {
				continue
			}
		}
	}

	whCount, err := uc.webhookRepo.Count(ctx)
	if err != nil {
		return err
	}
	if whCount == 0 {
		whSeeds := []CreateWebhookDTO{
			{Name: "Sales Order Notifier", TargetURL: "https://example.com/hooks/sales-order", EventType: "sales.order.created"},
			{Name: "Low Stock Alert", TargetURL: "https://example.com/hooks/low-stock", EventType: "inventory.stock.low"},
			{Name: "PO Approval Sync", TargetURL: "https://partner.example.org/webhooks/po-approved", EventType: "procurement.po.approved"},
			{Name: "Support Ticket Bridge", TargetURL: "https://helpdesk.example.net/incoming", EventType: "support.ticket.created"},
		}
		var firstID uuid.UUID
		for i, seed := range whSeeds {
			res, err := uc.CreateWebhook(ctx, seed)
			if err != nil {
				continue
			}
			if i == 0 {
				firstID = res.ID
			}
		}
		if firstID != uuid.Nil {
			_, _ = uc.TestWebhook(ctx, firstID)
			_, _ = uc.TestWebhook(ctx, firstID)
			_, _ = uc.TestWebhook(ctx, firstID)
		}
	}

	return nil
}
