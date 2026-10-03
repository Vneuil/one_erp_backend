package assets

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/assets/application"
	"github.com/divinecoid/one-backend/internal/modules/assets/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/assets/domain"
	"github.com/divinecoid/one-backend/internal/modules/assets/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the assets schema lives in each company's own
// tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database.
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "assets",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.FixedAsset{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewAssetsRepository(tenantDB)
			uc := application.NewAssetsUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("assets module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
