package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes mounts the logged-in management API on the versioned router
// and the machine-facing endpoints on the raw app (machines send no JWT and
// ADMS firmware calls fixed, unversioned paths).
func (h *Handler) RegisterRoutes(v1 fiber.Router, app fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	fp := v1.Group("/fingerprint", protected, tenantCtx)
	fp.Get("/devices", h.ListDevices)
	fp.Post("/devices", middleware.RequireRoles("admin"), h.RegisterDevice)
	fp.Put("/devices/:serial", middleware.RequireRoles("admin"), h.UpdateDevice)
	fp.Post("/devices/:serial/rotate-token", middleware.RequireRoles("admin"), h.RotateToken)
	fp.Delete("/devices/:serial", middleware.RequireRoles("admin"), h.DeleteDevice)
	fp.Get("/pin-maps", h.ListPinMaps)
	fp.Post("/pin-maps", h.MapPIN)
	fp.Delete("/pin-maps/:id", h.DeletePinMap)
	fp.Get("/punches", h.ListPunches)
	fp.Post("/import", h.Import)
	fp.Post("/reprocess", h.Reprocess)

	app.Post("/fingerprint-push", h.GenericPush)
	iclock := app.Group("/iclock")
	iclock.All("/cdata", h.ADMSData)
	iclock.Get("/getrequest", h.ADMSGetRequest)
	iclock.All("/devicecmd", h.ADMSDeviceCmd)
}
