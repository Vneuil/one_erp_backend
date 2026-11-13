package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	sales := router.Group("/sales", protected, tenantCtx)

	sales.Post("/orders/upload-attachment", h.UploadAttachment)
	sales.Get("/orders/attachment/download", h.DownloadAttachmentByKey)
	sales.Get("/orders/:id/attachment", h.DownloadAttachment)
	sales.Put("/orders/:id/invoice-document", h.AttachInvoiceDocument)
	sales.Get("/orders/:id/invoice-attachment", h.DownloadInvoiceAttachment)
	sales.Get("/orders/last-price", h.GetLastSalePrice)
	sales.Post("/orders", h.CreateOrder)
	sales.Get("/orders", h.ListOrders)
	sales.Get("/orders/:id", h.GetOrderByID)
	sales.Put("/orders/:id", h.UpdateOrder)
	sales.Get("/orders/:id/revisions", h.ListOrderRevisions)
	sales.Post("/orders/:id/submit", h.SubmitOrder)
	sales.Post("/orders/:id/approve", h.ApproveOrder)
	sales.Post("/orders/:id/reject", h.RejectOrder)
	sales.Get("/orders/:id/billing-schedule", h.ListBillingSchedule)
	sales.Put("/orders/:id/billing-schedule", h.SetBillingSchedule)
	sales.Post("/billing-terms/:id/invoice", h.InvoiceBillingTerm)

	sales.Post("/quotations", h.CreateQuotation)
	sales.Get("/quotations", h.ListQuotations)
	sales.Get("/quotations/:id", h.GetQuotationByID)
	sales.Put("/quotations/:id", h.UpdateQuotation)
	sales.Post("/quotations/:id/convert", h.ConvertQuotationToOrder)

	// PI / CI menu: superadmin, admin and the accountants (see actor.TradeDocRoles).
	trade := sales.Group("/trade-invoices", middleware.RequireRoles(actor.TradeDocRoles...))
	trade.Post("/", h.CreateTradeInvoice)
	trade.Get("/", h.ListTradeInvoices)
	trade.Get("/:id", h.GetTradeInvoice)
	trade.Post("/:id/issue", h.IssueTradeInvoice)
	trade.Post("/:id/cancel", h.CancelTradeInvoice)
	trade.Post("/:id/create-order", h.CreateOrderFromTradeInvoice)

	sales.Post("/invoices", h.CreateInvoice)
	sales.Get("/invoices", h.ListInvoices)
	sales.Post("/invoices/:id/payments", h.RecordInvoicePayment)

	sales.Post("/deliveries", h.CreateDelivery)
	sales.Get("/deliveries", h.ListDeliveries)
	sales.Put("/deliveries/:id/shipping-cost", h.RecordShippingCost)

	sales.Post("/down-payments", h.CreateDownPayment)
	sales.Get("/down-payments", h.ListDownPayments)
	sales.Post("/down-payments/:id/apply", h.ApplyDownPayment)

	sales.Post("/returns", h.CreateSalesReturn)
	sales.Get("/returns", h.ListSalesReturns)
	sales.Get("/returns/:id", h.GetSalesReturnByID)
}
