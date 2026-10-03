package reimbursement

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/reimbursement/application"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	"github.com/divinecoid/one-backend/internal/modules/reimbursement/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "reimbursement",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.ReimbursementClaim{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewReimbursementRepository(tenantDB)
			uc := application.NewReimbursementUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("Reimbursement module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
