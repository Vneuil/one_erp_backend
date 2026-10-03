package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	inventory := router.Group("/inventory", protected, tenantCtx)

	warehouses := inventory.Group("/warehouses")
	warehouses.Post("/", h.CreateWarehouse)
	warehouses.Get("/", h.ListWarehouses)
	warehouses.Put("/:id", h.UpdateWarehouse)
	warehouses.Delete("/:id", h.DeleteWarehouse)

	stockLevels := inventory.Group("/stock-levels")
	stockLevels.Get("/", h.ListStockLevels)
	stockLevels.Post("/adjust", h.AdjustStock)
	stockLevels.Post("/buffer", h.SetBufferStock)

	batches := inventory.Group("/batches")
	batches.Get("/", h.ListBatches)
	batches.Get("/expiring", h.ExpiringBatches)
	batches.Post("/receive", h.ReceiveBatch)
	batches.Post("/:id/write-off", h.WriteOffBatch)

	movements := inventory.Group("/movements")
	movements.Get("/", h.ListMovements)
	movements.Get("/summary", h.MovementSummary)

	transfers := inventory.Group("/transfers")
	transfers.Post("/", h.CreateTransfer)
	transfers.Get("/", h.ListTransfers)
	transfers.Post("/:id/complete", h.CompleteTransfer)
	transfers.Post("/:id/cancel", h.CancelTransfer)

	opname := inventory.Group("/opname")
	opname.Post("/", h.CreateOpname)
	opname.Get("/", h.ListOpnames)
	opname.Get("/:id", h.GetOpnameByID)
	opname.Post("/:id/count", h.CountOpnameLine)
	opname.Post("/:id/finalize", h.FinalizeOpname)
}
