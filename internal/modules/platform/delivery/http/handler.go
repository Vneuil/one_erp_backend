package http

import (
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/license"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/platform/subscriptions")
	group.Get("/:companyId", h.GetSubscription)
	group.Post("/:companyId/plan", h.ChoosePlan)
	group.Post("/:companyId/payment", h.CreatePayment)
}

func (h *Handler) GetSubscription(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Params("companyId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid company ID",
		})
	}

	var sub license.PlatformSubscription
	err = h.db.WithContext(c.UserContext()).
		Where("company_id = ?", companyID).
		Take(&sub).Error

	if err == gorm.ErrRecordNotFound {
		// Provide default active subscription for company
		now := time.Now().UTC()
		endsAt := now.AddDate(1, 0, 0)
		trialEndsAt := now.AddDate(0, 1, 0)
		sub = license.PlatformSubscription{
			ID:                 uuid.New(),
			CompanyID:          companyID,
			Status:             "active",
			PlanDurationMonths: 12,
			PricePerMonth:      1000000,
			TotalPrice:         10000000,
			TrialEndsAt:        &trialEndsAt,
			StartsAt:           &now,
			EndsAt:             &endsAt,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		_ = h.db.WithContext(c.UserContext()).Create(&sub).Error
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to query subscription",
		})
	}

	return response.OK(c, "Subscription retrieved", sub)
}

type choosePlanRequest struct {
	PlanDurationMonths int `json:"planDurationMonths"`
}

func (h *Handler) ChoosePlan(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Params("companyId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid company ID",
		})
	}

	var req choosePlanRequest
	if err := c.BodyParser(&req); err != nil || req.PlanDurationMonths <= 0 {
		req.PlanDurationMonths = 12
	}

	pricePerMonth := 1000000.0
	totalPrice := pricePerMonth * float64(req.PlanDurationMonths)
	if req.PlanDurationMonths >= 12 {
		totalPrice = 10000000.0 // 17% discount for annual
	}

	var sub license.PlatformSubscription
	now := time.Now().UTC()
	err = h.db.WithContext(c.UserContext()).Where("company_id = ?", companyID).Take(&sub).Error
	if err == gorm.ErrRecordNotFound {
		endsAt := now.AddDate(0, req.PlanDurationMonths, 0)
		sub = license.PlatformSubscription{
			ID:                 uuid.New(),
			CompanyID:          companyID,
			Status:             "pending_payment",
			PlanDurationMonths: req.PlanDurationMonths,
			PricePerMonth:      pricePerMonth,
			TotalPrice:         totalPrice,
			StartsAt:           &now,
			EndsAt:             &endsAt,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := h.db.WithContext(c.UserContext()).Create(&sub).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"message": "Failed to create subscription",
			})
		}
	} else if err == nil {
		sub.PlanDurationMonths = req.PlanDurationMonths
		sub.PricePerMonth = pricePerMonth
		sub.TotalPrice = totalPrice
		sub.Status = "pending_payment"
		sub.UpdatedAt = now
		_ = h.db.WithContext(c.UserContext()).Save(&sub).Error
	}

	return response.OK(c, "Plan updated", sub)
}

func (h *Handler) CreatePayment(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Params("companyId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid company ID",
		})
	}

	var sub license.PlatformSubscription
	if err := h.db.WithContext(c.UserContext()).Where("company_id = ?", companyID).Take(&sub).Error; err == nil {
		now := time.Now().UTC()
		endsAt := now.AddDate(0, sub.PlanDurationMonths, 0)
		sub.Status = "active"
		sub.StartsAt = &now
		sub.EndsAt = &endsAt
		sub.UpdatedAt = now
		_ = h.db.WithContext(c.UserContext()).Save(&sub).Error
	}

	paymentID := uuid.New().String()
	data := fiber.Map{
		"id":          paymentID,
		"amount":      sub.TotalPrice,
		"checkoutUrl": "/billing/subscription?paid=true",
		"status":      "COMPLETED",
	}

	return response.OK(c, "Payment processed", data)
}
