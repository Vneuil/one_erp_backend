package approval

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/approval/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/approval/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "approval",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(
				&domain.ApprovalWorkflow{},
				&domain.ApprovalWorkflowLevel{},
				&domain.ApprovalRequest{},
				&domain.ApprovalStep{},
			)
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("approval module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
