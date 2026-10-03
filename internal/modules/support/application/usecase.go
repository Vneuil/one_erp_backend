package application

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/support/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var ticketNumberRand = rand.New(rand.NewSource(time.Now().UnixNano()))

var validPriorities = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
	"urgent": true,
}

var validStatuses = map[string]bool{
	"open":             true,
	"in_progress":      true,
	"waiting_customer": true,
	"resolved":         true,
	"closed":           true,
}

var validAuthorTypes = map[string]bool{
	"customer": true,
	"agent":    true,
}

var slaDurations = map[string]time.Duration{
	"urgent": 4 * time.Hour,
	"high":   24 * time.Hour,
	"medium": 48 * time.Hour,
	"low":    72 * time.Hour,
}

type TicketsUseCase interface {
	CreateTicket(ctx context.Context, dto CreateTicketDTO) (*TicketResponseDTO, error)
	GetTicketByID(ctx context.Context, id uuid.UUID) (*TicketResponseDTO, error)
	ListTickets(ctx context.Context, query types.PaginationQuery) ([]TicketResponseDTO, types.PaginationMeta, error)
	UpdateTicket(ctx context.Context, id uuid.UUID, dto UpdateTicketDTO) (*TicketResponseDTO, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, dto UpdateStatusDTO) (*TicketResponseDTO, error)
	AddReply(ctx context.Context, ticketID uuid.UUID, dto ReplyDTO) (*ReplyResponseDTO, error)
	ListReplies(ctx context.Context, ticketID uuid.UUID) ([]ReplyResponseDTO, error)
	GetSummary(ctx context.Context) (*domain.TicketSummary, error)

	SeedInitialData(ctx context.Context) error
}

type ticketsUseCase struct {
	repo domain.TicketsRepository
}

func NewTicketsUseCase(repo domain.TicketsRepository) TicketsUseCase {
	return &ticketsUseCase{repo: repo}
}

func generateTicketNumber() string {
	year := time.Now().Year()
	seq := ticketNumberRand.Intn(1000000)
	return fmt.Sprintf("TCK-%d-%06d", year, seq)
}

func (uc *ticketsUseCase) CreateTicket(ctx context.Context, dto CreateTicketDTO) (*TicketResponseDTO, error) {
	if dto.Subject == "" {
		return nil, apperrors.NewBadRequest("Ticket subject is required")
	}
	if dto.CustomerName == "" {
		return nil, apperrors.NewBadRequest("Customer name is required")
	}
	if dto.Category == "" {
		return nil, apperrors.NewBadRequest("Category is required")
	}
	if !validPriorities[dto.Priority] {
		return nil, apperrors.NewBadRequest("Priority must be one of low, medium, high, urgent")
	}

	now := time.Now()
	t := &domain.Ticket{
		CompanyID:    dto.CompanyID,
		TicketNumber: generateTicketNumber(),
		Subject:      dto.Subject,
		Description:  dto.Description,
		CustomerName: dto.CustomerName,
		Priority:     dto.Priority,
		Status:       "open",
		Category:     dto.Category,
		AssignedTo:   dto.AssignedTo,
		SLADueAt:     now.Add(slaDurations[dto.Priority]),
	}

	if err := uc.repo.CreateTicket(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create ticket")
	}
	return ToTicketResponse(t), nil
}

func (uc *ticketsUseCase) GetTicketByID(ctx context.Context, id uuid.UUID) (*TicketResponseDTO, error) {
	t, err := uc.repo.GetTicketByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get ticket")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Ticket not found")
	}
	return ToTicketResponse(t), nil
}

func (uc *ticketsUseCase) ListTickets(ctx context.Context, query types.PaginationQuery) ([]TicketResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	tickets, total, err := uc.repo.ListTickets(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list tickets")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToTicketResponseList(tickets), meta, nil
}

func (uc *ticketsUseCase) UpdateTicket(ctx context.Context, id uuid.UUID, dto UpdateTicketDTO) (*TicketResponseDTO, error) {
	t, err := uc.repo.GetTicketByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get ticket")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Ticket not found")
	}
	if dto.Subject != "" {
		t.Subject = dto.Subject
	}
	if dto.Description != "" {
		t.Description = dto.Description
	}
	if dto.CustomerName != "" {
		t.CustomerName = dto.CustomerName
	}
	if dto.Priority != "" {
		if !validPriorities[dto.Priority] {
			return nil, apperrors.NewBadRequest("Priority must be one of low, medium, high, urgent")
		}
		t.Priority = dto.Priority
	}
	if dto.Category != "" {
		t.Category = dto.Category
	}
	if dto.AssignedTo != "" {
		t.AssignedTo = dto.AssignedTo
	}

	if err := uc.repo.UpdateTicket(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update ticket")
	}
	return ToTicketResponse(t), nil
}

func (uc *ticketsUseCase) UpdateStatus(ctx context.Context, id uuid.UUID, dto UpdateStatusDTO) (*TicketResponseDTO, error) {
	t, err := uc.repo.GetTicketByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get ticket")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Ticket not found")
	}
	if !validStatuses[dto.Status] {
		return nil, apperrors.NewBadRequest("Status must be one of open, in_progress, waiting_customer, resolved, closed")
	}
	t.Status = dto.Status
	if dto.Status == "resolved" {
		now := time.Now()
		t.ResolvedAt = &now
	}

	if err := uc.repo.UpdateTicket(ctx, t); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update ticket status")
	}
	return ToTicketResponse(t), nil
}

func (uc *ticketsUseCase) AddReply(ctx context.Context, ticketID uuid.UUID, dto ReplyDTO) (*ReplyResponseDTO, error) {
	t, err := uc.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get ticket")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Ticket not found")
	}
	if dto.AuthorName == "" {
		return nil, apperrors.NewBadRequest("Author name is required")
	}
	if !validAuthorTypes[dto.AuthorType] {
		return nil, apperrors.NewBadRequest("Author type must be one of customer, agent")
	}
	if dto.Message == "" {
		return nil, apperrors.NewBadRequest("Message is required")
	}

	r := &domain.TicketReply{
		TicketID:   ticketID,
		AuthorName: dto.AuthorName,
		AuthorType: dto.AuthorType,
		Message:    dto.Message,
	}
	if err := uc.repo.CreateReply(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to add reply")
	}
	return ToReplyResponse(r), nil
}

func (uc *ticketsUseCase) ListReplies(ctx context.Context, ticketID uuid.UUID) ([]ReplyResponseDTO, error) {
	t, err := uc.repo.GetTicketByID(ctx, ticketID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get ticket")
	}
	if t == nil {
		return nil, apperrors.NewNotFound("Ticket not found")
	}
	replies, err := uc.repo.ListRepliesByTicket(ctx, ticketID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list replies")
	}
	return ToReplyResponseList(replies), nil
}

func (uc *ticketsUseCase) GetSummary(ctx context.Context) (*domain.TicketSummary, error) {
	tickets, err := uc.repo.ListAllTickets(ctx)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list tickets")
	}
	summary := &domain.TicketSummary{
		ByPriority: map[string]int64{"low": 0, "medium": 0, "high": 0, "urgent": 0},
	}
	var totalResolutionHours float64
	var resolvedCount int64
	now := time.Now()
	for _, t := range tickets {
		summary.TotalTickets++
		summary.ByPriority[t.Priority]++
		switch t.Status {
		case "open":
			summary.OpenTickets++
		case "in_progress":
			summary.InProgressTickets++
		case "waiting_customer":
			summary.WaitingTickets++
		case "resolved":
			summary.ResolvedTickets++
		case "closed":
			summary.ClosedTickets++
		}
		if t.Status != "resolved" && t.Status != "closed" && now.After(t.SLADueAt) {
			summary.OverdueTickets++
		}
		if t.ResolvedAt != nil {
			totalResolutionHours += t.ResolvedAt.Sub(t.CreatedAt).Hours()
			resolvedCount++
		}
	}
	if resolvedCount > 0 {
		summary.AvgResolutionHours = totalResolutionHours / float64(resolvedCount)
	}
	return summary, nil
}

// SeedInitialData populates a few sample tickets on first boot
func (uc *ticketsUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountTickets(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()
	seeds := []struct {
		dto         CreateTicketDTO
		replies     []ReplyDTO
		overrideDue *time.Time
		resolve     bool
	}{
		{
			dto: CreateTicketDTO{
				Subject:      "Tidak bisa login ke aplikasi",
				Description:  "Pelanggan melaporkan error saat login sejak pagi ini",
				CustomerName: "PT Sinar Abadi",
				Priority:     "urgent",
				Category:     "Technical",
				AssignedTo:   "Rizky Pratama",
			},
			overrideDue: func() *time.Time { d := now.Add(-2 * time.Hour); return &d }(),
		},
		{
			dto: CreateTicketDTO{
				Subject:      "Tagihan bulan ini tidak sesuai",
				Description:  "Selisih jumlah tagihan dengan yang disepakati",
				CustomerName: "CV Makmur Jaya",
				Priority:     "high",
				Category:     "Billing",
				AssignedTo:   "Dewi Lestari",
			},
			replies: []ReplyDTO{
				{AuthorName: "CV Makmur Jaya", AuthorType: "customer", Message: "Mohon segera diperiksa, ini sudah terjadi 2 bulan berturut-turut"},
				{AuthorName: "Dewi Lestari", AuthorType: "agent", Message: "Baik, kami sedang mengecek ke tim finance, mohon ditunggu"},
			},
		},
		{
			dto: CreateTicketDTO{
				Subject:      "Permintaan informasi paket layanan",
				Description:  "Ingin tahu detail paket enterprise",
				CustomerName: "Budi Santoso",
				Priority:     "low",
				Category:     "General Inquiry",
				AssignedTo:   "Sarah Amelia",
			},
		},
		{
			dto: CreateTicketDTO{
				Subject:      "Fitur ekspor laporan error",
				Description:  "Ekspor ke Excel menghasilkan file kosong",
				CustomerName: "PT Cahaya Nusantara",
				Priority:     "medium",
				Category:     "Technical",
				AssignedTo:   "Rizky Pratama",
			},
			resolve: true,
		},
		{
			dto: CreateTicketDTO{
				Subject:      "Konfirmasi pembatalan langganan",
				Description:  "Pelanggan meminta konfirmasi status pembatalan",
				CustomerName: "Toko Berkah",
				Priority:     "medium",
				Category:     "Billing",
				AssignedTo:   "Dewi Lestari",
			},
		},
	}

	for _, seed := range seeds {
		res, err := uc.CreateTicket(ctx, seed.dto)
		if err != nil {
			continue
		}
		if seed.overrideDue != nil {
			t, gErr := uc.repo.GetTicketByID(ctx, res.ID)
			if gErr == nil && t != nil {
				t.SLADueAt = *seed.overrideDue
				_ = uc.repo.UpdateTicket(ctx, t)
			}
		}
		for _, rp := range seed.replies {
			if _, err := uc.AddReply(ctx, res.ID, rp); err != nil {
				continue
			}
		}
		if seed.resolve {
			_, _ = uc.UpdateStatus(ctx, res.ID, UpdateStatusDTO{Status: "resolved"})
		}
	}
	return nil
}
