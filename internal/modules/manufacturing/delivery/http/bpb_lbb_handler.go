package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ApproveRequisitionByWarehouse handles POST /api/v1/manufacturing/requisitions/:id/acc-gudang
func (h *Handler) ApproveRequisitionByWarehouse(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}

	var dto application.ApproveRequisitionByWarehouseDTO
	_ = c.BodyParser(&dto)

	if dto.WarehouseApprovedBy == "" {
		claims := middleware.CurrentUser(c)
		if claims != nil && claims.Email != "" {
			dto.WarehouseApprovedBy = claims.Email
		} else {
			dto.WarehouseApprovedBy = "Manager Gudang"
		}
	}

	approved, err := uc.ApproveRequisitionByWarehouse(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Permintaan bahan baku berhasil di-ACC Gudang dan stok telah dipotong", approved)
}

// EnterProductionFromRequisition handles POST /api/v1/manufacturing/requisitions/:id/enter-production
func (h *Handler) EnterProductionFromRequisition(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}

	var dto application.EnterProductionDTO
	_ = c.BodyParser(&dto)

	if dto.EnteredProductionBy == "" {
		claims := middleware.CurrentUser(c)
		if claims != nil && claims.Email != "" {
			dto.EnteredProductionBy = claims.Email
		} else {
			dto.EnteredProductionBy = "Produksi"
		}
	}

	res, err := uc.EnterProductionFromRequisition(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Bahan baku berhasil diserahkan ke lini produksi dan ketertelusuran dicatat", res)
}

// RejectRequisition handles POST /api/v1/manufacturing/requisitions/:id/reject
func (h *Handler) RejectRequisition(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}

	var dto application.RejectRequisitionDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload penolakan tidak valid")
	}

	rejected, err := uc.RejectRequisition(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Permintaan bahan baku berhasil ditolak dan reservasi stok dilepas", rejected)
}

// ─────────────────────────────────────────────────────────────────────────────
// LBB (Laporan Permintaan Bahan Baku — GDG-013-LBB) Handlers
// ─────────────────────────────────────────────────────────────────────────────

// ListLBBReports handles GET /api/v1/manufacturing/lbb
func (h *Handler) ListLBBReports(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var reqID *uuid.UUID
	if reqIDStr := c.Query("requisitionId"); reqIDStr != "" {
		if parsed, err := uuid.Parse(reqIDStr); err == nil {
			reqID = &parsed
		}
	}
	variantType := c.Query("variantType")
	status := c.Query("status")

	list, err := uc.ListLBBReports(h.ctx(c), reqID, variantType, status)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar Laporan Permintaan Bahan Baku (LBB) berhasil dimuat", list)
}

// GetLBBReportByID handles GET /api/v1/manufacturing/lbb/:id
func (h *Handler) GetLBBReportByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}

	rep, err := uc.GetLBBReportByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Detail Laporan Permintaan Bahan Baku (LBB) berhasil dimuat", rep)
}

// CreateLBBReport handles POST /api/v1/manufacturing/lbb
func (h *Handler) CreateLBBReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateLBBReportDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload LBB tidak valid")
	}

	claims := middleware.CurrentUser(c)
	if dto.PreparedBy == "" && claims != nil {
		dto.PreparedBy = claims.Email
	}
	if dto.PreparedDate == "" {
		dto.PreparedDate = dto.ReportDate
	}

	created, err := uc.CreateLBBReport(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Laporan Permintaan Bahan Baku (LBB) berhasil dibuat", created)
}

// UpdateLBBReport handles PUT /api/v1/manufacturing/lbb/:id
func (h *Handler) UpdateLBBReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}

	var dto application.UpdateLBBReportDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update LBB tidak valid")
	}

	updated, err := uc.UpdateLBBReport(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Laporan Permintaan Bahan Baku (LBB) berhasil diperbarui", updated)
}

// DeleteLBBReport handles DELETE /api/v1/manufacturing/lbb/:id
func (h *Handler) DeleteLBBReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}

	if err := uc.DeleteLBBReport(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Laporan Permintaan Bahan Baku (LBB) berhasil dihapus", nil)
}
