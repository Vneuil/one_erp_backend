package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	finance := router.Group("/finance", protected, tenantCtx)

	accounts := finance.Group("/accounts")
	accounts.Post("/", h.CreateAccount)
	accounts.Get("/", h.ListAccounts)
	accounts.Get("/:id", h.GetAccountByID)
	accounts.Put("/:id", h.UpdateAccount)

	journal := finance.Group("/journal-entries")
	journal.Post("/", h.CreateJournalEntry)
	journal.Get("/", h.ListJournalEntries)
	journal.Get("/:id", h.GetJournalEntryByID)
	journal.Post("/:id/post", h.PostJournalEntry)
	journal.Post("/:id/reverse", h.ReverseJournalEntry)

	payables := finance.Group("/payables")
	payables.Post("/", h.CreatePayable)
	payables.Get("/", h.ListPayables)
	payables.Get("/aging", h.APAgingReport)
	payables.Get("/:id", h.GetPayableByID)
	payables.Post("/:id/payments", h.RecordPayablePayment)

	receivables := finance.Group("/receivables")
	receivables.Post("/", h.CreateReceivable)
	receivables.Get("/", h.ListReceivables)
	receivables.Get("/aging", h.ARAgingReport)
	receivables.Get("/:id", h.GetReceivableByID)
	receivables.Post("/:id/payments", h.RecordReceivablePayment)

	pettyCash := finance.Group("/petty-cash")
	pettyCash.Post("/", h.CreatePettyCashFund)
	pettyCash.Get("/", h.ListPettyCashFunds)
	pettyCash.Post("/:id/transactions", h.CreatePettyCashTransaction)
	pettyCash.Get("/:id/transactions", h.ListPettyCashTransactions)
	pettyCash.Post("/:id/transactions/:txId/approve", h.ApprovePettyCashTransaction)
	pettyCash.Post("/:id/transactions/:txId/reject", h.RejectPettyCashTransaction)

	capital := finance.Group("/capital")
	capital.Get("/", h.ListCapital)
	capital.Post("/injections", h.recordCapital("injection"))
	capital.Post("/drawings", h.recordCapital("drawing"))

	otherIncome := finance.Group("/other-income")
	otherIncome.Get("/", h.ListOtherIncome)
	otherIncome.Post("/", h.RecordOtherIncome)

	budgets := finance.Group("/budgets")
	budgets.Post("/", h.CreateBudget)
	budgets.Get("/", h.ListBudgets)
	budgets.Put("/:id", h.UpdateBudget)
	budgets.Delete("/:id", h.DeleteBudget)

	cashVouchers := finance.Group("/cash-vouchers")
	cashVouchers.Post("/", h.CreateCashVoucher)
	cashVouchers.Get("/", h.ListCashVouchers)
	cashVouchers.Get("/:id", h.GetCashVoucher)
	cashVouchers.Post("/:id/void", h.VoidCashVoucher)

	tempAdvances := finance.Group("/temp-advances")
	tempAdvances.Post("/", h.IssueTempAdvance)
	tempAdvances.Get("/", h.ListTempAdvances)
	tempAdvances.Get("/outstanding", h.OutstandingTempAdvances)
	tempAdvances.Get("/:id", h.GetTempAdvance)
	tempAdvances.Post("/:id/settle", h.SettleTempAdvance)
	tempAdvances.Post("/:id/cancel", h.CancelTempAdvance)

	cashAdvances := finance.Group("/cash-advances")
	cashAdvances.Get("/", h.ListCashAdvances)
	cashAdvances.Get("/reports/outstanding", h.GetOutstandingCashAdvancesReport)
	cashAdvances.Get("/reports/differences", h.GetCashAdvanceDifferencesReport)
	cashAdvances.Get("/:id", h.GetCashAdvance)
	cashAdvances.Post("/cash-request", h.CreateCashRequest)
	cashAdvances.Post("/:id/approve-cr", h.ApproveCashRequest)
	cashAdvances.Post("/:id/reject-cr", h.RejectCashRequest)
	cashAdvances.Post("/:id/fill-form", h.FillCashAdvanceForm)
	cashAdvances.Post("/:id/approve-form", h.ApproveCashAdvanceForm)
	cashAdvances.Post("/:id/reject-form", h.RejectCashAdvanceForm)
	cashAdvances.Post("/:id/disburse", h.DisburseCashAdvance)
	cashAdvances.Post("/:id/submit-settlement", h.SubmitSettlement)
	cashAdvances.Post("/:id/approve-settlement", h.ApproveSettlement)
	cashAdvances.Post("/:id/reject-settlement", h.RejectSettlement)

	reports := finance.Group("/reports")
	reports.Get("/general-ledger", h.GeneralLedger)
	reports.Get("/cash-book", h.CashBook)
	reports.Get("/expense-breakdown", h.ExpenseBreakdown)
	reports.Get("/non-operating", h.NonOperating)
	reports.Get("/trial-balance", h.TrialBalance)
	reports.Get("/insights", h.Insights)
	reports.Get("/profit-loss", h.ProfitAndLoss)
	reports.Get("/balance-sheet", h.BalanceSheet)
	reports.Get("/cash-flow", h.CashFlow)
	reports.Get("/memorial-journals", h.MemorialJournals)
}
