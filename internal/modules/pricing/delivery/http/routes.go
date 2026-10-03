package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	paymentTerms := router.Group("/payment-terms", protected, tenantCtx)
	paymentTerms.Post("/", h.CreatePaymentTerm)
	paymentTerms.Get("/", h.ListPaymentTerms)
	paymentTerms.Put("/:id", h.UpdatePaymentTerm)
	paymentTerms.Delete("/:id", h.DeletePaymentTerm)

	customerTypes := router.Group("/customer-types", protected, tenantCtx)
	customerTypes.Post("/", h.CreateCustomerType)
	customerTypes.Get("/", h.ListCustomerTypes)
	customerTypes.Put("/:id", h.UpdateCustomerType)
	customerTypes.Delete("/:id", h.DeleteCustomerType)

	taxRates := router.Group("/tax-rates", protected, tenantCtx)
	taxRates.Post("/", h.CreateTaxRate)
	taxRates.Get("/", h.ListTaxRates)
	taxRates.Put("/:id", h.UpdateTaxRate)
	taxRates.Delete("/:id", h.DeleteTaxRate)
}
