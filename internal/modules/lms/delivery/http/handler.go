package http

import (
	"context"

	"github.com/divinecoid/one-backend/internal/foundation/middleware"
	"github.com/divinecoid/one-backend/internal/foundation/response"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/lms/application"
	"github.com/divinecoid/one-backend/internal/modules/lms/infrastructure"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
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
func (h *Handler) resolve(c *fiber.Ctx) (application.LMSUseCase, error) {
	tenantDB := middleware.TenantDB(c)
	if tenantDB == nil {
		return nil, apperrors.NewBadRequest("No active company. Please select or provision a company first.")
	}
	repo := infrastructure.NewLMSRepository(tenantDB)
	hrmRepo := hrminfra.NewHRMRepository(tenantDB)
	return application.NewLMSUseCase(repo, hrmRepo), nil
}

// ctx returns the request context with the active Tenant ID (see
// modules/workspace) attached, so tenant-aware repository queries can read
// it via tenantctx.FromContext without every usecase method needing an
// extra parameter. A nil tenant ID (the common case) means "no filter".
func (h *Handler) ctx(c *fiber.Ctx) context.Context {
	return tenantctx.WithTenantID(c.UserContext(), middleware.CurrentTenantID(c))
}

func (h *Handler) CreateCourse(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateCourseDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	course, err := uc.CreateCourse(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Course created successfully", course)
}

func (h *Handler) GetCourseByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	course, err := uc.GetCourseByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Course retrieved successfully", course)
}

func (h *Handler) ListCourses(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListCourses(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Courses retrieved successfully", items, meta)
}

func (h *Handler) UpdateCourse(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateCourseDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	course, err := uc.UpdateCourse(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Course updated successfully", course)
}

func (h *Handler) CreateEnrollment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateEnrollmentDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	enrollment, err := uc.CreateEnrollment(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Enrollment created successfully", enrollment)
}

func (h *Handler) GetEnrollmentByID(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	enrollment, err := uc.GetEnrollmentByID(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Enrollment retrieved successfully", enrollment)
}

func (h *Handler) ListEnrollments(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListEnrollments(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Enrollments retrieved successfully", items, meta)
}

func (h *Handler) UpdateProgress(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.UpdateProgressDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	enrollment, err := uc.UpdateProgress(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.OK(c, "Enrollment progress updated successfully", enrollment)
}

func (h *Handler) CompleteEnrollment(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	enrollment, err := uc.CompleteEnrollment(h.ctx(c), id)
	if err != nil {
		return err
	}
	return response.OK(c, "Enrollment completed successfully", enrollment)
}

func (h *Handler) CreateQuiz(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var dto application.CreateQuizDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	quiz, err := uc.CreateQuiz(h.ctx(c), dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Quiz created successfully", quiz)
}

func (h *Handler) ListQuizzes(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListQuizzes(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Quizzes retrieved successfully", items, meta)
}

func (h *Handler) CreateQuizAttempt(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	id, err := parseID(c, "id")
	if err != nil {
		return err
	}
	var dto application.CreateQuizAttemptDTO
	if err := c.BodyParser(&dto); err != nil {
		return apperrors.NewBadRequest("Invalid request body")
	}
	attempt, err := uc.CreateQuizAttempt(h.ctx(c), id, dto)
	if err != nil {
		return err
	}
	return response.Created(c, "Quiz attempt recorded successfully", attempt)
}

func (h *Handler) ListCertificates(c *fiber.Ctx) error {
	uc, err := h.resolve(c)
	if err != nil {
		return err
	}

	var query types.PaginationQuery
	if err := c.QueryParser(&query); err != nil {
		return apperrors.NewBadRequest("Invalid query parameters")
	}
	items, meta, err := uc.ListCertificates(h.ctx(c), query)
	if err != nil {
		return err
	}
	return response.SuccessWithMeta(c, fiber.StatusOK, "Certificates retrieved successfully", items, meta)
}
