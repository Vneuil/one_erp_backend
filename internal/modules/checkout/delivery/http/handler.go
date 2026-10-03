package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/checkout/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct {
	uc           application.CheckoutUseCase
	webhookToken string
}

func NewHandler(uc application.CheckoutUseCase, webhookToken string) *Handler {
	return &Handler{uc: uc, webhookToken: webhookToken}
}

// CreateOrder handles POST /checkout/orders - public, unauthenticated (a
// prospective customer has no account yet). Rejects a client-supplied
// amount implicitly: CreateCheckoutOrderDTO has no amount field at all, the
// use case prices plan+cycle server-side.
func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	var dto application.CreateCheckoutOrderDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}

	result, err := h.uc.CreateOrder(c.UserContext(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Checkout order created", result)
}

// GetOrderStatus handles GET /checkout/orders/:id - public, used by the
// checkout success/pending page to poll payment status. Returns only
// non-sensitive fields (see application.CheckoutOrderStatusDTO) - no
// contact PII, no provider reference.
func (h *Handler) GetOrderStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return apperrors.NewBadRequest("Invalid order id")
	}
	result, err := h.uc.GetOrderStatus(c.UserContext(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Checkout order status", result)
}

// XenditWebhook handles POST /checkout/xendit-webhook. Authenticated the
// way every Xendit integration guide documents: a shared secret in the
// X-Callback-Token header, compared against XENDIT_WEBHOOK_TOKEN - this is
// the ONLY thing standing between this public endpoint and anyone posting
// a fake "PAID" status, so a missing/blank configured token must fail
// closed (reject everything), not silently accept every request.
func (h *Handler) XenditWebhook(c *fiber.Ctx) error {
	if h.webhookToken == "" {
		return apperrors.NewInternal(nil, "Webhook is not configured")
	}
	if c.Get("X-Callback-Token") != h.webhookToken {
		return apperrors.NewUnauthorized("Invalid callback token")
	}

	var payload struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return apperrors.NewBadRequest("Invalid webhook payload")
	}
	if payload.ID == "" {
		return apperrors.NewBadRequest("Missing invoice id")
	}

	if err := h.uc.HandleXenditWebhook(c.UserContext(), payload.ID, payload.Status); err != nil {
		return err
	}
	return response.OK(c, "Webhook processed", nil)
}
