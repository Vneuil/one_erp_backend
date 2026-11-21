package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/manufacturing/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// --- LHP (PRO-002-LHP) Handlers ---

// ListLHP handles GET /api/v1/manufacturing/lhp
func (h *Handler) ListLHP(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var orderID *uuid.UUID
	if oStr := c.Query("productionOrderId"); oStr != "" {
		if uid, parseErr := uuid.Parse(oStr); parseErr == nil {
			orderID = &uid
		}
	}
	batchNo := c.Query("batchNo")
	status := c.Query("status")

	reports, err := uc.ListLHP(h.ctx(c), orderID, batchNo, status)
	if err != nil {
		return err
	}
	return response.OK(c, "LHP reports retrieved", reports)
}

// GetLHP handles GET /api/v1/manufacturing/lhp/:id
func (h *Handler) GetLHP(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	rep, err := uc.GetLHP(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "LHP report retrieved", rep)
}

// CreateLHP handles POST /api/v1/manufacturing/lhp
func (h *Handler) CreateLHP(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var input application.CreateLHPInput
	if err := c.BodyParser(&input); err != nil {
		return apperrors.NewBadRequest("invalid payload: " + err.Error())
	}

	claims := middleware.CurrentUser(c)
	if input.ProductionPIC == "" && claims != nil {
		input.ProductionPIC = claims.Email
	}
	if input.AdminPIC == "" && claims != nil {
		input.AdminPIC = claims.Email
	}

	created, err := uc.CreateLHP(h.ctx(c), input)
	if err != nil {
		return err
	}
	return response.Created(c, "LHP report created successfully", created)
}

// UpdateLHP handles PUT /api/v1/manufacturing/lhp/:id
func (h *Handler) UpdateLHP(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var input application.UpdateLHPInput
	if err := c.BodyParser(&input); err != nil {
		return apperrors.NewBadRequest("invalid payload: " + err.Error())
	}
	updated, err := uc.UpdateLHP(h.ctx(c), id, input)
	if err != nil {
		return err
	}
	return response.OK(c, "LHP report updated successfully", updated)
}

// DeleteLHP handles DELETE /api/v1/manufacturing/lhp/:id
func (h *Handler) DeleteLHP(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteLHP(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "LHP report deleted successfully", fiber.Map{"id": id})
}

// GenerateLSTFromLHP handles POST /api/v1/manufacturing/lhp/:id/generate-lst
func (h *Handler) GenerateLSTFromLHP(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	userEmail := "Produksi"
	if claims := middleware.CurrentUser(c); claims != nil && claims.Email != "" {
		userEmail = claims.Email
	}
	lst, err := uc.GenerateLSTFromLHP(h.ctx(c), id, userEmail)
	if err != nil {
		return err
	}
	return response.Created(c, "LST report generated successfully from LHP", lst)
}

// --- LST (GDG-009-LST) Handlers ---

// ListLST handles GET /api/v1/manufacturing/lst
func (h *Handler) ListLST(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var lhpID *uuid.UUID
	if lStr := c.Query("lhpId"); lStr != "" {
		if uid, parseErr := uuid.Parse(lStr); parseErr == nil {
			lhpID = &uid
		}
	}
	batchNo := c.Query("batchNo")
	status := c.Query("status")

	reports, err := uc.ListLST(h.ctx(c), lhpID, batchNo, status)
	if err != nil {
		return err
	}
	return response.OK(c, "LST reports retrieved", reports)
}

// GetLST handles GET /api/v1/manufacturing/lst/:id
func (h *Handler) GetLST(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	rep, err := uc.GetLST(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "LST report retrieved", rep)
}

// CreateLST handles POST /api/v1/manufacturing/lst
func (h *Handler) CreateLST(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var input application.CreateLSTInput
	if err := c.BodyParser(&input); err != nil {
		return apperrors.NewBadRequest("invalid payload: " + err.Error())
	}

	claims := middleware.CurrentUser(c)
	if input.HandedOverBy == "" && claims != nil {
		input.HandedOverBy = claims.Email
	}

	created, err := uc.CreateLST(h.ctx(c), input)
	if err != nil {
		return err
	}
	return response.Created(c, "LST report created successfully", created)
}

// UpdateLST handles PUT /api/v1/manufacturing/lst/:id
func (h *Handler) UpdateLST(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var input application.UpdateLSTInput
	if err := c.BodyParser(&input); err != nil {
		return apperrors.NewBadRequest("invalid payload: " + err.Error())
	}
	updated, err := uc.UpdateLST(h.ctx(c), id, input)
	if err != nil {
		return err
	}
	return response.OK(c, "LST report updated successfully", updated)
}

// VerifyLSTByQC handles POST /api/v1/manufacturing/lst/:id/verify-qc
func (h *Handler) VerifyLSTByQC(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	qcName := "QC Verifier"
	if claims := middleware.CurrentUser(c); claims != nil && claims.Email != "" {
		qcName = claims.Email
	}
	verified, err := uc.VerifyLSTByQC(h.ctx(c), id, qcName)
	if err != nil {
		return err
	}
	return response.OK(c, "LST report verified by QC", verified)
}

// AcceptLSTByWarehouse handles POST /api/v1/manufacturing/lst/:id/accept-warehouse
func (h *Handler) AcceptLSTByWarehouse(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	whName := "Admin Gudang"
	if claims := middleware.CurrentUser(c); claims != nil && claims.Email != "" {
		whName = claims.Email
	}
	accepted, err := uc.AcceptLSTByWarehouse(h.ctx(c), id, whName)
	if err != nil {
		return err
	}
	return response.OK(c, "LST report accepted by Warehouse (WIP updated)", accepted)
}

// DeleteLST handles DELETE /api/v1/manufacturing/lst/:id
func (h *Handler) DeleteLST(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteLST(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "LST report deleted successfully", fiber.Map{"id": id})
}
