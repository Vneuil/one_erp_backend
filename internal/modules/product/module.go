package product

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/product/application"
	"github.com/divinecoid/one-backend/internal/modules/product/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/product/domain"
	"github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "product",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Product{}, &domain.ProductCategory{}, &domain.UnitOfMeasure{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewProductRepository(tenantDB)
			uc := application.NewProductUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("product module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
