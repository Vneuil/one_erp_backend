package domain

import (
	"context"
	"fmt"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Course represents a learning course offered to employees
type Course struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this course to one business unit within the company
	// (see modules/workspace). Nil means the company's default tenant.
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Description    string     `gorm:"type:text" json:"description"`
	Category       string     `gorm:"type:varchar(100)" json:"category"`
	InstructorName string     `gorm:"type:varchar(255)" json:"instructorName"`
	DurationHours  int        `gorm:"not null;default:0" json:"durationHours"`
	Status         string     `gorm:"type:varchar(20);not null;default:'draft';index" json:"status"`
	EnrolledCount  int        `gorm:"not null;default:0" json:"enrolledCount"`
}

func (Course) TableName() string {
	return "lms_courses"
}

// Enrollment tracks an employee's progress through a course
type Enrollment struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this enrollment to one business unit within the
	// company (see modules/workspace). Nil means the default tenant.
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	CourseID uuid.UUID  `gorm:"type:uuid;not null;index" json:"courseId"`
	// EmployeeID links this enrollment to a real HRM employee record (see
	// modules/hrm). It is optional for backward compatibility with legacy
	// free-typed enrollments, but new enrollments should always set it.
	EmployeeID      *uuid.UUID `gorm:"type:uuid;index" json:"employeeId,omitempty"`
	EmployeeName    string     `gorm:"type:varchar(255);not null" json:"employeeName"`
	EnrolledDate    string     `gorm:"type:varchar(50)" json:"enrolledDate"`
	ProgressPercent int        `gorm:"not null;default:0" json:"progressPercent"`
	Status          string     `gorm:"type:varchar(20);not null;default:'in_progress';index" json:"status"`
	CompletedDate   *string    `gorm:"type:varchar(50)" json:"completedDate,omitempty"`
}

func (Enrollment) TableName() string {
	return "lms_enrollments"
}

// Quiz represents a simple quiz shell attached to a course
type Quiz struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this quiz to one business unit within the company
	// (see modules/workspace). Nil means the company's default tenant.
	TenantID            *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	CourseID            uuid.UUID  `gorm:"type:uuid;not null;index" json:"courseId"`
	Title               string     `gorm:"type:varchar(255);not null" json:"title"`
	PassingScorePercent int        `gorm:"not null;default:70" json:"passingScorePercent"`
}

func (Quiz) TableName() string {
	return "lms_quizzes"
}

// QuizAttempt records an employee's attempt at a quiz
type QuizAttempt struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this quiz attempt to one business unit within the
	// company (see modules/workspace). Nil means the default tenant.
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	QuizID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"quizId"`
	EmployeeName  string     `gorm:"type:varchar(255);not null" json:"employeeName"`
	ScorePercent  int        `gorm:"not null;default:0" json:"scorePercent"`
	Passed        bool       `gorm:"not null;default:false" json:"passed"`
	AttemptedDate string     `gorm:"type:varchar(50)" json:"attemptedDate"`
}

func (QuizAttempt) TableName() string {
	return "lms_quiz_attempts"
}

// Certificate is auto-generated when an enrollment reaches completed status
type Certificate struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this certificate to one business unit within the
	// company (see modules/workspace). Nil means the default tenant.
	TenantID          *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	EnrollmentID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"enrollmentId"`
	CertificateNumber string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"certificateNumber"`
	IssuedDate        string     `gorm:"type:varchar(50)" json:"issuedDate"`
}

func (Certificate) TableName() string {
	return "lms_certificates"
}

type LMSRepository interface {
	CreateCourse(ctx context.Context, c *Course) error
	GetCourseByID(ctx context.Context, id uuid.UUID) (*Course, error)
	ListCourses(ctx context.Context, query types.PaginationQuery) ([]Course, int64, error)
	UpdateCourse(ctx context.Context, c *Course) error
	CountCourses(ctx context.Context) (int64, error)

	CreateEnrollment(ctx context.Context, e *Enrollment) error
	GetEnrollmentByID(ctx context.Context, id uuid.UUID) (*Enrollment, error)
	ListEnrollments(ctx context.Context, query types.PaginationQuery) ([]Enrollment, int64, error)
	ListEnrollmentsByEmployeeID(ctx context.Context, employeeID uuid.UUID) ([]Enrollment, error)
	UpdateEnrollment(ctx context.Context, e *Enrollment) error

	CreateQuiz(ctx context.Context, q *Quiz) error
	GetQuizByID(ctx context.Context, id uuid.UUID) (*Quiz, error)
	ListQuizzes(ctx context.Context, query types.PaginationQuery) ([]Quiz, int64, error)

	CreateQuizAttempt(ctx context.Context, a *QuizAttempt) error

	CreateCertificate(ctx context.Context, cert *Certificate) error
	ListCertificates(ctx context.Context, query types.PaginationQuery) ([]Certificate, int64, error)
	CountCertificates(ctx context.Context) (int64, error)
}

// GenerateCertificateNumber builds a certificate number like CERT-2026-0001
func GenerateCertificateNumber(year int, seq int64) string {
	return fmt.Sprintf("CERT-%d-%04d", year, seq)
}
