package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/modules/hrops/application"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// wrap resolves the service and caller, runs fn and writes an envelope.
func (h *Handler) wrap(msg string, status int, fn func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		s, caller, err := h.resolve(c)
		if err != nil {
			return err
		}
		data, err := fn(c, s, caller)
		if err != nil {
			return err
		}
		return response.Success(c, status, msg, data)
	}
}

func qid(c *fiber.Ctx, name string) (uuid.UUID, error) {
	v, err := uuid.Parse(c.Query(name))
	if err != nil {
		return uuid.Nil, apperrors.NewBadRequest(name + " is required")
	}
	return v, nil
}

// ---- reimbursement evidence

func (h *Handler) AddEvidence() fiber.Handler {
	return h.wrap("Evidence added", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		var in struct {
			ClaimID uuid.UUID `json:"claimId"`
			Title   string    `json:"title"`
			FileRef string    `json:"fileRef"`
			Amount  float64   `json:"amount"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.AddEvidence(h.ctx(c), caller, application.EvidenceInput{ClaimID: in.ClaimID, Title: in.Title, FileRef: in.FileRef, Amount: in.Amount})
	})
}

func (h *Handler) ListEvidence() fiber.Handler {
	return h.wrap("Evidence retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		id, err := qid(c, "claimId")
		if err != nil {
			return nil, err
		}
		return s.ListEvidence(h.ctx(c), caller, id)
	})
}

// ---- canteen

func (h *Handler) CreateCanteenItem() fiber.Handler {
	return h.wrap("Menu item created", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		var in struct {
			Name  string  `json:"name"`
			Price float64 `json:"price"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.CreateCanteenItem(h.ctx(c), in.Name, in.Price)
	})
}

func (h *Handler) UpdateCanteenItem() fiber.Handler {
	return h.wrap("Menu item updated", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		id, err := parseParam(c)
		if err != nil {
			return nil, err
		}
		var in struct {
			Price    *float64 `json:"price"`
			IsActive *bool    `json:"isActive"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.UpdateCanteenItem(h.ctx(c), id, in.Price, in.IsActive)
	})
}

func parseParam(c *fiber.Ctx) (uuid.UUID, error) { return id(c) }

func (h *Handler) ListCanteenItems() fiber.Handler {
	return h.wrap("Menu retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		return s.ListCanteenItems(h.ctx(c), c.Query("all") != "true")
	})
}

func (h *Handler) OrderMeal() fiber.Handler {
	return h.wrap("Meal ordered", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		var in struct {
			NIP      string    `json:"nip"`
			ItemID   uuid.UUID `json:"itemId"`
			Quantity int       `json:"quantity"`
			Date     string    `json:"date"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.OrderMeal(h.ctx(c), caller, in.NIP, in.ItemID, in.Quantity, in.Date)
	})
}

func (h *Handler) CancelMeal() fiber.Handler {
	return h.wrap("Order cancelled", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		oid, err := parseParam(c)
		if err != nil {
			return nil, err
		}
		return s.CancelMeal(h.ctx(c), caller, oid)
	})
}

func (h *Handler) MyMeals() fiber.Handler {
	return h.wrap("Orders retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		list, total, err := s.MyMeals(h.ctx(c), caller, c.Query("period"))
		if err != nil {
			return nil, err
		}
		return fiber.Map{"orders": list, "total": total}, nil
	})
}

func (h *Handler) CanteenSummary() fiber.Handler {
	return h.wrap("Canteen summary retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		return s.CanteenSummary(h.ctx(c), c.Query("period"))
	})
}

// ---- visits

func (h *Handler) PlanVisits() fiber.Handler {
	return h.wrap("Visits planned", fiber.StatusCreated, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		var in struct {
			NIP   string `json:"nip"`
			Date  string `json:"date"`
			Stops []struct {
				CustomerName string  `json:"customerName"`
				Address      string  `json:"address"`
				Latitude     float64 `json:"latitude"`
				Longitude    float64 `json:"longitude"`
				RadiusMeters int     `json:"radiusMeters"`
			} `json:"stops"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		stops := make([]application.StopInput, len(in.Stops))
		for i, st := range in.Stops {
			stops[i] = application.StopInput{CustomerName: st.CustomerName, Address: st.Address, Latitude: st.Latitude, Longitude: st.Longitude, RadiusMeters: st.RadiusMeters}
		}
		return s.PlanVisits(h.ctx(c), caller, in.NIP, in.Date, stops)
	})
}

func (h *Handler) MyPlan() fiber.Handler {
	return h.wrap("Visit plan retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		return s.MyPlan(h.ctx(c), caller, c.Query("date"))
	})
}

func (h *Handler) TeamPlans() fiber.Handler {
	return h.wrap("Visit plans retrieved", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, _ application.Caller) (any, error) {
		return s.TeamPlans(h.ctx(c), c.Query("date"), c.Query("nip"))
	})
}

func (h *Handler) OptimizeRoute() fiber.Handler {
	return h.wrap("Route optimised", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		var in struct {
			Date      string  `json:"date"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.OptimizeRoute(h.ctx(c), caller, in.Date, application.GeoPoint{Lat: in.Latitude, Lng: in.Longitude})
	})
}

func (h *Handler) CheckIn() fiber.Handler {
	return h.wrap("Checked in", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		sid, err := parseParam(c)
		if err != nil {
			return nil, err
		}
		var in struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.CheckIn(h.ctx(c), caller, sid, application.GeoPoint{Lat: in.Latitude, Lng: in.Longitude})
	})
}

func (h *Handler) SkipStop() fiber.Handler {
	return h.wrap("Stop skipped", fiber.StatusOK, func(c *fiber.Ctx, s *application.Service, caller application.Caller) (any, error) {
		sid, err := parseParam(c)
		if err != nil {
			return nil, err
		}
		var in struct {
			Reason string `json:"reason"`
		}
		if err := body(c, &in); err != nil {
			return nil, err
		}
		return s.SkipStop(h.ctx(c), caller, sid, in.Reason)
	})
}
