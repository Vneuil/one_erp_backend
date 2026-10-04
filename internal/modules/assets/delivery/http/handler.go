package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/assets/application"
	"github.com/divinecoid/one-backend/internal/modules/assets/infrastructure"
	financeApp "github.com/divinecoid/one-backend/internal/modules/finance/application"
	financeInfra "github.com/divinecoid/one-backend/internal/modules/finance/infrastructure"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.AssetsUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewAssetsRepository(tenantDB)
	ledger := financeApp.NewLedgerPoster(financeInfra.NewFinanceRepository(tenantDB))
	return application.NewAssetsUseCase(repo, application.WithLedger(ledger)), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateAsset(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateAssetDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	a, err := uc.CreateAsset(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Asset created successfully", a)
}

func (h *Handler) GetAssetByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	a, err := uc.GetAssetByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Asset retrieved successfully", a)
}

func (h *Handler) ListAssets(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListAssets(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Assets retrieved successfully", items, meta)
}

func (h *Handler) UpdateAsset(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateAssetDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	a, err := uc.UpdateAsset(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Asset updated successfully", a)
}

func (h *Handler) UpdateAssetStatus(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateAssetStatusDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	a, err := uc.UpdateAssetStatus(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Asset status updated successfully", a)
}

func (h *Handler) RecalculateDepreciation(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	a, err := uc.RecalculateDepreciation(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Depreciation recalculated successfully", a)
}

func (h *Handler) RecalculateAllDepreciation(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	items, err := uc.RecalculateAllDepreciation(h.ctx(c))
	if err != nil {
		return err
	}
	return response.OK(c, "Depreciation recalculated for all assets successfully", items)
}

func (h *Handler) PostDepreciation(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		Period  string `json:"period"`
		CatchUp bool   `json:"catchUp"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	res, err := uc.PostDepreciation(h.ctx(c), in.Period, in.CatchUp)
	if err != nil {
		return err
	}
	return response.OK(c, "Depreciation posted to the ledger", res)
}

func (h *Handler) DepreciationReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	res, err := uc.DepreciationReport(h.ctx(c), c.Query("period"))
	if err != nil {
		return err
	}
	return response.OK(c, "Depreciation report retrieved successfully", res)
}
