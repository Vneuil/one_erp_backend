package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/approval/domain"
	"github.com/google/uuid"
)

type WorkflowLevelDTO struct {
	LevelOrder   int    `json:"levelOrder"`
	ApproverRole string `json:"approverRole"`
}

type CreateWorkflowDTO struct {
	DocumentType string             `json:"documentType"`
	Name         string             `json:"name"`
	MinAmount    float64            `json:"minAmount"`
	Levels       []WorkflowLevelDTO `json:"levels"`
}

type WorkflowResponseDTO struct {
	ID           uuid.UUID          `json:"id"`
	DocumentType string             `json:"documentType"`
	Name         string             `json:"name"`
	MinAmount    float64            `json:"minAmount"`
	IsActive     bool               `json:"isActive"`
	Levels       []WorkflowLevelDTO `json:"levels"`
	CreatedAt    time.Time          `json:"createdAt"`
}

func ToWorkflowResponse(w *domain.ApprovalWorkflow) WorkflowResponseDTO {
	levels := make([]WorkflowLevelDTO, len(w.Levels))
	for i, l := range w.Levels {
		levels[i] = WorkflowLevelDTO{LevelOrder: l.LevelOrder, ApproverRole: l.ApproverRole}
	}
	return WorkflowResponseDTO{
		ID: w.ID, DocumentType: w.DocumentType, Name: w.Name, MinAmount: w.MinAmount,
		IsActive: w.IsActive, Levels: levels, CreatedAt: w.CreatedAt,
	}
}

func ToWorkflowResponseList(items []domain.ApprovalWorkflow) []WorkflowResponseDTO {
	result := make([]WorkflowResponseDTO, len(items))
	for i, w := range items {
		result[i] = ToWorkflowResponse(&w)
	}
	return result
}

type StepResponseDTO struct {
	ID           uuid.UUID  `json:"id"`
	LevelOrder   int        `json:"levelOrder"`
	ApproverRole string     `json:"approverRole"`
	Status       string     `json:"status"`
	ActedByName  string     `json:"actedByName,omitempty"`
	ActedAt      *time.Time `json:"actedAt,omitempty"`
	Comments     string     `json:"comments,omitempty"`
}

type RequestResponseDTO struct {
	ID             uuid.UUID         `json:"id"`
	WorkflowID     uuid.UUID         `json:"workflowId"`
	DocumentType   string            `json:"documentType"`
	DocumentID     uuid.UUID         `json:"documentId"`
	DocumentNumber string            `json:"documentNumber"`
	Amount         float64           `json:"amount"`
	RequesterName  string            `json:"requesterName"`
	Status         string            `json:"status"`
	CurrentLevel   int               `json:"currentLevel"`
	Steps          []StepResponseDTO `json:"steps"`
	CreatedAt      time.Time         `json:"createdAt"`
}

func ToRequestResponse(r *domain.ApprovalRequest) RequestResponseDTO {
	steps := make([]StepResponseDTO, len(r.Steps))
	for i, s := range r.Steps {
		steps[i] = StepResponseDTO{
			ID: s.ID, LevelOrder: s.LevelOrder, ApproverRole: s.ApproverRole, Status: s.Status,
			ActedByName: s.ActedByName, ActedAt: s.ActedAt, Comments: s.Comments,
		}
	}
	return RequestResponseDTO{
		ID: r.ID, WorkflowID: r.WorkflowID, DocumentType: r.DocumentType, DocumentID: r.DocumentID,
		DocumentNumber: r.DocumentNumber, Amount: r.Amount, RequesterName: r.RequesterName,
		Status: r.Status, CurrentLevel: r.CurrentLevel, Steps: steps, CreatedAt: r.CreatedAt,
	}
}

func ToRequestResponseList(items []domain.ApprovalRequest) []RequestResponseDTO {
	result := make([]RequestResponseDTO, len(items))
	for i, r := range items {
		result[i] = ToRequestResponse(&r)
	}
	return result
}

type SubmitDocumentDTO struct {
	DocumentType   string
	DocumentID     uuid.UUID
	DocumentNumber string
	Amount         float64
	RequesterName  string
}

type SubmitResultDTO struct {
	RequiresApproval bool                `json:"requiresApproval"`
	Request          *RequestResponseDTO `json:"request,omitempty"`
}

type ActOnStepDTO struct {
	ActorName string `json:"actorName"`
	ActorRole string `json:"actorRole"`
	Comments  string `json:"comments"`
}
