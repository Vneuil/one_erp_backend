package projecttask

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/projecttask/application"
	"github.com/divinecoid/one-backend/internal/modules/projecttask/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/projecttask/domain"
	"github.com/divinecoid/one-backend/internal/modules/projecttask/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the projecttask schema lives in each company's
// own tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database.
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "projecttask",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.ProjectTask{}, &domain.TaskChecklistItem{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewProjectTaskRepository(tenantDB)
			uc := application.NewProjectTaskUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("projecttask module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
