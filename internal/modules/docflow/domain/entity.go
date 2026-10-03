package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Form represents a dynamic form definition
type Form struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this form to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant.
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title         string     `gorm:"type:varchar(255);not null" json:"title"`
	Description   string     `gorm:"type:text" json:"description"`
	Fields        string     `gorm:"type:text;not null" json:"fields"`
	CreatedBy     string     `gorm:"type:varchar(255);not null" json:"createdBy"`
	Status        string     `gorm:"type:varchar(20);not null;default:'draft'" json:"status"`
	ResponseCount int64      `gorm:"not null;default:0" json:"responseCount"`
}

func (Form) TableName() string {
	return "docflow_forms"
}

// FormResponse represents a single submission against a form
type FormResponse struct {
	types.BaseEntity
	FormID      uuid.UUID `gorm:"type:uuid;not null;index" json:"formId"`
	SubmittedBy string    `gorm:"type:varchar(255);not null" json:"submittedBy"`
	Answers     string    `gorm:"type:text;not null" json:"answers"`
	SubmittedAt string    `gorm:"type:varchar(50);not null" json:"submittedAt"`
}

func (FormResponse) TableName() string {
	return "docflow_form_responses"
}

// SignatureDocument represents a document routed for e-signature, optionally
// backed by real object storage (StorageKey is empty for legacy
// metadata-only documents or when R2 is not configured).
type SignatureDocument struct {
	types.BaseEntity
	CompanyID  *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID   *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title      string     `gorm:"type:varchar(255);not null" json:"title"`
	FileName   string     `gorm:"type:varchar(255);not null" json:"fileName"`
	Status     string     `gorm:"type:varchar(30);not null;default:'draft'" json:"status"`
	UploadedBy string     `gorm:"type:varchar(255);not null" json:"uploadedBy"`
	UploadedAt string     `gorm:"type:varchar(50);not null" json:"uploadedAt"`
	StorageKey string     `gorm:"type:varchar(500)" json:"storageKey,omitempty"`
	MimeType   string     `gorm:"type:varchar(150)" json:"mimeType,omitempty"`
}

func (SignatureDocument) TableName() string {
	return "docflow_signature_documents"
}

// SignatureRequest represents one signer's request against a document
type SignatureRequest struct {
	types.BaseEntity
	DocumentID  uuid.UUID `gorm:"type:uuid;not null;index" json:"documentId"`
	SignerName  string    `gorm:"type:varchar(255);not null" json:"signerName"`
	SignerEmail string    `gorm:"type:varchar(255);not null" json:"signerEmail"`
	Status      string    `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	OrderIndex  int       `gorm:"not null;default:0" json:"orderIndex"`
	SignedAt    *string   `gorm:"type:varchar(50)" json:"signedAt,omitempty"`
}

func (SignatureRequest) TableName() string {
	return "docflow_signature_requests"
}

// Meeting represents a scheduled meeting
type Meeting struct {
	types.BaseEntity
	CompanyID       *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title           string     `gorm:"type:varchar(255);not null" json:"title"`
	Description     string     `gorm:"type:text" json:"description"`
	ScheduledAt     string     `gorm:"type:varchar(50);not null" json:"scheduledAt"`
	DurationMinutes int        `gorm:"not null;default:30" json:"durationMinutes"`
	OrganizerName   string     `gorm:"type:varchar(255);not null" json:"organizerName"`
	MeetingURL      string     `gorm:"type:varchar(255);not null" json:"meetingUrl"`
	Status          string     `gorm:"type:varchar(20);not null;default:'scheduled'" json:"status"`
	AttendeeNames   string     `gorm:"type:text" json:"attendeeNames"`
}

func (Meeting) TableName() string {
	return "docflow_meetings"
}

// MeetingNote represents a note recorded against a meeting
type MeetingNote struct {
	types.BaseEntity
	MeetingID   uuid.UUID `gorm:"type:uuid;not null;index" json:"meetingId"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	AIGenerated bool      `gorm:"not null;default:false" json:"aiGenerated"`
}

func (MeetingNote) TableName() string {
	return "docflow_meeting_notes"
}

type FormsRepository interface {
	CreateForm(ctx context.Context, f *Form) error
	GetFormByID(ctx context.Context, id uuid.UUID) (*Form, error)
	ListForms(ctx context.Context, query types.PaginationQuery) ([]Form, int64, error)
	UpdateForm(ctx context.Context, f *Form) error
	CountForms(ctx context.Context) (int64, error)

	CreateResponse(ctx context.Context, r *FormResponse) error
	ListResponsesByForm(ctx context.Context, formID uuid.UUID) ([]FormResponse, error)
}

type SignaturesRepository interface {
	CreateDocument(ctx context.Context, d *SignatureDocument) error
	GetDocumentByID(ctx context.Context, id uuid.UUID) (*SignatureDocument, error)
	ListDocuments(ctx context.Context, query types.PaginationQuery) ([]SignatureDocument, int64, error)
	UpdateDocument(ctx context.Context, d *SignatureDocument) error
	CountDocuments(ctx context.Context) (int64, error)

	CreateRequest(ctx context.Context, r *SignatureRequest) error
	GetRequestByID(ctx context.Context, id uuid.UUID) (*SignatureRequest, error)
	ListRequestsByDocument(ctx context.Context, documentID uuid.UUID) ([]SignatureRequest, error)
	UpdateRequest(ctx context.Context, r *SignatureRequest) error
}

type MeetingsRepository interface {
	CreateMeeting(ctx context.Context, m *Meeting) error
	GetMeetingByID(ctx context.Context, id uuid.UUID) (*Meeting, error)
	ListMeetings(ctx context.Context, query types.PaginationQuery) ([]Meeting, int64, error)
	UpdateMeeting(ctx context.Context, m *Meeting) error
	CountMeetings(ctx context.Context) (int64, error)

	CreateNote(ctx context.Context, n *MeetingNote) error
	ListNotesByMeeting(ctx context.Context, meetingID uuid.UUID) ([]MeetingNote, error)
}
