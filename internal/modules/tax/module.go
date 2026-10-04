package tax

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/tax/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

// NewModule registers the tenant-scoped tax schema (Faktur Pajak, serial
// ranges, settings) and its routes.
func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "tax",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.TaxSettings{},
				&domain.SerialRange{},
				&domain.TaxInvoice{},
				&domain.TaxInvoiceLine{},
			)
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("tax module initialized (tenant-scoped)")
	return &Module{Handler: handler}
}
