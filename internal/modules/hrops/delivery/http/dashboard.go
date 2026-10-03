package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) Dashboard(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.Dashboard(h.ctx(c), caller)
	if err != nil {
		return err
	}
	return response.OK(c, "HR dashboard retrieved", v)
}
