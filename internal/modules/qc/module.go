package qc

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/qc/application"
	"github.com/divinecoid/one-backend/internal/modules/qc/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/divinecoid/one-backend/internal/modules/qc/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "qc",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.QCParameter{},
				&domain.QCInspection{},
				&domain.QCInspectionItem{},
				&domain.PersonalHygieneCheck{},
				&domain.PreOpSanitationCheck{},
			)
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewQCRepository(tenantDB)
			uc := application.NewQCUseCase(repo)
			return uc.SeedDefaultParameters(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("quality control (qc) module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
