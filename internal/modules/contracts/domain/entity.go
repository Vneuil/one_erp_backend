package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Contract represents a customer or vendor agreement tracked for renewal and reminders
type Contract struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this contract to a branch/subsidiary Tenant within the
	// Company's database (nil for companies with only their default Tenant).
	TenantID       *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	ContractNumber string     `gorm:"type:varchar(50);not null;index" json:"contractNumber"`
	PartyType      string     `gorm:"type:varchar(20);not null" json:"partyType"`
	PartyID        *uuid.UUID `gorm:"type:uuid;index" json:"partyId,omitempty"`
	PartyName      string     `gorm:"type:varchar(255);not null" json:"partyName"`
	Title          string     `gorm:"type:varchar(255);not null" json:"title"`
	Description    string     `gorm:"type:text" json:"description"`
	ContractValue  float64    `gorm:"type:decimal(15,2);not null" json:"contractValue"`
	StartDate      string     `gorm:"type:varchar(50);not null" json:"startDate"`
	EndDate        string     `gorm:"type:varchar(50);not null" json:"endDate"`
	Status         string     `gorm:"type:varchar(50);not null;default:'draft'" json:"status"`
	PaymentTerms   string     `gorm:"type:varchar(255)" json:"paymentTerms"`
	Attachments    string     `gorm:"type:text" json:"attachments"`
	RenewedFromID  *uuid.UUID `gorm:"type:uuid;index" json:"renewedFromId,omitempty"`
}

func (Contract) TableName() string {
	return "contracts_contracts"
}

type ContractsRepository interface {
	CreateContract(ctx context.Context, c *Contract) error
	GetContractByID(ctx context.Context, id uuid.UUID) (*Contract, error)
	ListContracts(ctx context.Context, query types.PaginationQuery) ([]Contract, int64, error)
	ListAllContracts(ctx context.Context) ([]Contract, error)
	UpdateContract(ctx context.Context, c *Contract) error
	CountContracts(ctx context.Context) (int64, error)
}
