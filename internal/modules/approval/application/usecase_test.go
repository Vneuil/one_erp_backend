package application

import (
	"context"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/approval/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/google/uuid"
)

type mockApprovalRepo struct {
	domain.ApprovalRepository
	workflows map[uuid.UUID]*domain.ApprovalWorkflow
	requests  map[uuid.UUID]*domain.ApprovalRequest
}

func newMockApprovalRepo() *mockApprovalRepo {
	return &mockApprovalRepo{
		workflows: make(map[uuid.UUID]*domain.ApprovalWorkflow),
		requests:  make(map[uuid.UUID]*domain.ApprovalRequest),
	}
}

func (m *mockApprovalRepo) CreateWorkflow(ctx context.Context, w *domain.ApprovalWorkflow) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	w.CreatedAt = time.Now()
	m.workflows[w.ID] = w
	return nil
}

func (m *mockApprovalRepo) GetWorkflowByID(ctx context.Context, id uuid.UUID) (*domain.ApprovalWorkflow, error) {
	w, ok := m.workflows[id]
	if !ok {
		return nil, nil
	}
	return w, nil
}

func (m *mockApprovalRepo) UpdateWorkflow(ctx context.Context, w *domain.ApprovalWorkflow) error {
	m.workflows[w.ID] = w
	return nil
}

func (m *mockApprovalRepo) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	delete(m.workflows, id)
	return nil
}

func (m *mockApprovalRepo) ListWorkflows(ctx context.Context, documentType string) ([]domain.ApprovalWorkflow, error) {
	var out []domain.ApprovalWorkflow
	for _, w := range m.workflows {
		if documentType == "" || w.DocumentType == documentType {
			out = append(out, *w)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].MinAmount < out[j].MinAmount
	})
	return out, nil
}

func (m *mockApprovalRepo) ListMatchingWorkflows(ctx context.Context, documentType string, amount float64) ([]domain.ApprovalWorkflow, error) {
	var out []domain.ApprovalWorkflow
	for _, w := range m.workflows {
		if w.DocumentType == documentType && w.IsActive && w.MinAmount <= amount && (w.MaxAmount == nil || *w.MaxAmount >= amount) {
			out = append(out, *w)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].MinAmount > out[j].MinAmount
	})
	return out, nil
}

func (m *mockApprovalRepo) CreateRequest(ctx context.Context, req *domain.ApprovalRequest) error {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	for i := range req.Steps {
		if req.Steps[i].ID == uuid.Nil {
			req.Steps[i].ID = uuid.New()
		}
		req.Steps[i].RequestID = req.ID
	}
	m.requests[req.ID] = req
	return nil
}

func (m *mockApprovalRepo) GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.ApprovalRequest, error) {
	r, ok := m.requests[id]
	if !ok {
		return nil, nil
	}
	return r, nil
}

func (m *mockApprovalRepo) UpdateRequest(ctx context.Context, r *domain.ApprovalRequest) error {
	m.requests[r.ID] = r
	return nil
}

func (m *mockApprovalRepo) UpdateStep(ctx context.Context, s *domain.ApprovalStep) error {
	if r, ok := m.requests[s.RequestID]; ok {
		for i := range r.Steps {
			if r.Steps[i].ID == s.ID {
				r.Steps[i] = *s
			}
		}
	}
	return nil
}

func (m *mockApprovalRepo) ListPendingStepsForRole(ctx context.Context, role string) ([]domain.ApprovalStep, error) {
	var out []domain.ApprovalStep
	for _, r := range m.requests {
		if r.Status != domain.StatusPending {
			continue
		}
		for _, s := range r.Steps {
			if s.Status == domain.StatusPending && s.LevelOrder == r.CurrentLevel {
				if strings.EqualFold(role, "admin") || strings.EqualFold(s.ApproverRole, role) {
					out = append(out, s)
				}
			}
		}
	}
	return out, nil
}

func TestUpdateWorkflow(t *testing.T) {
	repo := newMockApprovalRepo()
	uc := NewApprovalUseCase(repo)

	// Create initial workflow
	created, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "purchase_order",
		Name:         "PO Approval 1",
		MinAmount:    10000000,
		Levels: []WorkflowLevelDTO{
			{LevelOrder: 1, ApproverRole: "manager"},
		},
	})
	if err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	// Test successful update
	isActive := false
	updated, err := uc.UpdateWorkflow(context.Background(), created.ID, UpdateWorkflowDTO{
		DocumentType: "purchase_order",
		Name:         "PO Approval Revised",
		MinAmount:    25000000,
		IsActive:     &isActive,
		Levels: []WorkflowLevelDTO{
			{LevelOrder: 1, ApproverRole: "manager"},
			{LevelOrder: 2, ApproverRole: "admin"},
		},
	})
	if err != nil {
		t.Fatalf("failed to update workflow: %v", err)
	}

	if updated.Name != "PO Approval Revised" {
		t.Errorf("expected name 'PO Approval Revised', got %q", updated.Name)
	}
	if updated.MinAmount != 25000000 {
		t.Errorf("expected minAmount 25000000, got %f", updated.MinAmount)
	}
	if updated.IsActive != false {
		t.Errorf("expected isActive false, got %v", updated.IsActive)
	}
	if len(updated.Levels) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(updated.Levels))
	}
	if updated.Levels[1].ApproverRole != "admin" {
		t.Errorf("expected level 2 role 'admin', got %q", updated.Levels[1].ApproverRole)
	}

	// Test non-existent workflow
	_, err = uc.UpdateWorkflow(context.Background(), uuid.New(), UpdateWorkflowDTO{
		DocumentType: "purchase_order",
		Name:         "Ghost",
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "manager"}},
	})
	if err == nil {
		t.Error("expected error for non-existent workflow, got nil")
	}

	// Test invalid empty levels
	_, err = uc.UpdateWorkflow(context.Background(), created.ID, UpdateWorkflowDTO{
		DocumentType: "purchase_order",
		Name:         "No Levels",
		Levels:       []WorkflowLevelDTO{},
	})
	if err == nil {
		t.Error("expected error for empty levels, got nil")
	}
}

func TestSubmitAndActWithCustomRBACRoleInActorContext(t *testing.T) {
	repo := newMockApprovalRepo()
	uc := NewApprovalUseCase(repo)

	max499k := 499999.0
	_, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO Minimal",
		MinAmount:    0,
		MaxAmount:    &max499k,
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "manager"}},
	})
	if err != nil {
		t.Fatalf("failed to create SO minimal workflow: %v", err)
	}
	_, err = uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO >= 500rb",
		MinAmount:    500000,
		Levels: []WorkflowLevelDTO{
			{LevelOrder: 1, ApproverRole: "manager"},
			{LevelOrder: 2, ApproverRole: "admin"},
		},
	})
	if err != nil {
		t.Fatalf("failed to create SO >= 500rb workflow: %v", err)
	}

	// Amount 6000 matches the 0-threshold workflow (1 level: manager).
	res, err := uc.SubmitDocument(context.Background(), SubmitDocumentDTO{
		DocumentType:   "sales_order",
		DocumentID:     uuid.New(),
		DocumentNumber: "SO-202611-0001",
		Amount:         6000,
		RequesterName:  "Customer A",
	})
	if err != nil {
		t.Fatalf("SubmitDocument failed: %v", err)
	}
	if !res.RequiresApproval || res.Request == nil {
		t.Fatal("expected RequiresApproval=true for 6000 SO")
	}
	if len(res.Request.Steps) != 1 || res.Request.Steps[0].ApproverRole != "manager" {
		t.Fatalf("expected 1 manager step, got %+v", res.Request.Steps)
	}

	// Plain staff without RBAC Manager role cannot list or approve.
	staffCtx := actor.WithRoles(context.Background(), []string{"staff"})
	pending, err := uc.ListMyPendingApprovals(staffCtx, "staff")
	if err != nil {
		t.Fatalf("ListMyPendingApprovals failed: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("expected 0 pending approvals for plain staff, got %d", len(pending))
	}
	if _, err := uc.ApproveStep(staffCtx, res.Request.ID, ActOnStepDTO{ActorName: "staff@x.com", ActorRole: "staff"}); err == nil {
		t.Fatal("expected error when plain staff tries to approve manager step")
	}

	// Staff assigned RBAC "Manager" role can list and approve.
	rbacMgrCtx := actor.WithRoles(context.Background(), []string{"staff", "Manager"})
	pending, err = uc.ListMyPendingApprovals(rbacMgrCtx, "staff")
	if err != nil {
		t.Fatalf("ListMyPendingApprovals failed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending approval for RBAC Manager, got %d", len(pending))
	}
	approved, err := uc.ApproveStep(rbacMgrCtx, res.Request.ID, ActOnStepDTO{ActorName: "mgr@x.com", ActorRole: "staff"})
	if err != nil {
		t.Fatalf("expected RBAC Manager to approve step, got error: %v", err)
	}
	if approved.Status != domain.StatusApproved {
		t.Fatalf("expected final status approved, got %q", approved.Status)
	}
}

func TestWorkflowAmountRangeAndOverlapValidation(t *testing.T) {
	ptr := func(v float64) *float64 { return &v }
	repo := newMockApprovalRepo()
	uc := NewApprovalUseCase(repo)

	// Invalid: maxAmount < minAmount
	if _, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "Invalid Range",
		MinAmount:    500000,
		MaxAmount:    ptr(100000),
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "manager"}},
	}); err == nil {
		t.Fatal("expected error when maxAmount < minAmount")
	}

	// Valid Tier 1: SO 0 - 500.000
	tier1, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO Tier 1",
		MinAmount:    0,
		MaxAmount:    ptr(500000),
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "manager"}},
	})
	if err != nil {
		t.Fatalf("failed to create SO Tier 1: %v", err)
	}

	// Overlap at exact boundary 500.000 should be rejected
	if _, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO Boundary Collision",
		MinAmount:    500000,
		MaxAmount:    ptr(1000000),
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "admin"}},
	}); err == nil {
		t.Fatal("expected overlap error at boundary 500000")
	}

	// Overlap with unlimited upper bound should be rejected
	if _, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO Unlimited Collision",
		MinAmount:    400000,
		MaxAmount:    nil,
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "admin"}},
	}); err == nil {
		t.Fatal("expected overlap error when unlimited range overlaps Tier 1")
	}

	// Same range for different documentType (purchase_order) is allowed
	if _, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "purchase_order",
		Name:         "PO Tier 1",
		MinAmount:    0,
		MaxAmount:    ptr(500000),
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "manager"}},
	}); err != nil {
		t.Fatalf("same range on different documentType should succeed, got: %v", err)
	}

	// Non-overlapping Tier 2: SO 500.001 - unlimited
	tier2, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO Tier 2",
		MinAmount:    500001,
		MaxAmount:    nil,
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "admin"}},
	})
	if err != nil {
		t.Fatalf("failed to create non-overlapping SO Tier 2: %v", err)
	}

	// Updating Tier 1 to collide with Tier 2 should fail
	if _, err := uc.UpdateWorkflow(context.Background(), tier1.ID, UpdateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO Tier 1 Expanded",
		MinAmount:    0,
		MaxAmount:    ptr(600000),
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "manager"}},
	}); err == nil {
		t.Fatal("expected error when updating Tier 1 to overlap Tier 2")
	}

	// Updating Tier 1 within valid range should succeed
	if _, err := uc.UpdateWorkflow(context.Background(), tier1.ID, UpdateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO Tier 1 Updated",
		MinAmount:    0,
		MaxAmount:    ptr(499999),
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "manager"}},
	}); err != nil {
		t.Fatalf("expected valid update on Tier 1 to succeed, got: %v", err)
	}

	// Deactivating Tier 2 allows creating a new workflow in its range,
	// and re-activating Tier 2 while the new one is active fails.
	inactive := false
	if _, err := uc.UpdateWorkflow(context.Background(), tier2.ID, UpdateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         tier2.Name,
		MinAmount:    tier2.MinAmount,
		MaxAmount:    tier2.MaxAmount,
		IsActive:     &inactive,
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "admin"}},
	}); err != nil {
		t.Fatalf("failed to deactivate Tier 2: %v", err)
	}

	if _, err := uc.CreateWorkflow(context.Background(), CreateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         "SO Tier 2 Replacement",
		MinAmount:    500000,
		MaxAmount:    nil,
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "manager"}},
	}); err != nil {
		t.Fatalf("creating replacement while old tier is inactive should succeed, got: %v", err)
	}

	active := true
	if _, err := uc.UpdateWorkflow(context.Background(), tier2.ID, UpdateWorkflowDTO{
		DocumentType: "sales_order",
		Name:         tier2.Name,
		MinAmount:    tier2.MinAmount,
		MaxAmount:    tier2.MaxAmount,
		IsActive:     &active,
		Levels:       []WorkflowLevelDTO{{LevelOrder: 1, ApproverRole: "admin"}},
	}); err == nil {
		t.Fatal("expected overlap error when re-activating Tier 2")
	}

	// Verify SubmitDocument matches exact tier by amount
	lowRes, err := uc.SubmitDocument(context.Background(), SubmitDocumentDTO{
		DocumentType: "sales_order", DocumentID: uuid.New(), DocumentNumber: "SO-1", Amount: 6000,
	})
	if err != nil || !lowRes.RequiresApproval || lowRes.Request.WorkflowID != tier1.ID {
		t.Fatalf("expected amount 6000 to match Tier 1 (%s), got %+v (err=%v)", tier1.ID, lowRes, err)
	}
	highRes, err := uc.SubmitDocument(context.Background(), SubmitDocumentDTO{
		DocumentType: "sales_order", DocumentID: uuid.New(), DocumentNumber: "SO-2", Amount: 500000,
	})
	if err != nil || !highRes.RequiresApproval || highRes.Request.WorkflowID == tier1.ID {
		t.Fatalf("expected amount 500000 to match Tier 2 Replacement, got %+v (err=%v)", highRes, err)
	}
}

