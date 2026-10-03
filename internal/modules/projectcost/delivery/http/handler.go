package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	projectinfra "github.com/divinecoid/one-backend/internal/modules/project/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/projectcost/application"
	"github.com/divinecoid/one-backend/internal/modules/projectcost/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) wrap(msg string, status int, fn func(c *fiber.Ctx, s *application.Service) (any, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		db := middleware.TenantDB(c)
		if db == nil {
			return apperrors.NewBadRequest("No active company. Please select or provision a company first.")
		}
		s := application.NewService(infrastructure.NewRepository(db), projectinfra.NewProjectRepository(db))
		data, err := fn(c, s)
		if err != nil {
			return err
		}
		return response.Success(c, status, msg, data)
	}
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func param(c *fiber.Ctx, name string) (uuid.UUID, error) {
	v, err := uuid.Parse(c.Params(name))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid " + name)
	}
	return v, nil
}

func body(c *fiber.Ctx, v any) error {
	if err := c.BodyParser(v); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	return nil
}

// ---- budget and costs

func (h *Handler) Budget() fiber.Handler {
	return h.wrap("Budget retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service) (any, error) {
		id, err := param(c, "id")
		if err != nil {
			return nil, err
		}
		return s.Budget(h.ctx(c), id)
	})
}

func (h *Handler) ImportBudget() fiber.Handler {
	return h.wrap("Budget lines imported", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service) (any, error) {
		id, err := param(c, "id")
		if err != nil {
			return nil, err
		}
		var in struct {
			Mode  string                        `json:"mode"`
			Lines []application.BudgetLineInput `json:"lines"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.AddBudgetLines(h.ctx(c), id, in.Mode, in.Lines)
	})
}

func (h *Handler) DeleteBudgetLine() fiber.Handler {
	return h.wrap("Budget line deleted", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service) (any, error) {
		pid, err := param(c, "id")
		if err != nil {
			return nil, err
		}
		lid, err := param(c, "lineId")
		if err != nil {
			return nil, err
		}
		return nil, s.DeleteBudgetLine(h.ctx(c), pid, lid)
	})
}

func (h *Handler) RecordCost() fiber.Handler {
	return h.wrap("Cost recorded", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service) (any, error) {
		id, err := param(c, "id")
		if err != nil {
			return nil, err
		}
		var in struct {
			Category    string  `json:"category"`
			Description string  `json:"description"`
			Date        string  `json:"date"`
			Amount      float64 `json:"amount"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.RecordCost(h.ctx(c), id, application.CostInput{Category: in.Category, Description: in.Description, Date: in.Date, Amount: in.Amount})
	})
}

func (h *Handler) ListCosts() fiber.Handler {
	return h.wrap("Costs retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service) (any, error) {
		id, err := param(c, "id")
		if err != nil {
			return nil, err
		}
		return s.ListCosts(h.ctx(c), id)
	})
}

func (h *Handler) DeleteCost() fiber.Handler {
	return h.wrap("Cost deleted", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service) (any, error) {
		pid, err := param(c, "id")
		if err != nil {
			return nil, err
		}
		cid, err := param(c, "costId")
		if err != nil {
			return nil, err
		}
		return nil, s.DeleteCost(h.ctx(c), pid, cid)
	})
}

// ---- work orders

type workOrderBody struct {
	ProjectID     *uuid.UUID `json:"projectId"`
	SalesOrderID  *uuid.UUID `json:"salesOrderId"`
	CustomerName  string     `json:"customerName"`
	Title         string     `json:"title"`
	Scope         string     `json:"scope"`
	ContractValue float64    `json:"contractValue"`
	StartDate     string     `json:"startDate"`
	DueDate       string     `json:"dueDate"`
	AssignedTo    string     `json:"assignedTo"`
	Notes         string     `json:"notes"`
}

func (h *Handler) CreateWorkOrder() fiber.Handler {
	return h.wrap("Work order created", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service) (any, error) {
		var in workOrderBody
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.CreateWorkOrder(h.ctx(c), application.WorkOrderInput(in))
	})
}

func (h *Handler) Profitability() fiber.Handler {
	return h.wrap("Project profitability retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service) (any, error) {
		return s.Profitability(h.ctx(c))
	})
}

func (h *Handler) ListWorkOrders() fiber.Handler {
	return h.wrap("Work orders retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service) (any, error) {
		var pid *uuid.UUID
		if raw := c.Query("projectId"); raw != "" {
			v, err := uuid.Parse(raw)
			if err != nil {
				return nil, apperrors.NewBadRequest("Invalid projectId")
			}
			pid = &v
		}
		return s.ListWorkOrders(h.ctx(c), c.Query("status"), pid)
	})
}

func (h *Handler) Advance(to string) fiber.Handler {
	return h.wrap("Work order updated", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service) (any, error) {
		id, err := param(c, "id")
		if err != nil {
			return nil, err
		}
		return s.Advance(h.ctx(c), id, to)
	})
}

func (h *Handler) SetProgress() fiber.Handler {
	return h.wrap("Progress updated", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service) (any, error) {
		id, err := param(c, "id")
		if err != nil {
			return nil, err
		}
		var in struct {
			Progress int `json:"progress"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.SetProgress(h.ctx(c), id, in.Progress)
	})
}
