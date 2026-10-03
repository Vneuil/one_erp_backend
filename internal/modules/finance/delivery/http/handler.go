package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/finance/application"
	"github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func parseID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid " + param + " format")
	}
	return id, nil
}

// resolve builds a use-case bound to the caller's own tenant database. The
// schema is migrated and seeded once, at tenant-provision time (see
// module.go's registration with foundation/tenant.RegisterSchema).
func (h *Handler) resolve(c *fiber.Ctx) (application.FinanceUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewFinanceRepository(tenantDB)
	return application.NewFinanceUseCase(repo), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// Accounts

func (h *Handler) CreateAccount(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateAccountDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	a, err := uc.CreateAccount(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Account created successfully", a)
}

func (h *Handler) GetAccountByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	a, err := uc.GetAccountByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Account retrieved successfully", a)
}

func (h *Handler) ListAccounts(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListAccounts(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Accounts retrieved successfully", items, meta)
}

func (h *Handler) UpdateAccount(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.CreateAccountDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	a, err := uc.UpdateAccount(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Account updated successfully", a)
}

// Journal entries

func (h *Handler) CreateJournalEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateJournalEntryDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	je, err := uc.CreateJournalEntry(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Journal entry created successfully", je)
}

func (h *Handler) GetJournalEntryByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	je, err := uc.GetJournalEntryByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Journal entry retrieved successfully", je)
}

func (h *Handler) ListJournalEntries(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListJournalEntries(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Journal entries retrieved successfully", items, meta)
}

func (h *Handler) PostJournalEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	je, err := uc.PostJournalEntry(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Journal entry posted successfully", je)
}

func (h *Handler) ReverseJournalEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	je, err := uc.ReverseJournalEntry(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.Created(c, "Reversing journal entry created successfully", je)
}

// Payables

func (h *Handler) CreatePayable(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePayableDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	p, err := uc.CreatePayable(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Payable created successfully", p)
}

func (h *Handler) GetPayableByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	p, err := uc.GetPayableByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Payable retrieved successfully", p)
}

func (h *Handler) ListPayables(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListPayables(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Payables retrieved successfully", items, meta)
}

func (h *Handler) RecordPayablePayment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.RecordPaymentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	p, err := uc.RecordPayablePayment(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Payment recorded successfully", p)
}

func (h *Handler) APAgingReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	report, err := uc.APAgingReport(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "AP aging report retrieved successfully", report)
}

// Receivables

func (h *Handler) CreateReceivable(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateReceivableDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	r, err := uc.CreateReceivable(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Receivable created successfully", r)
}

func (h *Handler) GetReceivableByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	r, err := uc.GetReceivableByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Receivable retrieved successfully", r)
}

func (h *Handler) ListReceivables(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListReceivables(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Receivables retrieved successfully", items, meta)
}

func (h *Handler) RecordReceivablePayment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.RecordPaymentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	r, err := uc.RecordReceivablePayment(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Payment recorded successfully", r)
}

func (h *Handler) ARAgingReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	report, err := uc.ARAgingReport(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "AR aging report retrieved successfully", report)
}

// Petty cash

func (h *Handler) CreatePettyCashFund(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreatePettyCashFundDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	f, err := uc.CreatePettyCashFund(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Petty cash fund created successfully", f)
}

func (h *Handler) ListPettyCashFunds(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListPettyCashFunds(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Petty cash funds retrieved successfully", items, meta)
}

func (h *Handler) CreatePettyCashTransaction(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	fundID, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.PettyCashTransactionDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	f, err := uc.CreatePettyCashTransaction(h.ctx(c), fundID, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Petty cash transaction recorded successfully", f)
}

func (h *Handler) ListPettyCashTransactions(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	fundID, err := parseID(c, "id")
	if err != nil {
		return err
	}
	txs, err := uc.ListPettyCashTransactions(h.ctx(c), fundID)
	if err != nil {
		return err
	}
	return response.OK(c, "Petty cash transactions retrieved successfully", txs)
}

func (h *Handler) ApprovePettyCashTransaction(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	fundID, err := parseID(c, "id")
	if err != nil {
		return err
	}
	txID, err := parseID(c, "txId")
	if err != nil {
		return err
	}
	f, err := uc.ApprovePettyCashTransaction(h.ctx(c), fundID, txID)
	if err != nil {
		return err
	}
	return response.OK(c, "Petty cash transaction approved successfully", f)
}

func (h *Handler) RejectPettyCashTransaction(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	fundID, err := parseID(c, "id")
	if err != nil {
		return err
	}
	txID, err := parseID(c, "txId")
	if err != nil {
		return err
	}
	f, err := uc.RejectPettyCashTransaction(h.ctx(c), fundID, txID)
	if err != nil {
		return err
	}
	return response.OK(c, "Petty cash transaction rejected successfully", f)
}

// Budgets

func (h *Handler) CreateBudget(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateBudgetDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	b, err := uc.CreateBudget(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Budget created successfully", b)
}

func (h *Handler) UpdateBudget(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid budget ID format")
	}
	var dto application.UpdateBudgetDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	b, err := uc.UpdateBudget(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Budget updated successfully", b)
}

func (h *Handler) DeleteBudget(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid budget ID format")
	}
	if err := uc.DeleteBudget(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Budget deleted successfully", nil)
}

func (h *Handler) ListBudgets(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListBudgets(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Budgets retrieved successfully", items, meta)
}

// Reports

func (h *Handler) TrialBalance(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	report, err := uc.TrialBalance(h.ctx(c), c.Query("asOf"))
	if err != nil {
		return err
	}
	return response.OK(c, "Trial balance retrieved successfully", report)
}

func (h *Handler) ProfitAndLoss(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	report, err := uc.ProfitAndLoss(h.ctx(c), c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	return response.OK(c, "Profit and loss statement retrieved successfully", report)
}

func (h *Handler) BalanceSheet(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	report, err := uc.BalanceSheet(h.ctx(c), c.Query("asOf"))
	if err != nil {
		return err
	}
	return response.OK(c, "Balance sheet retrieved successfully", report)
}

func (h *Handler) CashFlow(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	report, err := uc.CashFlow(h.ctx(c), c.Query("from"), c.Query("to"))
	if err != nil {
		return err
	}
	return response.OK(c, "Cash flow statement retrieved successfully", report)
}

// ---- owner capital and other income

func (h *Handler) recordCapital(typ string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		uc, err := h.resolve(c)
		if err != nil {
			return err
		}
		var in struct {
			OwnerName          string  `json:"ownerName"`
			Amount             float64 `json:"amount"`
			Date               string  `json:"date"`
			Description        string  `json:"description"`
			PaymentAccountCode string  `json:"paymentAccountCode"`
		}
		if err := c.BodyParser(&in); err != nil {
			return apperrors.NewBadRequest("Invalid request body")
		}
		v, err := uc.RecordCapital(h.ctx(c), typ, application.CapitalInput{OwnerName: in.OwnerName, Date: in.Date, Description: in.Description, PaymentAccountCode: in.PaymentAccountCode, Amount: in.Amount})
		if err != nil {
			return err
		}
		return response.Created(c, "Capital transaction recorded", v)
	}
}

func (h *Handler) ListCapital(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := uc.ListCapital(h.ctx(c), c.Query("type"), c.Query("period"))
	if err != nil {
		return err
	}
	return response.OK(c, "Capital transactions retrieved", v)
}

func (h *Handler) RecordOtherIncome(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		Category           string  `json:"category"`
		Amount             float64 `json:"amount"`
		Date               string  `json:"date"`
		Description        string  `json:"description"`
		PaymentAccountCode string  `json:"paymentAccountCode"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	v, err := uc.RecordOtherIncome(h.ctx(c), application.OtherIncomeInput{Category: in.Category, Date: in.Date, Description: in.Description, PaymentAccountCode: in.PaymentAccountCode, Amount: in.Amount})
	if err != nil {
		return err
	}
	return response.Created(c, "Other income recorded", v)
}

func (h *Handler) ListOtherIncome(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := uc.ListOtherIncome(h.ctx(c), c.Query("category"), c.Query("period"))
	if err != nil {
		return err
	}
	return response.OK(c, "Other income retrieved", v)
}

func (h *Handler) Insights(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	out, err := uc.Insights(h.ctx(c), c.Query("from"))
	if err != nil {
		return err
	}
	return response.OK(c, "Financial insights retrieved", out)
}
