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

// ─────────────────────────────────────────────────────────────────────────────
// Personal Hygiene (Doc CPH) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListHygieneChecks(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	date := c.Query("date")
	line := c.Query("line")
	checks, err := uc.ListHygieneChecks(c.Context(), date, line)
	if err != nil {
		return err
	}
	return response.OK(c, "Data personal hygiene berhasil dimuat", checks)
}

func (h *Handler) CreateHygieneCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateHygieneCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload pemeriksaan hygiene tidak valid")
	}
	if dto.InspectorName == "" {
		claims := middleware.CurrentUser(c)
		if claims != nil {
			dto.InspectorName = claims.Email
		} else {
			dto.InspectorName = "Adin (QC)"
		}
	}
	check, err := uc.CreateHygieneCheck(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Catatan personal hygiene berhasil disimpan", check)
}

func (h *Handler) DeleteHygieneCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteHygieneCheck(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Catatan personal hygiene berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Pre-Op Sanitation (Doc SSO) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListPreOpChecks(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	date := c.Query("date")
	room := c.Query("room")
	checks, err := uc.ListPreOpChecks(c.Context(), date, room)
	if err != nil {
		return err
	}
	return response.OK(c, "Data sarana pra-operasional berhasil dimuat", checks)
}

func (h *Handler) CreatePreOpCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreatePreOpCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload pemeriksaan sarana operasional tidak valid")
	}
	if dto.InspectorName == "" {
		claims := middleware.CurrentUser(c)
		if claims != nil {
			dto.InspectorName = claims.Email
		} else {
			dto.InspectorName = "Adin (QC)"
		}
	}
	check, err := uc.CreatePreOpCheck(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Catatan sarana pra-operasional berhasil disimpan", check)
}

func (h *Handler) DeletePreOpCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeletePreOpCheck(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Catatan sarana pra-operasional berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Packaging Quality Check (PBK — QC-003) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListPackagingChecks(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	lbmNo := c.Query("lbmNo")
	checks, err := uc.ListPackagingChecks(c.Context(), month, lbmNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Data pemeriksaan mutu bahan kemas berhasil dimuat", checks)
}

func (h *Handler) GetPackagingCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	check, err := uc.GetPackagingCheckByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Detail pemeriksaan kemasan berhasil dimuat", check)
}

func (h *Handler) CreatePackagingCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreatePackagingCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload pemeriksaan kemasan tidak valid")
	}
	if dto.QCName == "" {
		claims := middleware.CurrentUser(c)
		if claims != nil {
			dto.QCName = claims.Email
		} else {
			dto.QCName = "Staff QC"
		}
	}
	check, err := uc.CreatePackagingCheck(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Pemeriksaan mutu bahan kemas berhasil disimpan", check)
}

func (h *Handler) UpdatePackagingCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdatePackagingCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update kemasan tidak valid")
	}
	check, err := uc.UpdatePackagingCheck(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan mutu bahan kemas berhasil diperbarui", check)
}

func (h *Handler) DeletePackagingCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeletePackagingCheck(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan kemasan berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Supplier Armada Check (CAS — QC-007) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListArmadaChecks(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	date := c.Query("date")
	lbmNo := c.Query("lbmNo")
	checks, err := uc.ListArmadaChecks(c.Context(), date, lbmNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Data checklist armada supplier berhasil dimuat", checks)
}

func (h *Handler) GetArmadaCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	check, err := uc.GetArmadaCheckByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Detail checklist armada berhasil dimuat", check)
}

func (h *Handler) CreateArmadaCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateArmadaCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload checklist armada tidak valid")
	}
	if dto.QCName == "" {
		claims := middleware.CurrentUser(c)
		if claims != nil {
			dto.QCName = claims.Email
		} else {
			dto.QCName = "Staff QC"
		}
	}
	check, err := uc.CreateArmadaCheck(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Checklist armada supplier berhasil disimpan", check)
}

func (h *Handler) UpdateArmadaCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateArmadaCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update checklist armada tidak valid")
	}
	check, err := uc.UpdateArmadaCheck(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Checklist armada supplier berhasil diperbarui", check)
}

func (h *Handler) DeleteArmadaCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteArmadaCheck(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Checklist armada berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Raw Material Incoming Check (PBD — QC-002-PBD) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListIncomingChecks(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	date := c.Query("date")
	lbmNo := c.Query("lbmNo")
	checks, err := uc.ListIncomingChecks(c.Context(), month, date, lbmNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Data pemeriksaan kualitas barang datang berhasil dimuat", checks)
}

func (h *Handler) GetIncomingCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	check, err := uc.GetIncomingCheckByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Detail pemeriksaan bahan datang berhasil dimuat", check)
}

func (h *Handler) CreateIncomingCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateIncomingCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload pemeriksaan bahan datang tidak valid")
	}
	if dto.QCName == "" {
		claims := middleware.CurrentUser(c)
		if claims != nil {
			dto.QCName = claims.Email
		} else {
			dto.QCName = "Staff QC"
		}
	}
	check, err := uc.CreateIncomingCheck(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Pemeriksaan bahan datang berhasil disimpan", check)
}

func (h *Handler) UpdateIncomingCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateIncomingCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update pemeriksaan bahan datang tidak valid")
	}
	check, err := uc.UpdateIncomingCheck(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan bahan datang berhasil diperbarui", check)
}

func (h *Handler) DeleteIncomingCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteIncomingCheck(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan bahan datang berhasil dihapus", fiber.Map{"id": id})
}

// TraceLot follows one lot number: receipt -> QC -> production genealogy -> finished
// batches -> their QC inspections (the basis of the COI), plus the supplier's certificates.
func (h *Handler) TraceLot(c *fiber.Ctx) error {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	res, err := application.TraceLot(c.UserContext(), infrastructure.NewTraceReader(tenantDB), c.Params("lot"))
	if err != nil {
		return err
	}
	return response.OK(c, "Lot trace retrieved successfully", res)
}

// ─────────────────────────────────────────────────────────────────────────────
// Sieving Monitoring (MNA — Form QC-009-MNA.01) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListSievingLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	category := c.Query("category")
	machineNo := c.Query("machineNo")
	batchNo := c.Query("batchNo")
	list, err := uc.ListSievingLogs(c.Context(), month, category, machineNo, batchNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring proses ayak berhasil diambil", list)
}

func (h *Handler) GetSievingLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetSievingLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring proses ayak berhasil diambil", item)
}

func (h *Handler) CreateSievingLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateSievingLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring proses ayak tidak valid")
	}
	item, err := uc.CreateSievingLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Monitoring proses ayak berhasil disimpan", item)
}

func (h *Handler) UpdateSievingLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateSievingLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring proses ayak tidak valid")
	}
	item, err := uc.UpdateSievingLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Monitoring proses ayak berhasil diperbarui", item)
}

func (h *Handler) DeleteSievingLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteSievingLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Monitoring proses ayak berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Magnet Trap (CMT — Form QC-038-CMT) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListMagnetTrapLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	room := c.Query("room")
	machineNo := c.Query("machineNo")
	list, err := uc.ListMagnetTrapLogs(c.Context(), month, room, machineNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar pemakaian magnet trap berhasil diambil", list)
}

func (h *Handler) GetMagnetTrapLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetMagnetTrapLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data pemakaian magnet trap berhasil diambil", item)
}

func (h *Handler) CreateMagnetTrapLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateMagnetTrapLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload catatan magnet trap tidak valid")
	}
	item, err := uc.CreateMagnetTrapLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Catatan pemakaian magnet trap berhasil disimpan", item)
}

func (h *Handler) UpdateMagnetTrapLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateMagnetTrapLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update catatan magnet trap tidak valid")
	}
	item, err := uc.UpdateMagnetTrapLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Catatan pemakaian magnet trap berhasil diperbarui", item)
}

func (h *Handler) DeleteMagnetTrapLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteMagnetTrapLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Catatan magnet trap berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Mesh Bolt Monitoring (MNM — Form QC-010-MNM.01) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListMeshBoltLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	room := c.Query("room")
	machineNo := c.Query("machineNo")
	list, err := uc.ListMeshBoltLogs(c.Context(), month, room, machineNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring mur & baut berhasil diambil", list)
}

func (h *Handler) GetMeshBoltLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetMeshBoltLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring mur & baut berhasil diambil", item)
}

func (h *Handler) CreateMeshBoltLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateMeshBoltLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring mur & baut tidak valid")
	}
	item, err := uc.CreateMeshBoltLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Monitoring mur & baut berhasil disimpan", item)
}

func (h *Handler) UpdateMeshBoltLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateMeshBoltLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring mur & baut tidak valid")
	}
	item, err := uc.UpdateMeshBoltLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Monitoring mur & baut berhasil diperbarui", item)
}

func (h *Handler) DeleteMeshBoltLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteMeshBoltLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Monitoring mur & baut berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Oven Monitoring (MNO — Form QC-012-MNO.05) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListOvenLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	ovenNo := c.Query("ovenNo")
	batchNo := c.Query("batchNo")
	list, err := uc.ListOvenLogs(c.Context(), month, ovenNo, batchNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring proses oven berhasil diambil", list)
}

func (h *Handler) GetOvenLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetOvenLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring oven berhasil diambil", item)
}

func (h *Handler) CreateOvenLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateOvenLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring proses oven tidak valid")
	}
	item, err := uc.CreateOvenLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Monitoring proses oven berhasil disimpan", item)
}

func (h *Handler) UpdateOvenLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateOvenLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring oven tidak valid")
	}
	item, err := uc.UpdateOvenLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Monitoring proses oven berhasil diperbarui", item)
}

func (h *Handler) DeleteOvenLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteOvenLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Monitoring proses oven berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Filter Mesh Monitoring (MNP - Gula Cair & MMV - VCO) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListFilterMeshLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	category := c.Query("category")
	location := c.Query("location")
	list, err := uc.ListFilterMeshLogs(c.Context(), month, category, location)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring proses penyaringan berhasil diambil", list)
}

func (h *Handler) GetFilterMeshLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetFilterMeshLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring proses penyaringan berhasil diambil", item)
}

func (h *Handler) CreateFilterMeshLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateFilterMeshLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring proses penyaringan tidak valid")
	}
	item, err := uc.CreateFilterMeshLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Monitoring proses penyaringan berhasil disimpan", item)
}

func (h *Handler) UpdateFilterMeshLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateFilterMeshLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring proses penyaringan tidak valid")
	}
	item, err := uc.UpdateFilterMeshLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Monitoring proses penyaringan berhasil diperbarui", item)
}

func (h *Handler) DeleteFilterMeshLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteFilterMeshLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Monitoring proses penyaringan berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// Brontol Blade Monitoring (MPB — Form QC-011-MPB.01) Handlers
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListBrontolBladeLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	productCode := c.Query("productCode")
	list, err := uc.ListBrontolBladeLogs(c.Context(), month, productCode)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring bilah pisau berhasil diambil", list)
}

func (h *Handler) GetBrontolBladeLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetBrontolBladeLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring bilah pisau berhasil diambil", item)
}

func (h *Handler) CreateBrontolBladeLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateBrontolBladeLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring bilah pisau tidak valid")
	}
	item, err := uc.CreateBrontolBladeLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Monitoring bilah pisau berhasil disimpan", item)
}

func (h *Handler) UpdateBrontolBladeLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateBrontolBladeLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring bilah pisau tidak valid")
	}
	item, err := uc.UpdateBrontolBladeLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Monitoring bilah pisau berhasil diperbarui", item)
}

func (h *Handler) DeleteBrontolBladeLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteBrontolBladeLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Monitoring bilah pisau berhasil dihapus", fiber.Map{"id": id})
}


