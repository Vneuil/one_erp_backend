package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/reimbursement/domain"
	"github.com/google/uuid"
)

type CreateClaimDTO struct {
	EmployeeName    string  `json:"employeeName"`
	Department      string  `json:"department"`
	Category        string  `json:"category"`
	Amount          float64 `json:"amount"`
	Description     string  `json:"description"`
	Date            string  `json:"date"`
	ReceiptAttached bool    `json:"receiptAttached"`
}

type ClaimResponseDTO struct {
	ID              uuid.UUID `json:"id"`
	ClaimNo         string    `json:"claimNo"`
	EmployeeName    string    `json:"employeeName"`
	Department      string    `json:"department"`
	Category        string    `json:"category"`
	Amount          float64   `json:"amount"`
	Description     string    `json:"description"`
	RequesterEmail  string    `json:"requesterEmail,omitempty"`
	Date            string    `json:"date"`
	ReceiptAttached bool      `json:"receiptAttached"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
}

func ToClaimResponse(cl *domain.ReimbursementClaim) *ClaimResponseDTO {
	if cl == nil {
		return nil
	}
	return &ClaimResponseDTO{
		ID:              cl.ID,
		ClaimNo:         cl.ClaimNo,
		EmployeeName:    cl.EmployeeName,
		Department:      cl.Department,
		Category:        cl.Category,
		Amount:          cl.Amount,
		Description:     cl.Description,
		RequesterEmail:  cl.RequesterEmail,
		Date:            cl.Date,
		ReceiptAttached: cl.ReceiptAttached,
		Status:          cl.Status,
		CreatedAt:       cl.CreatedAt,
	}
}

func ToClaimResponseList(claims []domain.ReimbursementClaim) []ClaimResponseDTO {
	result := make([]ClaimResponseDTO, len(claims))
	for i, cl := range claims {
		result[i] = *ToClaimResponse(&cl)
	}
	return result
}
