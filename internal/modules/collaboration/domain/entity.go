package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Conversation represents a chat conversation between display-name participants
type Conversation struct {
	types.BaseEntity
	// CompanyID scopes this row to one company. This module stores every
	// company's data in the single shared control-plane database (no
	// per-tenant database like most other modules), so CompanyID is the
	// only thing preventing one company from seeing another's - see
	// foundation/companyctx. Unlike TenantID below, this must never be nil
	// on a real row.
	CompanyID        *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID         *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name             string     `gorm:"type:varchar(255);not null" json:"name"`
	Type             string     `gorm:"type:varchar(20);not null" json:"type"`
	ParticipantNames string     `gorm:"type:text;not null" json:"participantNames"`
}

func (Conversation) TableName() string {
	return "collaboration_conversations"
}

// Message represents a single chat message within a conversation
type Message struct {
	types.BaseEntity
	ConversationID uuid.UUID `gorm:"type:uuid;not null;index" json:"conversationId"`
	SenderName     string    `gorm:"type:varchar(255);not null" json:"senderName"`
	Content        string    `gorm:"type:text;not null" json:"content"`
	AttachmentName *string   `gorm:"type:varchar(255)" json:"attachmentName,omitempty"`
	SentAt         time.Time `gorm:"not null" json:"sentAt"`
}

func (Message) TableName() string {
	return "collaboration_messages"
}

// Folder represents a drive folder, optionally nested under a parent
type Folder struct {
	types.BaseEntity
	CompanyID      *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name           string     `gorm:"type:varchar(255);not null" json:"name"`
	ParentFolderID *uuid.UUID `gorm:"type:uuid;index" json:"parentFolderId,omitempty"`
}

func (Folder) TableName() string {
	return "collaboration_folders"
}

// File represents a drive file's metadata, optionally backed by real object
// storage (StorageKey is empty for legacy metadata-only files or when R2 is
// not configured).
type File struct {
	types.BaseEntity
	CompanyID  *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID   *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	FolderID   *uuid.UUID `gorm:"type:uuid;index" json:"folderId,omitempty"`
	Name       string     `gorm:"type:varchar(255);not null" json:"name"`
	SizeBytes  int64      `gorm:"not null;default:0" json:"sizeBytes"`
	MimeType   string     `gorm:"type:varchar(150)" json:"mimeType"`
	UploadedBy string     `gorm:"type:varchar(255);not null" json:"uploadedBy"`
	StorageKey string     `gorm:"type:varchar(500)" json:"storageKey,omitempty"`
}

func (File) TableName() string {
	return "collaboration_files"
}

// ExportJob represents a data export job run against a named dataset
type ExportJob struct {
	types.BaseEntity
	CompanyID   *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID    *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	DatasetName string     `gorm:"type:varchar(150);not null" json:"datasetName"`
	Format      string     `gorm:"type:varchar(20);not null" json:"format"`
	Status      string     `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	RequestedBy string     `gorm:"type:varchar(255);not null" json:"requestedBy"`
	RowCount    *int64     `json:"rowCount,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

func (ExportJob) TableName() string {
	return "collaboration_export_jobs"
}

type ChatRepository interface {
	CreateConversation(ctx context.Context, c *Conversation) error
	GetConversationByID(ctx context.Context, id uuid.UUID) (*Conversation, error)
	ListConversations(ctx context.Context) ([]Conversation, error)
	CountConversations(ctx context.Context) (int64, error)

	CreateMessage(ctx context.Context, m *Message) error
	ListMessagesByConversation(ctx context.Context, conversationID uuid.UUID) ([]Message, error)
}

type DriveRepository interface {
	CreateFolder(ctx context.Context, f *Folder) error
	GetFolderByID(ctx context.Context, id uuid.UUID) (*Folder, error)
	ListFolders(ctx context.Context) ([]Folder, error)
	CountFolders(ctx context.Context) (int64, error)
	DeleteFolder(ctx context.Context, id uuid.UUID) error

	CreateFile(ctx context.Context, f *File) error
	GetFileByID(ctx context.Context, id uuid.UUID) (*File, error)
	ListFiles(ctx context.Context, folderID *uuid.UUID) ([]File, error)
	DeleteFile(ctx context.Context, id uuid.UUID) error
	CountFiles(ctx context.Context) (int64, error)
}

type ExportsRepository interface {
	CreateJob(ctx context.Context, j *ExportJob) error
	GetJobByID(ctx context.Context, id uuid.UUID) (*ExportJob, error)
	ListJobs(ctx context.Context) ([]ExportJob, error)
	UpdateJob(ctx context.Context, j *ExportJob) error
	CountJobs(ctx context.Context) (int64, error)
}
