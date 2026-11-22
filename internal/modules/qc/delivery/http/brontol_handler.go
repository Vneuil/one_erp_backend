package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/qc/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ─── Brontol Handlers (Form REP-013-KSB) ────────────────────────────────────

func (h *Handler) ListBrontolReports(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	month := c.Query("month")
	stage := c.Query("stage")
	action := c.Query("action")
	status := c.Query("status")
	search := c.Query("search")

	list, err := uc.ListBrontolReports(c.Context(), month, stage, action, status, search)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar laporan brontol berhasil dimuat", list)
}

func (h *Handler) CreateBrontolReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateBrontolReportDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload data laporan brontol tidak valid: " + err.Error())
	}

	item, err := uc.CreateBrontolReport(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Laporan brontol berhasil dibuat", item)
}

func (h *Handler) GetBrontolReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Format ID laporan brontol tidak valid")
	}

	item, err := uc.GetBrontolReportByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data laporan brontol berhasil dimuat", item)
}

func (h *Handler) UpdateBrontolReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Format ID laporan brontol tidak valid")
	}

	var dto application.UpdateBrontolReportDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update tidak valid: " + err.Error())
	}

	item, err := uc.UpdateBrontolReport(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Laporan brontol berhasil diperbarui", item)
}

func (h *Handler) DeleteBrontolReport(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Format ID laporan brontol tidak valid")
	}

	if err := uc.DeleteBrontolReport(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Laporan brontol berhasil dihapus", fiber.Map{"id": id})
}

func (h *Handler) GetBrontolStats(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	stats, err := uc.GetBrontolStats(c.Context())
	if err != nil {
		return err
	}
	return response.OK(c, "Statistik brontol berhasil dimuat", stats)
}

func (h *Handler) GetBrontolStockCard(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	month := c.Query("month")
	productCode := c.Query("productCode")

	cards, err := uc.GetBrontolStockCard(c.Context(), month, productCode)
	if err != nil {
		return err
	}
	return response.OK(c, "Kartu stok brontol berhasil dimuat", cards)
}

// ─── Laporan Retur Barang Reject Handlers ────────────────────────────────────

func (h *Handler) CreateRejectReturn(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	brontolID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Format ID laporan brontol tidak valid")
	}

	var dto application.CreateRejectReturnDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload laporan retur tidak valid: " + err.Error())
	}

	ret, err := uc.CreateRejectReturn(c.Context(), brontolID, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Laporan retur barang reject berhasil dibuat", ret)
}

func (h *Handler) ListRejectReturns(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	month := c.Query("month")
	status := c.Query("status")
	search := c.Query("search")

	list, err := uc.ListRejectReturns(c.Context(), month, status, search)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar laporan retur barang reject berhasil dimuat", list)
}

func (h *Handler) GetRejectReturn(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Format ID laporan retur tidak valid")
	}

	ret, err := uc.GetRejectReturnByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data laporan retur barang reject berhasil dimuat", ret)
}

func (h *Handler) UpdateRejectReturn(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Format ID laporan retur tidak valid")
	}

	var dto application.UpdateRejectReturnDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update retur tidak valid: " + err.Error())
	}

	ret, err := uc.UpdateRejectReturn(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Laporan retur barang reject berhasil diperbarui", ret)
}

func (h *Handler) DeleteRejectReturn(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Format ID laporan retur tidak valid")
	}

	if err := uc.DeleteRejectReturn(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Laporan retur barang reject berhasil dihapus", fiber.Map{"id": id})
}
