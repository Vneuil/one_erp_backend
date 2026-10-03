package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// JobVacancy represents an open position a company is recruiting for
type JobVacancy struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this vacancy to one business unit within the company
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Department     string     `gorm:"type:varchar(100);not null" json:"department"`
	EmploymentType string     `gorm:"type:varchar(30);not null;default:'Full-time'" json:"employmentType"`
	Location       string     `gorm:"type:varchar(255)" json:"location"`
	Description    string     `gorm:"type:text" json:"description"`
	Status         string     `gorm:"type:varchar(20);not null;default:'open';index" json:"status"`
	OpeningsCount  int        `gorm:"not null;default:1" json:"openingsCount"`
	PostedDate     string     `gorm:"type:varchar(50)" json:"postedDate"`
}

func (JobVacancy) TableName() string {
	return "recruitment_job_vacancies"
}

// Candidate represents an applicant against a job vacancy, tracked through the pipeline
type Candidate struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this candidate to one business unit within the company
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	JobVacancyID uuid.UUID  `gorm:"type:uuid;not null;index" json:"jobVacancyId"`
	Name         string     `gorm:"type:varchar(255);not null" json:"name"`
	Email        string     `gorm:"type:varchar(255)" json:"email"`
	Phone        string     `gorm:"type:varchar(50)" json:"phone"`
	ResumeNote   string     `gorm:"type:text" json:"resumeNote"`
	Source       string     `gorm:"type:varchar(100)" json:"source"`
	AppliedDate  string     `gorm:"type:varchar(50)" json:"appliedDate"`
	Stage        string     `gorm:"type:varchar(20);not null;default:'applied';index" json:"stage"`
	StageNotes   string     `gorm:"type:text" json:"stageNotes"`
}

func (Candidate) TableName() string {
	return "recruitment_candidates"
}

// Interview represents a scheduled or completed interview for a candidate
type Interview struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this interview to one business unit within the company
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	CandidateID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"candidateId"`
	ScheduledAt     string     `gorm:"type:varchar(50);not null" json:"scheduledAt"`
	InterviewerName string     `gorm:"type:varchar(255);not null" json:"interviewerName"`
	Status          string     `gorm:"type:varchar(20);not null;default:'scheduled';index" json:"status"`
	FeedbackNote    string     `gorm:"type:text" json:"feedbackNote"`
	Rating          *int       `gorm:"type:int" json:"rating,omitempty"`
}

func (Interview) TableName() string {
	return "recruitment_interviews"
}

type RecruitmentRepository interface {
	CreateVacancy(ctx context.Context, v *JobVacancy) error
	GetVacancyByID(ctx context.Context, id uuid.UUID) (*JobVacancy, error)
	ListVacancies(ctx context.Context, query types.PaginationQuery) ([]JobVacancy, int64, error)
	UpdateVacancy(ctx context.Context, v *JobVacancy) error
	CountVacancies(ctx context.Context) (int64, error)

	CreateCandidate(ctx context.Context, c *Candidate) error
	GetCandidateByID(ctx context.Context, id uuid.UUID) (*Candidate, error)
	ListCandidates(ctx context.Context, query types.PaginationQuery) ([]Candidate, int64, error)
	UpdateCandidate(ctx context.Context, c *Candidate) error
	CountCandidates(ctx context.Context) (int64, error)

	CreateInterview(ctx context.Context, i *Interview) error
	GetInterviewByID(ctx context.Context, id uuid.UUID) (*Interview, error)
	ListInterviews(ctx context.Context, query types.PaginationQuery) ([]Interview, int64, error)
	UpdateInterview(ctx context.Context, i *Interview) error
}
