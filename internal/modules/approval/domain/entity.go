package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// ApprovalWorkflow is an admin-configured, dynamic approval rule: "for
// documents of type X above amount Y, route through these N ordered
// approver roles before the document is considered approved." Any module
// (procurement, sales, ...) can submit a document against this engine by
// document type string - no schema change needed here to support a new
// document type, which is what makes this "dynamic".
type ApprovalWorkflow struct {
	types.BaseEntity
	CompanyID    *uuid.UUID              `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID     *uuid.UUID              `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DocumentType string                  `gorm:"type:varchar(100);not null;index" json:"documentType"`
	Name         string                  `gorm:"type:varchar(150);not null" json:"name"`
	MinAmount    float64                 `gorm:"type:decimal(15,2);not null;default:0" json:"minAmount"`
	IsActive     bool                    `gorm:"default:true" json:"isActive"`
	Levels       []ApprovalWorkflowLevel `gorm:"foreignKey:WorkflowID" json:"levels,omitempty"`
}

func (ApprovalWorkflow) TableName() string {
	return "approval_workflows"
}

// ApprovalWorkflowLevel is one ordered step in a workflow. ApproverRole is
// a free-text role name matched against the acting user's legacy Role
// string (admin/manager/staff) or a custom modules/rbac Role name - kept
// as a string rather than an FK since Role lives in a different database
// (control-plane) than this tenant-scoped workflow.
type ApprovalWorkflowLevel struct {
	types.BaseEntity
	WorkflowID   uuid.UUID `gorm:"type:uuid;not null;index" json:"workflowId"`
	LevelOrder   int       `gorm:"not null" json:"levelOrder"`
	ApproverRole string    `gorm:"type:varchar(100);not null" json:"approverRole"`
}

func (ApprovalWorkflowLevel) TableName() string {
	return "approval_workflow_levels"
}

// Approval request/step statuses
const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

// ApprovalRequest is one instance of a document going through a matched
// ApprovalWorkflow.
type ApprovalRequest struct {
	types.BaseEntity
	CompanyID      *uuid.UUID     `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID       *uuid.UUID     `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	WorkflowID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"workflowId"`
	DocumentType   string         `gorm:"type:varchar(100);not null;index" json:"documentType"`
	DocumentID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"documentId"`
	DocumentNumber string         `gorm:"type:varchar(50)" json:"documentNumber"`
	Amount         float64        `gorm:"type:decimal(15,2)" json:"amount"`
	RequesterName  string         `gorm:"type:varchar(150)" json:"requesterName"`
	Status         string         `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	CurrentLevel   int            `gorm:"not null;default:1" json:"currentLevel"`
	Steps          []ApprovalStep `gorm:"foreignKey:RequestID" json:"steps,omitempty"`
}

func (ApprovalRequest) TableName() string {
	return "approval_requests"
}

type ApprovalStep struct {
	types.BaseEntity
	RequestID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"requestId"`
	LevelOrder   int        `gorm:"not null" json:"levelOrder"`
	ApproverRole string     `gorm:"type:varchar(100);not null" json:"approverRole"`
	Status       string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ActedByName  string     `gorm:"type:varchar(150)" json:"actedByName,omitempty"`
	ActedAt      *time.Time `json:"actedAt,omitempty"`
	Comments     string     `gorm:"type:varchar(255)" json:"comments,omitempty"`
}

func (ApprovalStep) TableName() string {
	return "approval_steps"
}

type ApprovalRepository interface {
	CreateWorkflow(ctx context.Context, w *ApprovalWorkflow) error
	GetWorkflowByID(ctx context.Context, id uuid.UUID) (*ApprovalWorkflow, error)
	ListWorkflows(ctx context.Context, documentType string) ([]ApprovalWorkflow, error)
	// ListMatchingWorkflows returns active workflows for a document type
	// whose MinAmount is <= amount, ordered by MinAmount descending so the
	// caller's first result is the most specific (highest threshold) match.
	ListMatchingWorkflows(ctx context.Context, documentType string, amount float64) ([]ApprovalWorkflow, error)
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error

	CreateRequest(ctx context.Context, r *ApprovalRequest) error
	GetRequestByID(ctx context.Context, id uuid.UUID) (*ApprovalRequest, error)
	GetActiveRequestForDocument(ctx context.Context, documentType string, documentID uuid.UUID) (*ApprovalRequest, error)
	ListRequests(ctx context.Context, query types.PaginationQuery, documentType string) ([]ApprovalRequest, int64, error)
	UpdateRequest(ctx context.Context, r *ApprovalRequest) error

	UpdateStep(ctx context.Context, s *ApprovalStep) error
	ListPendingStepsForRole(ctx context.Context, role string) ([]ApprovalStep, error)
}
