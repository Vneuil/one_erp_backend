package application

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/crm/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type CRMUseCase interface {
	Create(ctx context.Context, dto CreateLeadDTO) (*LeadResponseDTO, error)
	GetByID(ctx context.Context, id uuid.UUID) (*LeadResponseDTO, error)
	List(ctx context.Context, query types.PaginationQuery) ([]LeadResponseDTO, types.PaginationMeta, error)
	Update(ctx context.Context, id uuid.UUID, dto UpdateLeadDTO) (*LeadResponseDTO, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*LeadResponseDTO, error)
	Delete(ctx context.Context, id uuid.UUID) error
	SeedInitialData(ctx context.Context) error

	CreateDeal(ctx context.Context, dto CreateDealDTO) (*DealResponseDTO, error)
	GetDealByID(ctx context.Context, id uuid.UUID) (*DealResponseDTO, error)
	ListDeals(ctx context.Context, query types.PaginationQuery) ([]DealResponseDTO, types.PaginationMeta, error)
	UpdateDeal(ctx context.Context, id uuid.UUID, dto UpdateDealDTO) (*DealResponseDTO, error)
	UpdateDealStage(ctx context.Context, id uuid.UUID, dto UpdateDealStageDTO) (*DealResponseDTO, error)
	DeleteDeal(ctx context.Context, id uuid.UUID) error
}

type crmUseCase struct {
	repo   domain.CRMRepository
	stages StageProvider
}

// StageInfo is a pipeline stage as far as deal handling is concerned.
type StageInfo struct {
	Key         string
	Probability int
	Kind        string // open | won | lost
}

// StageProvider supplies the company's configured pipeline stages. Without one
// (or when it returns none) the built-in five stages apply.
type StageProvider interface {
	ActiveStages(ctx context.Context) ([]StageInfo, error)
}

type Option func(*crmUseCase)

// WithStages makes deal stages configurable per company.
func WithStages(p StageProvider) Option { return func(uc *crmUseCase) { uc.stages = p } }

func NewCRMUseCase(repo domain.CRMRepository, opts ...Option) CRMUseCase {
	uc := &crmUseCase{repo: repo}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

var defaultStages = []StageInfo{
	{"discovery", 30, "open"}, {"quotation", 60, "open"}, {"negotiation", 80, "open"}, {"won", 100, "won"}, {"lost", 0, "lost"},
}

// stageInfo resolves a stage key against the configured pipeline, falling back
// to the built-in stages when none is configured.
func (uc *crmUseCase) stageInfo(ctx context.Context, key string) (StageInfo, bool, error) {
	stages := defaultStages
	if uc.stages != nil {
		configured, err := uc.stages.ActiveStages(ctx)
		if err != nil {
			return StageInfo{}, false, err
		}
		if len(configured) > 0 {
			stages = configured
		}
	}
	for _, s := range stages {
		if s.Key == key {
			return s, true, nil
		}
	}
	return StageInfo{}, false, nil
}

// defaultPIC is who a new lead or deal is assigned to when none is named: the
// caller (by login email), or nobody when there is no caller.
func defaultPIC(ctx context.Context) string { return actor.EmailFrom(ctx) }

func (uc *crmUseCase) Create(ctx context.Context, dto CreateLeadDTO) (*LeadResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	company := strings.TrimSpace(dto.Company)

	if name == "" || company == "" {
		return nil, apperrors.NewBadRequest("Lead Name and Company are required")
	}

	status := dto.Status
	if status == "" {
		status = "New"
	}
	segment := dto.Segment
	if segment == "" {
		segment = "Enterprise B2B"
	}
	source := dto.Source
	if source == "" {
		source = "whatsapp"
	}
	pic := dto.PIC
	if pic == "" {
		pic = defaultPIC(ctx)
	}

	lead := &domain.Lead{
		Name:           name,
		Company:        company,
		Email:          dto.Email,
		Phone:          dto.Phone,
		Segment:        segment,
		Source:         source,
		EstimatedValue: dto.EstimatedValue,
		Status:         status,
		PIC:            pic,
	}

	if err := uc.repo.Create(ctx, lead); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create lead")
	}

	return ToLeadResponse(lead), nil
}

func (uc *crmUseCase) GetByID(ctx context.Context, id uuid.UUID) (*LeadResponseDTO, error) {
	lead, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve lead")
	}
	if lead == nil {
		return nil, apperrors.NewNotFound("Lead not found")
	}
	return ToLeadResponse(lead), nil
}

func (uc *crmUseCase) List(ctx context.Context, query types.PaginationQuery) ([]LeadResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	leads, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list leads")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToLeadResponseList(leads), meta, nil
}

func (uc *crmUseCase) Update(ctx context.Context, id uuid.UUID, dto UpdateLeadDTO) (*LeadResponseDTO, error) {
	lead, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get lead")
	}
	if lead == nil {
		return nil, apperrors.NewNotFound("Lead not found")
	}

	if dto.Name != nil {
		lead.Name = *dto.Name
	}
	if dto.Company != nil {
		lead.Company = *dto.Company
	}
	if dto.Email != nil {
		lead.Email = *dto.Email
	}
	if dto.Phone != nil {
		lead.Phone = *dto.Phone
	}
	if dto.Segment != nil {
		lead.Segment = *dto.Segment
	}
	if dto.Source != nil {
		lead.Source = *dto.Source
	}
	if dto.EstimatedValue != nil {
		lead.EstimatedValue = *dto.EstimatedValue
	}
	if dto.Status != nil {
		lead.Status = *dto.Status
	}
	if dto.PIC != nil {
		lead.PIC = *dto.PIC
	}

	if err := uc.repo.Update(ctx, lead); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update lead")
	}

	return ToLeadResponse(lead), nil
}

func (uc *crmUseCase) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (*LeadResponseDTO, error) {
	lead, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get lead")
	}
	if lead == nil {
		return nil, apperrors.NewNotFound("Lead not found")
	}

	lead.Status = status
	if err := uc.repo.Update(ctx, lead); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update lead status")
	}

	return ToLeadResponse(lead), nil
}

func (uc *crmUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	return uc.repo.Delete(ctx, id)
}

func (uc *crmUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		if err := uc.seedLeads(ctx); err != nil {
			return err
		}
	}

	dealCount, err := uc.repo.CountDeals(ctx)
	if err != nil {
		return err
	}
	if dealCount == 0 {
		if err := uc.seedDeals(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (uc *crmUseCase) seedLeads(ctx context.Context) error {
	initial := []CreateLeadDTO{
		{
			Name:           "Bambang Sudiro",
			Company:        "PT Graha Sentosa Teknik",
			Email:          "bambang@grahasentosa.co.id",
			Phone:          "+62 812-3456-7890",
			Segment:        "Enterprise B2B",
			Source:         "whatsapp",
			EstimatedValue: 450000000,
			Status:         "Proposal",
			PIC:            "Nicholas Tantra",
		},
		{
			Name:           "Rina Suryani",
			Company:        "CV Indo Makmur Abadi",
			Email:          "rina@indomakmur.com",
			Phone:          "+62 813-9876-5432",
			Segment:        "SME Fast Growth",
			Source:         "web",
			EstimatedValue: 120000000,
			Status:         "Contacted",
			PIC:            "Rizky Ramadhan",
		},
		{
			Name:           "Ir. Hendra Gunawan",
			Company:        "PT Mega Baja Nusantara",
			Email:          "hendra@megabaja.co.id",
			Phone:          "+62 811-2233-4455",
			Segment:        "Enterprise B2B",
			Source:         "referral",
			EstimatedValue: 850000000,
			Status:         "Qualified",
			PIC:            "Nicholas Tantra",
		},
	}

	for _, item := range initial {
		_, _ = uc.Create(ctx, item)
	}
	return nil
}

func (uc *crmUseCase) seedDeals(ctx context.Context) error {
	initialDeals := []CreateDealDTO{
		{Title: "Pengadaan 500 Unit Rak Gudang Heavy Duty", Customer: "PT Graha Konstruksi Nusantara", Value: 450000000, Stage: "negotiation", PIC: "Nicholas Tantra", ExpectedClosing: "2026-09-15"},
		{Title: "Kontrak Distribusi Bahan Bangunan Tahunan", Customer: "CV Surya Makmur Logistik", Value: 185000000, Stage: "quotation", PIC: "Nicholas Tantra", ExpectedClosing: "2026-09-22"},
		{Title: "Upgrade Sistem POS Retail 12 Cabang", Customer: "PT Global Retailindo Jaya", Value: 85000000, Stage: "discovery", PIC: "Budi Santoso", ExpectedClosing: "2026-10-05"},
	}
	for _, item := range initialDeals {
		_, _ = uc.CreateDeal(ctx, item)
	}
	return nil
}

func (uc *crmUseCase) CreateDeal(ctx context.Context, dto CreateDealDTO) (*DealResponseDTO, error) {
	title := strings.TrimSpace(dto.Title)
	customer := strings.TrimSpace(dto.Customer)
	if title == "" || customer == "" {
		return nil, apperrors.NewBadRequest("Deal title and customer are required")
	}
	stageKey := dto.Stage
	if stageKey == "" {
		stageKey = "discovery"
		if uc.stages != nil {
			if configured, err := uc.stages.ActiveStages(ctx); err == nil && len(configured) > 0 {
				stageKey = configured[0].Key
			}
		}
	}
	stage, ok, err := uc.stageInfo(ctx, stageKey)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load pipeline stages")
	}
	if !ok {
		return nil, apperrors.NewBadRequest("Invalid deal stage")
	}
	pic := dto.PIC
	if pic == "" {
		pic = defaultPIC(ctx)
	}

	deal := &domain.Deal{
		Title:           title,
		Customer:        customer,
		Value:           dto.Value,
		Probability:     stage.Probability,
		Stage:           stage.Key,
		PIC:             pic,
		ExpectedClosing: dto.ExpectedClosing,
		LeadID:          dto.LeadID,
	}
	if stage.Kind != "open" {
		now := time.Now()
		deal.ClosedAt = &now
	}
	if err := uc.repo.CreateDeal(ctx, deal); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create deal")
	}
	return ToDealResponse(deal), nil
}

func (uc *crmUseCase) GetDealByID(ctx context.Context, id uuid.UUID) (*DealResponseDTO, error) {
	deal, err := uc.repo.GetDealByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve deal")
	}
	if deal == nil {
		return nil, apperrors.NewNotFound("Deal not found")
	}
	return ToDealResponse(deal), nil
}

func (uc *crmUseCase) ListDeals(ctx context.Context, query types.PaginationQuery) ([]DealResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	deals, total, err := uc.repo.ListDeals(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list deals")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToDealResponseList(deals), meta, nil
}

func (uc *crmUseCase) UpdateDeal(ctx context.Context, id uuid.UUID, dto UpdateDealDTO) (*DealResponseDTO, error) {
	deal, err := uc.repo.GetDealByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get deal")
	}
	if deal == nil {
		return nil, apperrors.NewNotFound("Deal not found")
	}

	if dto.Title != nil {
		deal.Title = *dto.Title
	}
	if dto.Customer != nil {
		deal.Customer = *dto.Customer
	}
	if dto.Value != nil {
		deal.Value = *dto.Value
	}
	if dto.PIC != nil {
		deal.PIC = *dto.PIC
	}
	if dto.ExpectedClosing != nil {
		deal.ExpectedClosing = *dto.ExpectedClosing
	}

	if err := uc.repo.UpdateDeal(ctx, deal); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update deal")
	}
	return ToDealResponse(deal), nil
}

func (uc *crmUseCase) UpdateDealStage(ctx context.Context, id uuid.UUID, dto UpdateDealStageDTO) (*DealResponseDTO, error) {
	stage, ok, err := uc.stageInfo(ctx, dto.Stage)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load pipeline stages")
	}
	if !ok {
		return nil, apperrors.NewBadRequest("Invalid deal stage")
	}
	reason := strings.TrimSpace(dto.LostReason)
	// A lost deal without a reason is useless for win/loss analysis.
	if stage.Kind == "lost" && reason == "" {
		return nil, apperrors.NewBadRequest("A reason is required when a deal is lost")
	}
	deal, err := uc.repo.GetDealByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get deal")
	}
	if deal == nil {
		return nil, apperrors.NewNotFound("Deal not found")
	}

	deal.Stage = stage.Key
	deal.Probability = stage.Probability
	if stage.Kind == "lost" {
		deal.LostReason = reason
	} else {
		deal.LostReason = ""
	}
	// ClosedAt marks when the deal was settled; reopening a deal clears it.
	switch {
	case stage.Kind == "open":
		deal.ClosedAt = nil
	case deal.ClosedAt == nil:
		now := time.Now()
		deal.ClosedAt = &now
	}

	if err := uc.repo.UpdateDeal(ctx, deal); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update deal stage")
	}
	return ToDealResponse(deal), nil
}

func (uc *crmUseCase) DeleteDeal(ctx context.Context, id uuid.UUID) error {
	return uc.repo.DeleteDeal(ctx, id)
}
