package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	banking := router.Group("/banking", protected, tenantCtx)

	accounts := banking.Group("/accounts")
	accounts.Post("/", h.CreateBankAccount)
	accounts.Get("/", h.ListBankAccounts)
	accounts.Get("/:id", h.GetBankAccountByID)
	accounts.Get("/:id/statement-lines", h.ListStatementLines)
	accounts.Post("/:id/statement-lines/import", h.ImportStatementLines)
	accounts.Post("/:id/reconcile", h.AutoReconcile)

	statementLines := banking.Group("/statement-lines")
	statementLines.Post("/:id/match", h.ManualMatch)
}
