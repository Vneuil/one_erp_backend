package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/docflow/domain"
	"github.com/google/uuid"
)

// Forms

type CreateFormDTO struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Fields      string     `json:"fields"`
	CreatedBy   string     `json:"createdBy"`
	CompanyID   *uuid.UUID `json:"companyId,omitempty"`
}

type SubmitResponseDTO struct {
	SubmittedBy string `json:"submittedBy"`
	Answers     string `json:"answers"`
}

type FormResponseDTO struct {
	ID            uuid.UUID  `json:"id"`
	CompanyID     *uuid.UUID `json:"companyId,omitempty"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Fields        string     `json:"fields"`
	CreatedBy     string     `json:"createdBy"`
	Status        string     `json:"status"`
	ResponseCount int64      `json:"responseCount"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type FormSubmissionResponseDTO struct {
	ID          uuid.UUID `json:"id"`
	FormID      uuid.UUID `json:"formId"`
	SubmittedBy string    `json:"submittedBy"`
	Answers     string    `json:"answers"`
	SubmittedAt string    `json:"submittedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

func ToFormResponse(f *domain.Form) *FormResponseDTO {
	if f == nil {
		return nil
	}
	return &FormResponseDTO{
		ID:            f.ID,
		CompanyID:     f.CompanyID,
		Title:         f.Title,
		Description:   f.Description,
		Fields:        f.Fields,
		CreatedBy:     f.CreatedBy,
		Status:        f.Status,
		ResponseCount: f.ResponseCount,
		CreatedAt:     f.CreatedAt,
	}
}

func ToFormResponseList(forms []domain.Form) []FormResponseDTO {
	result := make([]FormResponseDTO, len(forms))
	for i, f := range forms {
		result[i] = *ToFormResponse(&f)
	}
	return result
}

func ToFormSubmissionResponse(r *domain.FormResponse) *FormSubmissionResponseDTO {
	if r == nil {
		return nil
	}
	return &FormSubmissionResponseDTO{
		ID:          r.ID,
		FormID:      r.FormID,
		SubmittedBy: r.SubmittedBy,
		Answers:     r.Answers,
		SubmittedAt: r.SubmittedAt,
		CreatedAt:   r.CreatedAt,
	}
}

func ToFormSubmissionResponseList(items []domain.FormResponse) []FormSubmissionResponseDTO {
	result := make([]FormSubmissionResponseDTO, len(items))
	for i, r := range items {
		result[i] = *ToFormSubmissionResponse(&r)
	}
	return result
}

// Signatures

type UploadDocumentDTO struct {
	Title    string `json:"title"`
	FileName string `json:"fileName"`
	// UploadedBy is set by the handler from the authenticated caller, never
	// trusted from client input.
	UploadedBy string     `json:"-"`
	CompanyID  *uuid.UUID `json:"companyId,omitempty"`
	Content    []byte     `json:"-"`
	MimeType   string     `json:"-"`
}

type DocumentDownloadDTO struct {
	SignatureDocumentResponseDTO
	ContentType string `json:"-"`
	Binary      []byte `json:"-"`
}

type AddSignerDTO struct {
	SignerName  string `json:"signerName"`
	SignerEmail string `json:"signerEmail"`
	OrderIndex  int    `json:"orderIndex"`
}

type SignatureDocumentResponseDTO struct {
	ID         uuid.UUID  `json:"id"`
	CompanyID  *uuid.UUID `json:"companyId,omitempty"`
	Title      string     `json:"title"`
	FileName   string     `json:"fileName"`
	Status     string     `json:"status"`
	UploadedBy string     `json:"uploadedBy"`
	UploadedAt string     `json:"uploadedAt"`
	CreatedAt  time.Time  `json:"createdAt"`
}

type SignatureRequestResponseDTO struct {
	ID          uuid.UUID `json:"id"`
	DocumentID  uuid.UUID `json:"documentId"`
	SignerName  string    `json:"signerName"`
	SignerEmail string    `json:"signerEmail"`
	Status      string    `json:"status"`
	OrderIndex  int       `json:"orderIndex"`
	SignedAt    *string   `json:"signedAt,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

func ToSignatureDocumentResponse(d *domain.SignatureDocument) *SignatureDocumentResponseDTO {
	if d == nil {
		return nil
	}
	return &SignatureDocumentResponseDTO{
		ID:         d.ID,
		CompanyID:  d.CompanyID,
		Title:      d.Title,
		FileName:   d.FileName,
		Status:     d.Status,
		UploadedBy: d.UploadedBy,
		UploadedAt: d.UploadedAt,
		CreatedAt:  d.CreatedAt,
	}
}

func ToSignatureDocumentResponseList(items []domain.SignatureDocument) []SignatureDocumentResponseDTO {
	result := make([]SignatureDocumentResponseDTO, len(items))
	for i, d := range items {
		result[i] = *ToSignatureDocumentResponse(&d)
	}
	return result
}

func ToSignatureRequestResponse(r *domain.SignatureRequest) *SignatureRequestResponseDTO {
	if r == nil {
		return nil
	}
	return &SignatureRequestResponseDTO{
		ID:          r.ID,
		DocumentID:  r.DocumentID,
		SignerName:  r.SignerName,
		SignerEmail: r.SignerEmail,
		Status:      r.Status,
		OrderIndex:  r.OrderIndex,
		SignedAt:    r.SignedAt,
		CreatedAt:   r.CreatedAt,
	}
}

func ToSignatureRequestResponseList(items []domain.SignatureRequest) []SignatureRequestResponseDTO {
	result := make([]SignatureRequestResponseDTO, len(items))
	for i, r := range items {
		result[i] = *ToSignatureRequestResponse(&r)
	}
	return result
}

type DocumentWithSignersDTO struct {
	SignatureDocumentResponseDTO
	Signers []SignatureRequestResponseDTO `json:"signers"`
}

// Meetings

type CreateMeetingDTO struct {
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	ScheduledAt     string     `json:"scheduledAt"`
	DurationMinutes int        `json:"durationMinutes"`
	OrganizerName   string     `json:"organizerName"`
	AttendeeNames   string     `json:"attendeeNames"`
	CompanyID       *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateMeetingStatusDTO struct {
	Status string `json:"status"`
}

type AddNoteDTO struct {
	Content string `json:"content"`
}

type MeetingResponseDTO struct {
	ID              uuid.UUID  `json:"id"`
	CompanyID       *uuid.UUID `json:"companyId,omitempty"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	ScheduledAt     string     `json:"scheduledAt"`
	DurationMinutes int        `json:"durationMinutes"`
	OrganizerName   string     `json:"organizerName"`
	MeetingURL      string     `json:"meetingUrl"`
	Status          string     `json:"status"`
	AttendeeNames   string     `json:"attendeeNames"`
	CreatedAt       time.Time  `json:"createdAt"`
}

type MeetingNoteResponseDTO struct {
	ID          uuid.UUID `json:"id"`
	MeetingID   uuid.UUID `json:"meetingId"`
	Content     string    `json:"content"`
	AIGenerated bool      `json:"aiGenerated"`
	CreatedAt   time.Time `json:"createdAt"`
}

func ToMeetingResponse(m *domain.Meeting) *MeetingResponseDTO {
	if m == nil {
		return nil
	}
	return &MeetingResponseDTO{
		ID:              m.ID,
		CompanyID:       m.CompanyID,
		Title:           m.Title,
		Description:     m.Description,
		ScheduledAt:     m.ScheduledAt,
		DurationMinutes: m.DurationMinutes,
		OrganizerName:   m.OrganizerName,
		MeetingURL:      m.MeetingURL,
		Status:          m.Status,
		AttendeeNames:   m.AttendeeNames,
		CreatedAt:       m.CreatedAt,
	}
}

func ToMeetingResponseList(items []domain.Meeting) []MeetingResponseDTO {
	result := make([]MeetingResponseDTO, len(items))
	for i, m := range items {
		result[i] = *ToMeetingResponse(&m)
	}
	return result
}

func ToMeetingNoteResponse(n *domain.MeetingNote) *MeetingNoteResponseDTO {
	if n == nil {
		return nil
	}
	return &MeetingNoteResponseDTO{
		ID:          n.ID,
		MeetingID:   n.MeetingID,
		Content:     n.Content,
		AIGenerated: n.AIGenerated,
		CreatedAt:   n.CreatedAt,
	}
}

func ToMeetingNoteResponseList(items []domain.MeetingNote) []MeetingNoteResponseDTO {
	result := make([]MeetingNoteResponseDTO, len(items))
	for i, n := range items {
		result[i] = *ToMeetingNoteResponse(&n)
	}
	return result
}
