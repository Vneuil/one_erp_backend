package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/hrletters/application"
	"github.com/divinecoid/one-backend/internal/modules/hrletters/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrletters/infrastructure"
	hrmInfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func parseID(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid id format")
	}
	return id, nil
}

// resolve builds a use-case bound to the caller's own tenant database.
func (h *Handler) resolve(c *fiber.Ctx) (application.UseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	return application.NewUseCase(infrastructure.NewRepository(tenantDB), hrmInfra.NewHRMRepository(tenantDB)), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func call[T any](h *Handler, c *fiber.Ctx, message string, fn func(application.UseCase, context.Context) (T, error)) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := fn(uc, h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, message, v)
}

func body[T any](c *fiber.Ctx) (T, error) {
	var in T
	if err := c.BodyParser(&in); err != nil {
		return in, apperrors.NewBadRequest("Invalid request body")
	}
	return in, nil
}

func (h *Handler) Create(c *fiber.Ctx) error {
	in, err := body[application.LetterInput](c)
	if err != nil {
		return err
	}
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	l, err := uc.Create(h.ctx(c), in)
	if err != nil {
		return err
	}
	return response.Created(c, "Letter drafted", l)
}

func (h *Handler) Preview(c *fiber.Ctx) error {
	in, err := body[application.LetterInput](c)
	if err != nil {
		return err
	}
	return call(h, c, "Letter preview", func(uc application.UseCase, ctx context.Context) (map[string]string, error) {
		text, err := uc.Preview(ctx, in)
		return map[string]string{"body": text}, err
	})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	in, err := body[application.LetterInput](c)
	if err != nil {
		return err
	}
	return call(h, c, "Letter updated", func(uc application.UseCase, ctx context.Context) (*domain.Letter, error) {
		return uc.Update(ctx, id, in)
	})
}

func (h *Handler) Get(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	return call(h, c, "Letter retrieved", func(uc application.UseCase, ctx context.Context) (*domain.Letter, error) { return uc.Get(ctx, id) })
}

func (h *Handler) List(c *fiber.Ctx) error {
	f := domain.Filter{Type: c.Query("type"), Status: c.Query("status"), From: c.Query("from"), To: c.Query("to"), Search: c.Query("search")}
	if raw := c.Query("employeeId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return apperrors.NewBadRequest("Invalid employeeId format")
		}
		f.EmployeeID = &id
	}
	return call(h, c, "Letters retrieved", func(uc application.UseCase, ctx context.Context) ([]domain.Letter, error) { return uc.List(ctx, f) })
}

func (h *Handler) Issue(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	return call(h, c, "Letter issued", func(uc application.UseCase, ctx context.Context) (*domain.Letter, error) { return uc.Issue(ctx, id) })
}

func (h *Handler) Cancel(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	in, err := body[struct {
		Reason string `json:"reason"`
	}](c)
	if err != nil {
		return err
	}
	return call(h, c, "Letter cancelled", func(uc application.UseCase, ctx context.Context) (*domain.Letter, error) {
		return uc.Cancel(ctx, id, in.Reason)
	})
}

func (h *Handler) Acknowledge(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	return call(h, c, "Acknowledgement recorded", func(uc application.UseCase, ctx context.Context) (*domain.Letter, error) {
		return uc.Acknowledge(ctx, id)
	})
}

func (h *Handler) Apply(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	return call(h, c, "Letter applied to the employee record", func(uc application.UseCase, ctx context.Context) (*domain.Letter, error) { return uc.Apply(ctx, id) })
}

func (h *Handler) EmployeeSummary(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	return call(h, c, "Employee letters retrieved", func(uc application.UseCase, ctx context.Context) (*application.EmployeeSummaryDTO, error) {
		return uc.EmployeeSummary(ctx, id)
	})
}

func (h *Handler) ExpiringContracts(c *fiber.Ctx) error {
	return call(h, c, "Expiring contracts retrieved", func(uc application.UseCase, ctx context.Context) ([]application.ExpiringContract, error) {
		return uc.ExpiringContracts(ctx, c.QueryInt("days", 60))
	})
}
