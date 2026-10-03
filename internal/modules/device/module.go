package device

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/device/application"
	"github.com/divinecoid/one-backend/internal/modules/device/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/device/domain"
	"github.com/divinecoid/one-backend/internal/modules/device/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the device schema lives in each company's own
// tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database.
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "device",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Device{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewDeviceRepository(tenantDB)
			uc := application.NewDeviceUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("device module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
