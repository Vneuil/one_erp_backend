package cooperative

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/cooperative/application"
	"github.com/divinecoid/one-backend/internal/modules/cooperative/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/cooperative/domain"
	"github.com/divinecoid/one-backend/internal/modules/cooperative/infrastructure"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "cooperative",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.CooperativeLoan{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewCooperativeRepository(tenantDB)
			hrmRepo := hrminfra.NewHRMRepository(tenantDB)
			uc := application.NewCooperativeUseCase(repo, hrmRepo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("Cooperative module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
