package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/project/domain"
	taskdomain "github.com/divinecoid/one-backend/internal/modules/projecttask/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type ProjectUseCase interface {
	Create(ctx context.Context, dto CreateProjectDTO) (*ProjectResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ProjectResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]ProjectResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateProjectDTO) (*ProjectResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	// RecalculateProgress derives real Progress/ActualCost from Kanban
	// tasks and logged timesheets (see modules/projecttask, TimeEntry
	// above) and persists them onto the Project, instead of those fields
	// staying stale manual entries (audit gap #5).
	RecalculateProgress(ctx context.Context, id uuid.UUID) (*ProjectResponseDTO, error)
	SeedInitialData(ctx context.Context) error
}

type projectUseCase struct {
	repo          domain.ProjectRepository
	timeEntryRepo domain.TimeEntryRepository
	taskRepo      taskdomain.ProjectTaskRepository
}

// NewProjectUseCase wires the project repository together with the
// project's own TimeEntryRepository and the ProjectTask repository, so
// RecalculateProgress can compute real progress/cost from Kanban tasks and
// timesheets. timeEntryRepo/taskRepo may be nil, in which case
// RecalculateProgress returns a bad-request error.
func NewProjectUseCase(repo domain.ProjectRepository, timeEntryRepo domain.TimeEntryRepository, taskRepo taskdomain.ProjectTaskRepository) ProjectUseCase {
	return &projectUseCase{repo: repo, timeEntryRepo: timeEntryRepo, taskRepo: taskRepo}
}

func (uc *projectUseCase) Create(ctx context.Context, dto CreateProjectDTO) (*ProjectResponseDTO, error) {
	code := strings.ToUpper(strings.TrimSpace(dto.Code))
	name := strings.TrimSpace(dto.Name)
	customer := strings.TrimSpace(dto.Customer)

	if name == "" || customer == "" {
		return nil, apperrors.NewBadRequest("Project Name and Customer are required")
	}
	// The code is generated when the client does not supply one, so the UI never
	// has to invent (and possibly collide on) project numbers.
	if code == "" {
		count, err := uc.repo.Count(ctx)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to generate project code")
		}
		code = fmt.Sprintf("PRJ-%d-%03d", time.Now().Year(), count+1)
	}

	status := dto.Status
	if status == "" {
		status = "Planning"
	}
	manager := dto.Manager

	project := &domain.Project{
		Code:       code,
		Name:       name,
		Customer:   customer,
		RABValue:   dto.RABValue,
		RAPValue:   dto.RAPValue,
		ActualCost: dto.ActualCost,
		Progress:   dto.Progress,
		Status:     status,
		Manager:    manager,
	}

	if err := uc.repo.Create(ctx, project); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create project")
	}

	return ToProjectResponse(project), nil
}

func (uc *projectUseCase) GetByID(ctx context.Context, id uuid.UUID) (*ProjectResponseDTO, error) {
	project, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve project")
	}
	if project == nil {
		return nil, apperrors.NewNotFound("Project not found")
	}
	return ToProjectResponse(project), nil
}

func (uc *projectUseCase) List(ctx context.Context, query types.PaginationQuery) ([]ProjectResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	projects, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list projects")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToProjectResponseList(projects), meta, nil
}

func (uc *projectUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateProjectDTO) (*ProjectResponseDTO, error) {
	project, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get project")
	}
	if project == nil {
		return nil, apperrors.NewNotFound("Project not found")
	}

	if dto.Name != nil {
		project.Name = *dto.Name
	}
	if dto.Customer != nil {
		project.Customer = *dto.Customer
	}
	if dto.RABValue != nil {
		project.RABValue = *dto.RABValue
	}
	if dto.RAPValue != nil {
		project.RAPValue = *dto.RAPValue
	}
	if dto.ActualCost != nil {
		project.ActualCost = *dto.ActualCost
	}
	if dto.Progress != nil {
		project.Progress = *dto.Progress
	}
	if dto.Status != nil {
		project.Status = *dto.Status
	}
	if dto.Manager != nil {
		project.Manager = *dto.Manager
	}

	if err := uc.repo.Update(ctx, project); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update project")
	}

	return ToProjectResponse(project), nil
}

func (uc *projectUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *projectUseCase) RecalculateProgress(ctx context.Context, id uuid.UUID) (*ProjectResponseDTO, error) {
	if uc.taskRepo == nil || uc.timeEntryRepo == nil {
		return nil, apperrors.NewBadRequest("Recalculation is not available")
	}

	project, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get project")
	}
	if project == nil {
		return nil, apperrors.NewNotFound("Project not found")
	}

	// Progress = completed tasks / total tasks for this project's Kanban
	// board. types.PaginationQuery{PerPage:1} is used purely to keep the
	// query cheap - the accurate count comes back as the second (total)
	// return value regardless of page size.
	countQuery := types.PaginationQuery{Page: 1, PerPage: 1}
	_, totalTasks, err := uc.taskRepo.List(ctx, countQuery, project.Code, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to count project tasks")
	}
	_, doneTasks, err := uc.taskRepo.List(ctx, countQuery, project.Code, "done")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to count completed project tasks")
	}

	progress := 0
	if totalTasks > 0 {
		progress = int((doneTasks * 100) / totalTasks)
	}

	// ActualCost = sum of (timesheet hours x hourly rate) across every
	// time entry logged against this project.
	entries, err := uc.timeEntryRepo.ListAllByProjectCode(ctx, project.Code)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load project time entries")
	}
	var actualCost float64
	for _, e := range entries {
		hours := float64(e.DurationMinutes) / 60.0
		actualCost += hours * e.HourlyRate
	}

	project.Progress = progress
	project.ActualCost = actualCost

	if err := uc.repo.Update(ctx, project); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save recalculated project progress")
	}

	return ToProjectResponse(project), nil
}

func (uc *projectUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	initial := []CreateProjectDTO{
		{
			Code:       "PRJ-2026-001",
			Name:       "Instalasi Sistem Rak Gudang Heavy Duty MM2100",
			Customer:   "PT Graha Konstruksi Nusantara",
			RABValue:   1250000000,
			RAPValue:   950000000,
			ActualCost: 620000000,
			Progress:   65,
			Status:     "In Progress",
			Manager:    "Nicholas Tantra",
		},
		{
			Code:       "PRJ-2026-002",
			Name:       "Pembangunan Conveyor Line Otomasi Sortir Box",
			Customer:   "CV Surya Makmur Logistik",
			RABValue:   850000000,
			RAPValue:   620000000,
			ActualCost: 590000000,
			Progress:   90,
			Status:     "In Progress",
			Manager:    "Ir. Hendra Gunawan",
		},
	}

	for _, item := range initial {
		_, _ = uc.Create(ctx, item)
	}
	return nil
}

type TimeEntryUseCase interface {
	Create(ctx context.Context, dto CreateTimeEntryDTO) (*TimeEntryResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery, projectCode string) ([]TimeEntryResponseDTO, types.PaginationMeta, error)
}

type timeEntryUseCase struct {
	repo domain.TimeEntryRepository
}

func NewTimeEntryUseCase(repo domain.TimeEntryRepository) TimeEntryUseCase {
	return &timeEntryUseCase{repo: repo}
}

func (uc *timeEntryUseCase) Create(ctx context.Context, dto CreateTimeEntryDTO) (*TimeEntryResponseDTO, error) {
	projectCode := strings.TrimSpace(dto.ProjectCode)
	if projectCode == "" {
		return nil, apperrors.NewBadRequest("ProjectCode is required")
	}

	isBillable := true
	if dto.IsBillable != nil {
		isBillable = *dto.IsBillable
	}

	entry := &domain.TimeEntry{
		ProjectCode:     projectCode,
		TaskTitle:       dto.TaskTitle,
		WorkerName:      dto.WorkerName,
		Date:            dto.Date,
		DurationMinutes: dto.DurationMinutes,
		HourlyRate:      dto.HourlyRate,
		IsBillable:      isBillable,
	}

	if err := uc.repo.Create(ctx, entry); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create time entry")
	}

	return ToTimeEntryResponse(entry), nil
}

func (uc *timeEntryUseCase) List(ctx context.Context, query types.PaginationQuery, projectCode string) ([]TimeEntryResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	entries, total, err := uc.repo.List(ctx, query, projectCode)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list time entries")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToTimeEntryResponseList(entries), meta, nil
}
