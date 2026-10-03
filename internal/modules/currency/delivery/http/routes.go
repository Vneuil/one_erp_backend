package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	currencies := router.Group("/currencies", protected, tenantCtx)
	currencies.Get("/", h.ListCurrencies)
	currencies.Post("/", h.CreateCurrency)
	currencies.Delete("/:id", h.DeleteCurrency)
	currencies.Post("/convert", h.Convert)

	rates := router.Group("/exchange-rates", protected, tenantCtx)
	rates.Get("/", h.ListExchangeRates)
	rates.Post("/", h.SetExchangeRate)
}
