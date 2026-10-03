package crm

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/crm/application"
	"github.com/divinecoid/one-backend/internal/modules/crm/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/modules/crm/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "crm",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Lead{}, &domain.Deal{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewCRMRepository(tenantDB)
			uc := application.NewCRMUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("CRM module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
