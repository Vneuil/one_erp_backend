package application

import (
	"context"
	"strings"
	"testing"
	"time"

	projectDomain "github.com/divinecoid/one-backend/internal/modules/project/domain"
	"github.com/divinecoid/one-backend/internal/modules/projectcost/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/google/uuid"
)

func TestValidateBudgetLinesReportsEveryBadRow(t *testing.T) {
	items, errs := ValidateBudgetLines([]BudgetLineInput{
		{Kind: "RAB", Category: " Material ", Description: "Semen", Quantity: 10, Unit: "sak", UnitPrice: 65_000},
		{Kind: "rap", Category: "Tenaga", Description: "Mandor", Quantity: 2.5, UnitPrice: 300_000.555},
		{Kind: "boq", Category: "X", Description: "x", Quantity: 1, UnitPrice: 1},
		{Kind: "rab", Category: "", Description: "x", Quantity: 1, UnitPrice: 1},
		{Kind: "rab", Category: "X", Description: "x", Quantity: 0, UnitPrice: 1},
		{Kind: "rab", Category: "X", Description: "x", Quantity: 1, UnitPrice: -5},
	})
	if len(items) != 2 || items[0].Kind != "rab" || items[0].Category != "Material" || items[0].Amount != 650_000 {
		t.Fatalf("valid rows: %+v", items)
	}
	if items[1].UnitPrice != 300_000.56 || items[1].Amount != 750_001.39 { // 2.5 x 300,000.56 rounded
		t.Fatalf("rounding: %+v", items[1])
	}
	if len(errs) != 4 || errs[0].Row != 3 || errs[3].Row != 6 {
		t.Fatalf("errors: %+v", errs)
	}
}

func TestBuildSummaryComparesPlanWithActuals(t *testing.T) {
	items := []domain.BudgetItem{
		{Kind: "rab", Category: "Material", Amount: 1_000_000},
		{Kind: "rab", Category: "Tenaga", Amount: 500_000},
		{Kind: "rap", Category: "Material", Amount: 600_000},
		{Kind: "rap", Category: "Tenaga", Amount: 200_000},
	}
	costs := []domain.CostEntry{
		{Category: "material", Amount: 450_000},
		{Category: "Material", Amount: 250_000}, // 700k against a 600k plan: over
		{Category: "Tenaga", Amount: 50_000},
		{Category: "Transport", Amount: 30_000}, // never planned
	}
	s := BuildSummary(items, costs)
	if s.RAB != 1_500_000 || s.RAP != 800_000 || s.PlannedMargin != 700_000 || s.PlannedMarginP != 46.7 {
		t.Fatalf("plan: %+v", s)
	}
	if s.Actual != 780_000 || s.ProjectedMargin != 720_000 || s.Uncategorised != 30_000 || !s.OverBudget {
		t.Fatalf("actuals: %+v", s)
	}
	byCat := map[string]CategoryRow{}
	for _, c := range s.Categories {
		byCat[c.Category] = c
	}
	if m := byCat["Material"]; m.Planned != 600_000 || m.Actual != 700_000 || m.Variance != -100_000 || !m.Over || m.UsedPct != 116.7 {
		t.Fatalf("material: %+v", m)
	}
	if tr := byCat["Transport"]; !tr.Over || tr.Planned != 0 {
		t.Fatalf("unplanned spend counts as over: %+v", tr)
	}
	if empty := BuildSummary(nil, nil); empty.OverBudget || empty.Categories == nil {
		t.Fatalf("empty: %+v", empty)
	}
}

type memRepo struct {
	domain.Repository
	items []domain.BudgetItem
	costs []domain.CostEntry
	wos   map[uuid.UUID]*domain.WorkOrder
	nums  map[string]bool
}

func newMem() *memRepo {
	return &memRepo{wos: map[uuid.UUID]*domain.WorkOrder{}, nums: map[string]bool{}}
}

func (m *memRepo) ListBudgetItems(context.Context, uuid.UUID, string) ([]domain.BudgetItem, error) {
	return m.items, nil
}
func (m *memRepo) AddBudgetItems(_ context.Context, _ uuid.UUID, replace []string, items []domain.BudgetItem) error {
	if len(replace) > 0 {
		var keep []domain.BudgetItem
		for _, it := range m.items {
			drop := false
			for _, k := range replace {
				drop = drop || it.Kind == k
			}
			if !drop {
				keep = append(keep, it)
			}
		}
		m.items = keep
	}
	m.items = append(m.items, items...)
	return nil
}
func (m *memRepo) ListCosts(context.Context, uuid.UUID) ([]domain.CostEntry, error) {
	return m.costs, nil
}
func (m *memRepo) CreateCost(_ context.Context, c *domain.CostEntry) error {
	m.costs = append(m.costs, *c)
	return nil
}
func (m *memRepo) CreateWorkOrder(_ context.Context, w *domain.WorkOrder) error {
	if m.nums[w.Number] {
		return context.DeadlineExceeded // stands in for a unique-violation
	}
	w.ID = uuid.New()
	m.nums[w.Number] = true
	m.wos[w.ID] = w
	return nil
}
func (m *memRepo) GetWorkOrder(_ context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	if w := m.wos[id]; w != nil {
		c := *w
		return &c, nil
	}
	return nil, nil
}
func (m *memRepo) UpdateWorkOrder(_ context.Context, w *domain.WorkOrder) error {
	c := *w
	m.wos[w.ID] = &c
	return nil
}
func (m *memRepo) CountWorkOrdersWithPrefix(_ context.Context, prefix string) (int64, error) {
	var n int64
	for num := range m.nums {
		if strings.HasPrefix(num, prefix) {
			n++
		}
	}
	return n, nil
}

type memProjects struct {
	projectDomain.ProjectRepository
	p *projectDomain.Project
}

func (m *memProjects) GetByID(context.Context, uuid.UUID) (*projectDomain.Project, error) {
	return m.p, nil
}
func (m *memProjects) Update(_ context.Context, p *projectDomain.Project) error { m.p = p; return nil }

func newSvc() (*Service, *memRepo, *memProjects) {
	repo := newMem()
	p := &projectDomain.Project{Code: "PRJ-1", Name: "Gudang", Customer: "PT A"}
	p.ID = uuid.New()
	pr := &memProjects{p: p}
	s := NewService(repo, pr)
	s.now = func() time.Time { return time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC) }
	return s, repo, pr
}

func TestImportIsAllOrNothingAndReplaceOnlyTouchesUploadedKinds(t *testing.T) {
	s, repo, pr := newSvc()
	ctx := context.Background()
	id := pr.p.ID
	good := []BudgetLineInput{{Kind: "rab", Category: "Material", Description: "Baja", Quantity: 2, UnitPrice: 1_000_000}, {Kind: "rap", Category: "Material", Description: "Baja", Quantity: 2, UnitPrice: 700_000}}
	if res, err := s.AddBudgetLines(ctx, id, "append", good); err != nil || res.Added != 2 {
		t.Fatalf("first import: %+v %v", res, err)
	}
	if pr.p.RABValue != 2_000_000 || pr.p.RAPValue != 1_400_000 {
		t.Fatalf("project headline figures must follow the lines: %v / %v", pr.p.RABValue, pr.p.RAPValue)
	}
	// One bad row: nothing is stored and every problem is reported.
	res, err := s.AddBudgetLines(ctx, id, "append", append([]BudgetLineInput{good[0]}, BudgetLineInput{Kind: "x"}, BudgetLineInput{Kind: "rab"}))
	if err == nil || len(res.Errors) != 2 || len(repo.items) != 2 {
		t.Fatalf("a bad sheet must import nothing: %v %+v items=%d", err, res, len(repo.items))
	}
	// Re-uploading only the RAB in replace mode leaves the RAP alone.
	if _, err := s.AddBudgetLines(ctx, id, "replace", []BudgetLineInput{{Kind: "rab", Category: "Material", Description: "Baja rev", Quantity: 1, UnitPrice: 1_500_000}}); err != nil {
		t.Fatal(err)
	}
	if pr.p.RABValue != 1_500_000 || pr.p.RAPValue != 1_400_000 || len(repo.items) != 2 {
		t.Fatalf("replace: RAB %v RAP %v items %d", pr.p.RABValue, pr.p.RAPValue, len(repo.items))
	}
	if _, err := s.AddBudgetLines(ctx, id, "replace", nil); err == nil {
		t.Fatal("an empty import must be refused")
	}
	if _, err := s.AddBudgetLines(ctx, id, "merge", good); err == nil {
		t.Fatal("unknown mode must be refused")
	}
	// Recording a cost moves the project's actual figure.
	if _, err := s.RecordCost(ctx, id, CostInput{Category: "Material", Amount: 400_000}); err != nil {
		t.Fatal(err)
	}
	if pr.p.ActualCost != 400_000 {
		t.Fatalf("actual cost: %v", pr.p.ActualCost)
	}
	if _, err := s.RecordCost(ctx, id, CostInput{Category: "", Amount: 5}); err == nil {
		t.Fatal("cost needs a category")
	}
}

func TestWorkOrderNumbersLifecycleAndSeparationOfDuties(t *testing.T) {
	s, repo, _ := newSvc()
	maker := actor.WithEmail(context.Background(), "ani@x.com")
	boss := actor.WithEmail(context.Background(), "boss@x.com")

	w1, err := s.CreateWorkOrder(maker, WorkOrderInput{CustomerName: "PT A", Title: "Pasang rak", ContractValue: 50_000_000, StartDate: "2026-10-01", DueDate: "2026-10-31"})
	if err != nil || w1.Number != "SPK-2026-001" || w1.Status != "draft" {
		t.Fatalf("first: %+v %v", w1, err)
	}
	w2, _ := s.CreateWorkOrder(maker, WorkOrderInput{CustomerName: "PT B", Title: "Servis"})
	if w2.Number != "SPK-2026-002" {
		t.Fatalf("second number: %s", w2.Number)
	}
	for _, bad := range []WorkOrderInput{{Title: "x"}, {CustomerName: "y"}, {CustomerName: "y", Title: "x", ContractValue: -1}, {CustomerName: "y", Title: "x", StartDate: "2026-10-05", DueDate: "2026-10-01"}, {CustomerName: "y", Title: "x", DueDate: "31/10/2026"}} {
		if _, err := s.CreateWorkOrder(maker, bad); err == nil {
			t.Errorf("should be refused: %+v", bad)
		}
	}

	if _, err := s.Advance(maker, w1.ID, "in_progress"); err == nil {
		t.Fatal("cannot start a draft")
	}
	if _, err := s.Advance(maker, w1.ID, "issued"); err == nil {
		t.Fatal("the drafter must not issue their own SPK")
	}
	if got, err := s.Advance(boss, w1.ID, "issued"); err != nil || got.IssuedBy != "boss@x.com" {
		t.Fatalf("issue: %+v %v", got, err)
	}
	if _, err := s.SetProgress(boss, w1.ID, 40); err == nil {
		t.Fatal("progress only while in progress")
	}
	if _, err := s.Advance(boss, w1.ID, "in_progress"); err != nil {
		t.Fatal(err)
	}
	if got, err := s.SetProgress(boss, w1.ID, 60); err != nil || got.ProgressPct != 60 {
		t.Fatalf("progress: %+v %v", got, err)
	}
	if _, err := s.SetProgress(boss, w1.ID, 100); err == nil {
		t.Fatal("100 is reached by completing")
	}
	if got, err := s.Advance(boss, w1.ID, "completed"); err != nil || got.ProgressPct != 100 {
		t.Fatalf("complete: %+v %v", got, err)
	}
	if _, err := s.Advance(boss, w1.ID, "cancelled"); err == nil {
		t.Fatal("a completed SPK cannot be cancelled")
	}
	if !CanTransition("draft", "cancelled") || CanTransition("completed", "issued") {
		t.Fatal("transition table wrong")
	}
	_ = repo
}
