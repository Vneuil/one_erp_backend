package http

import (
	"strconv"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/qc/application"
	"github.com/divinecoid/one-backend/internal/modules/qc/domain"
	"github.com/divinecoid/one-backend/internal/modules/qc/infrastructure"
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
		return uuid.Nil, apperrors.NewBadRequest("Format " + param + " tidak valid")
	}
	return id, nil
}

func (h *Handler) resolve(c *fiber.Ctx) (application.QCUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewQCRepository(tenantDB)
	return application.NewQCUseCase(repo), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Parameter Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListParameters(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var productID *uuid.UUID
	if prodStr := c.Query("productId"); prodStr != "" {
		if id, err := uuid.Parse(prodStr); err == nil {
			productID = &id
		}
	}
	processCode := c.Query("processCode")

	params, err := uc.ListParameters(c.Context(), productID, processCode)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar parameter QC berhasil dimuat", params)
}

func (h *Handler) GetParameter(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	param, err := uc.GetParameterByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Parameter QC berhasil dimuat", param)
}

func (h *Handler) CreateParameter(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateQCParameterDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload data tidak valid")
	}
	created, err := uc.CreateParameter(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Parameter QC berhasil dibuat", created)
}

func (h *Handler) UpdateParameter(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateQCParameterDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload data tidak valid")
	}
	updated, err := uc.UpdateParameter(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Parameter QC berhasil diperbarui", updated)
}

func (h *Handler) DeleteParameter(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteParameter(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Parameter QC berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Inspection Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListInspections(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	filter := domain.ListInspectionFilter{
		BatchNo: c.Query("batchNo"),
		Page:    1,
		PerPage: 20,
	}

	if tt := c.Query("targetType"); tt != "" {
		targetType := domain.TargetType(tt)
		filter.TargetType = &targetType
	}
	if tid := c.Query("targetId"); tid != "" {
		if id, err := uuid.Parse(tid); err == nil {
			filter.TargetID = &id
		}
	}
	if poid := c.Query("productionOrderId"); poid != "" {
		if id, err := uuid.Parse(poid); err == nil {
			filter.ProductionOrderID = &id
		}
	}
	if pid := c.Query("productId"); pid != "" {
		if id, err := uuid.Parse(pid); err == nil {
			filter.ProductID = &id
		}
	}
	if st := c.Query("status"); st != "" {
		status := domain.InspectionStatus(st)
		filter.Status = &status
	}
	if sDate := c.Query("startDate"); sDate != "" {
		if t, err := time.Parse("2006-01-02", sDate); err == nil {
			filter.StartDate = &t
		}
	}
	if eDate := c.Query("endDate"); eDate != "" {
		if t, err := time.Parse("2006-01-02", eDate); err == nil {
			filter.EndDate = &t
		}
	}
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if perPageStr := c.Query("perPage"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 {
			filter.PerPage = pp
		}
	}

	items, total, err := uc.ListInspections(c.Context(), filter)
	if err != nil {
		return err
	}

	totalPages := int((total + int64(filter.PerPage) - 1) / int64(filter.PerPage))
	meta := types.PaginationMeta{
		CurrentPage: filter.Page,
		PerPage:     filter.PerPage,
		TotalItems:  total,
		TotalPages:  totalPages,
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Daftar inspeksi QC berhasil dimuat", items, meta)
}

func (h *Handler) GetInspection(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	insp, err := uc.GetInspectionByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Detail inspeksi QC berhasil dimuat", insp)
}

func (h *Handler) CreateInspection(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateQCInspectionDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload inspeksi tidak valid")
	}

	claims := middleware.CurrentUser(c)
	var inspectorID uuid.UUID
	var inspectorName string
	if claims != nil {
		inspectorID = claims.UserID
		inspectorName = claims.Email
	} else {
		inspectorID = uuid.New()
		inspectorName = "System"
	}

	created, err := uc.CreateInspection(c.Context(), inspectorID, inspectorName, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Hasil inspeksi QC berhasil dicatat", created)
}

func (h *Handler) VoidInspection(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.VoidQCInspectionDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload void tidak valid")
	}

	claims := middleware.CurrentUser(c)
	var actorID uuid.UUID
	if claims != nil {
		actorID = claims.UserID
	} else {
		actorID = uuid.New()
	}

	voided, err := uc.VoidInspection(c.Context(), id, actorID, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Inspeksi QC berhasil dibatalkan (void)", voided)
}

func (h *Handler) GetStats(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	stats, err := uc.GetStats(c.Context())
	if err != nil {
		return err
	}
	return response.OK(c, "Statistik QC berhasil dimuat", stats)
}
