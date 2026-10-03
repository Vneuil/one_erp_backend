package lms

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/lms/application"
	"github.com/divinecoid/one-backend/internal/modules/lms/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/lms/domain"
	"github.com/divinecoid/one-backend/internal/modules/lms/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the lms schema lives in each company's own
// tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database.
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "lms",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.Course{},
				&domain.Enrollment{},
				&domain.Quiz{},
				&domain.QuizAttempt{},
				&domain.Certificate{},
			)
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewLMSRepository(tenantDB)
			hrmRepo := hrminfra.NewHRMRepository(tenantDB)
			uc := application.NewLMSUseCase(repo, hrmRepo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("lms module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
