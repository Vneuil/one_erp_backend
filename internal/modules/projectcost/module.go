package projectcost

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/projectcost/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/projectcost/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct{ Handler *http.Handler }

// NewModule is tenant-scoped: its tables live in each company's own database.
func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "projectcost",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.BudgetItem{}, &domain.CostEntry{}, &domain.WorkOrder{})
		},
	})
	h := http.NewHandler()
	h.RegisterRoutes(router, jwtSecret, manager)
	slog.Info("projectcost module initialized (tenant-scoped)")
	return &Module{Handler: h}
}
