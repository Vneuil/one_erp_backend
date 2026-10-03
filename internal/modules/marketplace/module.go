package marketplace

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/marketplace/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/marketplace/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

// NewModule wires the TikTok Shop / Shopee marketplace integration. It is
// tenant-scoped like the sales module: each company's OAuth connection and
// sync bookkeeping live in that company's own database, never the
// control-plane DB.
func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager, frontendBaseURL string, tiktokCfg config.TikTokShopConfig, shopeeCfg config.ShopeeConfig, blibliCfg config.BlibliConfig, lazadaCfg config.LazadaConfig) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "marketplace",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.MarketplaceConnection{}, &domain.MarketplaceOrderSync{})
		},
	})

	handler := http.NewHandler(tiktokCfg, shopeeCfg, blibliCfg, lazadaCfg, frontendBaseURL, jwtSecret, manager)
	handler.RegisterRoutes(router, jwtSecret, manager)

	if !tiktokCfg.Enabled() {
		slog.Warn("marketplace module: TikTok Shop integration not configured (TIKTOK_SHOP_APP_KEY/TIKTOK_SHOP_APP_SECRET missing)")
	}
	if !shopeeCfg.Enabled() {
		slog.Warn("marketplace module: Shopee integration not configured (SHOPEE_PARTNER_ID/SHOPEE_PARTNER_KEY missing)")
	}
	if !blibliCfg.Enabled() {
		slog.Warn("marketplace module: Blibli integration not configured (BLIBLI_API_CLIENT_ID/BLIBLI_API_CLIENT_SECRET missing)")
	}
	if !lazadaCfg.Enabled() {
		slog.Warn("marketplace module: Lazada integration not configured (LAZADA_APP_KEY/LAZADA_APP_SECRET missing)")
	}
	slog.Info("marketplace module initialized (tenant-scoped)")

	return &Module{Handler: handler}
}
