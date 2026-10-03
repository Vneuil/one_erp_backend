package application

import (
	"context"
	"strings"
	"testing"
	"time"

	crmdomain "github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type fakeRepo struct {
	domain.Repository
	tasks    map[uuid.UUID]*domain.SalesTask
	tags     []domain.Tag
	contacts []domain.Contact
	stages   []domain.PipelineStage
	inter    []domain.Interaction
	forms    []domain.LeadForm
	submits  int
}

func newFakeRepo() *fakeRepo { return &fakeRepo{tasks: map[uuid.UUID]*domain.SalesTask{}} }

func (f *fakeRepo) CreateTask(_ context.Context, v *domain.SalesTask) error {
	v.ID = uuid.New()
	f.tasks[v.ID] = v
	return nil
}
func (f *fakeRepo) GetTask(_ context.Context, id uuid.UUID) (*domain.SalesTask, error) {
	return f.tasks[id], nil
}
func (f *fakeRepo) UpdateTask(context.Context, *domain.SalesTask) error { return nil }
func (f *fakeRepo) ListTags(context.Context) ([]domain.Tag, error)      { return f.tags, nil }
func (f *fakeRepo) CreateTag(_ context.Context, v *domain.Tag) error {
	f.tags = append(f.tags, *v)
	return nil
}
func (f *fakeRepo) CreateContact(_ context.Context, v *domain.Contact) error {
	v.ID = uuid.New()
	f.contacts = append(f.contacts, *v)
	return nil
}
func (f *fakeRepo) ListContacts(_ context.Context, company string, _ *uuid.UUID) ([]domain.Contact, error) {
	var out []domain.Contact
	for _, c := range f.contacts {
		if strings.EqualFold(c.CompanyName, company) {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeRepo) UpdateContact(_ context.Context, v *domain.Contact) error {
	for i := range f.contacts {
		if f.contacts[i].ID == v.ID {
			f.contacts[i] = *v
		}
	}
	return nil
}
func (f *fakeRepo) ListStages(_ context.Context, activeOnly bool) ([]domain.PipelineStage, error) {
	var out []domain.PipelineStage
	for _, s := range f.stages {
		if !activeOnly || s.IsActive {
			out = append(out, s)
		}
	}
	return out, nil
}
func (f *fakeRepo) CountStages(context.Context) (int64, error) { return int64(len(f.stages)), nil }
func (f *fakeRepo) CreateStage(_ context.Context, v *domain.PipelineStage) error {
	v.ID = uuid.New()
	f.stages = append(f.stages, *v)
	return nil
}
func (f *fakeRepo) GetStage(_ context.Context, id uuid.UUID) (*domain.PipelineStage, error) {
	for i := range f.stages {
		if f.stages[i].ID == id {
			c := f.stages[i] // a copy, like a real repository returns
			return &c, nil
		}
	}
	return nil, nil
}
func (f *fakeRepo) UpdateStage(_ context.Context, v *domain.PipelineStage) error {
	for i := range f.stages {
		if f.stages[i].ID == v.ID {
			f.stages[i] = *v
		}
	}
	return nil
}
func (f *fakeRepo) DeleteStage(_ context.Context, id uuid.UUID) error {
	for i := range f.stages {
		if f.stages[i].ID == id {
			f.stages = append(f.stages[:i], f.stages[i+1:]...)
			return nil
		}
	}
	return nil
}
func (f *fakeRepo) CreateInteraction(_ context.Context, v *domain.Interaction) error {
	f.inter = append(f.inter, *v)
	return nil
}
func (f *fakeRepo) IncrementSubmissions(context.Context, uuid.UUID) error { f.submits++; return nil }

type fakeCRM struct {
	crmdomain.CRMRepository
	leads []*crmdomain.Lead
	deals []crmdomain.Deal
}

func (f *fakeCRM) GetByID(context.Context, uuid.UUID) (*crmdomain.Lead, error) {
	return &crmdomain.Lead{Company: "PT Maju"}, nil
}
func (f *fakeCRM) Create(_ context.Context, l *crmdomain.Lead) error {
	l.ID = uuid.New()
	f.leads = append(f.leads, l)
	return nil
}
func (f *fakeCRM) ListDeals(context.Context, types.PaginationQuery) ([]crmdomain.Deal, int64, error) {
	return f.deals, int64(len(f.deals)), nil
}

type fakeNotifier struct{ sent []string }

func (n *fakeNotifier) Notify(_ context.Context, email, title, _, _ string) error {
	n.sent = append(n.sent, email+"|"+title)
	return nil
}

func newSvc() (*Service, *fakeRepo, *fakeCRM, *fakeNotifier) {
	r, c, n := newFakeRepo(), &fakeCRM{}, &fakeNotifier{}
	s := NewService(r, c, n, nil)
	s.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, zone) }
	return s, r, c, n
}

func TestStageKey(t *testing.T) {
	for in, want := range map[string]string{"Site Survey": "site-survey", "  Tender / RFP!! ": "tender-rfp", "Won": "won", "***": ""} {
		if got := StageKey(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestBucketizeSplitsAroundToday(t *testing.T) {
	b := Bucketize([]domain.SalesTask{
		{Status: "open", DueDate: "2026-09-29"}, {Status: "open", DueDate: "2026-09-30"}, {Status: "open", DueDate: "2026-10-01"}, {Status: "done", DueDate: "2026-09-01"},
	}, "2026-09-30")
	if len(b.Overdue) != 1 || len(b.Today) != 1 || len(b.Upcoming) != 1 {
		t.Fatalf("%+v", b)
	}
}

func TestTaskAssignmentNotifiesOnlyOthersAndRestrictsCompletion(t *testing.T) {
	s, _, _, n := newSvc()
	ctx := context.Background()
	own, err := s.CreateTask(ctx, Caller{Email: "ani@x.com"}, TaskInput{Title: "Call", DueDate: "2026-10-02"})
	if err != nil || own.AssigneeEmail != "ani@x.com" || len(n.sent) != 0 {
		t.Fatalf("self-assigned task must not notify: %+v %v %v", own, err, n.sent)
	}
	other, err := s.CreateTask(ctx, Caller{Email: "ani@x.com"}, TaskInput{Title: "Send quote", DueDate: "2026-10-02", AssigneeEmail: "bob@x.com"})
	if err != nil || len(n.sent) != 1 || !strings.HasPrefix(n.sent[0], "bob@x.com|") {
		t.Fatalf("assignee should be notified: %v %v", n.sent, err)
	}
	if _, err := s.SetTaskDone(ctx, Caller{Email: "eve@x.com"}, other.ID, true); err == nil {
		t.Fatal("a stranger must not complete someone else's task")
	}
	if _, err := s.SetTaskDone(ctx, Caller{Email: "BOB@x.com"}, other.ID, true); err != nil {
		t.Fatalf("assignee can complete: %v", err)
	}
	if other.Status != "done" || other.CompletedAt == nil {
		t.Fatalf("not completed: %+v", other)
	}
	if _, err := s.SetTaskDone(ctx, Caller{Email: "hr@x.com", Privileged: true}, other.ID, false); err != nil || other.Status != "open" {
		t.Fatalf("privileged reopen failed: %v %s", err, other.Status)
	}
	for _, bad := range []TaskInput{{Title: "", DueDate: "2026-10-02"}, {Title: "x", DueDate: "02-10-2026"}, {Title: "x", DueDate: "2026-10-02", Priority: "urgent"},
		{Title: "x", DueDate: "2026-10-02", AssigneeEmail: "not-an-email"}, {Title: "x", DueDate: "2026-10-02", ParentType: "lead"}} {
		if _, err := s.CreateTask(ctx, Caller{Email: "ani@x.com"}, bad); err == nil {
			t.Errorf("should be rejected: %+v", bad)
		}
	}
}

func TestTagNamesAreUniqueIgnoringCase(t *testing.T) {
	s, _, _, _ := newSvc()
	ctx := context.Background()
	if _, err := s.CreateTag(ctx, "VIP", "red"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTag(ctx, "vip", "blue"); err == nil {
		t.Fatal("duplicate tag differing by case must be rejected")
	}
	if _, err := s.CreateTag(ctx, "New", "neon"); err == nil {
		t.Fatal("unknown colour must be rejected")
	}
}

func TestOnlyOnePrimaryContactPerCompany(t *testing.T) {
	s, r, _, _ := newSvc()
	ctx := context.Background()
	a, _ := s.CreateContact(ctx, ContactInput{Name: "A", CompanyName: "PT Maju", IsPrimary: true})
	b, err := s.CreateContact(ctx, ContactInput{Name: "B", CompanyName: "pt maju", IsPrimary: true})
	if err != nil {
		t.Fatal(err)
	}
	primaries := 0
	for _, c := range r.contacts {
		if c.IsPrimary {
			primaries++
			if c.ID != b.ID {
				t.Fatalf("the newer primary should win, got %s (a=%s)", c.Name, a.Name)
			}
		}
	}
	if primaries != 1 {
		t.Fatalf("%d primary contacts", primaries)
	}
	if _, err := s.CreateContact(ctx, ContactInput{Name: "C", Email: "bad", CompanyName: "X"}); err == nil {
		t.Fatal("invalid email must be rejected")
	}
	// A lead id supplies the company name.
	lid := uuid.New()
	if c, err := s.CreateContact(ctx, ContactInput{Name: "D", LeadID: &lid}); err != nil || c.CompanyName != "PT Maju" {
		t.Fatalf("company from lead: %+v %v", c, err)
	}
}

func TestPipelineNeedsWonAndLostStages(t *testing.T) {
	s, r, c, _ := newSvc()
	ctx := context.Background()
	if err := s.SeedDefaultStages(ctx); err != nil || len(r.stages) != 5 {
		t.Fatalf("seed: %v %d", err, len(r.stages))
	}
	if err := s.SeedDefaultStages(ctx); err != nil || len(r.stages) != 5 {
		t.Fatalf("seeding twice must not duplicate: %d", len(r.stages))
	}
	var won, lost, disc domain.PipelineStage
	for _, st := range r.stages {
		switch st.Key {
		case "won":
			won = st
		case "lost":
			lost = st
		case "discovery":
			disc = st
		}
	}
	off := false
	if _, err := s.UpdateStage(ctx, won.ID, StageInput{IsActive: &off}, nil); err == nil {
		t.Fatal("deactivating the only won stage must be refused")
	}
	if err := s.DeleteStage(ctx, lost.ID); err == nil {
		t.Fatal("deleting the only lost stage must be refused")
	}
	// A stage still used by a deal cannot be deleted.
	c.deals = []crmdomain.Deal{{Stage: "discovery"}}
	if err := s.DeleteStage(ctx, disc.ID); err == nil {
		t.Fatal("a stage in use must not be deleted")
	}
	c.deals = nil
	if err := s.DeleteStage(ctx, disc.ID); err != nil {
		t.Fatalf("unused stage should delete: %v", err)
	}
	// New custom stage.
	st, err := s.CreateStage(ctx, StageInput{Name: "Site Survey", Probability: 45})
	if err != nil || st.Key != "site-survey" || st.Kind != "open" || st.Position != 6 {
		t.Fatalf("custom stage: %+v %v", st, err)
	}
	if _, err := s.CreateStage(ctx, StageInput{Name: "site survey", Probability: 10}); err == nil {
		t.Fatal("duplicate stage name must be rejected")
	}
	if _, err := s.CreateStage(ctx, StageInput{Name: "Signed", Kind: "won", Probability: 90}); err == nil {
		t.Fatal("a won stage must have probability 100")
	}
	// The configured stages feed the crm module.
	infos, _ := s.ActiveStages(ctx)
	if len(infos) != 5 { // 4 seeded remaining + custom
		t.Fatalf("active stages: %+v", infos)
	}
}

func TestPublicSubmissionValidationAndHoneypot(t *testing.T) {
	s, r, c, n := newSvc()
	ctx := context.Background()
	form := &domain.LeadForm{Name: "Kontak", Source: "web-form", DefaultPIC: "sales@x.com", IsActive: true}
	form.ID = uuid.New()

	// Honeypot filled: accepted silently, nothing stored.
	if err := s.AcceptSubmission(ctx, form, Submission{Name: "Bot", Email: "b@x.com", Honeypot: "http://spam"}); err != nil || len(c.leads) != 0 {
		t.Fatalf("honeypot: %v leads=%d", err, len(c.leads))
	}
	for _, bad := range []Submission{{}, {Name: "A"}, {Name: "A", Email: "nope"}, {Name: "A", Phone: "abc"}, {Name: strings.Repeat("x", 300), Email: "a@x.com"}} {
		if err := s.AcceptSubmission(ctx, form, bad); err == nil {
			t.Errorf("should be rejected: %+v", bad)
		}
	}
	if len(c.leads) != 0 {
		t.Fatal("rejected submissions must not create leads")
	}
	if err := s.AcceptSubmission(ctx, form, Submission{Name: " Budi ", Phone: "+62 812-3456", Message: "Butuh penawaran"}); err != nil {
		t.Fatal(err)
	}
	l := c.leads[0]
	if l.Name != "Budi" || l.Company != "Budi" || l.Source != "web-form" || l.Status != "New" || l.PIC != "sales@x.com" {
		t.Fatalf("lead: %+v", l)
	}
	if len(r.inter) != 1 || !strings.Contains(r.inter[0].Summary, "Butuh penawaran") || r.inter[0].ParentID != l.ID {
		t.Fatalf("message note: %+v", r.inter)
	}
	if r.submits != 1 || len(n.sent) != 1 || !strings.HasPrefix(n.sent[0], "sales@x.com|") {
		t.Fatalf("counter %d notified %v", r.submits, n.sent)
	}
	form.IsActive = false
	if err := s.AcceptSubmission(ctx, form, Submission{Name: "X", Email: "x@x.com"}); err == nil {
		t.Fatal("an inactive form must not accept submissions")
	}
}

func TestInteractionValidation(t *testing.T) {
	s, r, _, _ := newSvc()
	ctx := context.Background()
	id := uuid.New()
	if _, err := s.LogInteraction(ctx, Caller{Email: "a@x.com"}, InteractionInput{ParentType: "lead", ParentID: id, Kind: "call", Summary: "Discussed pricing"}); err != nil || len(r.inter) != 1 {
		t.Fatalf("valid interaction: %v", err)
	}
	future := time.Date(2026, 10, 30, 0, 0, 0, 0, zone)
	for _, bad := range []InteractionInput{
		{ParentType: "lead", ParentID: id, Kind: "telepathy", Summary: "x"},
		{ParentType: "lead", ParentID: id, Kind: "call", Summary: "  "},
		{ParentType: "invoice", ParentID: id, Kind: "call", Summary: "x"},
		{ParentType: "lead", ParentID: id, Kind: "call", Summary: "x", OccurredAt: &future},
	} {
		if _, err := s.LogInteraction(ctx, Caller{}, bad); err == nil {
			t.Errorf("should be rejected: %+v", bad)
		}
	}
}
