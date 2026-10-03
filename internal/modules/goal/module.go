package goal

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/goal/application"
	"github.com/divinecoid/one-backend/internal/modules/goal/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/goal/domain"
	"github.com/divinecoid/one-backend/internal/modules/goal/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: it has no startup-bound repository or use-case,
// since the goal schema lives in each company's own tenant database. Its
// migration+seed is registered once here (run at tenant-provision time by
// foundation/tenant.ProvisionAll), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database (see
// delivery/http.Handler.resolve).
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "goal",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Goal{}, &domain.GoalCheckIn{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewGoalsRepository(tenantDB)
			uc := application.NewGoalsUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("goal module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
