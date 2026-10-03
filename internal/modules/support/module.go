package support

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/support/application"
	"github.com/divinecoid/one-backend/internal/modules/support/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/support/domain"
	"github.com/divinecoid/one-backend/internal/modules/support/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the support schema lives in each company's own
// tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database.
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "support",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Ticket{}, &domain.TicketReply{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewTicketsRepository(tenantDB)
			uc := application.NewTicketsUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("support module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
