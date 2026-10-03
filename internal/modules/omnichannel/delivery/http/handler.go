package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/notify"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/omnichannel/application"
	"github.com/divinecoid/one-backend/internal/modules/omnichannel/domain"
	"github.com/divinecoid/one-backend/internal/modules/omnichannel/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/whatsappregistry"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	waCfg    config.WhatsAppConfig
	notifier notify.Notifier
	manager  *tenantMgr.Manager
	registry whatsappregistry.Repository
}

func NewHandler(waCfg config.WhatsAppConfig, notifier notify.Notifier, manager *tenantMgr.Manager, registry whatsappregistry.Repository) *Handler {
	return &Handler{waCfg: waCfg, notifier: notifier, manager: manager, registry: registry}
}

func (h *Handler) useCaseForDB(tenantDB *gorm.DB) application.OmnichannelUseCase {
	repo := infrastructure.NewOmnichannelRepository(tenantDB)
	connRepo := infrastructure.NewChannelConnectionRepository(tenantDB)
	return application.NewOmnichannelUseCase(repo, connRepo, h.registry, h.notifier, h.waCfg)
}

// resolve builds a use-case bound to the caller's own tenant database,
// following the same pattern as the sales/marketplace modules.
func (h *Handler) resolve(c *fiber.Ctx) (application.OmnichannelUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	return h.useCaseForDB(tenantDB), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) ListConversations(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var channel *domain.Channel
	if q := c.Query("channel"); q != "" {
		ch := domain.Channel(q)
		if !ch.Valid() {
			return apperrors.NewBadRequest("Unsupported channel")
		}
		channel = &ch
	}
	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	query.SetDefaults()

	conversations, total, err := uc.ListConversations(h.ctx(c), channel, query)
	if err != nil {
		return err
	}
	meta := types.PaginationMeta{
		CurrentPage: query.Page,
		PerPage:     query.PerPage,
		TotalItems:  total,
		TotalPages:  int((total + int64(query.PerPage) - 1) / int64(query.PerPage)),
	}
	return response.SuccessWithMeta(c, 200, "Conversations retrieved successfully", conversations, meta)
}

func (h *Handler) GetThread(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid conversation id")
	}
	thread, err := uc.GetConversationThread(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Conversation thread retrieved successfully", thread)
}

type sendMessageRequest struct {
	Content string `json:"content"`
}

func (h *Handler) SendMessage(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid conversation id")
	}
	var body sendMessageRequest
	if err := c.BodyParser(&body); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	msg, err := uc.SendMessage(h.ctx(c), id, body.Content)
	if err != nil {
		return err
	}
	return response.Created(c, "Message sent successfully", msg)
}

type connectWhatsAppRequest struct {
	PhoneNumberID     string `json:"phoneNumberId"`
	AccessToken       string `json:"accessToken"`
	BusinessAccountID string `json:"businessAccountId"`
	// Scope is "tenant" (default, only this tenant sees/uses it) or
	// "company" (shared - every tenant in this company sends/receives
	// through this same number). See domain.ConnectionScope*.
	Scope string `json:"scope"`
}

// ConnectWhatsApp handles POST /omnichannel/connections/whatsapp - the
// manual-connect-form + live-verification flow mirroring marketplace's
// BlibliConnect. Protected + tenant-scoped.
func (h *Handler) ConnectWhatsApp(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	claims := middleware.CurrentUser(c)
	if claims == nil || claims.CompanyID == nil {
		return apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	var body connectWhatsAppRequest
	if err := c.BodyParser(&body); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	conn, err := uc.ConnectWhatsApp(h.ctx(c), *claims.CompanyID, body.PhoneNumberID, body.AccessToken, body.BusinessAccountID, body.Scope)
	if err != nil {
		return err
	}
	return response.OK(c, "WhatsApp connected successfully", conn)
}

// ListConnections handles GET /omnichannel/connections. Protected +
// tenant-scoped.
func (h *Handler) ListConnections(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	conns, err := uc.ListChannelConnections(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Channel connections retrieved successfully", conns)
}

// DisconnectChannel handles POST /omnichannel/connections/:channel/disconnect.
// Protected + tenant-scoped.
func (h *Handler) DisconnectChannel(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	channel := domain.Channel(c.Params("channel"))
	if err := uc.DisconnectChannel(h.ctx(c), channel); err != nil {
		return err
	}
	return response.OK(c, "Channel disconnected successfully", nil)
}

// WhatsAppVerify handles Meta's webhook verification handshake:
// GET /omnichannel/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=...&hub.challenge=...
// Public, unauthenticated route - Meta calls this directly.
func (h *Handler) WhatsAppVerify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && h.waCfg.WebhookVerifyToken != "" && token == h.waCfg.WebhookVerifyToken {
		c.Set("Content-Type", "text/plain")
		return c.Status(fiber.StatusOK).SendString(challenge)
	}
	return c.SendStatus(fiber.StatusForbidden)
}

// verifyWhatsAppSignature checks Meta's X-Hub-Signature-256 header
// (HMAC-SHA256 over the
// raw body, keyed by WHATSAPP_APP_SECRET) when that secret is configured -
// if it's empty, verification is skipped and a warning-level rejection does
// not apply (documented tradeoff: Meta requires the App Secret to be
// visible in the dashboard already, so any production deployment should
// set WHATSAPP_APP_SECRET; leaving it unset is only tolerated so the
// feature keeps working before that value is supplied, matching the noop-
// with-explicit-error pattern for genuinely unconfigured integrations
// elsewhere in this codebase - here the "integration" is send-only until
// this is set).
func (h *Handler) verifyWhatsAppSignature(c *fiber.Ctx) bool {
	if h.waCfg.AppSecret == "" {
		return true
	}
	sig := c.Get("X-Hub-Signature-256")
	const prefix = "sha256="
	if !strings.HasPrefix(sig, prefix) {
		return false
	}
	expected := strings.TrimPrefix(sig, prefix)

	mac := hmac.New(sha256.New, []byte(h.waCfg.AppSecret))
	mac.Write(c.Body())
	computed := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(computed))
}

// webhookPhoneNumberIDPeek extracts just
// entry[0].changes[0].value.metadata.phone_number_id from a raw WhatsApp
// webhook body, without committing to the full application.WhatsAppWebhookPayload
// shape - used to resolve which tenant database to route to BEFORE parsing
// the rest of the payload (see WhatsAppWebhook below).
type webhookPhoneNumberIDPeek struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func (p webhookPhoneNumberIDPeek) phoneNumberID() string {
	for _, e := range p.Entry {
		for _, ch := range e.Changes {
			if ch.Value.Metadata.PhoneNumberID != "" {
				return ch.Value.Metadata.PhoneNumberID
			}
		}
	}
	return ""
}

// WhatsAppWebhook receives inbound WhatsApp messages from Meta. Public,
// unauthenticated route - there is no logged-in user, so it cannot use
// middleware.TenantDB/CurrentTenantID.
//
// Routing: the raw body is first peeked for
// entry[0].changes[0].value.metadata.phone_number_id (present on every real
// WhatsApp Cloud API webhook delivery) and looked up in the control-plane
// WhatsAppNumberIndex (see internal/modules/whatsappregistry) to find which
// company registered that number via POST /omnichannel/connections/whatsapp.
// If found, the tenant database is resolved dynamically via
// manager.GetDB(ctx, companyID) - true per-tenant WhatsApp routing.
// If the phone_number_id is unregistered (or missing from the payload),
// this falls back to WHATSAPP_DEFAULT_COMPANY_ID exactly as before, for
// backward compatibility with the single-platform-number deployment. If
// neither resolves, the message is dropped and 200 is returned so Meta
// doesn't retry forever.
//
// Signature verified below via X-Hub-Signature-256 (HMAC-SHA256 over the
// raw body, keyed by the single platform-level WHATSAPP_APP_SECRET) -
// unchanged by per-tenant routing. This assumes one shared Meta App at the
// platform level (an ISV/tech-provider pattern): tenants only ever supply
// their own phone_number_id + access token (see ConnectWhatsApp), never a
// separate App Secret, the same way TikTok Shop/Shopee/Blibli tenants
// authorize their own seller account against ONE ERP's single registered
// app/ISV credentials rather than each running their own app registration
// (confirmed against marketplace/infrastructure/*.go: TikTok's
// AppKey/AppSecret and Shopee's PartnerID/PartnerKey are both single
// platform-level config values in internal/foundation/config, with only the
// OAuth-granted shop token being per-tenant - Blibli likewise keys every
// call with one platform-level APIClientID/APIClientSecret Basic Auth pair
// alongside the tenant's own BusinessPartnerCode/ApiSellerKey).
func (h *Handler) WhatsAppWebhook(c *fiber.Ctx) error {
	if !h.verifyWhatsAppSignature(c) {
		return apperrors.NewUnauthorized("Invalid webhook signature")
	}

	rawBody := c.Body()

	var peek webhookPhoneNumberIDPeek
	_ = json.Unmarshal(rawBody, &peek) // best-effort; falls through to default-company below on failure

	var companyID *uuid.UUID
	var tenantID *uuid.UUID
	if phoneNumberID := peek.phoneNumberID(); phoneNumberID != "" && h.registry != nil {
		idx, err := h.registry.GetByPhoneNumberID(c.UserContext(), phoneNumberID)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to resolve WhatsApp number registry")
		}
		if idx != nil {
			companyID = &idx.CompanyID
			tenantID = idx.TenantID
		}
	}
	if companyID == nil {
		companyID = h.waCfg.DefaultCompanyID()
	}
	if companyID == nil {
		// Not configured/registered - swallow with 200 so Meta doesn't retry
		// forever, but this message is effectively dropped. Documented
		// limitation, not a silent success claim to any caller expecting
		// delivery.
		return c.SendStatus(fiber.StatusOK)
	}

	var payload application.WhatsAppWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return apperrors.NewBadRequest("Invalid webhook payload")
	}

	tenantDB, err := h.manager.GetDB(c.UserContext(), *companyID)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to resolve tenant database")
	}
	uc := h.useCaseForDB(tenantDB)
	// Attach the registry's TenantID (nil for a company-wide connection, or
	// unregistered numbers falling back to WHATSAPP_DEFAULT_COMPANY_ID) so
	// inbound conversations/messages land tagged to the right tenant, the
	// same way an authenticated request would via middleware.CurrentTenantID.
	ctx := tenantctx.WithTenantID(c.UserContext(), tenantID)
	if err := uc.HandleWhatsAppWebhook(ctx, payload); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusOK)
}
