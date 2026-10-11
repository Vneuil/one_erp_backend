package fingerprint

import (
	"log/slog"

	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/modules/fingerprint/delivery/http"
	"github.com/divinecoid/one-backend/internal/modules/fingerprint/domain"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// NewModule wires fingerprint-machine attendance: management endpoints on v1
// and unauthenticated machine endpoints on the raw app. db is the
// control-plane database holding the machine registry.
func NewModule(v1 fiber.Router, app fiber.Router, jwtSecret string, manager *tenantMgr.Manager, db *gorm.DB) {
	if err := db.AutoMigrate(&domain.Registry{}); err != nil {
		slog.Error("fingerprint: failed to migrate the machine registry", "error", err)
	}
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name: "fingerprint",
		Migrate: func(tenantDB *gorm.DB) error {
			return tenantDB.AutoMigrate(&domain.Device{}, &domain.PinMap{}, &domain.Punch{})
		},
	})
	http.NewHandler(db, manager).RegisterRoutes(v1, app, jwtSecret, manager)
	slog.Info("fingerprint module initialized")
}
