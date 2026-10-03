package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	hrm := router.Group("/hrm", protected, tenantCtx)

	hrm.Post("/employees", h.CreateEmployee)
	hrm.Get("/employees", h.ListEmployees)
	hrm.Get("/organization", h.Organization)
	hrm.Get("/employees/:id", h.GetEmployeeByID)
	hrm.Put("/employees/:id", h.UpdateEmployee)
	hrm.Delete("/employees/:id", h.DeleteEmployee)

	hrm.Post("/attendance/clock-in", h.ClockIn)
	hrm.Post("/attendance/clock-out", h.ClockOut)
	hrm.Get("/attendance", h.ListAttendance)

	hrm.Get("/attendance/locations", h.ListAttendanceLocations)
	hrm.Post("/attendance/locations", middleware.RequireRoles("admin"), h.CreateAttendanceLocation)
	hrm.Delete("/attendance/locations/:id", middleware.RequireRoles("admin"), h.DeleteAttendanceLocation)

	hrm.Get("/employees/:id/training-history", h.GetTrainingHistory)
}
