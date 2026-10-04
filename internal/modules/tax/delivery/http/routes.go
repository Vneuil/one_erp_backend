package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	tax := router.Group("/tax", protected, tenantCtx)

	tax.Get("/settings", h.GetSettings)
	tax.Put("/settings", middleware.RequireRoles("admin"), h.UpdateSettings)

	serials := tax.Group("/serial-ranges")
	serials.Get("/", h.ListSerialRanges)
	serials.Post("/", middleware.RequireRoles("admin"), h.CreateSerialRange)
	serials.Put("/:id/active", middleware.RequireRoles("admin"), h.SetSerialRangeActive)

	invoices := tax.Group("/invoices")
	invoices.Get("/", h.ListTaxInvoices)
	invoices.Get("/coretax-export", h.CoretaxExport)
	invoices.Post("/from-sales-invoice", h.CreateFromSalesInvoice)
	invoices.Post("/from-purchase-invoice", h.CreateFromPurchaseInvoice)
	invoices.Get("/:id", h.GetTaxInvoice)
	invoices.Put("/:id", h.UpdateDraft)
	invoices.Post("/:id/issue", h.Issue)
	invoices.Post("/:id/cancel", h.Cancel)
	invoices.Post("/:id/replace", h.Replace)
	invoices.Put("/:id/tax-number", h.SetTaxNumber)

	reports := tax.Group("/reports")
	reports.Get("/sales-book", h.SalesBook)
	reports.Get("/vat-summary", h.VATSummary)
}
