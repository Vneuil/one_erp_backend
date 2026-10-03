package activitylog

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"

	"github.com/divinecoid/one-backend/internal/modules/activitylog/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/activitylog/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct {
	Handler *http.Handler
}

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "activitylog",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.ActivityLog{})
		},
	})

	handler := http.NewHandler()
	handler.RegisterRoutes(router, jwtSecret, manager)

	slog.Info("activity log module initialized (tenant-scoped)")

	return &Module{
		Handler: handler,
	}
}
