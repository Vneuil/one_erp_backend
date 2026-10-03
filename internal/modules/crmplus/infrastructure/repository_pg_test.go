package infrastructure

import (
	"context"
	"os"
	"testing"
	"time"

	crmdomain "github.com/divinecoid/one-backend/internal/modules/crm/domain"
	crminfra "github.com/divinecoid/one-backend/internal/modules/crm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/application"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when CRMPLUS_TEST_DSN is set (a scratch database).
func TestCRMPlusAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("CRMPLUS_TEST_DSN")
	if dsn == "" {
		t.Skip("CRMPLUS_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&crmdomain.Lead{}, &crmdomain.Deal{}, &domain.Interaction{}, &domain.SalesTask{}, &domain.Document{}, &domain.Tag{},
		&domain.TagAssignment{}, &domain.Contact{}, &domain.PipelineStage{}, &domain.LeadForm{}, &domain.LeadFormIndex{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewRepository(db)
	crm := crminfra.NewCRMRepository(db)
	svc := application.NewService(repo, crm, nil, nil)
	if err := svc.SeedDefaultStages(ctx); err != nil {
		t.Fatal(err)
	}
	if err := svc.SeedDefaultStages(ctx); err != nil { // idempotent
		t.Fatal(err)
	}
	if n, _ := repo.CountStages(ctx); n != 5 {
		t.Fatalf("stages after double seed: %d", n)
	}

	lead := &crmdomain.Lead{Name: "Budi", Company: "PT Maju", Status: "New"}
	if err := crm.Create(ctx, lead); err != nil {
		t.Fatal(err)
	}
	deal := &crmdomain.Deal{Title: "Rak", Customer: "PT Maju", Value: 1000, Stage: "won", Probability: 100, PIC: "ani@x.com"}
	if err := crm.CreateDeal(ctx, deal); err != nil {
		t.Fatal(err)
	}

	// Tags: assigning twice keeps one row; the join returns the tag; deleting the tag clears assignments.
	tag, err := svc.CreateTag(ctx, "VIP", "red")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := svc.SetTag(ctx, tag.ID, "lead", lead.ID, true); err != nil {
			t.Fatal(err)
		}
	}
	if tags, _ := svc.TagsFor(ctx, "lead", lead.ID); len(tags) != 1 || tags[0].Name != "VIP" {
		t.Fatalf("tags for lead: %+v", tags)
	}
	if ids, _ := svc.ParentsWithTag(ctx, tag.ID, "lead"); len(ids) != 1 || ids[0] != lead.ID {
		t.Fatalf("parents with tag: %v", ids)
	}
	if err := svc.SetTag(ctx, tag.ID, "lead", lead.ID, false); err != nil {
		t.Fatal(err)
	}
	if tags, _ := svc.TagsFor(ctx, "lead", lead.ID); len(tags) != 0 {
		t.Fatalf("tag not removed: %+v", tags)
	}

	// Last interaction per parent.
	old, recent := time.Now().Add(-72*time.Hour), time.Now().Add(-1*time.Hour)
	for _, at := range []time.Time{old, recent} {
		at := at
		if _, err := svc.LogInteraction(ctx, application.Caller{Email: "ani@x.com"}, application.InteractionInput{ParentType: "deal", ParentID: deal.ID, Kind: "call", Summary: "x", OccurredAt: &at}); err != nil {
			t.Fatal(err)
		}
	}
	last, err := repo.LastInteractionByParent(ctx, "deal")
	if err != nil || last[deal.ID].Sub(recent) > time.Second || recent.Sub(last[deal.ID]) > time.Second {
		t.Fatalf("last interaction: %v %v", last, err)
	}

	// Tasks: assignee filter is case-insensitive.
	if _, err := svc.CreateTask(ctx, application.Caller{Email: "Ani@X.com"}, application.TaskInput{Title: "Call", DueDate: "2026-10-01"}); err != nil {
		t.Fatal(err)
	}
	if tasks, _ := svc.ListTasks(ctx, "open", "ani@x.com", "", nil); len(tasks) != 1 {
		t.Fatalf("tasks: %d", len(tasks))
	}

	// Analytics end to end.
	a, err := svc.GetAnalytics(ctx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Totals.Won != 1 || a.Totals.WonValue != 1000 || len(a.Funnel) != 5 || a.Tasks.Open != 1 {
		t.Fatalf("analytics: %+v", a.Totals)
	}

	// Control-plane registry upsert / lookup / delete.
	reg := NewFormRegistry(db)
	company := uuid.New()
	if err := reg.Upsert(ctx, "abc", company, nil); err != nil {
		t.Fatal(err)
	}
	other := uuid.New()
	if err := reg.Upsert(ctx, "abc", other, nil); err != nil {
		t.Fatal(err)
	}
	if idx, _ := reg.Lookup(ctx, "abc"); idx == nil || idx.CompanyID != other {
		t.Fatalf("lookup after upsert: %+v", idx)
	}
	_ = reg.Delete(ctx, "abc")
	if idx, _ := reg.Lookup(ctx, "abc"); idx != nil {
		t.Fatal("deleted key must not resolve")
	}

	// A form's public submission creates a lead in this database.
	form, err := svc.CreateLeadForm(ctx, reg, company, nil, application.LeadFormInput{Name: "Kontak"})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetLeadFormByKey(ctx, form.Key)
	if got == nil || !got.IsActive || len(form.Key) != 24 {
		t.Fatalf("form by key: %+v", got)
	}
	if err := svc.AcceptSubmission(ctx, got, application.Submission{Name: "Sari", Email: "sari@x.com", Message: "Halo"}); err != nil {
		t.Fatal(err)
	}
	leads, _, _ := crm.List(ctx, crmTypes())
	found := false
	for _, l := range leads {
		if l.Name == "Sari" && l.Source == "web" {
			found = true
		}
	}
	if !found {
		t.Fatal("submission did not create a lead")
	}
	if f, _ := repo.GetLeadFormByKey(ctx, form.Key); f.Submissions != 1 {
		t.Fatalf("submission counter: %d", f.Submissions)
	}
}

func crmTypes() types.PaginationQuery { return types.PaginationQuery{Page: 1, PerPage: 100} }
