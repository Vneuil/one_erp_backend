// Package projectcost adds cost control to projects: the RAB (client-facing
// budget) and RAP (internal cost plan) as itemised lines, recorded actual
// costs with variance, and SPK work orders.
package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

const (
	KindRAB = "rab"
	KindRAP = "rap"
)

// BudgetItem is one line of a project's RAB or RAP.
type BudgetItem struct {
	types.BaseEntity
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProjectID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_budget_project_kind" json:"projectId"`
	Kind        string     `gorm:"type:varchar(3);not null;index:idx_budget_project_kind" json:"kind"`
	Category    string     `gorm:"type:varchar(100);not null" json:"category"`
	Description string     `gorm:"type:varchar(255);not null" json:"description"`
	Quantity    float64    `gorm:"type:decimal(15,3);not null" json:"quantity"`
	Unit        string     `gorm:"type:varchar(30)" json:"unit"`
	UnitPrice   float64    `gorm:"type:decimal(15,2);not null" json:"unitPrice"`
	Amount      float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
}

func (BudgetItem) TableName() string { return "project_budget_items" }

// CostEntry is money actually spent on a project, against a budget category.
type CostEntry struct {
	types.BaseEntity
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ProjectID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"projectId"`
	Category       string     `gorm:"type:varchar(100);not null" json:"category"`
	Description    string     `gorm:"type:varchar(255)" json:"description"`
	Amount         float64    `gorm:"type:decimal(15,2);not null" json:"amount"`
	Date           string     `gorm:"type:varchar(10);not null" json:"date"`
	CreatedByEmail string     `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
}

func (CostEntry) TableName() string { return "project_cost_entries" }

// SPK statuses.
const (
	WODraft      = "draft"
	WOIssued     = "issued"
	WOInProgress = "in_progress"
	WOCompleted  = "completed"
	WOCancelled  = "cancelled"
)

// WorkOrder is an SPK (surat perintah kerja): an instruction to carry out a
// defined scope of work for a customer or project, with a value and a deadline.
type WorkOrder struct {
	types.BaseEntity
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Number         string     `gorm:"type:varchar(30);not null;uniqueIndex" json:"number"`
	ProjectID      *uuid.UUID `gorm:"type:uuid;index" json:"projectId,omitempty"`
	SalesOrderID   *uuid.UUID `gorm:"type:uuid;index" json:"salesOrderId,omitempty"`
	CustomerName   string     `gorm:"type:varchar(255);not null" json:"customerName"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Scope          string     `gorm:"type:text" json:"scope,omitempty"`
	ContractValue  float64    `gorm:"type:decimal(15,2);default:0" json:"contractValue"`
	StartDate      string     `gorm:"type:varchar(10)" json:"startDate,omitempty"`
	DueDate        string     `gorm:"type:varchar(10)" json:"dueDate,omitempty"`
	AssignedTo     string     `gorm:"type:varchar(255)" json:"assignedTo,omitempty"`
	Status         string     `gorm:"type:varchar(20);not null;default:'draft';index" json:"status"`
	ProgressPct    int        `gorm:"not null;default:0" json:"progressPct"`
	Notes          string     `gorm:"type:text" json:"notes,omitempty"`
	CreatedByEmail string     `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
	IssuedBy       string     `gorm:"type:varchar(255)" json:"issuedBy,omitempty"`
}

func (WorkOrder) TableName() string { return "project_work_orders" }

type Repository interface {
	ListBudgetItems(ctx context.Context, projectID uuid.UUID, kind string) ([]BudgetItem, error)
	// AddBudgetItems inserts all items in one transaction. Lines of the kinds in
	// replaceKinds are removed first, in the same transaction.
	AddBudgetItems(ctx context.Context, projectID uuid.UUID, replaceKinds []string, items []BudgetItem) error
	DeleteBudgetItem(ctx context.Context, projectID, id uuid.UUID) (bool, error)

	CreateCost(ctx context.Context, c *CostEntry) error
	ListCosts(ctx context.Context, projectID uuid.UUID) ([]CostEntry, error)
	DeleteCost(ctx context.Context, projectID, id uuid.UUID) (bool, error)

	CreateWorkOrder(ctx context.Context, w *WorkOrder) error
	GetWorkOrder(ctx context.Context, id uuid.UUID) (*WorkOrder, error)
	UpdateWorkOrder(ctx context.Context, w *WorkOrder) error
	ListWorkOrders(ctx context.Context, status string, projectID *uuid.UUID) ([]WorkOrder, error)
	CountWorkOrdersWithPrefix(ctx context.Context, prefix string) (int64, error)
}
