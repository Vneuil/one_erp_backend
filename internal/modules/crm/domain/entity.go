package domain

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type Lead struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this lead to one business unit within the company
	// (see modules/workspace). Nil means it belongs to no specific tenant.
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Name           string     `gorm:"type:varchar(255);not null" json:"name"`
	Company        string     `gorm:"type:varchar(255);not null" json:"company"`
	Email          string     `gorm:"type:varchar(255)" json:"email"`
	Phone          string     `gorm:"type:varchar(50)" json:"phone"`
	Segment        string     `gorm:"type:varchar(100);default:'Enterprise B2B'" json:"segment"`
	Source         string     `gorm:"type:varchar(50);default:'whatsapp'" json:"source"`
	EstimatedValue float64    `gorm:"type:decimal(15,2);default:0" json:"estimatedValue"`
	Status         string     `gorm:"type:varchar(50);default:'New'" json:"status"`
	PIC            string     `gorm:"type:varchar(100)" json:"pic"`
}

func (Lead) TableName() string {
	return "leads"
}

type Deal struct {
	types.BaseEntity
	CompanyID       *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID        *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	Title           string     `gorm:"type:varchar(255);not null" json:"title"`
	Customer        string     `gorm:"type:varchar(255);not null" json:"customer"`
	Value           float64    `gorm:"type:decimal(15,2);default:0" json:"value"`
	Probability     int        `gorm:"default:30" json:"probability"`
	Stage           string     `gorm:"type:varchar(50);default:'discovery'" json:"stage"`
	PIC             string     `gorm:"type:varchar(100)" json:"pic"`
	ExpectedClosing string     `gorm:"type:varchar(50)" json:"expectedClosing"`
	LostReason      string     `gorm:"type:varchar(255)" json:"lostReason"`
	// LeadID is the lead this deal came from; ProjectID the project started
	// from it once won. ClosedAt is when it reached a won or lost stage.
	LeadID    *uuid.UUID `gorm:"type:uuid;index" json:"leadId,omitempty"`
	ProjectID *uuid.UUID `gorm:"type:uuid;index" json:"projectId,omitempty"`
	ClosedAt  *time.Time `json:"closedAt,omitempty"`
}

func (Deal) TableName() string {
	return "crm_deals"
}

type CRMRepository interface {
	Create(ctx context.Context, lead *Lead) error
	GetByID(ctx context.Context, id uuid.UUID) (*Lead, error)
	List(ctx context.Context, query types.PaginationQuery) ([]Lead, int64, error)
	Update(ctx context.Context, lead *Lead) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int64, error)

	CreateDeal(ctx context.Context, deal *Deal) error
	GetDealByID(ctx context.Context, id uuid.UUID) (*Deal, error)
	ListDeals(ctx context.Context, query types.PaginationQuery) ([]Deal, int64, error)
	UpdateDeal(ctx context.Context, deal *Deal) error
	DeleteDeal(ctx context.Context, id uuid.UUID) error
	CountDeals(ctx context.Context) (int64, error)
}
