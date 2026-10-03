package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	procurement := router.Group("/procurement", protected, tenantCtx)

	requests := procurement.Group("/requests")
	requests.Post("/", h.CreatePurchaseRequest)
	requests.Get("/", h.ListPurchaseRequests)
	requests.Get("/:id", h.GetPurchaseRequestByID)
	requests.Post("/:id/submit", h.SubmitPurchaseRequest)
	requests.Post("/:id/approve", h.ApprovePurchaseRequest)
	requests.Post("/:id/reject", h.RejectPurchaseRequest)

	orders := procurement.Group("/orders")
	orders.Post("/", h.CreatePurchaseOrder)
	orders.Get("/", h.ListPurchaseOrders)
	orders.Get("/:id", h.GetPurchaseOrderByID)
	orders.Post("/:id/approve", h.ApprovePurchaseOrder)
	orders.Post("/:id/reject", h.RejectPurchaseOrder)
	orders.Post("/:id/cancel", h.CancelPurchaseOrder)

	receipts := procurement.Group("/receipts")
	receipts.Post("/", h.CreateGoodsReceipt)
	receipts.Get("/", h.ListGoodsReceipts)
	receipts.Get("/:id", h.GetGoodsReceiptByID)

	invoices := procurement.Group("/invoices")
	invoices.Post("/", h.CreatePurchaseInvoice)
	invoices.Get("/", h.ListPurchaseInvoices)
	invoices.Get("/:id", h.GetPurchaseInvoiceByID)
	invoices.Post("/:id/payments", h.RecordInvoicePayment)

	downPayments := procurement.Group("/down-payments")
	downPayments.Post("/", h.CreateDownPayment)
	downPayments.Get("/", h.ListDownPayments)

	returns := procurement.Group("/returns")
	returns.Post("/", h.CreatePurchaseReturn)
	returns.Get("/", h.ListPurchaseReturns)
	returns.Get("/:id", h.GetPurchaseReturnByID)
}
