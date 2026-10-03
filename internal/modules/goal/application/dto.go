package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/goal/domain"
	"github.com/google/uuid"
)

type CreateGoalDTO struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	GoalType    string     `json:"goalType"`
	OwnerName   string     `json:"ownerName"`
	Category    string     `json:"category"`
	TargetValue float64    `json:"targetValue"`
	Unit        string     `json:"unit"`
	PeriodStart string     `json:"periodStart"`
	PeriodEnd   string     `json:"periodEnd"`
	CompanyID   *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateGoalDTO struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	GoalType    string   `json:"goalType"`
	OwnerName   string   `json:"ownerName"`
	Category    string   `json:"category"`
	TargetValue *float64 `json:"targetValue,omitempty"`
	Unit        string   `json:"unit"`
	PeriodStart string   `json:"periodStart"`
	PeriodEnd   string   `json:"periodEnd"`
}

type CheckInDTO struct {
	ValueRecorded float64 `json:"valueRecorded"`
	Note          string  `json:"note"`
	CheckedInDate string  `json:"checkedInDate"`
}

type GoalResponseDTO struct {
	ID              uuid.UUID  `json:"id"`
	CompanyID       *uuid.UUID `json:"companyId,omitempty"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	GoalType        string     `json:"goalType"`
	OwnerName       string     `json:"ownerName"`
	Category        string     `json:"category"`
	TargetValue     float64    `json:"targetValue"`
	CurrentValue    float64    `json:"currentValue"`
	Unit            string     `json:"unit"`
	PeriodStart     string     `json:"periodStart"`
	PeriodEnd       string     `json:"periodEnd"`
	Status          string     `json:"status"`
	ProgressPercent float64    `json:"progressPercent"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type CheckInResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	GoalID        uuid.UUID `json:"goalId"`
	ValueRecorded float64   `json:"valueRecorded"`
	Note          string    `json:"note"`
	CheckedInDate string    `json:"checkedInDate"`
	CreatedAt     time.Time `json:"createdAt"`
}

func ToGoalResponse(g *domain.Goal) *GoalResponseDTO {
	if g == nil {
		return nil
	}
	return &GoalResponseDTO{
		ID:              g.ID,
		CompanyID:       g.CompanyID,
		Title:           g.Title,
		Description:     g.Description,
		GoalType:        g.GoalType,
		OwnerName:       g.OwnerName,
		Category:        g.Category,
		TargetValue:     g.TargetValue,
		CurrentValue:    g.CurrentValue,
		Unit:            g.Unit,
		PeriodStart:     g.PeriodStart,
		PeriodEnd:       g.PeriodEnd,
		Status:          g.Status,
		ProgressPercent: g.ProgressPercent,
		CreatedAt:       g.CreatedAt,
	}
}

func ToGoalResponseList(goals []domain.Goal) []GoalResponseDTO {
	result := make([]GoalResponseDTO, len(goals))
	for i, g := range goals {
		result[i] = *ToGoalResponse(&g)
	}
	return result
}

func ToCheckInResponse(ci *domain.GoalCheckIn) *CheckInResponseDTO {
	if ci == nil {
		return nil
	}
	return &CheckInResponseDTO{
		ID:            ci.ID,
		GoalID:        ci.GoalID,
		ValueRecorded: ci.ValueRecorded,
		Note:          ci.Note,
		CheckedInDate: ci.CheckedInDate,
		CreatedAt:     ci.CreatedAt,
	}
}
