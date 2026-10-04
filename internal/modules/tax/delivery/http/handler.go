package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/tax/application"
	"github.com/divinecoid/one-backend/internal/modules/tax/domain"
	"github.com/divinecoid/one-backend/internal/modules/tax/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.TaxUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	return application.NewTaxUseCase(infrastructure.NewTaxRepository(tenantDB), infrastructure.NewSourceReader(tenantDB)), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

// call resolves the use-case and runs fn; the result is wrapped as 200 OK.
func call[T any](h *Handler, c *fiber.Ctx, message string, fn func(application.TaxUseCase, context.Context) (T, error)) error {
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

// Settings and serial ranges

func (h *Handler) GetSettings(c *fiber.Ctx) error {
	return call(h, c, "Tax settings retrieved", func(uc application.TaxUseCase, ctx context.Context) (*domain.TaxSettings, error) {
		return uc.GetSettings(ctx)
	})
}

func (h *Handler) UpdateSettings(c *fiber.Ctx) error {
	in, err := body[application.SettingsInput](c)
	if err != nil {
		return err
	}
	return call(h, c, "Tax settings saved", func(uc application.TaxUseCase, ctx context.Context) (*domain.TaxSettings, error) {
		return uc.UpdateSettings(ctx, in)
	})
}

func (h *Handler) ListSerialRanges(c *fiber.Ctx) error {
	return call(h, c, "Serial ranges retrieved", func(uc application.TaxUseCase, ctx context.Context) ([]domain.SerialRange, error) {
		return uc.ListSerialRanges(ctx)
	})
}

func (h *Handler) CreateSerialRange(c *fiber.Ctx) error {
	in, err := body[application.SerialRangeInput](c)
	if err != nil {
		return err
	}
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := uc.CreateSerialRange(h.ctx(c), in)
	if err != nil {
		return err
	}
	return response.Created(c, "Serial range created", v)
}

func (h *Handler) SetSerialRangeActive(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	in, err := body[struct {
		IsActive bool `json:"isActive"`
	}](c)
	if err != nil {
		return err
	}
	return call(h, c, "Serial range updated", func(uc application.TaxUseCase, ctx context.Context) (*domain.SerialRange, error) {
		return uc.SetSerialRangeActive(ctx, id, in.IsActive)
	})
}

// Faktur Pajak

func (h *Handler) ListTaxInvoices(c *fiber.Ctx) error {
	f := domain.InvoiceFilter{Direction: c.Query("direction"), Status: c.Query("status"), From: c.Query("from"), To: c.Query("to"), Search: c.Query("search")}
	return call(h, c, "Faktur Pajak retrieved", func(uc application.TaxUseCase, ctx context.Context) ([]domain.TaxInvoice, error) {
		return uc.ListTaxInvoices(ctx, f)
	})
}

func (h *Handler) GetTaxInvoice(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	return call(h, c, "Faktur Pajak retrieved", func(uc application.TaxUseCase, ctx context.Context) (*domain.TaxInvoice, error) {
		return uc.GetTaxInvoice(ctx, id)
	})
}

func (h *Handler) CreateFromSalesInvoice(c *fiber.Ctx) error {
	in, err := body[application.FromSalesInput](c)
	if err != nil {
		return err
	}
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := uc.CreateFromSalesInvoice(h.ctx(c), in)
	if err != nil {
		return err
	}
	return response.Created(c, "Faktur Pajak Keluaran drafted", v)
}

func (h *Handler) CreateFromPurchaseInvoice(c *fiber.Ctx) error {
	in, err := body[application.FromPurchaseInput](c)
	if err != nil {
		return err
	}
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := uc.CreateFromPurchaseInvoice(h.ctx(c), in)
	if err != nil {
		return err
	}
	return response.Created(c, "Faktur Pajak Masukan drafted", v)
}

func (h *Handler) UpdateDraft(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	in, err := body[application.UpdateInput](c)
	if err != nil {
		return err
	}
	return call(h, c, "Faktur Pajak updated", func(uc application.TaxUseCase, ctx context.Context) (*domain.TaxInvoice, error) {
		return uc.UpdateDraft(ctx, id, in)
	})
}

func (h *Handler) Issue(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	return call(h, c, "Faktur Pajak issued", func(uc application.TaxUseCase, ctx context.Context) (*domain.TaxInvoice, error) {
		return uc.Issue(ctx, id)
	})
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
	return call(h, c, "Faktur Pajak cancelled", func(uc application.TaxUseCase, ctx context.Context) (*domain.TaxInvoice, error) {
		return uc.Cancel(ctx, id, in.Reason)
	})
}

func (h *Handler) Replace(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := uc.Replace(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.Created(c, "Replacement Faktur Pajak drafted", v)
}

func (h *Handler) SetTaxNumber(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	in, err := body[struct {
		TaxNumber string `json:"taxNumber"`
	}](c)
	if err != nil {
		return err
	}
	return call(h, c, "Nomor Faktur Pajak saved", func(uc application.TaxUseCase, ctx context.Context) (*domain.TaxInvoice, error) {
		return uc.SetTaxNumber(ctx, id, in.TaxNumber)
	})
}

// Reports

func (h *Handler) SalesBook(c *fiber.Ctx) error {
	return call(h, c, "Sales book retrieved", func(uc application.TaxUseCase, ctx context.Context) (*application.SalesBookDTO, error) {
		return uc.SalesBook(ctx, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) VATSummary(c *fiber.Ctx) error {
	return call(h, c, "VAT summary retrieved", func(uc application.TaxUseCase, ctx context.Context) (*application.VATSummaryDTO, error) {
		return uc.VATSummary(ctx, c.Query("from"), c.Query("to"))
	})
}

func (h *Handler) CoretaxExport(c *fiber.Ctx) error {
	return call(h, c, "Coretax export prepared", func(uc application.TaxUseCase, ctx context.Context) (*application.CoretaxExportDTO, error) {
		return uc.CoretaxExport(ctx, c.Query("from"), c.Query("to"))
	})
}
