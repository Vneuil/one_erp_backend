package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/project/domain"
	"github.com/google/uuid"
)

type CreateProjectDTO struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Customer   string  `json:"customer"`
	RABValue   float64 `json:"rabValue"`
	RAPValue   float64 `json:"rapValue"`
	ActualCost float64 `json:"actualCost"`
	Progress   int     `json:"progress"`
	Status     string  `json:"status"`
	Manager    string  `json:"manager"`
}

type UpdateProjectDTO struct {
	Name       *string  `json:"name,omitempty"`
	Customer   *string  `json:"customer,omitempty"`
	RABValue   *float64 `json:"rabValue,omitempty"`
	RAPValue   *float64 `json:"rapValue,omitempty"`
	ActualCost *float64 `json:"actualCost,omitempty"`
	Progress   *int     `json:"progress,omitempty"`
	Status     *string  `json:"status,omitempty"`
	Manager    *string  `json:"manager,omitempty"`
}

type ProjectResponseDTO struct {
	ID         uuid.UUID `json:"id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Customer   string    `json:"customer"`
	RABValue   float64   `json:"rabValue"`
	RAPValue   float64   `json:"rapValue"`
	ActualCost float64   `json:"actualCost"`
	Progress   int       `json:"progress"`
	Status     string    `json:"status"`
	Manager    string    `json:"manager"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func ToProjectResponse(p *domain.Project) *ProjectResponseDTO {
	if p == nil {
		return nil
	}
	return &ProjectResponseDTO{
		ID:         p.ID,
		Code:       p.Code,
		Name:       p.Name,
		Customer:   p.Customer,
		RABValue:   p.RABValue,
		RAPValue:   p.RAPValue,
		ActualCost: p.ActualCost,
		Progress:   p.Progress,
		Status:     p.Status,
		Manager:    p.Manager,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

func ToProjectResponseList(projects []domain.Project) []ProjectResponseDTO {
	result := make([]ProjectResponseDTO, len(projects))
	for i, p := range projects {
		result[i] = *ToProjectResponse(&p)
	}
	return result
}

type CreateTimeEntryDTO struct {
	ProjectCode     string  `json:"projectCode"`
	TaskTitle       string  `json:"taskTitle"`
	WorkerName      string  `json:"workerName"`
	Date            string  `json:"date"`
	DurationMinutes int     `json:"durationMinutes"`
	HourlyRate      float64 `json:"hourlyRate"`
	IsBillable      *bool   `json:"isBillable,omitempty"`
}

type TimeEntryResponseDTO struct {
	ID              uuid.UUID `json:"id"`
	ProjectCode     string    `json:"projectCode"`
	TaskTitle       string    `json:"taskTitle"`
	WorkerName      string    `json:"workerName"`
	Date            string    `json:"date"`
	DurationMinutes int       `json:"durationMinutes"`
	HourlyRate      float64   `json:"hourlyRate"`
	IsBillable      bool      `json:"isBillable"`
	CreatedAt       time.Time `json:"createdAt"`
}

func ToTimeEntryResponse(t *domain.TimeEntry) *TimeEntryResponseDTO {
	if t == nil {
		return nil
	}
	return &TimeEntryResponseDTO{
		ID:              t.ID,
		ProjectCode:     t.ProjectCode,
		TaskTitle:       t.TaskTitle,
		WorkerName:      t.WorkerName,
		Date:            t.Date,
		DurationMinutes: t.DurationMinutes,
		HourlyRate:      t.HourlyRate,
		IsBillable:      t.IsBillable,
		CreatedAt:       t.CreatedAt,
	}
}

func ToTimeEntryResponseList(entries []domain.TimeEntry) []TimeEntryResponseDTO {
	result := make([]TimeEntryResponseDTO, len(entries))
	for i, e := range entries {
		result[i] = *ToTimeEntryResponse(&e)
	}
	return result
}
