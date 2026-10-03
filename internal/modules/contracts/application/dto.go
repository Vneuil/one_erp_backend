package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/contracts/domain"
	"github.com/google/uuid"
)

type CreateContractDTO struct {
	ContractNumber string     `json:"contractNumber"`
	PartyType      string     `json:"partyType"`
	PartyID        *uuid.UUID `json:"partyId,omitempty"`
	PartyName      string     `json:"partyName"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	ContractValue  float64    `json:"contractValue"`
	StartDate      string     `json:"startDate"`
	EndDate        string     `json:"endDate"`
	PaymentTerms   string     `json:"paymentTerms"`
	Attachments    string     `json:"attachments"`
	CompanyID      *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateContractDTO struct {
	PartyType     string     `json:"partyType"`
	PartyID       *uuid.UUID `json:"partyId,omitempty"`
	PartyName     string     `json:"partyName"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	ContractValue *float64   `json:"contractValue,omitempty"`
	StartDate     string     `json:"startDate"`
	EndDate       string     `json:"endDate"`
	PaymentTerms  string     `json:"paymentTerms"`
	Attachments   string     `json:"attachments"`
}

type UpdateContractStatusDTO struct {
	Status string `json:"status"`
}

type ContractResponseDTO struct {
	ID             uuid.UUID  `json:"id"`
	CompanyID      *uuid.UUID `json:"companyId,omitempty"`
	ContractNumber string     `json:"contractNumber"`
	PartyType      string     `json:"partyType"`
	PartyID        *uuid.UUID `json:"partyId,omitempty"`
	PartyName      string     `json:"partyName"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	ContractValue  float64    `json:"contractValue"`
	StartDate      string     `json:"startDate"`
	EndDate        string     `json:"endDate"`
	Status         string     `json:"status"`
	PaymentTerms   string     `json:"paymentTerms"`
	Attachments    string     `json:"attachments"`
	RenewedFromID  *uuid.UUID `json:"renewedFromId,omitempty"`
	NeedsAttention bool       `json:"needsAttention"`
	CreatedAt      time.Time  `json:"createdAt"`
}

const expiringSoonDays = 30

func ToContractResponse(c *domain.Contract) *ContractResponseDTO {
	if c == nil {
		return nil
	}
	needsAttention := false
	if c.Status == "active" {
		if end, err := time.Parse("2006-01-02", c.EndDate); err == nil {
			daysLeft := time.Until(end).Hours() / 24
			if daysLeft >= 0 && daysLeft <= expiringSoonDays {
				needsAttention = true
			}
		}
	}
	return &ContractResponseDTO{
		ID:             c.ID,
		CompanyID:      c.CompanyID,
		ContractNumber: c.ContractNumber,
		PartyType:      c.PartyType,
		PartyID:        c.PartyID,
		PartyName:      c.PartyName,
		Title:          c.Title,
		Description:    c.Description,
		ContractValue:  c.ContractValue,
		StartDate:      c.StartDate,
		EndDate:        c.EndDate,
		Status:         c.Status,
		PaymentTerms:   c.PaymentTerms,
		Attachments:    c.Attachments,
		RenewedFromID:  c.RenewedFromID,
		NeedsAttention: needsAttention,
		CreatedAt:      c.CreatedAt,
	}
}

func ToContractResponseList(contracts []domain.Contract) []ContractResponseDTO {
	result := make([]ContractResponseDTO, len(contracts))
	for i, c := range contracts {
		result[i] = *ToContractResponse(&c)
	}
	return result
}
