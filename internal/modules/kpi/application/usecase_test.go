package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/kpi/domain"
	"github.com/google/uuid"
)

type fakeKpiRepo struct {
	domain.KpiRepository
	review *domain.KpiReview
}

func (f *fakeKpiRepo) Create(_ context.Context, r *domain.KpiReview) error {
	r.ID = uuid.New()
	f.review = r
	return nil
}
func (f *fakeKpiRepo) GetByID(context.Context, uuid.UUID) (*domain.KpiReview, error) {
	return f.review, nil
}
func (f *fakeKpiRepo) Update(context.Context, *domain.KpiReview) error { return nil }

func TestKpiValidationAndForwardOnlyTransitions(t *testing.T) {
	repo := &fakeKpiRepo{}
	uc := NewKpiUseCase(repo)
	ctx := context.Background()

	for _, dto := range []CreateKpiReviewDTO{
		{EmployeeName: "A", Period: "Q3", TargetScore: 0, ActualScore: 50},
		{EmployeeName: "A", Period: "Q3", TargetScore: 85, ActualScore: 150},
		{EmployeeName: "A", Period: "Q3", TargetScore: 85, ActualScore: -1},
	} {
		if _, err := uc.CreateKpiReview(ctx, dto); err == nil {
			t.Errorf("expected %+v to be rejected", dto)
		}
	}

	res, err := uc.CreateKpiReview(ctx, CreateKpiReviewDTO{EmployeeName: "A", Period: "Q3", TargetScore: 85, ActualScore: 90})
	if err != nil || res.Grade != "A" || res.Status != "draft" {
		t.Fatalf("create: %+v err=%v", res, err)
	}

	if _, err := uc.UpdateKpiStatus(ctx, res.ID, UpdateKpiStatusDTO{Status: "finalized"}); err == nil {
		t.Fatal("draft must not jump straight to finalized")
	}
	if _, err := uc.UpdateKpiStatus(ctx, res.ID, UpdateKpiStatusDTO{Status: "in_review"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.UpdateKpiStatus(ctx, res.ID, UpdateKpiStatusDTO{Status: "finalized"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.UpdateKpiStatus(ctx, res.ID, UpdateKpiStatusDTO{Status: "draft"}); err == nil {
		t.Fatal("a finalized review must not be reopened")
	}
}
