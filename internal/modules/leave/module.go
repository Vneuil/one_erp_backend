package leave

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/leave/application"
	"github.com/divinecoid/one-backend/internal/modules/leave/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/leave/domain"
	"github.com/divinecoid/one-backend/internal/modules/leave/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "leave",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.LeaveRequest{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewLeaveRepository(tenantDB)
			hrmRepo := hrminfra.NewHRMRepository(tenantDB)
			uc := application.NewLeaveUseCase(repo, hrmRepo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("Leave module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
