package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/divinecoid/one-backend/internal/foundation/notify"
	"github.com/divinecoid/one-backend/internal/modules/omnichannel/domain"
	"github.com/divinecoid/one-backend/internal/modules/whatsappregistry"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type OmnichannelUseCase interface {
	ListConversations(ctx context.Context, channel *domain.Channel, query types.PaginationQuery) ([]ConversationDTO, int64, error)
	GetConversationThread(ctx context.Context, conversationID uuid.UUID) (*ThreadDTO, error)
	SendMessage(ctx context.Context, conversationID uuid.UUID, content string) (*MessageDTO, error)
	HandleWhatsAppWebhook(ctx context.Context, payload WhatsAppWebhookPayload) error

	// ConnectWhatsApp verifies and stores a tenant's own WhatsApp Cloud API
	// credentials, mirroring marketplace's ConnectBlibli. companyID is
	// required (not read from ctx) because it also keys the control-plane
	// WhatsAppNumberIndex row used for webhook routing - see
	// internal/modules/whatsappregistry.
	ConnectWhatsApp(ctx context.Context, companyID uuid.UUID, phoneNumberID, accessToken, businessAccountID, scope string) (*ChannelConnectionResponseDTO, error)
	ListChannelConnections(ctx context.Context) ([]ChannelConnectionResponseDTO, error)
	DisconnectChannel(ctx context.Context, channel domain.Channel) error
}

type omnichannelUseCase struct {
	repo     domain.Repository
	connRepo domain.ChannelConnectionRepository
	registry whatsappregistry.Repository
	notifier notify.Notifier
	waCfg    config.WhatsAppConfig

	httpClient *http.Client
}

func NewOmnichannelUseCase(repo domain.Repository, connRepo domain.ChannelConnectionRepository, registry whatsappregistry.Repository, notifier notify.Notifier, waCfg config.WhatsAppConfig) OmnichannelUseCase {
	return &omnichannelUseCase{
		repo:       repo,
		connRepo:   connRepo,
		registry:   registry,
		notifier:   notifier,
		waCfg:      waCfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// apiVersion returns the WhatsApp Cloud API version to use for tenant
// connections - reuses the platform-level WHATSAPP_API_VERSION config value
// since it is not tenant-specific (Meta's Graph API version is a property of
// the calling app, not the phone number).
func (u *omnichannelUseCase) apiVersion() string {
	if u.waCfg.APIVersion != "" {
		return u.waCfg.APIVersion
	}
	return "v20.0"
}

// verifyWhatsAppCredentials calls the WhatsApp Cloud API's phone-number
// lookup endpoint (GET /{api_version}/{phone_number_id}?access_token=...) to
// confirm the phone_number_id + access token pair is valid before ever
// marking a connection "connected" - mirrors
// marketplace/infrastructure/blibli_client.go's TestConnection pattern
// (live call, any well-formed success response proves the credentials work,
// 401/403 or an error envelope means they don't).
//
// CONFIDENCE NOTE: this endpoint and response shape are well-documented and
// stable in Meta's WhatsApp Cloud API reference
// (https://developers.facebook.com/docs/graph-api/reference/whatsapp-business-account-to-number-current-status/)
// but have NOT been exercised against a live Meta Business/App - there were
// no WhatsApp Business credentials available in this environment to test
// with. Treat the exact JSON field names below (`display_phone_number`,
// `verified_name`) as best-effort; they are not consulted for pass/fail,
// only status-code/error-envelope handling is.
func (u *omnichannelUseCase) verifyWhatsAppCredentials(ctx context.Context, phoneNumberID, accessToken string) error {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s?access_token=%s", u.apiVersion(), phoneNumberID, accessToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to build verification request: %w", err)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call WhatsApp Cloud API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read WhatsApp Cloud API response: %w", err)
	}

	if resp.StatusCode >= 300 {
		var errEnvelope struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errEnvelope) == nil && errEnvelope.Error.Message != "" {
			return fmt.Errorf("WhatsApp rejected the credentials: %s", errEnvelope.Error.Message)
		}
		return fmt.Errorf("WhatsApp Cloud API returned status %d", resp.StatusCode)
	}
	return nil
}

// ConnectWhatsApp verifies a tenant's WhatsApp Cloud API credentials with a
// live call, then stores the ChannelConnection in the tenant DB and upserts
// the control-plane WhatsAppNumberIndex row so inbound webhooks for this
// phone_number_id route back to this tenant - see
// internal/modules/whatsappregistry and delivery/http/handler.go's
// WhatsAppWebhook.
func (u *omnichannelUseCase) ConnectWhatsApp(ctx context.Context, companyID uuid.UUID, phoneNumberID, accessToken, businessAccountID, scope string) (*ChannelConnectionResponseDTO, error) {
	if phoneNumberID == "" || accessToken == "" {
		return nil, apperrors.NewBadRequest("Phone Number ID and Access Token are required")
	}
	if scope == "" {
		scope = domain.ConnectionScopeTenant
	}
	if scope != domain.ConnectionScopeTenant && scope != domain.ConnectionScopeCompany {
		return nil, apperrors.NewBadRequest("scope must be \"tenant\" or \"company\"")
	}

	if err := u.verifyWhatsAppCredentials(ctx, phoneNumberID, accessToken); err != nil {
		return nil, apperrors.NewBadRequest("Could not verify WhatsApp credentials: " + err.Error())
	}

	now := time.Now()
	conn := &domain.ChannelConnection{
		Channel:           domain.ChannelWhatsApp,
		Scope:             scope,
		PhoneNumberID:     phoneNumberID,
		AccessToken:       accessToken,
		BusinessAccountID: businessAccountID,
		DisplayName:       phoneNumberID,
		Status:            "connected",
		ConnectedAt:       now,
	}
	if err := u.connRepo.UpsertConnection(ctx, conn); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to store WhatsApp connection")
	}

	if u.registry != nil {
		if err := u.registry.Upsert(ctx, phoneNumberID, companyID, conn.TenantID); err != nil {
			return nil, apperrors.NewInternal(err, "WhatsApp connected but failed to register it for inbound routing")
		}
	}

	resp := ToChannelConnectionResponse(conn)
	return &resp, nil
}

func (u *omnichannelUseCase) ListChannelConnections(ctx context.Context) ([]ChannelConnectionResponseDTO, error) {
	conns, err := u.connRepo.ListConnections(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list channel connections")
	}
	return ToChannelConnectionResponseList(conns), nil
}

// DisconnectChannel removes the tenant's connection and, for WhatsApp,
// deletes the corresponding control-plane WhatsAppNumberIndex row so inbound
// webhooks for that phone_number_id stop routing here (falling back to
// WHATSAPP_DEFAULT_COMPANY_ID, if configured, or being dropped otherwise -
// same as before any tenant ever connected that number).
func (u *omnichannelUseCase) DisconnectChannel(ctx context.Context, channel domain.Channel) error {
	if !channel.Valid() {
		return apperrors.NewBadRequest("Unsupported channel")
	}

	conn, err := u.connRepo.GetActiveConnection(ctx, channel)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load channel connection")
	}
	if conn == nil {
		return apperrors.NewNotFound("No active connection for this channel")
	}

	if channel == domain.ChannelWhatsApp && u.registry != nil && conn.PhoneNumberID != "" {
		_ = u.registry.Delete(ctx, conn.PhoneNumberID)
	}

	if err := u.connRepo.DeleteConnection(ctx, conn.ID); err != nil {
		return apperrors.NewInternal(err, "Failed to disconnect channel")
	}
	return nil
}

func toConversationDTO(c domain.Conversation) ConversationDTO {
	return ConversationDTO{
		ID:                  c.ID,
		Channel:             string(c.Channel),
		ChannelLabel:        c.Channel.Label(),
		ExternalContactID:   c.ExternalContactID,
		ExternalContactName: c.ExternalContactName,
		LastMessagePreview:  c.LastMessagePreview,
		LastMessageAt:       c.LastMessageAt,
		UnreadCount:         c.UnreadCount,
		Status:              c.Status,
	}
}

func toMessageDTO(m domain.Message) MessageDTO {
	return MessageDTO{
		ID:        m.ID,
		Direction: m.Direction,
		Content:   m.Content,
		SentAt:    m.SentAt,
		SentBy:    m.SentBy,
	}
}

func (u *omnichannelUseCase) ListConversations(ctx context.Context, channel *domain.Channel, query types.PaginationQuery) ([]ConversationDTO, int64, error) {
	query.SetDefaults()
	conversations, total, err := u.repo.ListConversations(ctx, channel, query)
	if err != nil {
		return nil, 0, apperrors.NewInternal(err, "Failed to list conversations")
	}
	dtos := make([]ConversationDTO, 0, len(conversations))
	for _, c := range conversations {
		dtos = append(dtos, toConversationDTO(c))
	}
	return dtos, total, nil
}

func (u *omnichannelUseCase) GetConversationThread(ctx context.Context, conversationID uuid.UUID) (*ThreadDTO, error) {
	conv, err := u.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load conversation")
	}
	if conv == nil {
		return nil, apperrors.NewNotFound("Conversation not found")
	}
	messages, err := u.repo.ListMessagesByConversation(ctx, conversationID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load messages")
	}
	if conv.UnreadCount != 0 {
		if err := u.repo.MarkConversationRead(ctx, conv.ID); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to mark conversation read")
		}
		conv.UnreadCount = 0
	}
	dtoMessages := make([]MessageDTO, 0, len(messages))
	for _, m := range messages {
		dtoMessages = append(dtoMessages, toMessageDTO(m))
	}
	return &ThreadDTO{Conversation: toConversationDTO(*conv), Messages: dtoMessages}, nil
}

func (u *omnichannelUseCase) SendMessage(ctx context.Context, conversationID uuid.UUID, content string) (*MessageDTO, error) {
	if content == "" {
		return nil, apperrors.NewBadRequest("Message content is required")
	}
	conv, err := u.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load conversation")
	}
	if conv == nil {
		return nil, apperrors.NewNotFound("Conversation not found")
	}

	switch conv.Channel {
	case domain.ChannelWhatsApp:
		// Prefer this tenant's own connected WhatsApp number over the
		// platform-level WhatsAppConfig fallback - see ConnectWhatsApp and
		// the Notifier.SendWhatsAppWithCredentials doc.
		tenantConn, err := u.connRepo.GetActiveConnection(ctx, domain.ChannelWhatsApp)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to load WhatsApp connection")
		}
		switch {
		case tenantConn != nil && tenantConn.Status == "connected" && tenantConn.PhoneNumberID != "" && tenantConn.AccessToken != "":
			if err := u.notifier.SendWhatsAppWithCredentials(ctx, tenantConn.PhoneNumberID, tenantConn.AccessToken, u.apiVersion(), conv.ExternalContactID, content); err != nil {
				return nil, apperrors.NewServiceUnavailable("Failed to send WhatsApp message: " + err.Error())
			}
		case u.notifier.WhatsAppEnabled():
			if err := u.notifier.SendWhatsApp(ctx, conv.ExternalContactID, content); err != nil {
				return nil, apperrors.NewServiceUnavailable("Failed to send WhatsApp message: " + err.Error())
			}
		default:
			return nil, apperrors.NewServiceUnavailable("WhatsApp is not connected. Configure it in Settings > Integration.")
		}
	default:
		return nil, apperrors.NewServiceUnavailable(conv.Channel.Label() + " is not connected yet. Configure it in Settings > Integration.")
	}

	now := time.Now()
	msg := &domain.Message{
		ConversationID: conv.ID,
		Direction:      domain.DirectionOutbound,
		Content:        content,
		SentAt:         now,
	}
	if err := u.repo.CreateMessage(ctx, msg); err != nil {
		return nil, apperrors.NewInternal(err, "Message sent but failed to record it")
	}

	conv.LastMessagePreview = content
	conv.LastMessageAt = &now
	if err := u.repo.UpdateConversation(ctx, conv); err != nil {
		return nil, apperrors.NewInternal(err, "Message sent but failed to update conversation")
	}

	dto := toMessageDTO(*msg)
	return &dto, nil
}

// HandleWhatsAppWebhook processes an inbound WhatsApp Cloud API webhook
// notification: find-or-creates the Conversation for each sender, dedupes on
// the platform's own message id, and bumps unread/last-message bookkeeping.
// The caller (delivery/http) is responsible for resolving which tenant
// database `ctx` is scoped to before calling this - see the handler and
// module.go for the current single-default-company limitation.
func (u *omnichannelUseCase) HandleWhatsAppWebhook(ctx context.Context, payload WhatsAppWebhookPayload) error {
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			value := change.Value
			contactNames := map[string]string{}
			for _, c := range value.Contacts {
				contactNames[c.WaID] = c.Profile.Name
			}
			for _, m := range value.Messages {
				if m.From == "" {
					continue
				}
				// Dedupe: webhook deliveries can be retried by Meta.
				if m.ID != "" {
					existing, err := u.repo.GetMessageByExternalID(ctx, m.ID)
					if err != nil {
						return apperrors.NewInternal(err, "Failed to check for duplicate message")
					}
					if existing != nil {
						continue
					}
				}

				conv, err := u.repo.GetConversationByChannelAndContact(ctx, domain.ChannelWhatsApp, m.From)
				if err != nil {
					return apperrors.NewInternal(err, "Failed to load conversation")
				}
				contactName := contactNames[m.From]
				if conv == nil {
					conv = &domain.Conversation{
						Channel:             domain.ChannelWhatsApp,
						ExternalContactID:   m.From,
						ExternalContactName: contactName,
						Status:              domain.ConversationStatusOpen,
					}
					if err := u.repo.CreateConversation(ctx, conv); err != nil {
						return apperrors.NewInternal(err, "Failed to create conversation")
					}
				} else if contactName != "" && conv.ExternalContactName != contactName {
					conv.ExternalContactName = contactName
				}

				sentAt := time.Now()
				if ts, err := strconv.ParseInt(m.Timestamp, 10, 64); err == nil && ts > 0 {
					sentAt = time.Unix(ts, 0)
				}

				msg := &domain.Message{
					ConversationID:    conv.ID,
					Direction:         domain.DirectionInbound,
					Content:           m.Text.Body,
					ExternalMessageID: m.ID,
					SentAt:            sentAt,
				}
				if err := u.repo.CreateMessage(ctx, msg); err != nil {
					return apperrors.NewInternal(err, "Failed to record inbound message")
				}

				conv.LastMessagePreview = m.Text.Body
				conv.LastMessageAt = &sentAt
				conv.UnreadCount++
				if err := u.repo.UpdateConversation(ctx, conv); err != nil {
					return apperrors.NewInternal(err, "Failed to update conversation")
				}
			}
		}
	}
	return nil
}
