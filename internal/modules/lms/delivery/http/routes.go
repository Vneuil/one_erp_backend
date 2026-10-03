package http

import (
	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	tenantMgr "github.com/divinecoid/one-backend/internal/foundation/tenant"
	"github.com/gofiber/fiber/v2"
)

func (h *Handler) RegisterRoutes(router fiber.Router, jwtSecret string, manager *tenantMgr.Manager) {
	protected := middleware.Protected(jwtSecret)
	tenantCtx := middleware.TenantContext(manager)

	lms := router.Group("/lms", protected, tenantCtx)

	lms.Post("/courses", h.CreateCourse)
	lms.Get("/courses", h.ListCourses)
	lms.Get("/courses/:id", h.GetCourseByID)
	lms.Put("/courses/:id", h.UpdateCourse)

	lms.Post("/enrollments", h.CreateEnrollment)
	lms.Get("/enrollments", h.ListEnrollments)
	lms.Get("/enrollments/:id", h.GetEnrollmentByID)
	lms.Put("/enrollments/:id/progress", h.UpdateProgress)
	lms.Post("/enrollments/:id/complete", h.CompleteEnrollment)

	lms.Post("/quizzes", h.CreateQuiz)
	lms.Get("/quizzes", h.ListQuizzes)
	lms.Post("/quizzes/:id/attempts", h.CreateQuizAttempt)

	lms.Get("/certificates", h.ListCertificates)
}
