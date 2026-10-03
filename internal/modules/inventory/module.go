package inventory

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	"github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the inventory schema lives in each company's own
// tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database (see
// delivery/http.Handler.resolve for how the Product dependency is handled).
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "inventory",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.Warehouse{},
				&domain.StockLevel{},
				&domain.StockMovement{},
				&domain.StockBatch{},
				&domain.StockTransfer{},
				&domain.StockOpname{},
				&domain.StockOpnameLine{},
			)
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewInventoryRepository(tenantDB)
			productRepo := productInfra.NewProductRepository(tenantDB)
			uc := application.NewInventoryUseCase(repo, productRepo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("inventory module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
