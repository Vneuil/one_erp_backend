package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/projecttask/domain"
	"github.com/google/uuid"
)

type CreateChecklistItemDTO struct {
	Text string `json:"text"`
}

type CreateProjectTaskDTO struct {
	ProjectCode string                   `json:"projectCode"`
	Title       string                   `json:"title"`
	Assignee    string                   `json:"assignee"`
	Priority    string                   `json:"priority"`
	Status      string                   `json:"status"`
	DueDate     string                   `json:"dueDate"`
	Checklist   []CreateChecklistItemDTO `json:"checklist,omitempty"`
}

type UpdateStatusDTO struct {
	Status string `json:"status"`
}

type ChecklistItemResponseDTO struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"taskId"`
	Text      string    `json:"text"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"createdAt"`
}

type ProjectTaskResponseDTO struct {
	ID          uuid.UUID                  `json:"id"`
	ProjectCode string                     `json:"projectCode"`
	Title       string                     `json:"title"`
	Assignee    string                     `json:"assignee"`
	Priority    string                     `json:"priority"`
	Status      string                     `json:"status"`
	DueDate     string                     `json:"dueDate"`
	Checklist   []ChecklistItemResponseDTO `json:"checklist,omitempty"`
	CreatedAt   time.Time                  `json:"createdAt"`
	UpdatedAt   time.Time                  `json:"updatedAt"`
}

func ToChecklistItemResponse(i *domain.TaskChecklistItem) *ChecklistItemResponseDTO {
	if i == nil {
		return nil
	}
	return &ChecklistItemResponseDTO{
		ID:        i.ID,
		TaskID:    i.TaskID,
		Text:      i.Text,
		Done:      i.Done,
		CreatedAt: i.CreatedAt,
	}
}

func ToChecklistItemResponseList(items []domain.TaskChecklistItem) []ChecklistItemResponseDTO {
	result := make([]ChecklistItemResponseDTO, len(items))
	for i, item := range items {
		result[i] = *ToChecklistItemResponse(&item)
	}
	return result
}

func ToProjectTaskResponse(t *domain.ProjectTask) *ProjectTaskResponseDTO {
	if t == nil {
		return nil
	}
	return &ProjectTaskResponseDTO{
		ID:          t.ID,
		ProjectCode: t.ProjectCode,
		Title:       t.Title,
		Assignee:    t.Assignee,
		Priority:    t.Priority,
		Status:      t.Status,
		DueDate:     t.DueDate,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func ToProjectTaskResponseList(tasks []domain.ProjectTask) []ProjectTaskResponseDTO {
	result := make([]ProjectTaskResponseDTO, len(tasks))
	for i, t := range tasks {
		result[i] = *ToProjectTaskResponse(&t)
	}
	return result
}
