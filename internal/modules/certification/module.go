package certification

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/certification/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/certification/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct{ Handler *http.Handler }

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "certification",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Certificate{})
		},
	})
	h := http.NewHandler()
	h.RegisterRoutes(router, jwtSecret, manager)
	slog.Info("certification module initialized (tenant-scoped)")
	return &Module{Handler: h}
}
