package sales

import (
	"context"
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	approvalApp "github.com/divinecoid/one-backend/internal/modules/approval/application"
	approvalInfra "github.com/divinecoid/one-backend/internal/modules/approval/infrastructure"
	currencyApp "github.com/divinecoid/one-backend/internal/modules/currency/application"
	currencyInfra "github.com/divinecoid/one-backend/internal/modules/currency/infrastructure"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	inventoryInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/sales/application"
	"github.com/divinecoid/one-backend/internal/modules/sales/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/sales/domain"
	"github.com/divinecoid/one-backend/internal/modules/sales/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "sales",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.SalesOrder{}, &domain.SalesOrderLine{}, &domain.Quotation{}, &domain.QuotationLine{}, &domain.Invoice{}, &domain.Delivery{}, &domain.DeliveryLine{},
				&domain.SalesDownPayment{}, &domain.BillingTerm{}, &domain.SalesReturn{}, &domain.SalesReturnLine{},
			)
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewSalesRepository(tenantDB)
			currencyRepo := currencyInfra.NewCurrencyRepository(tenantDB)
			inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
			productRepo := productInfra.NewProductRepository(tenantDB)
			approvalRepo := approvalInfra.NewApprovalRepository(tenantDB)
			uc := application.NewSalesUseCase(repo, currencyApp.NewCurrencyUseCase(currencyRepo), inventoryApp.NewInventoryUseCase(inventoryRepo, productRepo), approvalApp.NewApprovalUseCase(approvalRepo))
			return uc.SeedInitialData(context.Background())
		},
	})

	// Register a callback so that approving/rejecting a Sales Order from the
	// generic Approval Center ("/approval/requests/:id/approve") updates the
	// order's status and deducts stock exactly the way this module's own
	// "/sales/orders/:id/approve" endpoint would - see
	// application/callback.go. The factory is registered once at startup;
	// it is invoked per-request with that request's tenant DB.
	approvalApp.RegisterCallbackFactory(application.SalesOrderDocumentType, func(tenantDB *gorm.DB) approvalApp.DocumentStatusCallback {
		repo := infrastructure.NewSalesRepository(tenantDB)
		currencyRepo := currencyInfra.NewCurrencyRepository(tenantDB)
		inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
		productRepo := productInfra.NewProductRepository(tenantDB)
		approvalRepo := approvalInfra.NewApprovalRepository(tenantDB)
		uc := application.NewSalesUseCase(repo, currencyApp.NewCurrencyUseCase(currencyRepo), inventoryApp.NewInventoryUseCase(inventoryRepo, productRepo), approvalApp.NewApprovalUseCase(approvalRepo))
		return application.NewSalesOrderApprovalCallback(uc)
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("sales module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
