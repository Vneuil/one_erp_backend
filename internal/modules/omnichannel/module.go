package omnichannel

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/divinecoid/one-backend/internal/foundation/notify"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/omnichannel/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/omnichannel/domain"
	"github.com/divinecoid/one-backend/internal/modules/whatsappregistry"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

// NewModule wires the Omnichannel Inbox module. Conversations, messages and
// each tenant's own ChannelConnection are tenant-scoped like
// sales/marketplace: each company's data lives in that company's own
// database, never the control-plane DB. controlPlaneDB is the one exception
// - it backs the WhatsAppNumberIndex registry (see
// internal/modules/whatsappregistry) used for inbound webhook routing,
// which by necessity lives outside any tenant database.
func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager, notifier notify.Notifier, waCfg config.WhatsAppConfig, controlPlaneDB *gorm.DB) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "omnichannel",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Conversation{}, &domain.Message{}, &domain.ChannelConnection{})
		},
	})

	// Control-plane WhatsAppNumberIndex table - migrated directly here (NOT
	// via tenantMgr.RegisterSchema, which only ever runs against per-tenant
	// databases) since it lives in the control-plane DB itself. See
	// whatsappregistry package doc for why.
	if err := controlPlaneDB.AutoMigrate(&whatsappregistry.WhatsAppNumberIndex{}); err != nil {
		slog.Error("omnichannel module: failed to migrate control-plane whatsapp_number_index table", "error", err)
	}
	registry := whatsappregistry.NewRepository(controlPlaneDB)

	handler := http.NewHandler(waCfg, notifier, manager, registry)
	handler.RegisterRoutes(router, jwtSecret, manager)

	if !waCfg.Enabled() {
		slog.Warn("omnichannel module: WhatsApp not configured (WHATSAPP_PHONE_NUMBER_ID/WHATSAPP_ACCESS_TOKEN missing) - sending will return 503")
	}
	if waCfg.WebhookVerifyToken == "" {
		slog.Warn("omnichannel module: WHATSAPP_WEBHOOK_VERIFY_TOKEN not set - webhook verification handshake will always fail")
	}
	if waCfg.DefaultCompanyID() == nil {
		slog.Warn("omnichannel module: WHATSAPP_DEFAULT_COMPANY_ID not set - inbound WhatsApp webhook messages will be dropped")
	}
	slog.Info("omnichannel module initialized (tenant-scoped)")

	return &Module{Handler: handler}
}
