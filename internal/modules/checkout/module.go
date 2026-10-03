package checkout

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/config"
	"github.com/divinecoid/one-backend/internal/foundation/payment"
	"github.com/divinecoid/one-backend/internal/modules/checkout/application"
	"github.com/divinecoid/one-backend/internal/modules/checkout/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/checkout/domain"
	"github.com/divinecoid/one-backend/internal/modules/checkout/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	UseCase application.CheckoutUseCase
	Handler *http.Handler
}

// NewModule wires the public pricing-page checkout flow. Like
// company/user/tenant, CheckoutOrder lives in the central database (db) -
// a prospective customer has no tenant to scope it to yet.
func NewModule(db *gorm.DB, router fiber.Router, xenditCfg config.XenditConfig) *Module {
	if err := db.AutoMigrate(&domain.CheckoutOrder{}); err != nil {
		slog.Error("failed to migrate checkout schema", "error", err)
	}

	repo := infrastructure.NewCheckoutRepository(db)
	client := payment.NewClient(xenditCfg)
	if !client.Enabled() {
		slog.Warn("checkout module initialized with Xendit disabled (XENDIT_SECRET_KEY not set) - checkout will return an error until configured")
	}

	uc := application.NewCheckoutUseCase(repo, client)
	handler := http.NewHandler(uc, xenditCfg.WebhookToken)
	handler.RegisterRoutes(router)

	slog.Info("checkout module initialized (public, central db)")

	return &Module{UseCase: uc, Handler: handler}
}
