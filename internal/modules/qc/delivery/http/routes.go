package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	qc := router.Group("/qc", protected, tenantCtx)

	qc.Get("/stats", h.GetStats)

	params := qc.Group("/parameters")
	params.Get("/", h.ListParameters)
	params.Post("/", h.CreateParameter)
	params.Get("/:id", h.GetParameter)
	params.Put("/:id", h.UpdateParameter)
	params.Delete("/:id", h.DeleteParameter)

	inspections := qc.Group("/inspections")
	inspections.Get("/", h.ListInspections)
	inspections.Post("/", h.CreateInspection)
	inspections.Get("/:id", h.GetInspection)
	inspections.Post("/:id/void", h.VoidInspection)

	hygiene := qc.Group("/hygiene")
	hygiene.Get("/", h.ListHygieneChecks)
	hygiene.Post("/", h.CreateHygieneCheck)
	hygiene.Delete("/:id", h.DeleteHygieneCheck)

	preop := qc.Group("/preop")
	preop.Get("/", h.ListPreOpChecks)
	preop.Post("/", h.CreatePreOpCheck)
	preop.Delete("/:id", h.DeletePreOpCheck)

	packaging := qc.Group("/packaging")
	packaging.Get("/", h.ListPackagingChecks)
	packaging.Post("/", h.CreatePackagingCheck)
	packaging.Get("/:id", h.GetPackagingCheck)
	packaging.Put("/:id", h.UpdatePackagingCheck)
	packaging.Delete("/:id", h.DeletePackagingCheck)

	armada := qc.Group("/armada")
	armada.Get("/", h.ListArmadaChecks)
	armada.Post("/", h.CreateArmadaCheck)
	armada.Get("/:id", h.GetArmadaCheck)
	armada.Put("/:id", h.UpdateArmadaCheck)
	armada.Delete("/:id", h.DeleteArmadaCheck)

	incoming := qc.Group("/incoming")
	incoming.Get("/", h.ListIncomingChecks)
	incoming.Post("/", h.CreateIncomingCheck)
	incoming.Get("/:id", h.GetIncomingCheck)
	incoming.Put("/:id", h.UpdateIncomingCheck)
	incoming.Delete("/:id", h.DeleteIncomingCheck)
}
