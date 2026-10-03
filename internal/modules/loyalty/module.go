package loyalty

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/loyalty/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/loyalty/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "loyalty",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.LoyaltyConfig{},
				&domain.LoyaltyMember{},
				&domain.LoyaltyTransaction{},
			)
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("Loyalty module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
