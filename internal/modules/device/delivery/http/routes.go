package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	devices := router.Group("/devices", protected, tenantCtx)
	devices.Post("/", h.CreateDevice)
	devices.Get("/", h.ListDevices)
	devices.Get("/:id", h.GetDeviceByID)
	devices.Put("/:id", h.UpdateDevice)
	devices.Post("/:id/ping", h.PingDevice)
}
