package application

import (
	"context"
	"time"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/lms/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

func currentYear() int {
	return time.Now().Year()
}

var validCourseStatuses = map[string]bool{
	"draft":     true,
	"published": true,
	"archived":  true,
}

type LMSUseCase interface {
	CreateCourse(ctx context.Context, dto CreateCourseDTO) (*CourseResponseDTO, error)
	GetCourseByID(ctx context.Context, id uuid.UUID) (*CourseResponseDTO, error)
	ListCourses(ctx context.Context, query types.PaginationQuery) ([]CourseResponseDTO, types.PaginationMeta, error)
	UpdateCourse(ctx context.Context, id uuid.UUID, dto UpdateCourseDTO) (*CourseResponseDTO, error)

	CreateEnrollment(ctx context.Context, dto CreateEnrollmentDTO) (*EnrollmentResponseDTO, error)
	GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*EnrollmentResponseDTO, error)
	ListEnrollments(ctx context.Context, query types.PaginationQuery) ([]EnrollmentResponseDTO, types.PaginationMeta, error)
	UpdateProgress(ctx context.Context, id uuid.UUID, dto UpdateProgressDTO) (*EnrollmentResponseDTO, error)
	CompleteEnrollment(ctx context.Context, id uuid.UUID) (*EnrollmentResponseDTO, error)

	CreateQuiz(ctx context.Context, dto CreateQuizDTO) (*QuizResponseDTO, error)
	ListQuizzes(ctx context.Context, query types.PaginationQuery) ([]QuizResponseDTO, types.PaginationMeta, error)
	CreateQuizAttempt(ctx context.Context, quizID uuid.UUID, dto CreateQuizAttemptDTO) (*QuizAttemptResponseDTO, error)

	ListCertificates(ctx context.Context, query types.PaginationQuery) ([]CertificateResponseDTO, types.PaginationMeta, error)

	SeedInitialData(ctx context.Context) error
}

type lmsUseCase struct {
	repo    domain.LMSRepository
	hrmRepo hrmdomain.HRMRepository
}

// NewLMSUseCase wires the LMS repository together with the HRM repository so
// enrollments can be linked to real employee records (see EmployeeID on
// CreateEnrollmentDTO). hrmRepo may be nil, in which case enrollments fall
// back to the free-typed EmployeeName field.
func NewLMSUseCase(repo domain.LMSRepository, hrmRepo hrmdomain.HRMRepository) LMSUseCase {
	return &lmsUseCase{repo: repo, hrmRepo: hrmRepo}
}

func (uc *lmsUseCase) CreateCourse(ctx context.Context, dto CreateCourseDTO) (*CourseResponseDTO, error) {
	if dto.Title == "" {
		return nil, apperrors.NewBadRequest("Title is required")
	}
	if dto.DurationHours < 0 {
		return nil, apperrors.NewBadRequest("Duration hours cannot be negative")
	}

	course := &domain.Course{
		CompanyID:      dto.CompanyID,
		Title:          dto.Title,
		Description:    dto.Description,
		Category:       dto.Category,
		InstructorName: dto.InstructorName,
		DurationHours:  dto.DurationHours,
		Status:         "draft",
	}

	if err := uc.repo.CreateCourse(ctx, course); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create course")
	}
	return ToCourseResponse(course), nil
}

func (uc *lmsUseCase) GetCourseByID(ctx context.Context, id uuid.UUID) (*CourseResponseDTO, error) {
	course, err := uc.repo.GetCourseByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get course")
	}
	if course == nil {
		return nil, apperrors.NewNotFound("Course not found")
	}
	return ToCourseResponse(course), nil
}

func (uc *lmsUseCase) ListCourses(ctx context.Context, query types.PaginationQuery) ([]CourseResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	courses, total, err := uc.repo.ListCourses(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list courses")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToCourseResponseList(courses), meta, nil
}

func (uc *lmsUseCase) UpdateCourse(ctx context.Context, id uuid.UUID, dto UpdateCourseDTO) (*CourseResponseDTO, error) {
	course, err := uc.repo.GetCourseByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get course")
	}
	if course == nil {
		return nil, apperrors.NewNotFound("Course not found")
	}

	if dto.Title != "" {
		course.Title = dto.Title
	}
	if dto.Description != "" {
		course.Description = dto.Description
	}
	if dto.Category != "" {
		course.Category = dto.Category
	}
	if dto.InstructorName != "" {
		course.InstructorName = dto.InstructorName
	}
	if dto.DurationHours > 0 {
		course.DurationHours = dto.DurationHours
	}
	if dto.Status != "" {
		if !validCourseStatuses[dto.Status] {
			return nil, apperrors.NewBadRequest("Status must be one of draft, published, archived")
		}
		course.Status = dto.Status
	}

	if err := uc.repo.UpdateCourse(ctx, course); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update course")
	}
	return ToCourseResponse(course), nil
}

func (uc *lmsUseCase) CreateEnrollment(ctx context.Context, dto CreateEnrollmentDTO) (*EnrollmentResponseDTO, error) {
	if dto.CourseID == uuid.Nil {
		return nil, apperrors.NewBadRequest("Course is required")
	}

	employeeName := dto.EmployeeName
	if dto.EmployeeID != nil && uc.hrmRepo != nil {
		emp, err := uc.hrmRepo.GetEmployeeByID(ctx, *dto.EmployeeID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get employee")
		}
		if emp == nil {
			return nil, apperrors.NewNotFound("Employee not found")
		}
		// EmployeeName is never trusted from the client when EmployeeID is
		// provided - it is always derived from the real employee record.
		employeeName = emp.Name
	}
	if employeeName == "" {
		return nil, apperrors.NewBadRequest("Employee is required")
	}

	course, err := uc.repo.GetCourseByID(ctx, dto.CourseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get course")
	}
	if course == nil {
		return nil, apperrors.NewNotFound("Course not found")
	}

	enrollment := &domain.Enrollment{
		CompanyID:    dto.CompanyID,
		CourseID:     dto.CourseID,
		EmployeeID:   dto.EmployeeID,
		EmployeeName: employeeName,
		EnrolledDate: dto.EnrolledDate,
		Status:       "in_progress",
	}

	if err := uc.repo.CreateEnrollment(ctx, enrollment); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create enrollment")
	}

	course.EnrolledCount++
	if err := uc.repo.UpdateCourse(ctx, course); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update course")
	}

	return ToEnrollmentResponse(enrollment), nil
}

func (uc *lmsUseCase) GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*EnrollmentResponseDTO, error) {
	enrollment, err := uc.repo.GetEnrollmentByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get enrollment")
	}
	if enrollment == nil {
		return nil, apperrors.NewNotFound("Enrollment not found")
	}
	return ToEnrollmentResponse(enrollment), nil
}

func (uc *lmsUseCase) ListEnrollments(ctx context.Context, query types.PaginationQuery) ([]EnrollmentResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	enrollments, total, err := uc.repo.ListEnrollments(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list enrollments")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToEnrollmentResponseList(enrollments), meta, nil
}

func (uc *lmsUseCase) UpdateProgress(ctx context.Context, id uuid.UUID, dto UpdateProgressDTO) (*EnrollmentResponseDTO, error) {
	if dto.ProgressPercent < 0 || dto.ProgressPercent > 100 {
		return nil, apperrors.NewBadRequest("Progress percent must be between 0 and 100")
	}

	enrollment, err := uc.repo.GetEnrollmentByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get enrollment")
	}
	if enrollment == nil {
		return nil, apperrors.NewNotFound("Enrollment not found")
	}
	if enrollment.Status == "completed" {
		return nil, apperrors.NewBadRequest("Enrollment is already completed")
	}

	enrollment.ProgressPercent = dto.ProgressPercent
	if err := uc.repo.UpdateEnrollment(ctx, enrollment); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update enrollment")
	}
	return ToEnrollmentResponse(enrollment), nil
}

func (uc *lmsUseCase) CompleteEnrollment(ctx context.Context, id uuid.UUID) (*EnrollmentResponseDTO, error) {
	enrollment, err := uc.repo.GetEnrollmentByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get enrollment")
	}
	if enrollment == nil {
		return nil, apperrors.NewNotFound("Enrollment not found")
	}
	if enrollment.Status == "completed" {
		return nil, apperrors.NewBadRequest("Enrollment is already completed")
	}

	completedDate := time.Now().Format("2006-01-02")
	enrollment.Status = "completed"
	enrollment.ProgressPercent = 100
	enrollment.CompletedDate = &completedDate
	if err := uc.repo.UpdateEnrollment(ctx, enrollment); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update enrollment")
	}

	if err := uc.issueCertificate(ctx, enrollment); err != nil {
		return nil, err
	}

	return ToEnrollmentResponse(enrollment), nil
}

func (uc *lmsUseCase) issueCertificate(ctx context.Context, enrollment *domain.Enrollment) error {
	count, err := uc.repo.CountCertificates(ctx)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to count certificates")
	}

	year := currentYear()
	cert := &domain.Certificate{
		CompanyID:         enrollment.CompanyID,
		EnrollmentID:      enrollment.ID,
		CertificateNumber: domain.GenerateCertificateNumber(year, count+1),
		IssuedDate:        enrollment.EnrolledDate,
	}
	if err := uc.repo.CreateCertificate(ctx, cert); err != nil {
		return apperrors.NewInternal(err, "Failed to issue certificate")
	}
	return nil
}

func (uc *lmsUseCase) CreateQuiz(ctx context.Context, dto CreateQuizDTO) (*QuizResponseDTO, error) {
	if dto.CourseID == uuid.Nil {
		return nil, apperrors.NewBadRequest("Course is required")
	}
	if dto.Title == "" {
		return nil, apperrors.NewBadRequest("Title is required")
	}
	if dto.PassingScorePercent < 0 || dto.PassingScorePercent > 100 {
		return nil, apperrors.NewBadRequest("Passing score percent must be between 0 and 100")
	}
	if dto.PassingScorePercent == 0 {
		dto.PassingScorePercent = 70
	}

	course, err := uc.repo.GetCourseByID(ctx, dto.CourseID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get course")
	}
	if course == nil {
		return nil, apperrors.NewNotFound("Course not found")
	}

	quiz := &domain.Quiz{
		CompanyID:           dto.CompanyID,
		CourseID:            dto.CourseID,
		Title:               dto.Title,
		PassingScorePercent: dto.PassingScorePercent,
	}

	if err := uc.repo.CreateQuiz(ctx, quiz); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create quiz")
	}
	return ToQuizResponse(quiz), nil
}

func (uc *lmsUseCase) ListQuizzes(ctx context.Context, query types.PaginationQuery) ([]QuizResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	quizzes, total, err := uc.repo.ListQuizzes(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list quizzes")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToQuizResponseList(quizzes), meta, nil
}

func (uc *lmsUseCase) CreateQuizAttempt(ctx context.Context, quizID uuid.UUID, dto CreateQuizAttemptDTO) (*QuizAttemptResponseDTO, error) {
	if dto.EmployeeName == "" {
		return nil, apperrors.NewBadRequest("Employee name is required")
	}
	if dto.ScorePercent < 0 || dto.ScorePercent > 100 {
		return nil, apperrors.NewBadRequest("Score percent must be between 0 and 100")
	}

	quiz, err := uc.repo.GetQuizByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get quiz")
	}
	if quiz == nil {
		return nil, apperrors.NewNotFound("Quiz not found")
	}

	attempt := &domain.QuizAttempt{
		CompanyID:     quiz.CompanyID,
		QuizID:        quizID,
		EmployeeName:  dto.EmployeeName,
		ScorePercent:  dto.ScorePercent,
		Passed:        dto.ScorePercent >= quiz.PassingScorePercent,
		AttemptedDate: dto.AttemptedDate,
	}

	if err := uc.repo.CreateQuizAttempt(ctx, attempt); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to record quiz attempt")
	}
	return ToQuizAttemptResponse(attempt), nil
}

func (uc *lmsUseCase) ListCertificates(ctx context.Context, query types.PaginationQuery) ([]CertificateResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	certs, total, err := uc.repo.ListCertificates(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list certificates")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToCertificateResponseList(certs), meta, nil
}

// SeedInitialData populates a few sample courses and enrollments on first boot
func (uc *lmsUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountCourses(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	courses := []CreateCourseDTO{
		{Title: "Workplace Safety Fundamentals", Description: "Core occupational safety practices for factory floor staff", Category: "Safety", InstructorName: "Budi Santoso", DurationHours: 4},
		{Title: "Effective Communication Skills", Description: "Business communication and presentation skills", Category: "Soft Skills", InstructorName: "Sari Wulandari", DurationHours: 6},
		{Title: "Lean Manufacturing Principles", Description: "Introduction to lean and continuous improvement", Category: "Operations", InstructorName: "Agus Prasetyo", DurationHours: 8},
	}

	var created []*CourseResponseDTO
	for _, c := range courses {
		r, err := uc.CreateCourse(ctx, c)
		if err != nil {
			continue
		}
		if _, err := uc.UpdateCourse(ctx, r.ID, UpdateCourseDTO{Status: "published"}); err != nil {
			continue
		}
		created = append(created, r)
	}
	if len(created) < 3 {
		return nil
	}

	enrollments := []struct {
		dto      CreateEnrollmentDTO
		progress int
		complete bool
	}{
		{dto: CreateEnrollmentDTO{CourseID: created[0].ID, EmployeeName: "Dedi Setiawan", EnrolledDate: "2026-08-10"}, progress: 100, complete: true},
		{dto: CreateEnrollmentDTO{CourseID: created[1].ID, EmployeeName: "Maya Anggraini", EnrolledDate: "2026-08-20"}, progress: 45},
		{dto: CreateEnrollmentDTO{CourseID: created[2].ID, EmployeeName: "Fajar Nugraha", EnrolledDate: "2026-08-25"}, progress: 10},
	}

	for _, e := range enrollments {
		resp, err := uc.CreateEnrollment(ctx, e.dto)
		if err != nil {
			continue
		}
		if e.complete {
			if _, err := uc.CompleteEnrollment(ctx, resp.ID); err != nil {
				continue
			}
			continue
		}
		if _, err := uc.UpdateProgress(ctx, resp.ID, UpdateProgressDTO{ProgressPercent: e.progress}); err != nil {
			continue
		}
	}

	return nil
}
