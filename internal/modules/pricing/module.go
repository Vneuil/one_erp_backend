package pricing

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/pricing/application"
	"github.com/divinecoid/one-backend/internal/modules/pricing/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/pricing/domain"
	"github.com/divinecoid/one-backend/internal/modules/pricing/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name:     "pricing",
		SeedKind: tenantMgr.SeedReference,
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.PaymentTerm{}, &domain.CustomerType{}, &domain.TaxRate{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewPricingRepository(tenantDB)
			uc := application.NewPricingUseCase(repo)
			return uc.SeedReferenceData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("pricing module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
