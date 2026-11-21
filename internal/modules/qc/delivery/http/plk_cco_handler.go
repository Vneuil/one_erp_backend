package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/qc/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

// ─────────────────────────────────────────────────────────────────────────────
// PLK: Packaging Label Check Handlers (Form QC-018-PLK)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListPackagingLabelChecks(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	batchNo := c.Query("batchNo")
	decision := c.Query("decision")
	list, err := uc.ListPackagingLabelChecks(c.Context(), month, batchNo, decision)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar pemeriksaan label kemasan PLK berhasil diambil", list)
}

func (h *Handler) GetPackagingLabelCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetPackagingLabelCheckByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data pemeriksaan label kemasan PLK berhasil diambil", item)
}

func (h *Handler) CreatePackagingLabelCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreatePackagingLabelCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Format payload tidak valid: " + err.Error())
	}
	item, err := uc.CreatePackagingLabelCheck(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Pemeriksaan label kemasan PLK berhasil disimpan", item)
}

func (h *Handler) UpdatePackagingLabelCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdatePackagingLabelCheckDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Format payload tidak valid: " + err.Error())
	}
	item, err := uc.UpdatePackagingLabelCheck(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan label kemasan PLK berhasil diperbarui", item)
}

func (h *Handler) DeletePackagingLabelCheck(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeletePackagingLabelCheck(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Pemeriksaan label kemasan PLK berhasil dihapus", fiber.Map{"id": id})
}

// ─────────────────────────────────────────────────────────────────────────────
// CCO: Container Checklist Handlers (Form QC-008-CCO.02)
// ─────────────────────────────────────────────────────────────────────────────

func (h *Handler) ListContainerChecklists(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	month := c.Query("month")
	containerNo := c.Query("containerNo")
	destination := c.Query("destination")
	decision := c.Query("decision")
	list, err := uc.ListContainerChecklists(c.Context(), month, containerNo, destination, decision)
	if err != nil {
		return err
	}
	return response.OK(c, "Daftar checklist kontainer CCO berhasil diambil", list)
}

func (h *Handler) GetContainerChecklist(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetContainerChecklistByID(c.Context(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Data checklist kontainer CCO berhasil diambil", item)
}

func (h *Handler) CreateContainerChecklist(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateContainerChecklistDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Format payload tidak valid: " + err.Error())
	}
	item, err := uc.CreateContainerChecklist(c.Context(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Checklist kontainer CCO berhasil disimpan", item)
}

func (h *Handler) UpdateContainerChecklist(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateContainerChecklistDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Format payload tidak valid: " + err.Error())
	}
	item, err := uc.UpdateContainerChecklist(c.Context(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Checklist kontainer CCO berhasil diperbarui", item)
}

func (h *Handler) DeleteContainerChecklist(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteContainerChecklist(c.Context(), id); err != nil {
		return err
	}
	return response.OK(c, "Checklist kontainer CCO berhasil dihapus", fiber.Map{"id": id})
}
