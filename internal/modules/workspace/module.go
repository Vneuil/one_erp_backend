package workspace

import (
	"log/slog"
	"time"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/workspace/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/workspace/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

// NewModule wires the Tenant (business-unit) feature: every Company always
// has at least one Tenant, auto-seeded here on provisioning, so companies
// that never create additional tenants are completely unaffected - their
// JWT simply carries a nil TenantID and tenant-aware repositories treat
// that as "no filter".
func NewModule(router fiber.Router, jwtSecret string, jwtExpiry time.Duration, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "workspace",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Tenant{})
		},
		Seed: func(tenantDB *gorm.DB) error {
			var count int64
			if err := tenantDB.Model(&domain.Tenant{}).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return nil
			}
			return tenantDB.Create(&domain.Tenant{
				Name: "Head Office", Code: "MAIN", IsActive: true, IsDefault: true,
			}).Error
		},
	})

	handler := http.NewHandler(jwtSecret, jwtExpiry)
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("workspace (tenant) module initialized (tenant-scoped)")

	return &Module{Handler: handler}
}
