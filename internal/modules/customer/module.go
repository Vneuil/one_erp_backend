package customer

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/customer/application"
	"github.com/divinecoid/one-backend/internal/modules/customer/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/customer/domain"
	"github.com/divinecoid/one-backend/internal/modules/customer/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "customer",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Customer{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewCustomerRepository(tenantDB)
			uc := application.NewCustomerUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("customer module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
