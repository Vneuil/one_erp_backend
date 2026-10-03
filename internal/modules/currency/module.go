package currency

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/currency/application"
	"github.com/divinecoid/one-backend/internal/modules/currency/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/currency/domain"
	"github.com/divinecoid/one-backend/internal/modules/currency/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "currency",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Currency{}, &domain.ExchangeRate{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewCurrencyRepository(tenantDB)
			uc := application.NewCurrencyUseCase(repo)
			// IDR is the default base currency for every company - see
			// modules/company's own Currency field default. A company can
			// still add other active currencies and set daily rates; base
			// currency changes are a manual follow-up, not automated here.
			return uc.SeedBaseCurrency(context.Background(), "IDR", "Indonesian Rupiah", "Rp")
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("currency module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
