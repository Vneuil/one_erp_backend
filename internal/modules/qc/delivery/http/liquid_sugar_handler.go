package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/qc/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

// ─────────────────────────────────────────────────────────────────────────────
// Flow Gula Cair Status
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) GetLiquidSugarFlowStatus(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	batchNo := c.Query("batchNo")
	status, err := uc.GetLiquidSugarFlowStatus(c.Context(), month, batchNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Status alur QC gula cair berhasil diambil", status)
}

// ─────────────────────────────────────────────────────────────────────────────
// MSS: Sterilization Temperature Monitoring (Form QC-057-MSS)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListSterilizationTempLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	stage := c.Query("stage")
	batchNo := c.Query("batchNo")
	list, err := uc.ListSterilizationTempLogs(c.Context(), month, stage, batchNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring suhu sterilisasi MSS berhasil diambil", list)
}

func (h *Handler) GetSterilizationTempLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetSterilizationTempLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring suhu sterilisasi MSS berhasil diambil", item)
}

func (h *Handler) CreateSterilizationTempLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateSterilizationTempLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring suhu sterilisasi tidak valid")
	}
	item, err := uc.CreateSterilizationTempLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Monitoring suhu sterilisasi MSS berhasil disimpan", item)
}

func (h *Handler) UpdateSterilizationTempLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateSterilizationTempLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring suhu sterilisasi tidak valid")
	}
	item, err := uc.UpdateSterilizationTempLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Monitoring suhu sterilisasi MSS berhasil diperbarui", item)
}

func (h *Handler) DeleteSterilizationTempLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteSterilizationTempLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Monitoring suhu sterilisasi MSS berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// PVK: Liquid Packaging Visual Check (Form QC-041-PVK)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListLiquidPackagingVisualChecks(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	batchNo := c.Query("batchNo")
	list, err := uc.ListLiquidPackagingVisualChecks(c.Context(), month, batchNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar pengecekan visual kemasan PVK berhasil diambil", list)
}

func (h *Handler) GetLiquidPackagingVisualCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetLiquidPackagingVisualCheckByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data pengecekan visual kemasan PVK berhasil diambil", item)
}

func (h *Handler) CreateLiquidPackagingVisualCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateLiquidPackagingVisualCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload pengecekan visual kemasan PVK tidak valid")
	}
	item, err := uc.CreateLiquidPackagingVisualCheck(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Pengecekan visual kemasan PVK berhasil disimpan", item)
}

func (h *Handler) UpdateLiquidPackagingVisualCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateLiquidPackagingVisualCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update pengecekan visual kemasan PVK tidak valid")
	}
	item, err := uc.UpdateLiquidPackagingVisualCheck(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Pengecekan visual kemasan PVK berhasil diperbarui", item)
}

func (h *Handler) DeleteLiquidPackagingVisualCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteLiquidPackagingVisualCheck(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Pengecekan visual kemasan PVK berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// MNC: Liquid Sugar Processing Log (Form QC-014-MNC)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListLiquidSugarProcessingLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	batchNo := c.Query("batchNo")
	productCode := c.Query("productCode")
	list, err := uc.ListLiquidSugarProcessingLogs(c.Context(), month, batchNo, productCode)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring pengolahan gula cair MNC berhasil diambil", list)
}

func (h *Handler) GetLiquidSugarProcessingLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetLiquidSugarProcessingLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring pengolahan gula cair MNC berhasil diambil", item)
}

func (h *Handler) CreateLiquidSugarProcessingLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateLiquidSugarProcessingLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring pengolahan gula cair MNC tidak valid")
	}
	item, err := uc.CreateLiquidSugarProcessingLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Monitoring pengolahan gula cair MNC berhasil disimpan", item)
}

func (h *Handler) UpdateLiquidSugarProcessingLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateLiquidSugarProcessingLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring pengolahan gula cair MNC tidak valid")
	}
	item, err := uc.UpdateLiquidSugarProcessingLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Monitoring pengolahan gula cair MNC berhasil diperbarui", item)
}

func (h *Handler) DeleteLiquidSugarProcessingLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteLiquidSugarProcessingLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Monitoring pengolahan gula cair MNC berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// PJC: Liquid Sugar Finished Goods Inspection (Form QC-015-PJC)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListLiquidSugarFinishedGoodsLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	batchNo := c.Query("batchNo")
	decision := c.Query("decision")
	list, err := uc.ListLiquidSugarFinishedGoodsLogs(c.Context(), month, batchNo, decision)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar pemeriksaan produk jadi gula cair PJC berhasil diambil", list)
}

func (h *Handler) GetLiquidSugarFinishedGoodsLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetLiquidSugarFinishedGoodsLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data pemeriksaan produk jadi gula cair PJC berhasil diambil", item)
}

func (h *Handler) CreateLiquidSugarFinishedGoodsLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateLiquidSugarFinishedGoodsLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload pemeriksaan produk jadi gula cair PJC tidak valid")
	}
	item, err := uc.CreateLiquidSugarFinishedGoodsLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Pemeriksaan produk jadi gula cair PJC berhasil disimpan", item)
}

func (h *Handler) UpdateLiquidSugarFinishedGoodsLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateLiquidSugarFinishedGoodsLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update pemeriksaan produk jadi gula cair PJC tidak valid")
	}
	item, err := uc.UpdateLiquidSugarFinishedGoodsLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan produk jadi gula cair PJC berhasil diperbarui", item)
}

func (h *Handler) DeleteLiquidSugarFinishedGoodsLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteLiquidSugarFinishedGoodsLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan produk jadi gula cair PJC berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// VHC: Liquid Sugar Wash Verification (Form QC-055-VHC)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListLiquidSugarWashVerifications(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	batchNo := c.Query("batchNo")
	list, err := uc.ListLiquidSugarWashVerifications(c.Context(), month, batchNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar verifikasi hasil cuci VHC berhasil diambil", list)
}

func (h *Handler) GetLiquidSugarWashVerification(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetLiquidSugarWashVerificationByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data verifikasi hasil cuci VHC berhasil diambil", item)
}

func (h *Handler) CreateLiquidSugarWashVerification(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateLiquidSugarWashVerificationDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload verifikasi hasil cuci VHC tidak valid")
	}
	item, err := uc.CreateLiquidSugarWashVerification(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Verifikasi hasil cuci VHC berhasil disimpan", item)
}

func (h *Handler) UpdateLiquidSugarWashVerification(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateLiquidSugarWashVerificationDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update verifikasi hasil cuci VHC tidak valid")
	}
	item, err := uc.UpdateLiquidSugarWashVerification(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Verifikasi hasil cuci VHC berhasil diperbarui", item)
}

func (h *Handler) DeleteLiquidSugarWashVerification(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteLiquidSugarWashVerification(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Verifikasi hasil cuci VHC berhasil dihapus", fiber.Map{"id": id})
}
