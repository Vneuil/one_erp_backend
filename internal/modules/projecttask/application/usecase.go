package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/projecttask/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var validTaskStatuses = map[string]bool{
	"todo":        true,
	"in_progress": true,
	"review":      true,
	"done":        true,
}

type ProjectTaskUseCase interface {
	Create(ctx context.Context, dto CreateProjectTaskDTO) (*ProjectTaskResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ProjectTaskResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery, projectCode, status string) ([]ProjectTaskResponseDTO, types.PaginationMeta, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, dto UpdateStatusDTO) (*ProjectTaskResponseDTO, error)
	AddChecklistItem(ctx context.Context, taskID uuid.UUID, dto CreateChecklistItemDTO) (*ChecklistItemResponseDTO, error)
	ToggleChecklistItem(ctx context.Context, itemID uuid.UUID) (*ChecklistItemResponseDTO, error)
	SeedInitialData(ctx context.Context) error
}

type projectTaskUseCase struct {
	repo domain.ProjectTaskRepository
}

func NewProjectTaskUseCase(repo domain.ProjectTaskRepository) ProjectTaskUseCase {
	return &projectTaskUseCase{repo: repo}
}

func (uc *projectTaskUseCase) Create(ctx context.Context, dto CreateProjectTaskDTO) (*ProjectTaskResponseDTO, error) {
	title := strings.TrimSpace(dto.Title)
	projectCode := strings.TrimSpace(dto.ProjectCode)
	if title == "" || projectCode == "" {
		return nil, apperrors.NewBadRequest("ProjectCode and Title are required")
	}

	status := dto.Status
	if status == "" {
		status = "todo"
	} else if !validTaskStatuses[status] {
		return nil, apperrors.NewBadRequest("Status must be one of todo, in_progress, review, done")
	}

	priority := dto.Priority
	if priority == "" {
		priority = "Medium"
	}

	task := &domain.ProjectTask{
		ProjectCode: projectCode,
		Title:       title,
		Assignee:    dto.Assignee,
		Priority:    priority,
		Status:      status,
		DueDate:     dto.DueDate,
	}

	items := make([]domain.TaskChecklistItem, 0, len(dto.Checklist))
	for _, c := range dto.Checklist {
		text := strings.TrimSpace(c.Text)
		if text == "" {
			continue
		}
		items = append(items, domain.TaskChecklistItem{Text: text})
	}

	if err := uc.repo.CreateWithChecklist(ctx, task, items); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create project task")
	}

	resp := ToProjectTaskResponse(task)
	resp.Checklist = ToChecklistItemResponseList(items)
	return resp, nil
}

func (uc *projectTaskUseCase) GetByID(ctx context.Context, id uuid.UUID) (*ProjectTaskResponseDTO, error) {
	task, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve project task")
	}
	if task == nil {
		return nil, apperrors.NewNotFound("Project task not found")
	}
	items, err := uc.repo.ListChecklistByTask(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve checklist")
	}
	resp := ToProjectTaskResponse(task)
	resp.Checklist = ToChecklistItemResponseList(items)
	return resp, nil
}

func (uc *projectTaskUseCase) List(ctx context.Context, query types.PaginationQuery, projectCode, status string) ([]ProjectTaskResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	tasks, total, err := uc.repo.List(ctx, query, projectCode, status)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list project tasks")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToProjectTaskResponseList(tasks), meta, nil
}

func (uc *projectTaskUseCase) UpdateStatus(ctx context.Context, id uuid.UUID, dto UpdateStatusDTO) (*ProjectTaskResponseDTO, error) {
	task, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get project task")
	}
	if task == nil {
		return nil, apperrors.NewNotFound("Project task not found")
	}
	if !validTaskStatuses[dto.Status] {
		return nil, apperrors.NewBadRequest("Status must be one of todo, in_progress, review, done")
	}
	task.Status = dto.Status
	if err := uc.repo.Update(ctx, task); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update project task status")
	}
	return ToProjectTaskResponse(task), nil
}

func (uc *projectTaskUseCase) AddChecklistItem(ctx context.Context, taskID uuid.UUID, dto CreateChecklistItemDTO) (*ChecklistItemResponseDTO, error) {
	task, err := uc.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get project task")
	}
	if task == nil {
		return nil, apperrors.NewNotFound("Project task not found")
	}
	text := strings.TrimSpace(dto.Text)
	if text == "" {
		return nil, apperrors.NewBadRequest("Checklist item text is required")
	}
	item := &domain.TaskChecklistItem{TaskID: taskID, Text: text}
	if err := uc.repo.CreateChecklistItem(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to add checklist item")
	}
	return ToChecklistItemResponse(item), nil
}

func (uc *projectTaskUseCase) ToggleChecklistItem(ctx context.Context, itemID uuid.UUID) (*ChecklistItemResponseDTO, error) {
	item, err := uc.repo.GetChecklistItem(ctx, itemID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get checklist item")
	}
	if item == nil {
		return nil, apperrors.NewNotFound("Checklist item not found")
	}
	item.Done = !item.Done
	if err := uc.repo.UpdateChecklistItem(ctx, item); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to toggle checklist item")
	}
	return ToChecklistItemResponse(item), nil
}

func (uc *projectTaskUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return nil
}
