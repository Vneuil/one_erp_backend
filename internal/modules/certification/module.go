package certification

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/storage"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/certification/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/certification/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Module struct{ Handler *http.Handler }

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager, store ...storage.Storage) *Module {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "certification",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Certificate{})
		},
	})
	var s storage.Storage
	if len(store) > 0 {
		s = store[0]
	}
	h := http.NewHandler(s)
	h.RegisterRoutes(router, jwtSecret, manager)
	slog.Info("certification module initialized (tenant-scoped)")
	return &Module{Handler: h}
}
