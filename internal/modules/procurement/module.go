package procurement

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
	"github.com/divinecoid/one-backend/internal/modules/procurement/application"
	"github.com/divinecoid/one-backend/internal/modules/procurement/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/procurement/domain"
	"github.com/divinecoid/one-backend/internal/modules/procurement/infrastructure"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "procurement",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.PurchaseRequest{},
				&domain.PurchaseRequestLine{},
				&domain.PurchaseOrder{},
				&domain.PurchaseOrderLine{},
				&domain.GoodsReceipt{},
				&domain.GoodsReceiptLine{},
				&domain.PurchaseInvoice{},
				&domain.PurchaseDownPayment{},
				&domain.PurchaseReturn{},
				&domain.PurchaseReturnLine{},
				&domain.InvoiceReceipt{},
				&domain.InvoiceReceiptLine{},
			)
		},
		Seed: func(tenantDB *gorm.DB) error {
			repo := infrastructure.NewProcurementRepository(tenantDB)
			approvalRepo := approvalInfra.NewApprovalRepository(tenantDB)
			inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
			productRepo := productInfra.NewProductRepository(tenantDB)
			currencyRepo := currencyInfra.NewCurrencyRepository(tenantDB)
			uc := application.NewProcurementUseCase(repo, approvalApp.NewApprovalUseCase(approvalRepo), inventoryApp.NewInventoryUseCase(inventoryRepo, productRepo), currencyApp.NewCurrencyUseCase(currencyRepo))
			return uc.SeedInitialData(context.Background())
		},
	})

	// Register a callback so that approving/rejecting a Purchase Order from
	// the generic Approval Center ("/approval/requests/:id/approve") updates
	// the PO's status exactly the way this module's own
	// "/procurement/purchase-orders/:id/approve" endpoint would - see
	// application/callback.go. The factory is registered once at startup;
	// it is invoked per-request with that request's tenant DB.
	approvalApp.RegisterCallbackFactory(application.PurchaseOrderDocumentType, func(tenantDB *gorm.DB) approvalApp.DocumentStatusCallback {
		repo := infrastructure.NewProcurementRepository(tenantDB)
		approvalRepo := approvalInfra.NewApprovalRepository(tenantDB)
		inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
		productRepo := productInfra.NewProductRepository(tenantDB)
		currencyRepo := currencyInfra.NewCurrencyRepository(tenantDB)
		uc := application.NewProcurementUseCase(repo, approvalApp.NewApprovalUseCase(approvalRepo), inventoryApp.NewInventoryUseCase(inventoryRepo, productRepo), currencyApp.NewCurrencyUseCase(currencyRepo))
		return application.NewPurchaseOrderApprovalCallback(uc)
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("procurement module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
