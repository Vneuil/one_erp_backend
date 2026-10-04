package hrletters

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/hrletters/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/hrletters/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

// NewModule registers the tenant-scoped schema of HR letters (contracts,
// warnings, termination, mutation, memos, overtime orders) and its routes.
func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "hrletters",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Letter{}, &domain.Recipient{})
		},
	})
	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)
	slog.Info("hrletters module initialized (tenant-scoped)")
	return &Module{Handler: handler}
}
