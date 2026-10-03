package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/lms/domain"
	"github.com/google/uuid"
)

type CreateCourseDTO struct {
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	InstructorName string     `json:"instructorName"`
	DurationHours  int        `json:"durationHours"`
	CompanyID      *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateCourseDTO struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	InstructorName string `json:"instructorName"`
	DurationHours  int    `json:"durationHours"`
	Status         string `json:"status"`
}

type CourseResponseDTO struct {
	ID             uuid.UUID  `json:"id"`
	CompanyID      *uuid.UUID `json:"companyId,omitempty"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	InstructorName string     `json:"instructorName"`
	DurationHours  int        `json:"durationHours"`
	Status         string     `json:"status"`
	EnrolledCount  int        `json:"enrolledCount"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type CreateEnrollmentDTO struct {
	CourseID uuid.UUID `json:"courseId"`
	// EmployeeID should reference a real employee (see modules/hrm). When
	// set, EmployeeName is derived server-side from that employee record
	// rather than trusted from the client.
	EmployeeID   *uuid.UUID `json:"employeeId,omitempty"`
	EmployeeName string     `json:"employeeName"`
	EnrolledDate string     `json:"enrolledDate"`
	CompanyID    *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateProgressDTO struct {
	ProgressPercent int `json:"progressPercent"`
}

type EnrollmentResponseDTO struct {
	ID              uuid.UUID  `json:"id"`
	CompanyID       *uuid.UUID `json:"companyId,omitempty"`
	CourseID        uuid.UUID  `json:"courseId"`
	EmployeeID      *uuid.UUID `json:"employeeId,omitempty"`
	EmployeeName    string     `json:"employeeName"`
	EnrolledDate    string     `json:"enrolledDate"`
	ProgressPercent int        `json:"progressPercent"`
	Status          string     `json:"status"`
	CompletedDate   *string    `json:"completedDate,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type CreateQuizDTO struct {
	CourseID            uuid.UUID  `json:"courseId"`
	Title               string     `json:"title"`
	PassingScorePercent int        `json:"passingScorePercent"`
	CompanyID           *uuid.UUID `json:"companyId,omitempty"`
}

type QuizResponseDTO struct {
	ID                  uuid.UUID  `json:"id"`
	CompanyID           *uuid.UUID `json:"companyId,omitempty"`
	CourseID            uuid.UUID  `json:"courseId"`
	Title               string     `json:"title"`
	PassingScorePercent int        `json:"passingScorePercent"`
	CreatedAt           time.Time  `json:"createdAt"`
}

type CreateQuizAttemptDTO struct {
	EmployeeName  string `json:"employeeName"`
	ScorePercent  int    `json:"scorePercent"`
	AttemptedDate string `json:"attemptedDate"`
}

type QuizAttemptResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	QuizID        uuid.UUID `json:"quizId"`
	EmployeeName  string    `json:"employeeName"`
	ScorePercent  int       `json:"scorePercent"`
	Passed        bool      `json:"passed"`
	AttemptedDate string    `json:"attemptedDate"`
	CreatedAt     time.Time `json:"createdAt"`
}

type CertificateResponseDTO struct {
	ID                uuid.UUID `json:"id"`
	EnrollmentID      uuid.UUID `json:"enrollmentId"`
	CertificateNumber string    `json:"certificateNumber"`
	IssuedDate        string    `json:"issuedDate"`
	CreatedAt         time.Time `json:"createdAt"`
}

func ToCourseResponse(c *domain.Course) *CourseResponseDTO {
	if c == nil {
		return nil
	}
	return &CourseResponseDTO{
		ID:             c.ID,
		CompanyID:      c.CompanyID,
		Title:          c.Title,
		Description:    c.Description,
		Category:       c.Category,
		InstructorName: c.InstructorName,
		DurationHours:  c.DurationHours,
		Status:         c.Status,
		EnrolledCount:  c.EnrolledCount,
		CreatedAt:      c.CreatedAt,
	}
}

func ToCourseResponseList(courses []domain.Course) []CourseResponseDTO {
	result := make([]CourseResponseDTO, len(courses))
	for i, c := range courses {
		result[i] = *ToCourseResponse(&c)
	}
	return result
}

func ToEnrollmentResponse(e *domain.Enrollment) *EnrollmentResponseDTO {
	if e == nil {
		return nil
	}
	return &EnrollmentResponseDTO{
		ID:              e.ID,
		CompanyID:       e.CompanyID,
		CourseID:        e.CourseID,
		EmployeeID:      e.EmployeeID,
		EmployeeName:    e.EmployeeName,
		EnrolledDate:    e.EnrolledDate,
		ProgressPercent: e.ProgressPercent,
		Status:          e.Status,
		CompletedDate:   e.CompletedDate,
		CreatedAt:       e.CreatedAt,
	}
}

func ToEnrollmentResponseList(enrollments []domain.Enrollment) []EnrollmentResponseDTO {
	result := make([]EnrollmentResponseDTO, len(enrollments))
	for i, e := range enrollments {
		result[i] = *ToEnrollmentResponse(&e)
	}
	return result
}

func ToQuizResponse(q *domain.Quiz) *QuizResponseDTO {
	if q == nil {
		return nil
	}
	return &QuizResponseDTO{
		ID:                  q.ID,
		CompanyID:           q.CompanyID,
		CourseID:            q.CourseID,
		Title:               q.Title,
		PassingScorePercent: q.PassingScorePercent,
		CreatedAt:           q.CreatedAt,
	}
}

func ToQuizResponseList(quizzes []domain.Quiz) []QuizResponseDTO {
	result := make([]QuizResponseDTO, len(quizzes))
	for i, q := range quizzes {
		result[i] = *ToQuizResponse(&q)
	}
	return result
}

func ToQuizAttemptResponse(a *domain.QuizAttempt) *QuizAttemptResponseDTO {
	if a == nil {
		return nil
	}
	return &QuizAttemptResponseDTO{
		ID:            a.ID,
		QuizID:        a.QuizID,
		EmployeeName:  a.EmployeeName,
		ScorePercent:  a.ScorePercent,
		Passed:        a.Passed,
		AttemptedDate: a.AttemptedDate,
		CreatedAt:     a.CreatedAt,
	}
}

func ToCertificateResponse(cert *domain.Certificate) *CertificateResponseDTO {
	if cert == nil {
		return nil
	}
	return &CertificateResponseDTO{
		ID:                cert.ID,
		EnrollmentID:      cert.EnrollmentID,
		CertificateNumber: cert.CertificateNumber,
		IssuedDate:        cert.IssuedDate,
		CreatedAt:         cert.CreatedAt,
	}
}

func ToCertificateResponseList(certs []domain.Certificate) []CertificateResponseDTO {
	result := make([]CertificateResponseDTO, len(certs))
	for i, cert := range certs {
		result[i] = *ToCertificateResponse(&cert)
	}
	return result
}
