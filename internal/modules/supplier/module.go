package supplier

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/supplier/application"
	"github.com/divinecoid/one-backend/internal/modules/supplier/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/supplier/domain"
	"github.com/divinecoid/one-backend/internal/modules/supplier/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "supplier",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Supplier{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewSupplierRepository(tenantDB)
			uc := application.NewSupplierUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("supplier module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
