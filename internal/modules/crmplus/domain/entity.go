// Package crmplus extends CRM with the day-to-day selling workflow around
// leads and deals: interaction history, sales tasks and reminders, documents,
// tags, contacts, configurable pipeline stages, a public lead-capture form and
// analytics.
package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Parent types an interaction, task, document, tag or contact can hang off.
const (
	ParentLead     = "lead"
	ParentDeal     = "deal"
	ParentCustomer = "customer"
)

// Interaction is one logged touchpoint (call, meeting, email, WhatsApp, visit)
// or an internal note ("note") against a lead, deal or customer.
type Interaction struct {
	types.BaseEntity
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ParentType     string     `gorm:"type:varchar(20);not null;index:idx_crm_int_parent" json:"parentType"`
	ParentID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_crm_int_parent" json:"parentId"`
	Kind           string     `gorm:"type:varchar(20);not null" json:"kind"`
	Summary        string     `gorm:"type:text;not null" json:"summary"`
	OccurredAt     time.Time  `gorm:"not null" json:"occurredAt"`
	CreatedByEmail string     `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
}

func (Interaction) TableName() string { return "crm_interactions" }

// SalesTask is a follow-up to do, optionally tied to a lead/deal/customer.
type SalesTask struct {
	types.BaseEntity
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Notes          string     `gorm:"type:text" json:"notes,omitempty"`
	DueDate        string     `gorm:"type:varchar(10);not null;index" json:"dueDate"` // YYYY-MM-DD
	Priority       string     `gorm:"type:varchar(10);not null;default:'normal'" json:"priority"`
	Status         string     `gorm:"type:varchar(10);not null;default:'open';index" json:"status"` // open | done
	AssigneeEmail  string     `gorm:"type:varchar(255);index" json:"assigneeEmail"`
	ParentType     string     `gorm:"type:varchar(20)" json:"parentType,omitempty"`
	ParentID       *uuid.UUID `gorm:"type:uuid;index" json:"parentId,omitempty"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	CreatedByEmail string     `gorm:"type:varchar(255)" json:"createdByEmail,omitempty"`
}

func (SalesTask) TableName() string { return "crm_tasks" }

// Document is a file reference (proposal, contract, spec) kept with a lead,
// deal or customer. FileRef is a Drive file id or URL.
type Document struct {
	types.BaseEntity
	TenantID   *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ParentType string     `gorm:"type:varchar(20);not null;index:idx_crm_doc_parent" json:"parentType"`
	ParentID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_crm_doc_parent" json:"parentId"`
	Title      string     `gorm:"type:varchar(255);not null" json:"title"`
	DocType    string     `gorm:"type:varchar(50);not null" json:"docType"`
	FileRef    string     `gorm:"type:varchar(500)" json:"fileRef"`
	Notes      string     `gorm:"type:text" json:"notes,omitempty"`
	UploadedBy string     `gorm:"type:varchar(255)" json:"uploadedBy,omitempty"`
	// An uploaded file, when there is one (FileRef then stays free for an external link).
	FileName   string `gorm:"type:varchar(255)" json:"fileName,omitempty"`
	FileSize   int64  `json:"fileSize,omitempty"`
	StorageKey string `gorm:"type:varchar(500)" json:"-"`
}

func (Document) TableName() string { return "crm_documents" }

// Tag is a label; TagAssignment attaches it to a lead, deal or customer.
type Tag struct {
	types.BaseEntity
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name     string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"name"`
	Color    string     `gorm:"type:varchar(20);not null;default:'slate'" json:"color"`
}

func (Tag) TableName() string { return "crm_tags" }

type TagAssignment struct {
	types.BaseEntity
	TagID      uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_crm_tag_assign" json:"tagId"`
	ParentType string    `gorm:"type:varchar(20);not null;uniqueIndex:uq_crm_tag_assign" json:"parentType"`
	ParentID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:uq_crm_tag_assign;index" json:"parentId"`
}

func (TagAssignment) TableName() string { return "crm_tag_assignments" }

// Contact is a person at a lead or customer company.
type Contact struct {
	types.BaseEntity
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`
	JobTitle    string     `gorm:"type:varchar(100)" json:"jobTitle,omitempty"`
	Email       string     `gorm:"type:varchar(255)" json:"email,omitempty"`
	Phone       string     `gorm:"type:varchar(50)" json:"phone,omitempty"`
	CompanyName string     `gorm:"type:varchar(255);index" json:"companyName"`
	LeadID      *uuid.UUID `gorm:"type:uuid;index" json:"leadId,omitempty"`
	IsPrimary   bool       `gorm:"not null;default:false" json:"isPrimary"`
	Notes       string     `gorm:"type:text" json:"notes,omitempty"`
}

func (Contact) TableName() string { return "crm_contacts" }

// Stage kinds. A pipeline always has at least one won and one lost stage.
const (
	StageOpen = "open"
	StageWon  = "won"
	StageLost = "lost"
)

// PipelineStage is one configurable column of the deal pipeline.
type PipelineStage struct {
	types.BaseEntity
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Key         string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"key"`
	Name        string     `gorm:"type:varchar(100);not null" json:"name"`
	Position    int        `gorm:"not null" json:"position"`
	Probability int        `gorm:"not null" json:"probability"`
	Kind        string     `gorm:"type:varchar(10);not null;default:'open'" json:"kind"`
	IsActive    bool       `gorm:"not null;default:true" json:"isActive"`
}

func (PipelineStage) TableName() string { return "crm_pipeline_stages" }

// LeadForm is a public web form that drops submissions into the leads list.
type LeadForm struct {
	types.BaseEntity
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Key            string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"key"`
	Name           string     `gorm:"type:varchar(255);not null" json:"name"`
	Source         string     `gorm:"type:varchar(50);not null;default:'web'" json:"source"`
	DefaultPIC     string     `gorm:"type:varchar(100)" json:"defaultPic"`
	SuccessMessage string     `gorm:"type:varchar(500)" json:"successMessage"`
	IsActive       bool       `gorm:"not null;default:true" json:"isActive"`
	Submissions    int        `gorm:"not null;default:0" json:"submissions"`
}

func (LeadForm) TableName() string { return "crm_lead_forms" }

// LeadFormIndex maps a public form key to its owning company. It lives in the
// control-plane database: a public submission carries no session, so the
// company must be found before its own database can be opened (the same
// reason as whatsappregistry).
type LeadFormIndex struct {
	types.BaseEntity
	FormKey   string     `gorm:"type:varchar(64);uniqueIndex;not null"`
	CompanyID uuid.UUID  `gorm:"type:uuid;not null;index"`
	TenantID  *uuid.UUID `gorm:"type:uuid"`
}

func (LeadFormIndex) TableName() string { return "crm_lead_form_index" }

// Repository is the persistence contract for tenant-scoped CRM extras.
type Repository interface {
	CreateInteraction(ctx context.Context, v *Interaction) error
	ListInteractions(ctx context.Context, parentType string, parentID uuid.UUID) ([]Interaction, error)
	// LastInteractionByParent returns the latest interaction time per parent id.
	LastInteractionByParent(ctx context.Context, parentType string) (map[uuid.UUID]time.Time, error)

	CreateTask(ctx context.Context, v *SalesTask) error
	GetTask(ctx context.Context, id uuid.UUID) (*SalesTask, error)
	UpdateTask(ctx context.Context, v *SalesTask) error
	ListTasks(ctx context.Context, status, assignee, parentType string, parentID *uuid.UUID) ([]SalesTask, error)

	CreateDocument(ctx context.Context, v *Document) error
	ListDocuments(ctx context.Context, parentType string, parentID uuid.UUID) ([]Document, error)
	GetDocument(ctx context.Context, id uuid.UUID) (*Document, error)
	DeleteDocument(ctx context.Context, id uuid.UUID) (bool, error)

	CreateTag(ctx context.Context, v *Tag) error
	ListTags(ctx context.Context) ([]Tag, error)
	GetTag(ctx context.Context, id uuid.UUID) (*Tag, error)
	DeleteTag(ctx context.Context, id uuid.UUID) (bool, error)
	AssignTag(ctx context.Context, v *TagAssignment) error
	UnassignTag(ctx context.Context, tagID uuid.UUID, parentType string, parentID uuid.UUID) error
	TagsFor(ctx context.Context, parentType string, parentID uuid.UUID) ([]Tag, error)
	ParentsWithTag(ctx context.Context, tagID uuid.UUID, parentType string) ([]uuid.UUID, error)

	CreateContact(ctx context.Context, v *Contact) error
	GetContact(ctx context.Context, id uuid.UUID) (*Contact, error)
	UpdateContact(ctx context.Context, v *Contact) error
	DeleteContact(ctx context.Context, id uuid.UUID) (bool, error)
	ListContacts(ctx context.Context, companyName string, leadID *uuid.UUID) ([]Contact, error)

	CreateStage(ctx context.Context, v *PipelineStage) error
	GetStage(ctx context.Context, id uuid.UUID) (*PipelineStage, error)
	UpdateStage(ctx context.Context, v *PipelineStage) error
	DeleteStage(ctx context.Context, id uuid.UUID) error
	ListStages(ctx context.Context, activeOnly bool) ([]PipelineStage, error)
	CountStages(ctx context.Context) (int64, error)

	CreateLeadForm(ctx context.Context, v *LeadForm) error
	GetLeadFormByKey(ctx context.Context, key string) (*LeadForm, error)
	GetLeadForm(ctx context.Context, id uuid.UUID) (*LeadForm, error)
	UpdateLeadForm(ctx context.Context, v *LeadForm) error
	DeleteLeadForm(ctx context.Context, id uuid.UUID) error
	ListLeadForms(ctx context.Context) ([]LeadForm, error)
	IncrementSubmissions(ctx context.Context, id uuid.UUID) error
}

// FormRegistry is the control-plane index of public form keys.
type FormRegistry interface {
	Upsert(ctx context.Context, key string, companyID uuid.UUID, tenantID *uuid.UUID) error
	Lookup(ctx context.Context, key string) (*LeadFormIndex, error)
	Delete(ctx context.Context, key string) error
}
