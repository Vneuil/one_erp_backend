package project

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/project/application"
	"github.com/divinecoid/one-backend/internal/modules/project/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/project/domain"
	"github.com/divinecoid/one-backend/internal/modules/project/infrastructure"
	taskinfra "github.com/divinecoid/one-backend/internal/modules/projecttask/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "project",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Project{}, &domain.TimeEntry{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewProjectRepository(tenantDB)
			timeEntryRepo := infrastructure.NewTimeEntryRepository(tenantDB)
			taskRepo := taskinfra.NewProjectTaskRepository(tenantDB)
			uc := application.NewProjectUseCase(repo, timeEntryRepo, taskRepo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("project module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
