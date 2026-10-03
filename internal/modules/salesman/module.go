package salesman

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/salesman/application"
	"github.com/divinecoid/one-backend/internal/modules/salesman/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/salesman/domain"
	"github.com/divinecoid/one-backend/internal/modules/salesman/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "salesman",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Salesman{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewSalesmanRepository(tenantDB)
			uc := application.NewSalesmanUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("salesman module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
