package application

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	projectDomain "github.com/divinecoid/one-backend/internal/modules/project/domain"
	"github.com/divinecoid/one-backend/internal/modules/projectcost/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"github.com/google/uuid"
)

const maxImportRows = 1000

func round2(v float64) float64 { return math.Round(v*100) / 100 }

type Service struct {
	repo     domain.Repository
	projects projectDomain.ProjectRepository
	now      func() time.Time
}

func NewService(repo domain.Repository, projects projectDomain.ProjectRepository) *Service {
	return &Service{repo: repo, projects: projects, now: time.Now}
}

func (s *Service) project(ctx context.Context, id uuid.UUID) (*projectDomain.Project, error) {
	p, err := s.projects.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load project")
	}
	if p == nil {
		return nil, apperrors.NewNotFound("Project not found")
	}
	return p, nil
}

// ----------------------------------------------------------------- budget lines

// BudgetLineInput is one row to add, typically parsed from a spreadsheet.
type BudgetLineInput struct {
	Kind        string  `json:"kind"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unitPrice"`
}

// ImportError names the row that failed, so a person can fix the sheet.
type ImportError struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// ValidateBudgetLines checks every row and returns all problems at once (row
// numbers are 1-based, as they appear in a spreadsheet after its header). A
// bulk import is all or nothing, so one bad row must never leave half the sheet in.
func ValidateBudgetLines(rows []BudgetLineInput) ([]domain.BudgetItem, []ImportError) {
	var errs []ImportError
	items := make([]domain.BudgetItem, 0, len(rows))
	bad := func(i int, m string) { errs = append(errs, ImportError{Row: i + 1, Message: m}) }
	for i, r := range rows {
		kind := strings.ToLower(strings.TrimSpace(r.Kind))
		cat, desc := strings.TrimSpace(r.Category), strings.TrimSpace(r.Description)
		switch {
		case kind != domain.KindRAB && kind != domain.KindRAP:
			bad(i, "kind must be rab or rap")
		case cat == "" || len(cat) > 100:
			bad(i, "category is required (max 100 characters)")
		case desc == "" || len(desc) > 255:
			bad(i, "description is required (max 255 characters)")
		case r.Quantity <= 0 || math.IsNaN(r.Quantity) || math.IsInf(r.Quantity, 0) || r.Quantity > 1e9:
			bad(i, "quantity must be a positive number")
		case r.UnitPrice < 0 || math.IsNaN(r.UnitPrice) || math.IsInf(r.UnitPrice, 0) || r.UnitPrice > 1e12:
			bad(i, "unitPrice must be zero or more")
		default:
			items = append(items, domain.BudgetItem{Kind: kind, Category: cat, Description: desc, Quantity: r.Quantity,
				Unit: strings.TrimSpace(r.Unit), UnitPrice: round2(r.UnitPrice), Amount: round2(r.Quantity * r.UnitPrice)})
		}
	}
	return items, errs
}

// ImportResult reports what an import did.
type ImportResult struct {
	Added  int           `json:"added"`
	Errors []ImportError `json:"errors,omitempty"`
}

// AddBudgetLines validates and stores lines. mode "replace" first clears the
// kinds present in the upload (so re-importing an updated RAB sheet does not
// double it); "append" (default) only adds. Nothing is stored if any row is bad.
func (s *Service) AddBudgetLines(ctx context.Context, projectID uuid.UUID, mode string, rows []BudgetLineInput) (*ImportResult, error) {
	if len(rows) == 0 || len(rows) > maxImportRows {
		return nil, apperrors.NewBadRequest(fmt.Sprintf("Provide between 1 and %d rows", maxImportRows))
	}
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "" && mode != "append" && mode != "replace" {
		return nil, apperrors.NewBadRequest("mode must be append or replace")
	}
	p, err := s.project(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items, errs := ValidateBudgetLines(rows)
	if len(errs) > 0 {
		return &ImportResult{Errors: errs}, apperrors.NewBadRequest(fmt.Sprintf("%d row(s) are invalid; nothing was imported (first: row %d, %s)", len(errs), errs[0].Row, errs[0].Message), errs)
	}
	for i := range items {
		items[i].ProjectID = projectID
	}
	// In replace mode only the kinds actually uploaded are cleared.
	var replace []string
	if mode == "replace" {
		seen := map[string]bool{}
		for _, it := range items {
			if !seen[it.Kind] {
				seen[it.Kind] = true
				replace = append(replace, it.Kind)
			}
		}
	}
	if err := s.repo.AddBudgetItems(ctx, projectID, replace, items); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save budget lines")
	}
	if err := s.syncTotals(ctx, p); err != nil {
		return nil, err
	}
	return &ImportResult{Added: len(items)}, nil
}

func (s *Service) DeleteBudgetLine(ctx context.Context, projectID, id uuid.UUID) error {
	p, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	ok, err := s.repo.DeleteBudgetItem(ctx, projectID, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete budget line")
	}
	if !ok {
		return apperrors.NewNotFound("Budget line not found")
	}
	return s.syncTotals(ctx, p)
}

// syncTotals keeps the project's headline RAB/RAP/actual figures equal to the sums of its lines.
func (s *Service) syncTotals(ctx context.Context, p *projectDomain.Project) error {
	items, err := s.repo.ListBudgetItems(ctx, p.ID, "")
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load budget lines")
	}
	costs, err := s.repo.ListCosts(ctx, p.ID)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to load costs")
	}
	var rab, rap, actual float64
	for _, it := range items {
		if it.Kind == domain.KindRAB {
			rab += it.Amount
		} else {
			rap += it.Amount
		}
	}
	for _, c := range costs {
		actual += c.Amount
	}
	p.RABValue, p.RAPValue, p.ActualCost = round2(rab), round2(rap), round2(actual)
	if err := s.projects.Update(ctx, p); err != nil {
		return apperrors.NewInternal(err, "Failed to update project totals")
	}
	return nil
}

// ------------------------------------------------------------------- actual costs

type CostInput struct {
	Category, Description, Date string
	Amount                      float64
}

func (s *Service) RecordCost(ctx context.Context, projectID uuid.UUID, in CostInput) (*domain.CostEntry, error) {
	p, err := s.project(ctx, projectID)
	if err != nil {
		return nil, err
	}
	cat := strings.TrimSpace(in.Category)
	if cat == "" || in.Amount <= 0 || math.IsNaN(in.Amount) {
		return nil, apperrors.NewBadRequest("category and a positive amount are required")
	}
	date := in.Date
	if date == "" {
		date = s.now().Format("2006-01-02")
	} else if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, apperrors.NewBadRequest("date must be YYYY-MM-DD")
	}
	c := &domain.CostEntry{ProjectID: projectID, Category: cat, Description: strings.TrimSpace(in.Description), Amount: round2(in.Amount), Date: date, CreatedByEmail: actor.EmailFrom(ctx)}
	if err := s.repo.CreateCost(ctx, c); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save cost")
	}
	if err := s.syncTotals(ctx, p); err != nil {
		return c, err
	}
	return c, nil
}

func (s *Service) DeleteCost(ctx context.Context, projectID, id uuid.UUID) error {
	p, err := s.project(ctx, projectID)
	if err != nil {
		return err
	}
	ok, err := s.repo.DeleteCost(ctx, projectID, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to delete cost")
	}
	if !ok {
		return apperrors.NewNotFound("Cost entry not found")
	}
	return s.syncTotals(ctx, p)
}

func (s *Service) ListCosts(ctx context.Context, projectID uuid.UUID) ([]domain.CostEntry, error) {
	if _, err := s.project(ctx, projectID); err != nil {
		return nil, err
	}
	out, err := s.repo.ListCosts(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list costs")
	}
	return out, nil
}

// --------------------------------------------------------------------- summary

// CategoryRow compares the RAP plan with what was actually spent in one category.
type CategoryRow struct {
	Category string  `json:"category"`
	Planned  float64 `json:"planned"`
	Actual   float64 `json:"actual"`
	Variance float64 `json:"variance"` // planned - actual; negative = overspent
	UsedPct  float64 `json:"usedPct"`  // actual / planned, 0 when nothing planned
	Over     bool    `json:"over"`
}

// BudgetSummary is the RAB/RAP control view of a project.
type BudgetSummary struct {
	RAB             float64       `json:"rab"`
	RAP             float64       `json:"rap"`
	PlannedMargin   float64       `json:"plannedMargin"` // RAB - RAP
	PlannedMarginP  float64       `json:"plannedMarginPct"`
	Actual          float64       `json:"actual"`
	ProjectedMargin float64       `json:"projectedMargin"` // RAB - actual so far
	Categories      []CategoryRow `json:"categories"`
	Uncategorised   float64       `json:"unplannedSpend"` // actual in categories with no RAP line
	OverBudget      bool          `json:"overBudget"`
}

// BuildSummary compares plan and actuals. Categories match case-insensitively.
func BuildSummary(items []domain.BudgetItem, costs []domain.CostEntry) BudgetSummary {
	var sum BudgetSummary
	rows := map[string]*CategoryRow{}
	names := map[string]string{}
	get := func(name string) *CategoryRow {
		k := strings.ToLower(strings.TrimSpace(name))
		if rows[k] == nil {
			rows[k] = &CategoryRow{Category: strings.TrimSpace(name)}
			names[k] = name
		}
		return rows[k]
	}
	for _, it := range items {
		if it.Kind == domain.KindRAB {
			sum.RAB += it.Amount
		} else {
			sum.RAP += it.Amount
			get(it.Category).Planned += it.Amount
		}
	}
	for _, c := range costs {
		sum.Actual += c.Amount
		get(c.Category).Actual += c.Amount
	}
	sum.Categories = make([]CategoryRow, 0, len(rows))
	for _, r := range rows {
		r.Planned, r.Actual = round2(r.Planned), round2(r.Actual)
		r.Variance = round2(r.Planned - r.Actual)
		if r.Planned > 0 {
			r.UsedPct = math.Round(r.Actual/r.Planned*1000) / 10
		} else {
			sum.Uncategorised += r.Actual
		}
		r.Over = r.Actual > r.Planned
		if r.Over {
			sum.OverBudget = true
		}
		sum.Categories = append(sum.Categories, *r)
	}
	sort.Slice(sum.Categories, func(i, j int) bool { return sum.Categories[i].Category < sum.Categories[j].Category })
	sum.RAB, sum.RAP, sum.Actual, sum.Uncategorised = round2(sum.RAB), round2(sum.RAP), round2(sum.Actual), round2(sum.Uncategorised)
	sum.PlannedMargin = round2(sum.RAB - sum.RAP)
	if sum.RAB > 0 {
		sum.PlannedMarginP = math.Round(sum.PlannedMargin/sum.RAB*1000) / 10
	}
	sum.ProjectedMargin = round2(sum.RAB - sum.Actual)
	return sum
}

// BudgetView bundles the lines with their summary for one call.
type BudgetView struct {
	Items   []domain.BudgetItem `json:"items"`
	Summary BudgetSummary       `json:"summary"`
}

func (s *Service) Budget(ctx context.Context, projectID uuid.UUID) (*BudgetView, error) {
	if _, err := s.project(ctx, projectID); err != nil {
		return nil, err
	}
	items, err := s.repo.ListBudgetItems(ctx, projectID, "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load budget lines")
	}
	costs, err := s.repo.ListCosts(ctx, projectID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load costs")
	}
	return &BudgetView{Items: items, Summary: BuildSummary(items, costs)}, nil
}

// ------------------------------------------------------------------ work orders

type WorkOrderInput struct {
	ProjectID     *uuid.UUID
	SalesOrderID  *uuid.UUID
	CustomerName  string
	Title         string
	Scope         string
	ContractValue float64
	StartDate     string
	DueDate       string
	AssignedTo    string
	Notes         string
}

func validOptionalDate(s string) bool {
	if s == "" {
		return true
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func (s *Service) nextNumber(ctx context.Context) (string, error) {
	prefix := fmt.Sprintf("SPK-%d-", s.now().Year())
	n, err := s.repo.CountWorkOrdersWithPrefix(ctx, prefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%03d", prefix, n+1), nil
}

func (s *Service) CreateWorkOrder(ctx context.Context, in WorkOrderInput) (*domain.WorkOrder, error) {
	title, customer := strings.TrimSpace(in.Title), strings.TrimSpace(in.CustomerName)
	if title == "" || customer == "" {
		return nil, apperrors.NewBadRequest("title and customerName are required")
	}
	if in.ContractValue < 0 || math.IsNaN(in.ContractValue) || !validOptionalDate(in.StartDate) || !validOptionalDate(in.DueDate) {
		return nil, apperrors.NewBadRequest("contractValue cannot be negative and dates must be YYYY-MM-DD")
	}
	if in.StartDate != "" && in.DueDate != "" && in.DueDate < in.StartDate {
		return nil, apperrors.NewBadRequest("dueDate cannot be before startDate")
	}
	if in.ProjectID != nil {
		if _, err := s.project(ctx, *in.ProjectID); err != nil {
			return nil, err
		}
	}
	w := &domain.WorkOrder{ProjectID: in.ProjectID, SalesOrderID: in.SalesOrderID, CustomerName: customer, Title: title, Scope: strings.TrimSpace(in.Scope),
		ContractValue: round2(in.ContractValue), StartDate: in.StartDate, DueDate: in.DueDate, AssignedTo: strings.TrimSpace(in.AssignedTo), Notes: in.Notes,
		Status: domain.WODraft, CreatedByEmail: actor.EmailFrom(ctx)}
	// The number is unique in the database; retry if two people create at once.
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		if w.Number, err = s.nextNumber(ctx); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to number the work order")
		}
		w.ID = uuid.Nil
		if err = s.repo.CreateWorkOrder(ctx, w); err == nil {
			return w, nil
		}
	}
	return nil, apperrors.NewInternal(err, "Failed to save the work order")
}

func (s *Service) ListWorkOrders(ctx context.Context, status string, projectID *uuid.UUID) ([]domain.WorkOrder, error) {
	out, err := s.repo.ListWorkOrders(ctx, status, projectID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list work orders")
	}
	return out, nil
}

func (s *Service) load(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	w, err := s.repo.GetWorkOrder(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load work order")
	}
	if w == nil {
		return nil, apperrors.NewNotFound("Work order not found")
	}
	return w, nil
}

// transitions is the SPK lifecycle.
var transitions = map[string][]string{
	domain.WODraft:      {domain.WOIssued, domain.WOCancelled},
	domain.WOIssued:     {domain.WOInProgress, domain.WOCancelled},
	domain.WOInProgress: {domain.WOCompleted, domain.WOCancelled},
}

// CanTransition reports whether a work order may move between two statuses.
func CanTransition(from, to string) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// Advance moves a work order to a new status. Issuing needs a second person: the
// one who drafted the SPK cannot issue it. Completing sets progress to 100%.
func (s *Service) Advance(ctx context.Context, id uuid.UUID, to string) (*domain.WorkOrder, error) {
	w, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if !CanTransition(w.Status, to) {
		return nil, apperrors.NewConflict(fmt.Sprintf("A work order that is %s cannot become %s", w.Status, to))
	}
	if to == domain.WOIssued {
		if err := sod.ForbidSelfApproval(ctx, w.CreatedByEmail); err != nil {
			return nil, err
		}
		w.IssuedBy = actor.EmailFrom(ctx)
	}
	w.Status = to
	if to == domain.WOCompleted {
		w.ProgressPct = 100
	}
	if err := s.repo.UpdateWorkOrder(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update work order")
	}
	return w, nil
}

// SetProgress records how far along an SPK in progress is (0-99; completing sets 100).
func (s *Service) SetProgress(ctx context.Context, id uuid.UUID, pct int) (*domain.WorkOrder, error) {
	w, err := s.load(ctx, id)
	if err != nil {
		return nil, err
	}
	if w.Status != domain.WOInProgress {
		return nil, apperrors.NewConflict("Progress can only be updated while the work order is in progress")
	}
	if pct < 0 || pct > 99 {
		return nil, apperrors.NewBadRequest("progress must be 0-99; complete the work order to reach 100")
	}
	w.ProgressPct = pct
	if err := s.repo.UpdateWorkOrder(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update work order")
	}
	return w, nil
}
