package contracts

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/contracts/application"
	"github.com/divinecoid/one-backend/internal/modules/contracts/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/contracts/domain"
	"github.com/divinecoid/one-backend/internal/modules/contracts/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Module is tenant-scoped: the contracts schema lives in each company's own
// tenant database. Its migration+seed is registered once here (run at
// tenant-provision time), and each request resolves a fresh
// repository/use-case pair against the caller's tenant database.
type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "contracts",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Contract{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewContractsRepository(tenantDB)
			uc := application.NewContractsUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("contracts module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
