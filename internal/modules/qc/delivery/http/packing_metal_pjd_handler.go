package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/qc/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

// ─────────────────────────────────────────────────────────────────────────────
// MNJ: Needle Monitoring Log Handlers (Form QC-011-MNJ.01)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListNeedleLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	machineNo := c.Query("machineNo")
	list, err := uc.ListNeedleLogs(c.Context(), month, machineNo)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring jarum berhasil diambil", list)
}

func (h *Handler) GetNeedleLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetNeedleLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring jarum berhasil diambil", item)
}

func (h *Handler) CreateNeedleLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateNeedleLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring jarum tidak valid")
	}
	item, err := uc.CreateNeedleLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Catatan monitoring jarum berhasil disimpan", item)
}

func (h *Handler) UpdateNeedleLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateNeedleLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring jarum tidak valid")
	}
	item, err := uc.UpdateNeedleLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Catatan monitoring jarum berhasil diperbarui", item)
}

func (h *Handler) DeleteNeedleLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteNeedleLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Catatan monitoring jarum berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// MNG: Scissor Monitoring Log Handlers (Form QC-011-MNJ.01 Sheet MNG)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListScissorLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	location := c.Query("location")
	list, err := uc.ListScissorLogs(c.Context(), month, location)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar monitoring gunting berhasil diambil", list)
}

func (h *Handler) GetScissorLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetScissorLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data monitoring gunting berhasil diambil", item)
}

func (h *Handler) CreateScissorLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateScissorLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload monitoring gunting tidak valid")
	}
	item, err := uc.CreateScissorLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Catatan monitoring gunting berhasil disimpan", item)
}

func (h *Handler) UpdateScissorLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateScissorLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update monitoring gunting tidak valid")
	}
	item, err := uc.UpdateScissorLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Catatan monitoring gunting berhasil diperbarui", item)
}

func (h *Handler) DeleteScissorLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteScissorLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Catatan monitoring gunting berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// CPP: Metal Detector Cleaning Log Handlers (Form CPP)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListMetalCleaningLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	machineNo := c.Query("machineNo")
	location := c.Query("location")
	list, err := uc.ListMetalCleaningLogs(c.Context(), month, machineNo, location)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar pembersihan metal detector berhasil diambil", list)
}

func (h *Handler) GetMetalCleaningLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetMetalCleaningLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data pembersihan metal detector berhasil diambil", item)
}

func (h *Handler) CreateMetalCleaningLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateMetalCleaningLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload pembersihan metal detector tidak valid")
	}
	item, err := uc.CreateMetalCleaningLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Catatan pembersihan metal detector berhasil disimpan", item)
}

func (h *Handler) UpdateMetalCleaningLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateMetalCleaningLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update pembersihan metal detector tidak valid")
	}
	item, err := uc.UpdateMetalCleaningLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Catatan pembersihan metal detector berhasil diperbarui", item)
}

func (h *Handler) DeleteMetalCleaningLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteMetalCleaningLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Catatan pembersihan metal detector berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// VMD: Metal Detector Verification Log Handlers (Form QC-017-VMD)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListMetalVerificationLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	productType := c.Query("productType")
	location := c.Query("location")
	list, err := uc.ListMetalVerificationLogs(c.Context(), month, productType, location)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar verifikasi metal detector berhasil diambil", list)
}

func (h *Handler) GetMetalVerificationLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetMetalVerificationLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data verifikasi metal detector berhasil diambil", item)
}

func (h *Handler) CreateMetalVerificationLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateMetalVerificationLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload verifikasi metal detector tidak valid")
	}
	item, err := uc.CreateMetalVerificationLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Catatan verifikasi metal detector berhasil disimpan", item)
}

func (h *Handler) UpdateMetalVerificationLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateMetalVerificationLogDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update verifikasi metal detector tidak valid")
	}
	item, err := uc.UpdateMetalVerificationLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Catatan verifikasi metal detector berhasil diperbarui", item)
}

func (h *Handler) DeleteMetalVerificationLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteMetalVerificationLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Catatan verifikasi metal detector berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// PJD: Finished Goods Inspection Handlers (Form QC-004-PJD.05 Rev 05)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListFinishedGoodsLogs(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	productName := c.Query("productName")
	batchNo := c.Query("batchNo")
	decision := c.Query("decision")
	list, err := uc.ListFinishedGoodsLogs(c.Context(), month, productName, batchNo, decision)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar pemeriksaan produk jadi berhasil diambil", list)
}

func (h *Handler) GetFinishedGoodsLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetFinishedGoodsLogByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data pemeriksaan produk jadi berhasil diambil", item)
}

func (h *Handler) CreateFinishedGoodsLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateFinishedGoodsDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload pemeriksaan produk jadi tidak valid")
	}
	item, err := uc.CreateFinishedGoodsLog(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Pemeriksaan produk jadi berhasil disimpan", item)
}

func (h *Handler) UpdateFinishedGoodsLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateFinishedGoodsDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Payload update pemeriksaan produk jadi tidak valid")
	}
	item, err := uc.UpdateFinishedGoodsLog(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan produk jadi berhasil diperbarui", item)
}

func (h *Handler) DeleteFinishedGoodsLog(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteFinishedGoodsLog(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan produk jadi berhasil dihapus", fiber.Map{"id": id})
}
