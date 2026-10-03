package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type Project struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this project to one business unit within the company
	TenantID   *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Code       string     `gorm:"type:varchar(50);not null;index" json:"code"`
	Name       string     `gorm:"type:varchar(255);not null" json:"name"`
	Customer   string     `gorm:"type:varchar(255);not null" json:"customer"`
	RABValue   float64    `gorm:"type:decimal(15,2);default:0" json:"rabValue"`
	RAPValue   float64    `gorm:"type:decimal(15,2);default:0" json:"rapValue"`
	ActualCost float64    `gorm:"type:decimal(15,2);default:0" json:"actualCost"`
	Progress   int        `gorm:"default:0" json:"progress"`
	Status     string     `gorm:"type:varchar(50);default:'Planning'" json:"status"`
	Manager    string     `gorm:"type:varchar(100)" json:"manager"`
}

func (Project) TableName() string {
	return "projects"
}

type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	List(ctx context.Context, query types.PaginationQuery) ([]Project, int64, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)
}

// TimeEntry is a single finished timesheet log entry against a Project
// (loosely, via ProjectCode - see Project's own denormalization note).
type TimeEntry struct {
	types.BaseEntity
	CompanyID       *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProjectCode     string     `gorm:"type:varchar(50);not null;index" json:"projectCode"`
	TaskTitle       string     `gorm:"type:varchar(255)" json:"taskTitle"`
	WorkerName      string     `gorm:"type:varchar(255)" json:"workerName"`
	Date            string     `gorm:"type:varchar(20)" json:"date"`
	DurationMinutes int        `gorm:"default:0" json:"durationMinutes"`
	HourlyRate      float64    `gorm:"type:decimal(15,2);default:0" json:"hourlyRate"`
	IsBillable      bool       `gorm:"not null;default:true" json:"isBillable"`
}

func (TimeEntry) TableName() string {
	return "project_time_entries"
}

type TimeEntryRepository interface {
	Create(ctx context.Context, entry *TimeEntry) error
	List(ctx context.Context, query types.PaginationQuery, projectCode string) ([]TimeEntry, int64, error)
	Count(ctx context.Context) (int64, error)
	// ListAllByProjectCode returns every (unpaginated) time entry logged
	// against a project, used to compute real ActualCost (see
	// ProjectUseCase.RecalculateProgress).
	ListAllByProjectCode(ctx context.Context, projectCode string) ([]TimeEntry, error)
}
