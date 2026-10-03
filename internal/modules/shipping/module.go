package shipping

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/shipping/application"
	"github.com/divinecoid/one-backend/internal/modules/shipping/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/shipping/domain"
	"github.com/divinecoid/one-backend/internal/modules/shipping/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "shippingMethod",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.ShippingMethod{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewShippingMethodRepository(tenantDB)
			uc := application.NewShippingMethodUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("shippingMethod module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
