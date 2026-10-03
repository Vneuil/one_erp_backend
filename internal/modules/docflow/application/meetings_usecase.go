package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/ai"
	"github.com/divinecoid/one-backend/internal/modules/docflow/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var validMeetingStatuses = map[string]bool{
	"scheduled":   true,
	"in_progress": true,
	"completed":   true,
	"cancelled":   true,
}

type MeetingsUseCase interface {
	CreateMeeting(ctx context.Context, dto CreateMeetingDTO) (*MeetingResponseDTO, error)
	GetMeetingByID(ctx context.Context, id uuid.UUID) (*MeetingResponseDTO, error)
	ListMeetings(ctx context.Context, query types.PaginationQuery) ([]MeetingResponseDTO, types.PaginationMeta, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, dto UpdateMeetingStatusDTO) (*MeetingResponseDTO, error)
	AddNote(ctx context.Context, meetingID uuid.UUID, dto AddNoteDTO) (*MeetingNoteResponseDTO, error)
	GenerateAINote(ctx context.Context, meetingID uuid.UUID, audioBytes []byte, audioFilename string) (*MeetingNoteResponseDTO, error)
	ListNotes(ctx context.Context, meetingID uuid.UUID) ([]MeetingNoteResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type meetingsUseCase struct {
	repo domain.MeetingsRepository
	ai   ai.Client
}

func NewMeetingsUseCase(repo domain.MeetingsRepository, aiClient ai.Client) MeetingsUseCase {
	return &meetingsUseCase{repo: repo, ai: aiClient}
}

func generateSlug() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func (uc *meetingsUseCase) CreateMeeting(ctx context.Context, dto CreateMeetingDTO) (*MeetingResponseDTO, error) {
	if dto.Title == "" {
		return nil, apperrors.NewBadRequest("Meeting title is required")
	}
	if dto.ScheduledAt == "" {
		return nil, apperrors.NewBadRequest("Scheduled at is required")
	}
	if dto.OrganizerName == "" {
		return nil, apperrors.NewBadRequest("Organizer name is required")
	}
	if dto.DurationMinutes <= 0 {
		dto.DurationMinutes = 30
	}

	m := &domain.Meeting{
		CompanyID:       dto.CompanyID,
		Title:           dto.Title,
		Description:     dto.Description,
		ScheduledAt:     dto.ScheduledAt,
		DurationMinutes: dto.DurationMinutes,
		OrganizerName:   dto.OrganizerName,
		MeetingURL:      fmt.Sprintf("https://meet.one-erp.com/%s", generateSlug()),
		Status:          "scheduled",
		AttendeeNames:   dto.AttendeeNames,
	}
	if err := uc.repo.CreateMeeting(ctx, m); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create meeting")
	}
	return ToMeetingResponse(m), nil
}

func (uc *meetingsUseCase) GetMeetingByID(ctx context.Context, id uuid.UUID) (*MeetingResponseDTO, error) {
	m, err := uc.repo.GetMeetingByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get meeting")
	}
	if m == nil {
		return nil, apperrors.NewNotFound("Meeting not found")
	}
	return ToMeetingResponse(m), nil
}

func (uc *meetingsUseCase) ListMeetings(ctx context.Context, query types.PaginationQuery) ([]MeetingResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.ListMeetings(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list meetings")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToMeetingResponseList(items), meta, nil
}

func (uc *meetingsUseCase) UpdateStatus(ctx context.Context, id uuid.UUID, dto UpdateMeetingStatusDTO) (*MeetingResponseDTO, error) {
	m, err := uc.repo.GetMeetingByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get meeting")
	}
	if m == nil {
		return nil, apperrors.NewNotFound("Meeting not found")
	}
	if !validMeetingStatuses[dto.Status] {
		return nil, apperrors.NewBadRequest("Status must be one of scheduled, in_progress, completed, cancelled")
	}
	m.Status = dto.Status
	if err := uc.repo.UpdateMeeting(ctx, m); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update meeting status")
	}
	return ToMeetingResponse(m), nil
}

func (uc *meetingsUseCase) AddNote(ctx context.Context, meetingID uuid.UUID, dto AddNoteDTO) (*MeetingNoteResponseDTO, error) {
	m, err := uc.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get meeting")
	}
	if m == nil {
		return nil, apperrors.NewNotFound("Meeting not found")
	}
	if dto.Content == "" {
		return nil, apperrors.NewBadRequest("Note content is required")
	}
	n := &domain.MeetingNote{
		MeetingID:   m.ID,
		Content:     dto.Content,
		AIGenerated: false,
	}
	if err := uc.repo.CreateNote(ctx, n); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to add note")
	}
	return ToMeetingNoteResponse(n), nil
}

// GenerateAINote produces an AI-generated meeting note by transcribing the
// given audio (if any) and asking the chat model to summarize it into a
// structured note (summary, decisions, action items). Requires OpenAI to be
// configured — if it is not, this returns a clear ServiceUnavailable error
// rather than silently degrading to a templated note. Use AddNote for manual
// notes, which remain unaffected by AI configuration.
func (uc *meetingsUseCase) GenerateAINote(ctx context.Context, meetingID uuid.UUID, audioBytes []byte, audioFilename string) (*MeetingNoteResponseDTO, error) {
	if !uc.ai.Enabled() {
		return nil, apperrors.NewServiceUnavailable("AI note generation is not configured. Please set OPENAI_API_KEY in the backend .env file.")
	}

	m, err := uc.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get meeting")
	}
	if m == nil {
		return nil, apperrors.NewNotFound("Meeting not found")
	}

	attendees := m.AttendeeNames
	if attendees == "" {
		attendees = "belum ditentukan"
	}

	var content string
	if len(audioBytes) > 0 {
		content, err = uc.generateAINoteFromAudio(ctx, m, attendees, audioBytes, audioFilename)
	} else {
		content, err = uc.ai.Summarize(ctx, fmt.Sprintf(
			"You are an assistant summarizing a business meeting into structured notes.\n\n"+
				"Meeting title: %s\nDescription: %s\nAttendees: %s\n\n"+
				"No transcript is available for this meeting. Produce structured meeting notes with three sections: "+
				"\"Summary\", \"Key Decisions\", and \"Action Items\", based only on the information above.",
			m.Title, m.Description, attendees,
		))
	}
	if err != nil {
		slog.Warn("AI meeting note generation failed", "meetingId", m.ID, "error", err)
		return nil, apperrors.NewServiceUnavailable("AI note generation failed: " + err.Error())
	}

	n := &domain.MeetingNote{
		MeetingID:   m.ID,
		Content:     content,
		AIGenerated: true,
	}
	if err := uc.repo.CreateNote(ctx, n); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to generate AI note")
	}
	return ToMeetingNoteResponse(n), nil
}

func (uc *meetingsUseCase) generateAINoteFromAudio(ctx context.Context, m *domain.Meeting, attendees string, audioBytes []byte, audioFilename string) (string, error) {
	transcript, err := uc.ai.Transcribe(ctx, audioBytes, audioFilename)
	if err != nil {
		return "", fmt.Errorf("transcription failed: %w", err)
	}

	prompt := fmt.Sprintf(
		"You are an assistant summarizing a business meeting into structured notes.\n\n"+
			"Meeting title: %s\nDescription: %s\nAttendees: %s\n\nTranscript:\n%s\n\n"+
			"Produce structured meeting notes with three sections: \"Summary\", \"Key Decisions\", and \"Action Items\". "+
			"Be concise and factual, based only on the transcript.",
		m.Title, m.Description, attendees, transcript,
	)

	summary, err := uc.ai.Summarize(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summarization failed: %w", err)
	}
	return summary, nil
}

func (uc *meetingsUseCase) ListNotes(ctx context.Context, meetingID uuid.UUID) ([]MeetingNoteResponseDTO, error) {
	m, err := uc.repo.GetMeetingByID(ctx, meetingID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get meeting")
	}
	if m == nil {
		return nil, apperrors.NewNotFound("Meeting not found")
	}
	items, err := uc.repo.ListNotesByMeeting(ctx, meetingID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list notes")
	}
	return ToMeetingNoteResponseList(items), nil
}

// SeedInitialData populates a few sample meetings with notes on first boot
func (uc *meetingsUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountMeetings(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now()
	seeds := []struct {
		dto    CreateMeetingDTO
		status string
		notes  []string
		aiNote bool
	}{
		{
			dto: CreateMeetingDTO{
				Title:           "Koordinasi Operasional Roll-out WMS Cikarang & Surabaya",
				Description:     "Integrasi barcode scanner dan sinkronisasi stok marketplace",
				ScheduledAt:     now.AddDate(0, 0, -1).Format(time.RFC3339),
				DurationMinutes: 90,
				OrganizerName:   "Nicholas Tantra",
				AttendeeNames:   "Nicholas Tantra, Budi Santoso, Dewi Lestari",
			},
			status: "completed",
			notes:  []string{"Membahas integrasi barcode scanner di loading dock gudang Cikarang, sinkronisasi stok otomatis dengan marketplace, serta pengaturan cut-off akuntansi akhir bulan."},
			aiNote: true,
		},
		{
			dto: CreateMeetingDTO{
				Title:           "Evaluasi Kontrak Suplai Plat Baja Corrugated Q4",
				Description:     "Negosiasi diskon volume pemesanan plat baja",
				ScheduledAt:     now.AddDate(0, 0, -2).Format(time.RFC3339),
				DurationMinutes: 60,
				OrganizerName:   "Nicholas Tantra",
				AttendeeNames:   "Nicholas Tantra, Rizky Ramadhan, Ir. Bambang",
			},
			status: "completed",
			notes:  []string{"Negosiasi diskon volume pemesanan 500 ton plat baja dengan PT Indo Steel Perkasa."},
		},
		{
			dto: CreateMeetingDTO{
				Title:           "Review Pencapaian Target Closing Q3",
				Description:     "Evaluasi capaian target penjualan kuartal berjalan",
				ScheduledAt:     now.AddDate(0, 0, 3).Format(time.RFC3339),
				DurationMinutes: 45,
				OrganizerName:   "Nicholas Tantra",
				AttendeeNames:   "Nicholas Tantra, Budi Santoso",
			},
			status: "scheduled",
		},
	}

	for _, seed := range seeds {
		res, err := uc.CreateMeeting(ctx, seed.dto)
		if err != nil {
			continue
		}
		if seed.status != "" && seed.status != "scheduled" {
			if _, err := uc.UpdateStatus(ctx, res.ID, UpdateMeetingStatusDTO{Status: seed.status}); err != nil {
				continue
			}
		}
		for _, note := range seed.notes {
			if _, err := uc.AddNote(ctx, res.ID, AddNoteDTO{Content: note}); err != nil {
				continue
			}
		}
		if seed.aiNote {
			attendees := seed.dto.AttendeeNames
			if attendees == "" {
				attendees = "belum ditentukan"
			}
			_, _ = uc.AddNote(ctx, res.ID, AddNoteDTO{
				Content: fmt.Sprintf("Meeting covered: %s. Attendees: %s. Action items to be determined.", seed.dto.Title, attendees),
			})
		}
	}
	return nil
}
