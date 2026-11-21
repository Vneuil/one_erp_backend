package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	manufacturing := router.Group("/manufacturing", protected, tenantCtx)

	manufacturing.Get("/dashboard", h.GetDashboardSummary)

	bom := manufacturing.Group("/bom")
	bom.Post("/", h.CreateBOM)
	bom.Get("/", h.ListBOMs)
	bom.Get("/:id", h.GetBOMByID)

	orders := manufacturing.Group("/production-orders")
	orders.Post("/", h.CreateOrder)
	orders.Get("/", h.ListOrders)
	orders.Get("/:id", h.GetOrderByID)
	orders.Post("/:id/release", h.ReleaseOrder)
	orders.Post("/:id/pause", h.PauseOrder)
	orders.Post("/:id/resume", h.ResumeOrder)
	orders.Post("/:id/cancel", h.CancelOrder)
	orders.Post("/:id/complete-batch", h.CompleteBatch)
	orders.Get("/:id/batches", h.ListBatches)
	orders.Post("/:id/batches/:batchId/void", h.VoidBatch)
	orders.Post("/:id/steps/:stepId/log", h.LogStep)
	orders.Get("/:id/step-logs", h.ListStepLogs)
	orders.Post("/:id/step-logs/:logId/void", h.VoidStepLog)
	orders.Post("/:id/evaluate-conditions", h.EvaluateConditions)
	orders.Post("/:id/steps/:stepId/qc-pass", h.SetStepQCPass)

	reports := manufacturing.Group("/reports")
	reports.Get("/daily", h.DailyReport)
	reports.Get("/order-summary", h.OrderSummary)
	reports.Get("/process-summary", h.ProcessSummary)
	reports.Get("/wip", h.WIPReport)

	traceability := manufacturing.Group("/traceability")
	traceability.Get("/backward/:fgBatchNo", h.GetBackwardTraceability)
	traceability.Get("/forward/:materialBatchNo", h.GetForwardTraceability)

	requisitions := manufacturing.Group("/requisitions")
	requisitions.Get("/", h.ListRequisitions)
	requisitions.Post("/", h.CreateRequisition)
	requisitions.Get("/:id", h.GetRequisitionByID)
	requisitions.Put("/:id/status", h.UpdateRequisitionStatus)
	requisitions.Post("/:id/acc-gudang", h.ApproveRequisitionByWarehouse)
	requisitions.Post("/:id/enter-production", h.EnterProductionFromRequisition)
	requisitions.Post("/:id/reject", h.RejectRequisition)

	lbb := manufacturing.Group("/lbb")
	lbb.Get("/", h.ListLBBReports)
	lbb.Post("/", h.CreateLBBReport)
	lbb.Get("/:id", h.GetLBBReportByID)
	lbb.Put("/:id", h.UpdateLBBReport)
	lbb.Delete("/:id", h.DeleteLBBReport)

	oven := manufacturing.Group("/oven-logs")
	oven.Get("/", h.ListOvenLogs)
	oven.Post("/", h.CreateOvenLog)
	oven.Get("/:id", h.GetOvenLogByID)

	// LHP (Laporan Hasil Produksi — PRO-002-LHP)
	lhp := manufacturing.Group("/lhp")
	lhp.Get("/", h.ListLHP)
	lhp.Post("/", h.CreateLHP)
	lhp.Get("/:id", h.GetLHP)
	lhp.Put("/:id", h.UpdateLHP)
	lhp.Delete("/:id", h.DeleteLHP)
	lhp.Post("/:id/generate-lst", h.GenerateLSTFromLHP)

	// LST (Laporan Serah Terima — GDG-009-LST)
	lst := manufacturing.Group("/lst")
	lst.Get("/", h.ListLST)
	lst.Post("/", h.CreateLST)
	lst.Get("/:id", h.GetLST)
	lst.Put("/:id", h.UpdateLST)
	lst.Delete("/:id", h.DeleteLST)
	lst.Post("/:id/verify-qc", h.VerifyLSTByQC)
	lst.Post("/:id/accept-warehouse", h.AcceptLSTByWarehouse)
}
