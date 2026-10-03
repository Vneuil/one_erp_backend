package application

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/approval/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type ApprovalUseCase interface {
	CreateWorkflow(ctx context.Context, dto CreateWorkflowDTO) (*WorkflowResponseDTO, error)
	ListWorkflows(ctx context.Context, documentType string) ([]WorkflowResponseDTO, error)
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error

	// SubmitDocument is called by other modules (e.g. procurement) when a
	// document is created/submitted. It finds the most specific active
	// workflow matching the document type and amount; if none matches, the
	// document needs no approval (RequiresApproval=false) - this is what
	// makes the whole system "dynamic": a document type with no configured
	// rule is simply never gated.
	SubmitDocument(ctx context.Context, dto SubmitDocumentDTO) (*SubmitResultDTO, error)
	GetActiveRequestForDocument(ctx context.Context, documentType string, documentID uuid.UUID) (*RequestResponseDTO, error)

	ApproveStep(ctx context.Context, requestID uuid.UUID, dto ActOnStepDTO) (*RequestResponseDTO, error)
	RejectStep(ctx context.Context, requestID uuid.UUID, dto ActOnStepDTO) (*RequestResponseDTO, error)

	ListMyPendingApprovals(ctx context.Context, role string) ([]RequestResponseDTO, error)
	ListRequests(ctx context.Context, query types.PaginationQuery, documentType string) ([]RequestResponseDTO, types.PaginationMeta, error)

	// RegisterCallback wires a DocumentStatusCallback for a document type
	// into this use case instance. ApproveStep/RejectStep invoke it exactly
	// once whenever a request of that document type reaches a final
	// approved/rejected state, regardless of which HTTP endpoint drove it
	// there - see callback.go for why this matters.
	RegisterCallback(documentType string, cb DocumentStatusCallback)
}

type approvalUseCase struct {
	repo      domain.ApprovalRepository
	callbacks map[string]DocumentStatusCallback
}

func NewApprovalUseCase(repo domain.ApprovalRepository) ApprovalUseCase {
	return &approvalUseCase{repo: repo, callbacks: make(map[string]DocumentStatusCallback)}
}

func (uc *approvalUseCase) RegisterCallback(documentType string, cb DocumentStatusCallback) {
	if uc.callbacks == nil {
		uc.callbacks = make(map[string]DocumentStatusCallback)
	}
	uc.callbacks[documentType] = cb
}

func (uc *approvalUseCase) CreateWorkflow(ctx context.Context, dto CreateWorkflowDTO) (*WorkflowResponseDTO, error) {
	docType := strings.TrimSpace(dto.DocumentType)
	name := strings.TrimSpace(dto.Name)
	if docType == "" || name == "" {
		return nil, apperrors.NewBadRequest("documentType and name are required")
	}
	if len(dto.Levels) == 0 {
		return nil, apperrors.NewBadRequest("At least one approval level is required")
	}
	if dto.MinAmount < 0 {
		return nil, apperrors.NewBadRequest("minAmount cannot be negative")
	}

	levels := make([]domain.ApprovalWorkflowLevel, len(dto.Levels))
	for i, l := range dto.Levels {
		role := strings.TrimSpace(l.ApproverRole)
		if role == "" {
			return nil, apperrors.NewBadRequest("Each level requires an approverRole")
		}
		levels[i] = domain.ApprovalWorkflowLevel{LevelOrder: i + 1, ApproverRole: role}
	}

	w := &domain.ApprovalWorkflow{
		DocumentType: docType,
		Name:         name,
		MinAmount:    dto.MinAmount,
		IsActive:     true,
		Levels:       levels,
	}
	if err := uc.repo.CreateWorkflow(ctx, w); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create approval workflow")
	}
	res := ToWorkflowResponse(w)
	return &res, nil
}

func (uc *approvalUseCase) ListWorkflows(ctx context.Context, documentType string) ([]WorkflowResponseDTO, error) {
	items, err := uc.repo.ListWorkflows(ctx, documentType)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list approval workflows")
	}
	return ToWorkflowResponseList(items), nil
}

func (uc *approvalUseCase) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	if err := uc.repo.DeleteWorkflow(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete approval workflow")
	}
	return nil
}

func (uc *approvalUseCase) SubmitDocument(ctx context.Context, dto SubmitDocumentDTO) (*SubmitResultDTO, error) {
	matches, err := uc.repo.ListMatchingWorkflows(ctx, dto.DocumentType, dto.Amount)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to match approval workflow")
	}
	if len(matches) == 0 {
		return &SubmitResultDTO{RequiresApproval: false}, nil
	}
	workflow := matches[0]
	if len(workflow.Levels) == 0 {
		return &SubmitResultDTO{RequiresApproval: false}, nil
	}

	steps := make([]domain.ApprovalStep, len(workflow.Levels))
	for i, l := range workflow.Levels {
		steps[i] = domain.ApprovalStep{LevelOrder: l.LevelOrder, ApproverRole: l.ApproverRole, Status: domain.StatusPending}
	}

	req := &domain.ApprovalRequest{
		WorkflowID:     workflow.ID,
		DocumentType:   dto.DocumentType,
		DocumentID:     dto.DocumentID,
		DocumentNumber: dto.DocumentNumber,
		Amount:         dto.Amount,
		RequesterName:  dto.RequesterName,
		Status:         domain.StatusPending,
		CurrentLevel:   1,
		Steps:          steps,
	}
	if err := uc.repo.CreateRequest(ctx, req); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create approval request")
	}

	res := ToRequestResponse(req)
	return &SubmitResultDTO{RequiresApproval: true, Request: &res}, nil
}

func (uc *approvalUseCase) GetActiveRequestForDocument(ctx context.Context, documentType string, documentID uuid.UUID) (*RequestResponseDTO, error) {
	req, err := uc.repo.GetActiveRequestForDocument(ctx, documentType, documentID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get approval request")
	}
	if req == nil {
		return nil, nil
	}
	res := ToRequestResponse(req)
	return &res, nil
}

func (uc *approvalUseCase) currentStep(req *domain.ApprovalRequest) *domain.ApprovalStep {
	for i := range req.Steps {
		if req.Steps[i].LevelOrder == req.CurrentLevel {
			return &req.Steps[i]
		}
	}
	return nil
}

func roleMatches(actorRole, requiredRole string) bool {
	return strings.EqualFold(actorRole, requiredRole) || strings.EqualFold(actorRole, "admin")
}

func (uc *approvalUseCase) ApproveStep(ctx context.Context, requestID uuid.UUID, dto ActOnStepDTO) (*RequestResponseDTO, error) {
	req, err := uc.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get approval request")
	}
	if req == nil {
		return nil, apperrors.NewNotFound("Approval request not found")
	}
	if req.Status != domain.StatusPending {
		return nil, apperrors.NewConflict("This approval request is no longer pending")
	}

	step := uc.currentStep(req)
	if step == nil {
		return nil, apperrors.NewInternal(nil, "Approval request has no current step")
	}
	if !roleMatches(dto.ActorRole, step.ApproverRole) {
		return nil, apperrors.NewForbidden("You are not the approver for the current level")
	}

	now := time.Now()
	step.Status = domain.StatusApproved
	step.ActedByName = dto.ActorName
	step.ActedAt = &now
	step.Comments = dto.Comments
	if err := uc.repo.UpdateStep(ctx, step); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update approval step")
	}

	maxLevel := 0
	for _, s := range req.Steps {
		if s.LevelOrder > maxLevel {
			maxLevel = s.LevelOrder
		}
	}
	finalApproved := req.CurrentLevel >= maxLevel
	if finalApproved {
		req.Status = domain.StatusApproved
	} else {
		req.CurrentLevel++
	}
	if err := uc.repo.UpdateRequest(ctx, req); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update approval request")
	}

	if finalApproved {
		if cb, ok := uc.callbacks[req.DocumentType]; ok && cb != nil {
			if err := cb.OnApproved(ctx, req.DocumentID); err != nil {
				return nil, apperrors.NewInternal(err, "Approval request approved but failed to update the underlying document")
			}
		}
	}

	res := ToRequestResponse(req)
	return &res, nil
}

func (uc *approvalUseCase) RejectStep(ctx context.Context, requestID uuid.UUID, dto ActOnStepDTO) (*RequestResponseDTO, error) {
	req, err := uc.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get approval request")
	}
	if req == nil {
		return nil, apperrors.NewNotFound("Approval request not found")
	}
	if req.Status != domain.StatusPending {
		return nil, apperrors.NewConflict("This approval request is no longer pending")
	}

	step := uc.currentStep(req)
	if step == nil {
		return nil, apperrors.NewInternal(nil, "Approval request has no current step")
	}
	if !roleMatches(dto.ActorRole, step.ApproverRole) {
		return nil, apperrors.NewForbidden("You are not the approver for the current level")
	}

	now := time.Now()
	step.Status = domain.StatusRejected
	step.ActedByName = dto.ActorName
	step.ActedAt = &now
	step.Comments = dto.Comments
	if err := uc.repo.UpdateStep(ctx, step); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update approval step")
	}

	req.Status = domain.StatusRejected
	if err := uc.repo.UpdateRequest(ctx, req); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update approval request")
	}

	if cb, ok := uc.callbacks[req.DocumentType]; ok && cb != nil {
		if err := cb.OnRejected(ctx, req.DocumentID); err != nil {
			return nil, apperrors.NewInternal(err, "Approval request rejected but failed to update the underlying document")
		}
	}

	res := ToRequestResponse(req)
	return &res, nil
}

func (uc *approvalUseCase) ListMyPendingApprovals(ctx context.Context, role string) ([]RequestResponseDTO, error) {
	steps, err := uc.repo.ListPendingStepsForRole(ctx, role)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list pending approvals")
	}

	seen := make(map[uuid.UUID]bool)
	result := make([]RequestResponseDTO, 0, len(steps))
	for _, s := range steps {
		if seen[s.RequestID] {
			continue
		}
		seen[s.RequestID] = true
		req, err := uc.repo.GetRequestByID(ctx, s.RequestID)
		if err != nil || req == nil {
			continue
		}
		result = append(result, ToRequestResponse(req))
	}
	return result, nil
}

func (uc *approvalUseCase) ListRequests(ctx context.Context, query types.PaginationQuery, documentType string) ([]RequestResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListRequests(ctx, query, documentType)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list approval requests")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToRequestResponseList(items), meta, nil
}
