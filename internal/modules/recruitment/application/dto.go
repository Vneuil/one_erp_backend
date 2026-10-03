package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/recruitment/domain"
	"github.com/google/uuid"
)

type CreateJobVacancyDTO struct {
	Title          string     `json:"title"`
	Department     string     `json:"department"`
	EmploymentType string     `json:"employmentType"`
	Location       string     `json:"location"`
	Description    string     `json:"description"`
	OpeningsCount  int        `json:"openingsCount"`
	PostedDate     string     `json:"postedDate"`
	CompanyID      *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateJobVacancyDTO struct {
	Title          string `json:"title"`
	Department     string `json:"department"`
	EmploymentType string `json:"employmentType"`
	Location       string `json:"location"`
	Description    string `json:"description"`
	Status         string `json:"status"`
	OpeningsCount  int    `json:"openingsCount"`
	PostedDate     string `json:"postedDate"`
}

type JobVacancyResponseDTO struct {
	ID             uuid.UUID  `json:"id"`
	CompanyID      *uuid.UUID `json:"companyId,omitempty"`
	Title          string     `json:"title"`
	Department     string     `json:"department"`
	EmploymentType string     `json:"employmentType"`
	Location       string     `json:"location"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	OpeningsCount  int        `json:"openingsCount"`
	PostedDate     string     `json:"postedDate"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type CreateCandidateDTO struct {
	JobVacancyID uuid.UUID  `json:"jobVacancyId"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	ResumeNote   string     `json:"resumeNote"`
	Source       string     `json:"source"`
	AppliedDate  string     `json:"appliedDate"`
	CompanyID    *uuid.UUID `json:"companyId,omitempty"`
}

type AdvanceStageDTO struct {
	Stage string `json:"stage"`
	Notes string `json:"notes"`
}

type CandidateResponseDTO struct {
	ID           uuid.UUID  `json:"id"`
	CompanyID    *uuid.UUID `json:"companyId,omitempty"`
	JobVacancyID uuid.UUID  `json:"jobVacancyId"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	Phone        string     `json:"phone"`
	ResumeNote   string     `json:"resumeNote"`
	Source       string     `json:"source"`
	AppliedDate  string     `json:"appliedDate"`
	Stage        string     `json:"stage"`
	StageNotes   string     `json:"stageNotes"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type CreateInterviewDTO struct {
	CandidateID     uuid.UUID  `json:"candidateId"`
	ScheduledAt     string     `json:"scheduledAt"`
	InterviewerName string     `json:"interviewerName"`
	CompanyID       *uuid.UUID `json:"companyId,omitempty"`
}

type UpdateInterviewDTO struct {
	Status       string `json:"status"`
	FeedbackNote string `json:"feedbackNote"`
	Rating       *int   `json:"rating,omitempty"`
}

type InterviewResponseDTO struct {
	ID              uuid.UUID  `json:"id"`
	CompanyID       *uuid.UUID `json:"companyId,omitempty"`
	CandidateID     uuid.UUID  `json:"candidateId"`
	ScheduledAt     string     `json:"scheduledAt"`
	InterviewerName string     `json:"interviewerName"`
	Status          string     `json:"status"`
	FeedbackNote    string     `json:"feedbackNote"`
	Rating          *int       `json:"rating,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

func ToJobVacancyResponse(v *domain.JobVacancy) *JobVacancyResponseDTO {
	if v == nil {
		return nil
	}
	return &JobVacancyResponseDTO{
		ID:             v.ID,
		CompanyID:      v.CompanyID,
		Title:          v.Title,
		Department:     v.Department,
		EmploymentType: v.EmploymentType,
		Location:       v.Location,
		Description:    v.Description,
		Status:         v.Status,
		OpeningsCount:  v.OpeningsCount,
		PostedDate:     v.PostedDate,
		CreatedAt:      v.CreatedAt,
	}
}

func ToJobVacancyResponseList(vacancies []domain.JobVacancy) []JobVacancyResponseDTO {
	result := make([]JobVacancyResponseDTO, len(vacancies))
	for i, v := range vacancies {
		result[i] = *ToJobVacancyResponse(&v)
	}
	return result
}

func ToCandidateResponse(c *domain.Candidate) *CandidateResponseDTO {
	if c == nil {
		return nil
	}
	return &CandidateResponseDTO{
		ID:           c.ID,
		CompanyID:    c.CompanyID,
		JobVacancyID: c.JobVacancyID,
		Name:         c.Name,
		Email:        c.Email,
		Phone:        c.Phone,
		ResumeNote:   c.ResumeNote,
		Source:       c.Source,
		AppliedDate:  c.AppliedDate,
		Stage:        c.Stage,
		StageNotes:   c.StageNotes,
		CreatedAt:    c.CreatedAt,
	}
}

func ToCandidateResponseList(candidates []domain.Candidate) []CandidateResponseDTO {
	result := make([]CandidateResponseDTO, len(candidates))
	for i, c := range candidates {
		result[i] = *ToCandidateResponse(&c)
	}
	return result
}

func ToInterviewResponse(i *domain.Interview) *InterviewResponseDTO {
	if i == nil {
		return nil
	}
	return &InterviewResponseDTO{
		ID:              i.ID,
		CompanyID:       i.CompanyID,
		CandidateID:     i.CandidateID,
		ScheduledAt:     i.ScheduledAt,
		InterviewerName: i.InterviewerName,
		Status:          i.Status,
		FeedbackNote:    i.FeedbackNote,
		Rating:          i.Rating,
		CreatedAt:       i.CreatedAt,
	}
}

func ToInterviewResponseList(interviews []domain.Interview) []InterviewResponseDTO {
	result := make([]InterviewResponseDTO, len(interviews))
	for i, v := range interviews {
		result[i] = *ToInterviewResponse(&v)
	}
	return result
}
