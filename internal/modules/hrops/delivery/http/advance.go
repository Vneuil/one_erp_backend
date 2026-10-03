package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RequestAdvance(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	var in struct {
		NIP          string  `json:"nip"`
		Amount       float64 `json:"amount"`
		Installments int     `json:"installments"`
		StartPeriod  string  `json:"startPeriod"`
		Reason       string  `json:"reason"`
	}
	if err := body(c, &in); err != nil {
		return err
	}
	v, err := svc.RequestAdvance(h.ctx(c), caller, in.NIP, in.Amount, in.Installments, in.StartPeriod, in.Reason)
	if err != nil {
		return err
	}
	return response.Created(c, "Cash advance submitted", v)
}

func (h *Handler) ListOwnAdvances(c *fiber.Ctx) error {
	svc, caller, err := h.resolve(c)
	if err != nil {
		return err
	}
	nip, err := h.ownNIP(c, svc, caller)
	if err != nil {
		return err
	}
	v, err := svc.ListAdvances(h.ctx(c), nip, c.Query("status"))
	if err != nil {
		return err
	}
	return response.OK(c, "Cash advances retrieved", v)
}

func (h *Handler) ListAdvances(c *fiber.Ctx) error {
	svc, _, err := h.resolve(c)
	if err != nil {
		return err
	}
	v, err := svc.ListAdvances(h.ctx(c), c.Query("nip"), c.Query("status"))
	if err != nil {
		return err
	}
	return response.OK(c, "Cash advances retrieved", v)
}

func (h *Handler) decideAdvance(approve bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		svc, _, err := h.resolve(c)
		if err != nil {
			return err
		}
		rid, err := id(c)
		if err != nil {
			return err
		}
		v, err := svc.DecideAdvance(h.ctx(c), rid, approve)
		if err != nil {
			return err
		}
		return response.OK(c, "Cash advance "+v.Status, v)
	}
}
