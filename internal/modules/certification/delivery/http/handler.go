package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/certification/application"
	"github.com/divinecoid/one-backend/internal/modules/certification/domain"
	"github.com/divinecoid/one-backend/internal/modules/certification/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct{}

func NewHandler() *Handler { return &Handler{} }

func (h *Handler) db(c *fiber.Ctx) (*gorm.DB, error) {
	db := middleware.TenantDB(c)
	if db == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	return db, nil
}

func (h *Handler) uc(c *fiber.Ctx) (application.UseCase, error) {
	db, err := h.db(c)
	if err != nil {
		return nil, err
	}
	return application.NewUseCase(infrastructure.NewRepository(db)), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func idParam(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid certificate ID format")
	}
	return id, nil
}

func (h *Handler) Schemes(c *fiber.Ctx) error {
	return response.OK(c, "Schemes retrieved successfully", application.CommonSchemes)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	uc, err := h.uc(c)
	if err != nil {
		return err
	}
	var dto application.CertificateDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := uc.Create(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Certificate created successfully", res)
}

func (h *Handler) List(c *fiber.Ctx) error {
	uc, err := h.uc(c)
	if err != nil {
		return err
	}
	f := domain.ListFilter{Scheme: c.Query("scheme")}
	if v := c.Query("supplierId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return apperrors.NewBadRequest("Invalid supplierId")
		}
		f.SupplierID = &id
	}
	if v := c.Query("productId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return apperrors.NewBadRequest("Invalid productId")
		}
		f.ProductID = &id
	}
	res, err := uc.List(h.ctx(c), f)
	if err != nil {
		return err
	}
	return response.OK(c, "Certificates retrieved successfully", res)
}

func (h *Handler) Eligibility(c *fiber.Ctx) error {
	uc, err := h.uc(c)
	if err != nil {
		return err
	}
	res, err := uc.Eligibility(h.ctx(c), c.Query("scheme"))
	if err != nil {
		return err
	}
	return response.OK(c, "Supplier certification status retrieved successfully", res)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	uc, err := h.uc(c)
	if err != nil {
		return err
	}
	id, err := idParam(c)
	if err != nil {
		return err
	}
	res, err := uc.Get(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Certificate retrieved successfully", res)
}

func (h *Handler) Update(c *fiber.Ctx) error {
	uc, err := h.uc(c)
	if err != nil {
		return err
	}
	id, err := idParam(c)
	if err != nil {
		return err
	}
	var dto application.CertificateDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := uc.Update(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Certificate updated successfully", res)
}

type reasonDTO struct {
	Reason string `json:"reason"`
}

func (h *Handler) action(c *fiber.Ctx, msg string, fn func(application.UseCase, context.Context, uuid.UUID, string) (*application.CertificateResponse, error)) error {
	uc, err := h.uc(c)
	if err != nil {
		return err
	}
	id, err := idParam(c)
	if err != nil {
		return err
	}
	var body reasonDTO
	_ = c.BodyParser(&body)
	res, err := fn(uc, h.ctx(c), id, body.Reason)
	if err != nil {
		return err
	}
	return response.OK(c, msg, res)
}

func (h *Handler) Suspend(c *fiber.Ctx) error {
	return h.action(c, "Certificate suspended", func(uc application.UseCase, ctx context.Context, id uuid.UUID, r string) (*application.CertificateResponse, error) {
		return uc.Suspend(ctx, id, r)
	})
}

func (h *Handler) Resume(c *fiber.Ctx) error {
	return h.action(c, "Certificate resumed", func(uc application.UseCase, ctx context.Context, id uuid.UUID, _ string) (*application.CertificateResponse, error) {
		return uc.Resume(ctx, id)
	})
}

func (h *Handler) Revoke(c *fiber.Ctx) error {
	return h.action(c, "Certificate revoked", func(uc application.UseCase, ctx context.Context, id uuid.UUID, r string) (*application.CertificateResponse, error) {
		return uc.Revoke(ctx, id, r)
	})
}

// TraceLot follows one lot number: receipt -> QC -> production genealogy -> finished
// batches -> their QC inspections (the basis of the COI), plus the supplier's certificates.
func (h *Handler) TraceLot(c *fiber.Ctx) error {
	db, err := h.db(c)
	if err != nil {
		return err
	}
	res, err := application.TraceLot(h.ctx(c), infrastructure.NewTraceReader(db), c.Params("lot"))
	if err != nil {
		return err
	}
	return response.OK(c, "Lot trace retrieved successfully", res)
}
