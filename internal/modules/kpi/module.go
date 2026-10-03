package kpi

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/kpi/application"
	"github.com/divinecoid/one-backend/internal/modules/kpi/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/kpi/domain"
	"github.com/divinecoid/one-backend/internal/modules/kpi/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "kpi",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.KpiReview{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewKpiRepository(tenantDB)
			uc := application.NewKpiUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("KPI module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
