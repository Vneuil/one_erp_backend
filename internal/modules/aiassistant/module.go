package aiassistant

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name:    "aiassistant",
		Migrate: func(tenantDB *gorm.DB) error { return tenantDB.AutoMigrate(&Settings{}) },
	})

	h := &Handler{}
	g := router.Group("/ai", middleware.Protected(jwtSecret), middleware.TenantContext(manager))
	g.Get("/settings", h.Get)
	g.Put("/settings", middleware.RequireRoles("admin"), h.Save)
	g.Post("/chat", h.Chat)

	slog.Info("aiassistant module initialized (tenant-scoped)")
}
