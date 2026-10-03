package application

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/recruitment/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

var validVacancyStatuses = map[string]bool{
	"open":    true,
	"closed":  true,
	"on_hold": true,
}

var validEmploymentTypes = map[string]bool{
	"Full-time":  true,
	"Contract":   true,
	"Internship": true,
}

var validInterviewStatuses = map[string]bool{
	"scheduled": true,
	"completed": true,
	"cancelled": true,
}

// stageOrder defines the forward progression of the recruitment pipeline
var stageOrder = map[string]int{
	"applied":   0,
	"screening": 1,
	"interview": 2,
	"offer":     3,
	"hired":     4,
	"rejected":  5,
}

type RecruitmentUseCase interface {
	CreateVacancy(ctx context.Context, dto CreateJobVacancyDTO) (*JobVacancyResponseDTO, error)
	GetVacancyByID(ctx context.Context, id uuid.UUID) (*JobVacancyResponseDTO, error)
	ListVacancies(ctx context.Context, query types.PaginationQuery) ([]JobVacancyResponseDTO, types.PaginationMeta, error)
	UpdateVacancy(ctx context.Context, id uuid.UUID, dto UpdateJobVacancyDTO) (*JobVacancyResponseDTO, error)

	CreateCandidate(ctx context.Context, dto CreateCandidateDTO) (*CandidateResponseDTO, error)
	GetCandidateByID(ctx context.Context, id uuid.UUID) (*CandidateResponseDTO, error)
	ListCandidates(ctx context.Context, query types.PaginationQuery) ([]CandidateResponseDTO, types.PaginationMeta, error)
	AdvanceStage(ctx context.Context, id uuid.UUID, dto AdvanceStageDTO) (*CandidateResponseDTO, error)
	HireCandidate(ctx context.Context, id uuid.UUID) (*CandidateResponseDTO, error)

	CreateInterview(ctx context.Context, dto CreateInterviewDTO) (*InterviewResponseDTO, error)
	GetInterviewByID(ctx context.Context, id uuid.UUID) (*InterviewResponseDTO, error)
	ListInterviews(ctx context.Context, query types.PaginationQuery) ([]InterviewResponseDTO, types.PaginationMeta, error)
	UpdateInterview(ctx context.Context, id uuid.UUID, dto UpdateInterviewDTO) (*InterviewResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type recruitmentUseCase struct {
	repo    domain.RecruitmentRepository
	hrmRepo hrmdomain.HRMRepository
}

// NewRecruitmentUseCase wires the recruitment repository together with the
// HRM repository, so hiring a candidate automatically creates a matching
// Employee record (see hire()) instead of leaving recruitment and HRM
// disconnected. hrmRepo may be nil, in which case hiring skips Employee
// creation entirely (recruitment-only deployments).
func NewRecruitmentUseCase(repo domain.RecruitmentRepository, hrmRepo hrmdomain.HRMRepository) RecruitmentUseCase {
	return &recruitmentUseCase{repo: repo, hrmRepo: hrmRepo}
}

// employmentTypeToContractType maps a job vacancy's employment type to the
// HRM contract type vocabulary used on Employee.ContractType.
func employmentTypeToContractType(employmentType string) string {
	switch employmentType {
	case "Contract":
		return "PKWT Kontrak"
	case "Internship":
		return "Magang"
	default:
		return "PKWTT Tetap"
	}
}

func (uc *recruitmentUseCase) CreateVacancy(ctx context.Context, dto CreateJobVacancyDTO) (*JobVacancyResponseDTO, error) {
	if dto.Title == "" {
		return nil, apperrors.NewBadRequest("Title is required")
	}
	if dto.Department == "" {
		return nil, apperrors.NewBadRequest("Department is required")
	}
	if dto.EmploymentType == "" {
		dto.EmploymentType = "Full-time"
	}
	if !validEmploymentTypes[dto.EmploymentType] {
		return nil, apperrors.NewBadRequest("Employment type must be one of Full-time, Contract, Internship")
	}
	if dto.OpeningsCount <= 0 {
		dto.OpeningsCount = 1
	}
	if dto.PostedDate == "" {
		dto.PostedDate = time.Now().Format("2006-01-02")
	}

	vacancy := &domain.JobVacancy{
		CompanyID:      dto.CompanyID,
		Title:          dto.Title,
		Department:     dto.Department,
		EmploymentType: dto.EmploymentType,
		Location:       dto.Location,
		Description:    dto.Description,
		Status:         "open",
		OpeningsCount:  dto.OpeningsCount,
		PostedDate:     dto.PostedDate,
	}

	if err := uc.repo.CreateVacancy(ctx, vacancy); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create job vacancy")
	}
	return ToJobVacancyResponse(vacancy), nil
}

func (uc *recruitmentUseCase) GetVacancyByID(ctx context.Context, id uuid.UUID) (*JobVacancyResponseDTO, error) {
	vacancy, err := uc.repo.GetVacancyByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get job vacancy")
	}
	if vacancy == nil {
		return nil, apperrors.NewNotFound("Job vacancy not found")
	}
	return ToJobVacancyResponse(vacancy), nil
}

func (uc *recruitmentUseCase) ListVacancies(ctx context.Context, query types.PaginationQuery) ([]JobVacancyResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	vacancies, total, err := uc.repo.ListVacancies(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list job vacancies")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToJobVacancyResponseList(vacancies), meta, nil
}

func (uc *recruitmentUseCase) UpdateVacancy(ctx context.Context, id uuid.UUID, dto UpdateJobVacancyDTO) (*JobVacancyResponseDTO, error) {
	vacancy, err := uc.repo.GetVacancyByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get job vacancy")
	}
	if vacancy == nil {
		return nil, apperrors.NewNotFound("Job vacancy not found")
	}

	if dto.Title != "" {
		vacancy.Title = dto.Title
	}
	if dto.Department != "" {
		vacancy.Department = dto.Department
	}
	if dto.EmploymentType != "" {
		if !validEmploymentTypes[dto.EmploymentType] {
			return nil, apperrors.NewBadRequest("Employment type must be one of Full-time, Contract, Internship")
		}
		vacancy.EmploymentType = dto.EmploymentType
	}
	if dto.Location != "" {
		vacancy.Location = dto.Location
	}
	if dto.Description != "" {
		vacancy.Description = dto.Description
	}
	if dto.Status != "" {
		if !validVacancyStatuses[dto.Status] {
			return nil, apperrors.NewBadRequest("Status must be one of open, closed, on_hold")
		}
		vacancy.Status = dto.Status
	}
	if dto.PostedDate != "" {
		vacancy.PostedDate = dto.PostedDate
	}
	if dto.OpeningsCount > 0 {
		vacancy.OpeningsCount = dto.OpeningsCount
	}

	if err := uc.repo.UpdateVacancy(ctx, vacancy); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update job vacancy")
	}
	return ToJobVacancyResponse(vacancy), nil
}

func (uc *recruitmentUseCase) CreateCandidate(ctx context.Context, dto CreateCandidateDTO) (*CandidateResponseDTO, error) {
	if dto.Name == "" {
		return nil, apperrors.NewBadRequest("Name is required")
	}
	if dto.JobVacancyID == uuid.Nil {
		return nil, apperrors.NewBadRequest("Job vacancy is required")
	}

	vacancy, err := uc.repo.GetVacancyByID(ctx, dto.JobVacancyID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get job vacancy")
	}
	if vacancy == nil {
		return nil, apperrors.NewNotFound("Job vacancy not found")
	}

	candidate := &domain.Candidate{
		CompanyID:    dto.CompanyID,
		JobVacancyID: dto.JobVacancyID,
		Name:         dto.Name,
		Email:        dto.Email,
		Phone:        dto.Phone,
		ResumeNote:   dto.ResumeNote,
		Source:       dto.Source,
		AppliedDate:  dto.AppliedDate,
		Stage:        "applied",
	}

	if err := uc.repo.CreateCandidate(ctx, candidate); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create candidate")
	}
	return ToCandidateResponse(candidate), nil
}

func (uc *recruitmentUseCase) GetCandidateByID(ctx context.Context, id uuid.UUID) (*CandidateResponseDTO, error) {
	candidate, err := uc.repo.GetCandidateByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get candidate")
	}
	if candidate == nil {
		return nil, apperrors.NewNotFound("Candidate not found")
	}
	return ToCandidateResponse(candidate), nil
}

func (uc *recruitmentUseCase) ListCandidates(ctx context.Context, query types.PaginationQuery) ([]CandidateResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	candidates, total, err := uc.repo.ListCandidates(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list candidates")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToCandidateResponseList(candidates), meta, nil
}

func (uc *recruitmentUseCase) AdvanceStage(ctx context.Context, id uuid.UUID, dto AdvanceStageDTO) (*CandidateResponseDTO, error) {
	if _, ok := stageOrder[dto.Stage]; !ok {
		return nil, apperrors.NewBadRequest("Stage must be one of applied, screening, interview, offer, hired, rejected")
	}

	candidate, err := uc.repo.GetCandidateByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get candidate")
	}
	if candidate == nil {
		return nil, apperrors.NewNotFound("Candidate not found")
	}

	if dto.Stage != "rejected" && stageOrder[dto.Stage] < stageOrder[candidate.Stage] {
		return nil, apperrors.NewBadRequest("Cannot move candidate to an earlier pipeline stage")
	}

	if dto.Stage == "hired" {
		return uc.hire(ctx, candidate, dto.Notes)
	}

	candidate.Stage = dto.Stage
	if dto.Notes != "" {
		candidate.StageNotes = dto.Notes
	}
	if err := uc.repo.UpdateCandidate(ctx, candidate); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update candidate")
	}
	return ToCandidateResponse(candidate), nil
}

func (uc *recruitmentUseCase) hire(ctx context.Context, candidate *domain.Candidate, notes string) (*CandidateResponseDTO, error) {
	vacancy, err := uc.repo.GetVacancyByID(ctx, candidate.JobVacancyID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get job vacancy")
	}
	if vacancy == nil {
		return nil, apperrors.NewNotFound("Job vacancy not found")
	}
	if vacancy.OpeningsCount <= 0 || vacancy.Status == "closed" {
		return nil, apperrors.NewBadRequest("No openings remaining on this job vacancy")
	}

	candidate.Stage = "hired"
	if notes != "" {
		candidate.StageNotes = notes
	}
	if err := uc.repo.UpdateCandidate(ctx, candidate); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update candidate")
	}

	if vacancy.OpeningsCount > 0 {
		vacancy.OpeningsCount--
	}
	if vacancy.OpeningsCount == 0 {
		vacancy.Status = "closed"
	}
	if err := uc.repo.UpdateVacancy(ctx, vacancy); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update job vacancy")
	}

	// Hiring a candidate creates a matching Employee record in HRM (audit
	// gap #4). This is best-effort: a failure here is logged but must not
	// block the hire itself, since some HRM-required fields (NIP, salary)
	// have no equivalent on the candidate/vacancy and are defaulted below.
	if uc.hrmRepo != nil {
		if err := uc.createEmployeeFromHire(ctx, candidate, vacancy); err != nil {
			slog.Error("Failed to create employee record from hired candidate",
				"candidateId", candidate.ID, "error", err)
		}
	}

	return ToCandidateResponse(candidate), nil
}

// createEmployeeFromHire pre-fills a new Employee record from the hired
// candidate's data and the job vacancy's department/title. NIP and
// BaseSalary have no source on Candidate/JobVacancy, so they are defaulted
// (a sequential "EMP-NNN" NIP, and a zero BaseSalary to be filled in by HR).
func (uc *recruitmentUseCase) createEmployeeFromHire(ctx context.Context, candidate *domain.Candidate, vacancy *domain.JobVacancy) error {
	count, err := uc.hrmRepo.CountEmployees(ctx)
	if err != nil {
		return err
	}

	joinDate := time.Now().Format("2006-01-02")
	emp := &hrmdomain.Employee{
		NIP:            fmt.Sprintf("EMP-%03d", count+1),
		Name:           candidate.Name,
		Email:          candidate.Email,
		Phone:          candidate.Phone,
		Department:     vacancy.Department,
		Role:           vacancy.Title,
		ContractType:   employmentTypeToContractType(vacancy.EmploymentType),
		JoinDate:       joinDate,
		Status:         "Active",
		BaseSalary:     0, // not available from candidate/vacancy data - HR fills this in
		LeaveQuotaDays: 12,
	}

	return uc.hrmRepo.CreateEmployee(ctx, emp)
}

func (uc *recruitmentUseCase) HireCandidate(ctx context.Context, id uuid.UUID) (*CandidateResponseDTO, error) {
	candidate, err := uc.repo.GetCandidateByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get candidate")
	}
	if candidate == nil {
		return nil, apperrors.NewNotFound("Candidate not found")
	}
	return uc.hire(ctx, candidate, "")
}

func (uc *recruitmentUseCase) CreateInterview(ctx context.Context, dto CreateInterviewDTO) (*InterviewResponseDTO, error) {
	if dto.CandidateID == uuid.Nil {
		return nil, apperrors.NewBadRequest("Candidate is required")
	}
	if dto.ScheduledAt == "" {
		return nil, apperrors.NewBadRequest("Scheduled date/time is required")
	}
	if dto.InterviewerName == "" {
		return nil, apperrors.NewBadRequest("Interviewer name is required")
	}

	candidate, err := uc.repo.GetCandidateByID(ctx, dto.CandidateID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get candidate")
	}
	if candidate == nil {
		return nil, apperrors.NewNotFound("Candidate not found")
	}

	interview := &domain.Interview{
		CompanyID:       dto.CompanyID,
		CandidateID:     dto.CandidateID,
		ScheduledAt:     dto.ScheduledAt,
		InterviewerName: dto.InterviewerName,
		Status:          "scheduled",
	}

	if err := uc.repo.CreateInterview(ctx, interview); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create interview")
	}
	return ToInterviewResponse(interview), nil
}

func (uc *recruitmentUseCase) GetInterviewByID(ctx context.Context, id uuid.UUID) (*InterviewResponseDTO, error) {
	interview, err := uc.repo.GetInterviewByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get interview")
	}
	if interview == nil {
		return nil, apperrors.NewNotFound("Interview not found")
	}
	return ToInterviewResponse(interview), nil
}

func (uc *recruitmentUseCase) ListInterviews(ctx context.Context, query types.PaginationQuery) ([]InterviewResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	interviews, total, err := uc.repo.ListInterviews(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list interviews")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToInterviewResponseList(interviews), meta, nil
}

func (uc *recruitmentUseCase) UpdateInterview(ctx context.Context, id uuid.UUID, dto UpdateInterviewDTO) (*InterviewResponseDTO, error) {
	interview, err := uc.repo.GetInterviewByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get interview")
	}
	if interview == nil {
		return nil, apperrors.NewNotFound("Interview not found")
	}

	if dto.Status != "" {
		if !validInterviewStatuses[dto.Status] {
			return nil, apperrors.NewBadRequest("Status must be one of scheduled, completed, cancelled")
		}
		interview.Status = dto.Status
	}
	if dto.FeedbackNote != "" {
		interview.FeedbackNote = dto.FeedbackNote
	}
	if dto.Rating != nil {
		if *dto.Rating < 1 || *dto.Rating > 5 {
			return nil, apperrors.NewBadRequest("Rating must be between 1 and 5")
		}
		interview.Rating = dto.Rating
	}

	if err := uc.repo.UpdateInterview(ctx, interview); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update interview")
	}
	return ToInterviewResponse(interview), nil
}

// SeedInitialData populates a few sample job vacancies and candidates on first boot
func (uc *recruitmentUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountVacancies(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	vacancies := []CreateJobVacancyDTO{
		{Title: "Senior Production Engineer", Department: "Manufacturing", EmploymentType: "Full-time", Location: "Cikarang", OpeningsCount: 1, PostedDate: "2026-08-20"},
		{Title: "Key Account Executive B2B", Department: "Sales", EmploymentType: "Full-time", Location: "Jakarta", OpeningsCount: 2, PostedDate: "2026-08-18"},
		{Title: "Warehouse Forklift Operator", Department: "Warehouse & Logistics", EmploymentType: "Contract", Location: "Bekasi", OpeningsCount: 3, PostedDate: "2026-08-05"},
	}

	var created []*JobVacancyResponseDTO
	for _, v := range vacancies {
		r, err := uc.CreateVacancy(ctx, v)
		if err != nil {
			continue
		}
		created = append(created, r)
	}
	if len(created) < 3 {
		return nil
	}

	candidates := []struct {
		dto   CreateCandidateDTO
		stage string
	}{
		{dto: CreateCandidateDTO{JobVacancyID: created[0].ID, Name: "Fajar Nugraha, S.T.", Email: "fajar.nugraha@example.com", Source: "LinkedIn", AppliedDate: "2026-08-25"}, stage: "interview"},
		{dto: CreateCandidateDTO{JobVacancyID: created[1].ID, Name: "Maya Anggraini", Email: "maya.anggraini@example.com", Source: "Referral", AppliedDate: "2026-08-20"}, stage: "offer"},
		{dto: CreateCandidateDTO{JobVacancyID: created[2].ID, Name: "Dedi Setiawan", Email: "dedi.setiawan@example.com", Source: "Job Portal", AppliedDate: "2026-08-10"}, stage: "hired"},
		{dto: CreateCandidateDTO{JobVacancyID: created[0].ID, Name: "Rina Salsabila, S.Ak.", Email: "rina.salsabila@example.com", Source: "LinkedIn", AppliedDate: "2026-09-01"}, stage: "screening"},
	}

	for _, c := range candidates {
		resp, err := uc.CreateCandidate(ctx, c.dto)
		if err != nil {
			continue
		}
		if c.stage == "applied" {
			continue
		}
		id := resp.ID
		if _, err := uc.AdvanceStage(ctx, id, AdvanceStageDTO{Stage: c.stage}); err != nil {
			continue
		}
	}

	return nil
}
