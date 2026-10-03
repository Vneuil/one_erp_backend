package integration

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/integration/application"
	"github.com/divinecoid/one-backend/internal/modules/integration/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/integration/domain"
	"github.com/divinecoid/one-backend/internal/modules/integration/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the integration schema lives in each company's
// own tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database.
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "integration",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.APIKey{}, &domain.WebhookSubscription{}, &domain.WebhookDelivery{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			apiKeyRepo := infrastructure.NewAPIKeyRepository(tenantDB)
			webhookRepo := infrastructure.NewWebhookRepository(tenantDB)
			uc := application.NewIntegrationUseCase(apiKeyRepo, webhookRepo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("integration module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
