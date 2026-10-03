package application

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/projectticket/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var ticketNoRand = rand.New(rand.NewSource(time.Now().UnixNano()))

var validTicketStatuses = map[string]bool{
	"open":             true,
	"in_progress":      true,
	"pending_customer": true,
	"resolved":         true,
}

func generateTicketNo() string {
	year := time.Now().Year()
	seq := ticketNoRand.Intn(10000)
	return fmt.Sprintf("PTK-%d-%04d", year, seq)
}

type ProjectTicketUseCase interface {
	Create(ctx context.Context, dto CreateProjectTicketDTO) (*ProjectTicketResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*ProjectTicketResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery, status string) ([]ProjectTicketResponseDTO, types.PaginationMeta, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, dto UpdateStatusDTO) (*ProjectTicketResponseDTO, error)
	SeedInitialData(ctx context.Context) error
}

type projectTicketUseCase struct {
	repo domain.ProjectTicketRepository
}

func NewProjectTicketUseCase(repo domain.ProjectTicketRepository) ProjectTicketUseCase {
	return &projectTicketUseCase{repo: repo}
}

func (uc *projectTicketUseCase) Create(ctx context.Context, dto CreateProjectTicketDTO) (*ProjectTicketResponseDTO, error) {
	subject := strings.TrimSpace(dto.Subject)
	projectCode := strings.TrimSpace(dto.ProjectCode)
	customerName := strings.TrimSpace(dto.CustomerName)
	if subject == "" || projectCode == "" || customerName == "" {
		return nil, apperrors.NewBadRequest("ProjectCode, Subject, and CustomerName are required")
	}

	ticket := &domain.ProjectTicket{
		TicketNo:       generateTicketNo(),
		ProjectCode:    projectCode,
		Subject:        subject,
		CustomerName:   customerName,
		Category:       dto.Category,
		Priority:       dto.Priority,
		SLATargetHours: dto.SLATargetHours,
		Status:         "open",
		AssignedTo:     dto.AssignedTo,
	}

	if err := uc.repo.Create(ctx, ticket); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create project ticket")
	}
	return ToProjectTicketResponse(ticket), nil
}

func (uc *projectTicketUseCase) GetByID(ctx context.Context, id uuid.UUID) (*ProjectTicketResponseDTO, error) {
	ticket, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve project ticket")
	}
	if ticket == nil {
		return nil, apperrors.NewNotFound("Project ticket not found")
	}
	return ToProjectTicketResponse(ticket), nil
}

func (uc *projectTicketUseCase) List(ctx context.Context, query types.PaginationQuery, status string) ([]ProjectTicketResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	tickets, total, err := uc.repo.List(ctx, query, status)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list project tickets")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToProjectTicketResponseList(tickets), meta, nil
}

func (uc *projectTicketUseCase) UpdateStatus(ctx context.Context, id uuid.UUID, dto UpdateStatusDTO) (*ProjectTicketResponseDTO, error) {
	ticket, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get project ticket")
	}
	if ticket == nil {
		return nil, apperrors.NewNotFound("Project ticket not found")
	}
	if !validTicketStatuses[dto.Status] {
		return nil, apperrors.NewBadRequest("Status must be one of open, in_progress, pending_customer, resolved")
	}
	ticket.Status = dto.Status
	if err := uc.repo.Update(ctx, ticket); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update project ticket status")
	}
	return ToProjectTicketResponse(ticket), nil
}

func (uc *projectTicketUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return nil
}
