package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/integration/application"
	"github.com/divinecoid/one-backend/internal/modules/integration/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func parseID(c *fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest("Invalid " + param + " format")
	}
	return id, nil
}

// resolve builds a use-case bound to the caller's own tenant database. The
// schema is migrated and seeded once, at tenant-provision time (see
// module.go's registration with foundation/tenant.RegisterSchema).
func (h *Handler) resolve(c *fiber.Ctx) (application.IntegrationUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	apiKeyRepo := infrastructure.NewAPIKeyRepository(tenantDB)
	webhookRepo := infrastructure.NewWebhookRepository(tenantDB)
	return application.NewIntegrationUseCase(apiKeyRepo, webhookRepo), nil
}

func (h *Handler) CreateAPIKey(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateAPIKeyDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	key, err := uc.CreateAPIKey(c.UserContext(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "API key created successfully", key)
}

func (h *Handler) ListAPIKeys(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	keys, err := uc.ListAPIKeys(c.UserContext())
	if err != nil {
		return err
	}
	return response.OK(c, "API keys retrieved successfully", keys)
}

func (h *Handler) RevokeAPIKey(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	key, err := uc.RevokeAPIKey(c.UserContext(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "API key revoked successfully", key)
}

func (h *Handler) CreateWebhook(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	var dto application.CreateWebhookDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	webhook, err := uc.CreateWebhook(c.UserContext(), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Webhook subscription created successfully", webhook)
}

func (h *Handler) ListWebhooks(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	webhooks, err := uc.ListWebhooks(c.UserContext())
	if err != nil {
		return err
	}
	return response.OK(c, "Webhook subscriptions retrieved successfully", webhooks)
}

func (h *Handler) UpdateWebhook(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateWebhookDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	webhook, err := uc.UpdateWebhook(c.UserContext(), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Webhook subscription updated successfully", webhook)
}

func (h *Handler) TestWebhook(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	delivery, err := uc.TestWebhook(c.UserContext(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Simulated webhook delivery attempted", delivery)
}

func (h *Handler) ListDeliveries(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	deliveries, err := uc.ListDeliveries(c.UserContext(), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Webhook deliveries retrieved successfully", deliveries)
}
