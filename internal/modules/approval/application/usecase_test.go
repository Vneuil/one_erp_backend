package application

import (
	"context"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/approval/domain"
	"github.com/google/uuid"
)

type mockApprovalRepo struct {
	domain.ApprovalRepository
	workflows map[uuid.UUID]*domain.ApprovalWorkflow
}

func newMockApprovalRepo() *mockApprovalRepo {
	return &mockApprovalRepo{workflows: make(map[uuid.UUID]*domain.ApprovalWorkflow)}
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
