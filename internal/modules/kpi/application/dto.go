package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/kpi/domain"
	"github.com/google/uuid"
)

type CreateKpiReviewDTO struct {
	EmployeeName  string  `json:"employeeName"`
	Department    string  `json:"department"`
	Period        string  `json:"period"`
	TargetScore   float64 `json:"targetScore"`
	ActualScore   float64 `json:"actualScore"`
	WeightFormula string  `json:"weightFormula"`
	Evaluator     string  `json:"evaluator"`
}

type UpdateKpiStatusDTO struct {
	Status string `json:"status"`
}

type KpiReviewResponseDTO struct {
	ID            uuid.UUID `json:"id"`
	EmployeeName  string    `json:"employeeName"`
	Department    string    `json:"department"`
	Period        string    `json:"period"`
	TargetScore   float64   `json:"targetScore"`
	ActualScore   float64   `json:"actualScore"`
	WeightFormula string    `json:"weightFormula"`
	Grade         string    `json:"grade"`
	Status        string    `json:"status"`
	Evaluator     string    `json:"evaluator"`
	CreatedAt     time.Time `json:"createdAt"`
}

func ToKpiReviewResponse(r *domain.KpiReview) *KpiReviewResponseDTO {
	if r == nil {
		return nil
	}
	return &KpiReviewResponseDTO{
		ID:            r.ID,
		EmployeeName:  r.EmployeeName,
		Department:    r.Department,
		Period:        r.Period,
		TargetScore:   r.TargetScore,
		ActualScore:   r.ActualScore,
		WeightFormula: r.WeightFormula,
		Grade:         r.Grade,
		Status:        r.Status,
		Evaluator:     r.Evaluator,
		CreatedAt:     r.CreatedAt,
	}
}

func ToKpiReviewResponseList(items []domain.KpiReview) []KpiReviewResponseDTO {
	result := make([]KpiReviewResponseDTO, len(items))
	for i, item := range items {
		result[i] = *ToKpiReviewResponse(&item)
	}
	return result
}
