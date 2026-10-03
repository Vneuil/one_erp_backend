package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/google/uuid"
)

type dealRepo struct {
	domain.CRMRepository
	deal *domain.Deal
}

func (r *dealRepo) CreateDeal(_ context.Context, d *domain.Deal) error {
	d.ID = uuid.New()
	r.deal = d
	return nil
}
func (r *dealRepo) GetDealByID(context.Context, uuid.UUID) (*domain.Deal, error) { return r.deal, nil }
func (r *dealRepo) UpdateDeal(context.Context, *domain.Deal) error               { return nil }

type customStages []StageInfo

func (c customStages) ActiveStages(context.Context) ([]StageInfo, error) { return c, nil }

func TestLostDealNeedsAReasonAndClosedAtTracksSettling(t *testing.T) {
	repo := &dealRepo{}
	uc := NewCRMUseCase(repo)
	ctx := context.Background()
	d, err := uc.CreateDeal(ctx, CreateDealDTO{Title: "Rak", Customer: "PT A"})
	if err != nil {
		t.Fatal(err)
	}
	if d.ClosedAt != nil || d.Stage != "discovery" {
		t.Fatalf("new deal: %+v", d)
	}
	if _, err := uc.UpdateDealStage(ctx, d.ID, UpdateDealStageDTO{Stage: "lost"}); err == nil {
		t.Fatal("losing a deal without a reason must be refused")
	}
	if repo.deal.Stage == "lost" {
		t.Fatal("a refused change must not be applied")
	}
	got, err := uc.UpdateDealStage(ctx, d.ID, UpdateDealStageDTO{Stage: "lost", LostReason: " Harga "})
	if err != nil || got.LostReason != "Harga" || got.ClosedAt == nil || got.Probability != 0 {
		t.Fatalf("lost: %+v %v", got, err)
	}
	// Reopening clears the close date and the stale reason.
	got, err = uc.UpdateDealStage(ctx, d.ID, UpdateDealStageDTO{Stage: "negotiation"})
	if err != nil || got.ClosedAt != nil || got.LostReason != "" || got.Probability != 80 {
		t.Fatalf("reopened: %+v %v", got, err)
	}
	if _, err := uc.UpdateDealStage(ctx, d.ID, UpdateDealStageDTO{Stage: "nonsense"}); err == nil {
		t.Fatal("unknown stage must be refused")
	}
}

func TestConfiguredStagesReplaceTheBuiltInOnes(t *testing.T) {
	repo := &dealRepo{}
	uc := NewCRMUseCase(repo, WithStages(customStages{
		{"survey", 20, "open"}, {"tender", 55, "open"}, {"signed", 100, "won"}, {"dropped", 0, "lost"},
	}))
	ctx := context.Background()
	d, err := uc.CreateDeal(ctx, CreateDealDTO{Title: "Gedung", Customer: "PT B"})
	if err != nil || d.Stage != "survey" || d.Probability != 20 {
		t.Fatalf("default stage should be the first configured one: %+v %v", d, err)
	}
	if _, err := uc.UpdateDealStage(ctx, d.ID, UpdateDealStageDTO{Stage: "negotiation"}); err == nil {
		t.Fatal("a built-in stage that is not configured must be refused")
	}
	got, err := uc.UpdateDealStage(ctx, d.ID, UpdateDealStageDTO{Stage: "signed"})
	if err != nil || got.Probability != 100 || got.ClosedAt == nil {
		t.Fatalf("custom won stage: %+v %v", got, err)
	}
	if _, err := uc.CreateDeal(ctx, CreateDealDTO{Title: "X", Customer: "Y", Stage: "won"}); err == nil {
		t.Fatal("creating a deal in an unconfigured stage must be refused")
	}
}

func TestNewDealsAreAssignedToTheCallerNotAFakeName(t *testing.T) {
	uc := NewCRMUseCase(&dealRepo{})
	d, err := uc.CreateDeal(actor.WithEmail(context.Background(), "ani@x.com"), CreateDealDTO{Title: "T", Customer: "C"})
	if err != nil || d.PIC != "ani@x.com" {
		t.Fatalf("PIC should default to the caller: %+v %v", d, err)
	}
	d, _ = NewCRMUseCase(&dealRepo{}).CreateDeal(context.Background(), CreateDealDTO{Title: "T", Customer: "C"})
	if d.PIC != "" {
		t.Fatalf("with no caller the deal is unassigned, got %q", d.PIC)
	}
}
