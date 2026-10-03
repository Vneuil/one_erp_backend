package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	warehouse := router.Group("/warehouse", protected, tenantCtx)

	pickWaves := warehouse.Group("/pick-waves")
	pickWaves.Post("/", h.CreatePickWave)
	pickWaves.Get("/", h.ListPickWaves)
	pickWaves.Get("/:id", h.GetPickWaveByID)
	pickWaves.Post("/:id/pick", h.RecordPick)
	pickWaves.Post("/:id/complete", h.CompletePickWave)
	pickWaves.Post("/:id/cancel", h.CancelPickWave)

	packingSessions := warehouse.Group("/packing-sessions")
	packingSessions.Post("/", h.CreatePackingSession)
	packingSessions.Get("/", h.ListPackingSessions)
	packingSessions.Post("/:id/complete", h.CompletePackingSession)

	shipments := warehouse.Group("/shipments")
	shipments.Post("/", h.CreateShipment)
	shipments.Get("/", h.ListShipments)
	shipments.Post("/:id/dispatch", h.DispatchShipment)
}
