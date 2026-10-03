package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// ProjectTask is a kanban task scoped to a Project (loosely, via ProjectCode -
// this codebase denormalizes across module boundaries rather than using hard
// DB foreign keys between modules).
type ProjectTask struct {
	types.BaseEntity
	CompanyID   *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProjectCode string     `gorm:"type:varchar(50);not null;index" json:"projectCode"`
	Title       string     `gorm:"type:varchar(255);not null" json:"title"`
	Assignee    string     `gorm:"type:varchar(255)" json:"assignee"`
	Priority    string     `gorm:"type:varchar(20);not null;default:'Medium'" json:"priority"`
	Status      string     `gorm:"type:varchar(20);not null;default:'todo'" json:"status"`
	DueDate     string     `gorm:"type:varchar(20)" json:"dueDate"`
}

func (ProjectTask) TableName() string {
	return "project_tasks"
}

// TaskChecklistItem is one checklist line item belonging to a ProjectTask.
type TaskChecklistItem struct {
	types.BaseEntity
	TaskID uuid.UUID `gorm:"type:uuid;not null;index" json:"taskId"`
	Text   string    `gorm:"type:varchar(255);not null" json:"text"`
	Done   bool      `gorm:"not null;default:false" json:"done"`
}

func (TaskChecklistItem) TableName() string {
	return "task_checklist_items"
}

type ProjectTaskRepository interface {
	Create(ctx context.Context, task *ProjectTask) error
	CreateWithChecklist(ctx context.Context, task *ProjectTask, items []TaskChecklistItem) error
	GetByID(ctx context.Context, id uuid.UUID) (*ProjectTask, error)
	List(ctx context.Context, query types.PaginationQuery, projectCode, status string) ([]ProjectTask, int64, error)
	Update(ctx context.Context, task *ProjectTask) error
	Count(ctx context.Context) (int64, error)

	CreateChecklistItem(ctx context.Context, item *TaskChecklistItem) error
	GetChecklistItem(ctx context.Context, id uuid.UUID) (*TaskChecklistItem, error)
	UpdateChecklistItem(ctx context.Context, item *TaskChecklistItem) error
	ListChecklistByTask(ctx context.Context, taskID uuid.UUID) ([]TaskChecklistItem, error)
}
