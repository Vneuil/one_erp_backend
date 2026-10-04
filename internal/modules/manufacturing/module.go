package manufacturing

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	inventoryInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/application"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/domain"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/infrastructure"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the manufacturing schema lives in each company's
// own tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database (see
// delivery/http.Handler.resolve for how the Inventory/Product dependencies
// are handled).
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "manufacturing",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.BillOfMaterial{},
				&domain.BOMLine{},
				&domain.BOMProcess{},
				&domain.ProductionOrder{},
				&domain.ProductionBatch{},
				&domain.ProductionStep{},
				&domain.ProductionStepLog{},
			)
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewManufacturingRepository(tenantDB)
			inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
			productRepo := productInfra.NewProductRepository(tenantDB)
			uc := application.NewManufacturingUseCase(repo, inventoryRepo, productRepo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("manufacturing module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
