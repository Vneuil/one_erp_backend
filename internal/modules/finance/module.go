package finance

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/finance/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/finance/domain"
	"github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "finance",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.Account{},
				&domain.JournalEntry{},
				&domain.JournalLine{},
				&domain.Payable{},
				&domain.Receivable{},
				&domain.CapitalTransaction{},
				&domain.OtherIncome{},
				&domain.PettyCashFund{},
				&domain.PettyCashTransaction{},
				&domain.Budget{},
				&domain.CashVoucher{},
				&domain.CashVoucherLine{},
			)
		},
		SeedKind: tenantMgr.SeedReference, // the default chart of accounts is needed by every company
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewFinanceRepository(tenantDB)
			uc := application.NewFinanceUseCase(repo)
			return uc.SeedInitialData(context.Background())
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("finance module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
