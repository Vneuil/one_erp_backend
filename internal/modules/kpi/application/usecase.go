package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/kpi/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type KpiUseCase interface {
	CreateKpiReview(ctx context.Context, dto CreateKpiReviewDTO) (*KpiReviewResponseDTO, error)
	ListKpiReviews(ctx context.Context, query types.PaginationQuery) ([]KpiReviewResponseDTO, types.PaginationMeta, error)
	UpdateKpiStatus(ctx context.Context, id uuid.UUID, dto UpdateKpiStatusDTO) (*KpiReviewResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type kpiUseCase struct {
	repo domain.KpiRepository
}

func NewKpiUseCase(repo domain.KpiRepository) KpiUseCase {
	return &kpiUseCase{repo: repo}
}

// computeGrade derives a letter grade from the ratio of actual to target
// score: >=100% -> A, >=85% -> B, >=70% -> C, else D.
func computeGrade(target, actual float64) string {
	if target <= 0 {
		return "D"
	}
	ratio := actual / target

	switch {
	case ratio >= 1.0:
		return "A"
	case ratio >= 0.85:
		return "B"
	case ratio >= 0.70:
		return "C"
	default:
		return "D"
	}
}

func (uc *kpiUseCase) CreateKpiReview(ctx context.Context, dto CreateKpiReviewDTO) (*KpiReviewResponseDTO, error) {
	name := strings.TrimSpace(dto.EmployeeName)
	if name == "" {
		return nil, apperrors.NewBadRequest("Employee Name is required")
	}
	if dto.Period == "" {
		return nil, apperrors.NewBadRequest("Period is required")
	}
	if dto.TargetScore <= 0 || dto.TargetScore > 100 {
		return nil, apperrors.NewBadRequest("Target score must be greater than 0 and at most 100")
	}
	if dto.ActualScore < 0 || dto.ActualScore > 100 {
		return nil, apperrors.NewBadRequest("Actual score must be between 0 and 100")
	}

	review := &domain.KpiReview{
		EmployeeName:  name,
		Department:    dto.Department,
		Period:        dto.Period,
		TargetScore:   dto.TargetScore,
		ActualScore:   dto.ActualScore,
		WeightFormula: dto.WeightFormula,
		Grade:         computeGrade(dto.TargetScore, dto.ActualScore),
		Status:        "draft",
		Evaluator:     dto.Evaluator,
	}

	if err := uc.repo.Create(ctx, review); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create KPI review")
	}

	return ToKpiReviewResponse(review), nil
}

func (uc *kpiUseCase) ListKpiReviews(ctx context.Context, query types.PaginationQuery) ([]KpiReviewResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list KPI reviews")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToKpiReviewResponseList(items), meta, nil
}

var validKpiTransitions = map[string]string{
	"draft":     "in_review",
	"in_review": "finalized",
}

func (uc *kpiUseCase) UpdateKpiStatus(ctx context.Context, id uuid.UUID, dto UpdateKpiStatusDTO) (*KpiReviewResponseDTO, error) {
	review, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get KPI review")
	}
	if review == nil {
		return nil, apperrors.NewNotFound("KPI review not found")
	}

	switch dto.Status {
	case "draft", "in_review", "finalized":
		// allowed values
	default:
		return nil, apperrors.NewBadRequest("Invalid status value")
	}
	// Reviews only move forward one step at a time; a finalized review is final.
	if validKpiTransitions[review.Status] != dto.Status {
		return nil, apperrors.NewBadRequest("Invalid status transition from " + review.Status + " to " + dto.Status)
	}

	review.Status = dto.Status

	if err := uc.repo.Update(ctx, review); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update KPI review status")
	}

	return ToKpiReviewResponse(review), nil
}

func (uc *kpiUseCase) SeedInitialData(ctx context.Context) error {
	// No seed data required for the kpi module.
	return nil
}
