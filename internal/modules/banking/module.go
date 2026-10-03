package banking

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/banking/application"
	"github.com/divinecoid/one-backend/internal/modules/banking/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/banking/domain"
	"github.com/divinecoid/one-backend/internal/modules/banking/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "banking",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.BankAccount{},
				&domain.BankStatementLine{},
			)
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewBankingRepository(tenantDB)
			uc := application.NewBankingUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("banking module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
