package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	assets := router.Group("/assets", protected, tenantCtx)
	assets.Post("/", h.CreateAsset)
	assets.Get("/", h.ListAssets)
	assets.Post("/recalculate-all", h.RecalculateAllDepreciation)
	assets.Get("/:id", h.GetAssetByID)
	assets.Put("/:id", h.UpdateAsset)
	assets.Put("/:id/status", h.UpdateAssetStatus)
	assets.Post("/:id/recalculate-depreciation", h.RecalculateDepreciation)
}
