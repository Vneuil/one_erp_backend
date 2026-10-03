package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	products := router.Group("/products", protected, tenantCtx)

	products.Post("/", h.Create)
	products.Get("/", h.List)
	products.Get("/copy-sources", h.CopySources)
	products.Get("/copy-sources/:tenantId", h.CopySourceProducts)
	products.Post("/copy", h.CopyProducts)
	products.Get("/:id", h.GetByID)
	products.Put("/:id", h.Update)
	products.Delete("/:id", h.Delete)

	categories := router.Group("/product-categories", protected, tenantCtx)
	categories.Post("/", h.CreateCategory)
	categories.Get("/", h.ListCategories)
	categories.Put("/:id", h.UpdateCategory)
	categories.Delete("/:id", h.DeleteCategory)

	units := router.Group("/units-of-measure", protected, tenantCtx)
	units.Post("/", h.CreateUnit)
	units.Get("/", h.ListUnits)
	units.Put("/:id", h.UpdateUnit)
	units.Delete("/:id", h.DeleteUnit)
}
