package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// ReimbursementClaim is an employee expense reimbursement claim.
type ReimbursementClaim struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this claim to one business unit within the company
	// (see modules/workspace). Nil means the company's default tenant.
	TenantID     *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ClaimNo      string     `gorm:"type:varchar(50);not null;index" json:"claimNo"`
	EmployeeName string     `gorm:"type:varchar(255);not null" json:"employeeName"`
	// RequesterEmail is the login email of whoever submitted the claim, set
	// server-side from the authenticated caller (never taken from the client).
	// It is used to stop a person from approving or paying their own claim.
	RequesterEmail  string  `gorm:"type:varchar(255);index" json:"-"`
	Department      string  `gorm:"type:varchar(100)" json:"department"`
	Category        string  `gorm:"type:varchar(100)" json:"category"`
	Amount          float64 `gorm:"type:decimal(15,2);not null" json:"amount"`
	Description     string  `gorm:"type:varchar(500)" json:"description"`
	Date            string  `gorm:"type:varchar(50)" json:"date"`
	ReceiptAttached bool    `gorm:"default:false" json:"receiptAttached"`
	Status          string  `gorm:"type:varchar(50);default:'pending_approval'" json:"status"` // pending_approval | approved | paid | rejected
}

func (ReimbursementClaim) TableName() string {
	return "reimbursement_claims"
}

type ReimbursementRepository interface {
	CreateClaim(ctx context.Context, claim *ReimbursementClaim) error
	GetClaimByID(ctx context.Context, id uuid.UUID) (*ReimbursementClaim, error)
	UpdateClaim(ctx context.Context, claim *ReimbursementClaim) error
	ListClaims(ctx context.Context, query types.PaginationQuery) ([]ReimbursementClaim, int64, error)
	CountClaims(ctx context.Context) (int64, error)
}
