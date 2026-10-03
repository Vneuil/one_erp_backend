package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
	inventoryApp "github.com/divinecoid/one-backend/internal/modules/inventory/application"
	inventoryInfra "github.com/divinecoid/one-backend/internal/modules/inventory/infrastructure"
	loyaltyApp "github.com/divinecoid/one-backend/internal/modules/loyalty/application"
	loyaltyInfra "github.com/divinecoid/one-backend/internal/modules/loyalty/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/pos/application"
	"github.com/divinecoid/one-backend/internal/modules/pos/infrastructure"
	productInfra "github.com/divinecoid/one-backend/internal/modules/product/infrastructure"
	salesInfra "github.com/divinecoid/one-backend/internal/modules/sales/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// resolve builds a use-case bound to the caller's own tenant database. The
// schema is migrated and seeded once, at tenant-provision time (see
// module.go's registration with foundation/tenant.RegisterSchema).
func (h *Handler) resolve(c *fiber.Ctx) (application.POSUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewPOSRepository(tenantDB)
	salesRepo := salesInfra.NewSalesRepository(tenantDB)
	loyaltyRepo := loyaltyInfra.NewLoyaltyRepository(tenantDB)
	loyaltyUC := loyaltyApp.NewLoyaltyUseCase(loyaltyRepo)
	inventoryRepo := inventoryInfra.NewInventoryRepository(tenantDB)
	productRepo := productInfra.NewProductRepository(tenantDB)
	inventoryUC := inventoryApp.NewInventoryUseCase(inventoryRepo, productRepo)
	ledger := financeApp.NewLedgerPoster(financeInfra.NewFinanceRepository(tenantDB))
	return application.NewPOSUseCase(repo, salesRepo, loyaltyUC, inventoryUC, application.WithLedger(ledger), application.WithProducts(productRepo)), nil
}

func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) Checkout(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CheckoutDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	tx, err := uc.Checkout(h.ctx(c), dto)
	if err != nil {
		return err
	}

	return response.Created(c, "POS transaction completed", tx)
}

func (h *Handler) GetByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return apperrors.NewBadRequest("Invalid transaction ID format")
	}

	tx, err := uc.GetByID(h.ctx(c), id)
	if err != nil {
		return err
	}

	return response.OK(c, "Transaction retrieved successfully", tx)
}

func (h *Handler) List(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}

	transactions, meta, err := uc.List(h.ctx(c), query)
	if err != nil {
		return err
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "POS transactions retrieved successfully", transactions, meta)
}

func txID(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid transaction ID format")
	}
	return id, nil
}

func (h *Handler) Void(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := txID(c)
	if err != nil {
		return err
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	tx, err := uc.Void(h.ctx(c), id, in.Reason)
	if err != nil {
		return err
	}
	return response.OK(c, "Transaction voided", tx)
}

func (h *Handler) Refund(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := txID(c)
	if err != nil {
		return err
	}
	var in application.RefundDTO
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	tx, err := uc.Refund(h.ctx(c), id, in)
	if err != nil {
		return err
	}
	return response.OK(c, "Refund recorded", tx)
}

func (h *Handler) ListRefunds(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := txID(c)
	if err != nil {
		return err
	}
	out, err := uc.ListRefunds(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Refunds retrieved", out)
}

func (h *Handler) GetSettings(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	s, err := uc.GetSettings(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "POS settings retrieved", s)
}

func (h *Handler) UpdateSettings(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in application.UpdateSettingsDTO
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	s, err := uc.UpdateSettings(h.ctx(c), in)
	if err != nil {
		return err
	}
	return response.OK(c, "POS settings saved", s)
}

func (h *Handler) SalesReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	rep, err := uc.SalesReport(h.ctx(c), c.Query("from"), c.Query("to"), c.Query("outlet"))
	if err != nil {
		return err
	}
	return response.OK(c, "POS sales report retrieved", rep)
}
