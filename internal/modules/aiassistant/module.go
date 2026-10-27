package aiassistant

import (
	"log/slog"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func NewModule(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager, central *gorm.DB) {
	tenantMgr.RegisterSchema(tenantMgr.SchemaMigrator{
		Name:    "aiassistant",
		Migrate: func(tenantDB *gorm.DB) error { return tenantDB.AutoMigrate(&Settings{}, &MCPToken{}) },
	})

	h := &Handler{manager: manager, central: central}
	g := router.Group("/ai", middleware.Protected(jwtSecret), middleware.TenantContext(manager))
	g.Get("/settings", h.Get)
	g.Put("/settings", middleware.RequireRoles("admin"), h.Save)
	g.Post("/chat", h.Chat)
	admin := middleware.RequireRoles("admin")
	g.Get("/mcp-tokens", admin, h.ListTokens)
	g.Post("/mcp-tokens", admin, h.CreateToken)
	g.Delete("/mcp-tokens/:id", admin, h.RevokeToken)

	// MCP endpoint (Streamable HTTP). Authenticated by MCP token, not JWT.
	router.Post("/mcp", h.MCP)
	router.Get("/mcp", h.MCPMethodNotAllowed)

	slog.Info("aiassistant module initialized (tenant-scoped)")
}
