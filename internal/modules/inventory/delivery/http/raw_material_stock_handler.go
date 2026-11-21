package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/inventory/application"
	"github.com/divinecoid/one-backend/internal/modules/inventory/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
)

// Raw material warehouse stock (Form Stok Gudang Bahan Baku — GDG-002-SBB)

func (h *Handler) GetLBMInboundContext(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	grID, err := parseID(c, "goodsReceiptId")
	if err != nil {
		return err
	}
	res, err := uc.GetLBMInboundContext(h.ctx(c), grID)
	if err != nil {
		return err
	}
	return response.OK(c, "LBM inbound context retrieved", res)
}

func (h *Handler) GetRawMaterialStockSummary(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	wid, err := optionalID(c, "warehouseId")
	if err != nil {
		return err
	}
	res, err := uc.GetRawMaterialStockSummary(h.ctx(c), wid)
	if err != nil {
		return err
	}
	return response.OK(c, "Raw material stock summary retrieved", res)
}

func (h *Handler) ListAvailableRawMaterialLots(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	pid, err := optionalID(c, "productId")
	if err != nil {
		return err
	}
	wid, err := optionalID(c, "warehouseId")
	if err != nil {
		return err
	}
	items, err := uc.ListAvailableRawMaterialLots(h.ctx(c), pid, wid)
	if err != nil {
		return err
	}
	return response.OK(c, "Available raw material lots retrieved", items)
}

func (h *Handler) ListRawMaterialStockEntries(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	wid, err := optionalID(c, "warehouseId")
	if err != nil {
		return err
	}
	pid, err := optionalID(c, "productId")
	if err != nil {
		return err
	}
	grID, err := optionalID(c, "goodsReceiptId")
	if err != nil {
		return err
	}
	filter := domain.RawMaterialStockFilter{
		Status:         c.Query("status"),
		WarehouseID:    wid,
		ProductID:      pid,
		StorageZone:    c.Query("storageZone"),
		OrganicStatus:  c.Query("organicStatus"),
		LBMNo:          c.Query("lbmNo"),
		GoodsReceiptID: grID,
		FromDate:       c.Query("fromDate"),
		ToDate:         c.Query("toDate"),
		Search:         c.Query("search"),
	}
	items, err := uc.ListRawMaterialStockEntries(h.ctx(c), filter)
	if err != nil {
		return err
	}
	return response.OK(c, "Raw material stock entries retrieved", items)
}

func (h *Handler) GetRawMaterialStockEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	item, err := uc.GetRawMaterialStockEntry(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Raw material stock entry retrieved", item)
}

func (h *Handler) CreateRawMaterialStockEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateRawMaterialStockDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.CreateRawMaterialStockEntry(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Form Stok Gudang Bahan Baku berhasil dibuat", item)
}

func (h *Handler) UpdateRawMaterialStockEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateRawMaterialStockDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.UpdateRawMaterialStockEntry(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Form Stok Gudang Bahan Baku berhasil diperbarui", item)
}

func (h *Handler) PostRawMaterialStockEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.PostRawMaterialStockDTO
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&dto); err != nil {
			return apperrors.NewBadRequest("Invalid request body")
		}
	}
	item, err := uc.PostRawMaterialStockEntry(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Form Stok Gudang Bahan Baku berhasil diposting", item)
}

func (h *Handler) ReleaseRawMaterialQuarantine(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.ReleaseQuarantineDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.ReleaseQuarantine(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Status karantina lot berhasil diperbarui", item)
}

func (h *Handler) VoidRawMaterialStockEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&in); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	item, err := uc.VoidRawMaterialStockEntry(h.ctx(c), id, in.Reason)
	if err != nil {
		return err
	}
	return response.OK(c, "Form Stok Gudang Bahan Baku berhasil di-void", item)
}

func (h *Handler) DeleteRawMaterialStockEntry(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	if err := uc.DeleteRawMaterialStockEntry(h.ctx(c), id); err != nil {
		return err
	}
	return response.OK(c, "Draft Form Stok Gudang Bahan Baku berhasil dihapus", nil)
}
