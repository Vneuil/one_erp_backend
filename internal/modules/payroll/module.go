package payroll

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	coopinfra "github.com/divinecoid/one-backend/internal/modules/cooperative/infrastructure"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/payroll/application"
	"github.com/divinecoid/one-backend/internal/modules/payroll/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	"github.com/divinecoid/one-backend/internal/modules/payroll/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "payroll",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.PayrollEntry{}, &domain.PayrollPolicy{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewPayrollRepository(tenantDB)
			hrmRepo := hrminfra.NewHRMRepository(tenantDB)
			coopRepo := coopinfra.NewCooperativeRepository(tenantDB)
			uc := application.NewPayrollUseCase(repo, hrmRepo, coopRepo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("Payroll module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
