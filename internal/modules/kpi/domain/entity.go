package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type KpiReview struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this KPI review to one business unit within the
	// company (see modules/workspace). Nil means the company's default tenant.
	TenantID      *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	EmployeeName  string     `gorm:"type:varchar(255);not null" json:"employeeName"`
	Department    string     `gorm:"type:varchar(100);not null" json:"department"`
	Period        string     `gorm:"type:varchar(50);not null" json:"period"`
	TargetScore   float64    `gorm:"type:decimal(10,2);default:0" json:"targetScore"`
	ActualScore   float64    `gorm:"type:decimal(10,2);default:0" json:"actualScore"`
	WeightFormula string     `gorm:"type:text" json:"weightFormula"`
	Grade         string     `gorm:"type:varchar(5)" json:"grade"`
	Status        string     `gorm:"type:varchar(50);default:'draft'" json:"status"`
	Evaluator     string     `gorm:"type:varchar(255)" json:"evaluator"`
}

func (KpiReview) TableName() string {
	return "kpi_reviews"
}

type KpiRepository interface {
	Create(ctx context.Context, review *KpiReview) error
	GetByID(ctx context.Context, id uuid.UUID) (*KpiReview, error)
	List(ctx context.Context, query types.PaginationQuery) ([]KpiReview, int64, error)
	Update(ctx context.Context, review *KpiReview) error
	Count(ctx context.Context) (int64, error)
}
