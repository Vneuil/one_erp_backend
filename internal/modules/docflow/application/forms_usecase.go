package application

import (
	"context"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/docflow/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type FormsUseCase interface {
	CreateForm(ctx context.Context, dto CreateFormDTO) (*FormResponseDTO, error)
	GetFormByID(ctx context.Context, id uuid.UUID) (*FormResponseDTO, error)
	ListForms(ctx context.Context, query types.PaginationQuery) ([]FormResponseDTO, types.PaginationMeta, error)
	PublishForm(ctx context.Context, id uuid.UUID) (*FormResponseDTO, error)
	SubmitResponse(ctx context.Context, formID uuid.UUID, dto SubmitResponseDTO) (*FormSubmissionResponseDTO, error)
	ListResponses(ctx context.Context, formID uuid.UUID) ([]FormSubmissionResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type formsUseCase struct {
	repo domain.FormsRepository
}

func NewFormsUseCase(repo domain.FormsRepository) FormsUseCase {
	return &formsUseCase{repo: repo}
}

func (uc *formsUseCase) CreateForm(ctx context.Context, dto CreateFormDTO) (*FormResponseDTO, error) {
	if dto.Title == "" {
		return nil, apperrors.NewBadRequest("Form title is required")
	}
	if dto.CreatedBy == "" {
		return nil, apperrors.NewBadRequest("Created by is required")
	}
	if dto.Fields == "" {
		dto.Fields = "[]"
	}

	f := &domain.Form{
		CompanyID:   dto.CompanyID,
		Title:       dto.Title,
		Description: dto.Description,
		Fields:      dto.Fields,
		CreatedBy:   dto.CreatedBy,
		Status:      "draft",
	}
	if err := uc.repo.CreateForm(ctx, f); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create form")
	}
	return ToFormResponse(f), nil
}

func (uc *formsUseCase) GetFormByID(ctx context.Context, id uuid.UUID) (*FormResponseDTO, error) {
	f, err := uc.repo.GetFormByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get form")
	}
	if f == nil {
		return nil, apperrors.NewNotFound("Form not found")
	}
	return ToFormResponse(f), nil
}

func (uc *formsUseCase) ListForms(ctx context.Context, query types.PaginationQuery) ([]FormResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	forms, total, err := uc.repo.ListForms(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list forms")
	}
	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToFormResponseList(forms), meta, nil
}

func (uc *formsUseCase) PublishForm(ctx context.Context, id uuid.UUID) (*FormResponseDTO, error) {
	f, err := uc.repo.GetFormByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get form")
	}
	if f == nil {
		return nil, apperrors.NewNotFound("Form not found")
	}
	if f.Status == "published" {
		return nil, apperrors.NewBadRequest("Form is already published")
	}
	f.Status = "published"
	if err := uc.repo.UpdateForm(ctx, f); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to publish form")
	}
	return ToFormResponse(f), nil
}

func (uc *formsUseCase) SubmitResponse(ctx context.Context, formID uuid.UUID, dto SubmitResponseDTO) (*FormSubmissionResponseDTO, error) {
	f, err := uc.repo.GetFormByID(ctx, formID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get form")
	}
	if f == nil {
		return nil, apperrors.NewNotFound("Form not found")
	}
	if dto.SubmittedBy == "" {
		return nil, apperrors.NewBadRequest("Submitted by is required")
	}
	if dto.Answers == "" {
		return nil, apperrors.NewBadRequest("Answers are required")
	}

	r := &domain.FormResponse{
		FormID:      f.ID,
		SubmittedBy: dto.SubmittedBy,
		Answers:     dto.Answers,
		SubmittedAt: time.Now().Format(time.RFC3339),
	}
	if err := uc.repo.CreateResponse(ctx, r); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to submit response")
	}

	f.ResponseCount++
	if err := uc.repo.UpdateForm(ctx, f); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to update form response count")
	}
	return ToFormSubmissionResponse(r), nil
}

func (uc *formsUseCase) ListResponses(ctx context.Context, formID uuid.UUID) ([]FormSubmissionResponseDTO, error) {
	f, err := uc.repo.GetFormByID(ctx, formID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get form")
	}
	if f == nil {
		return nil, apperrors.NewNotFound("Form not found")
	}
	items, err := uc.repo.ListResponsesByForm(ctx, formID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list responses")
	}
	return ToFormSubmissionResponseList(items), nil
}

// SeedInitialData populates a few sample forms with responses on first boot
func (uc *formsUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountForms(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	seeds := []struct {
		dto       CreateFormDTO
		publish   bool
		responses []SubmitResponseDTO
	}{
		{
			dto: CreateFormDTO{
				Title:       "Formulir Pengajuan Lembur & SPK Lapangan",
				Description: "Form pengajuan lembur karyawan dan surat perintah kerja lapangan",
				Fields:      `[{"label":"Nama Karyawan","type":"text","required":true},{"label":"Tanggal Lembur","type":"date","required":true},{"label":"Alasan Lembur","type":"textarea","required":true}]`,
				CreatedBy:   "Nicholas Tantra",
			},
			publish: true,
			responses: []SubmitResponseDTO{
				{SubmittedBy: "Budi Santoso", Answers: `{"Nama Karyawan":"Budi Santoso","Tanggal Lembur":"2026-08-28","Alasan Lembur":"Penyelesaian laporan bulanan"}`},
				{SubmittedBy: "Dewi Lestari", Answers: `{"Nama Karyawan":"Dewi Lestari","Tanggal Lembur":"2026-08-29","Alasan Lembur":"Persiapan audit"}`},
			},
		},
		{
			dto: CreateFormDTO{
				Title:       "Survei Kepuasan Klien Pengiriman WMS",
				Description: "Survei kepuasan klien terhadap layanan pengiriman warehouse",
				Fields:      `[{"label":"Nama Klien","type":"text","required":true},{"label":"Skor Kepuasan","type":"number","required":true},{"label":"Komentar","type":"textarea","required":false}]`,
				CreatedBy:   "Dewi Lestari",
			},
			publish: true,
			responses: []SubmitResponseDTO{
				{SubmittedBy: "PT Graha Konstruksi", Answers: `{"Nama Klien":"PT Graha Konstruksi","Skor Kepuasan":"9","Komentar":"Pengiriman tepat waktu"}`},
			},
		},
		{
			dto: CreateFormDTO{
				Title:       "Inspeksi K3 & Safety Check Mesin Pabrik",
				Description: "Checklist inspeksi keselamatan kerja mesin produksi",
				Fields:      `[{"label":"Nama Inspektor","type":"text","required":true},{"label":"Kondisi Mesin","type":"text","required":true}]`,
				CreatedBy:   "Rizky Ramadhan",
			},
			publish: false,
		},
	}

	for _, seed := range seeds {
		res, err := uc.CreateForm(ctx, seed.dto)
		if err != nil {
			continue
		}
		if seed.publish {
			if _, err := uc.PublishForm(ctx, res.ID); err != nil {
				continue
			}
		}
		for _, r := range seed.responses {
			if _, err := uc.SubmitResponse(ctx, res.ID, r); err != nil {
				continue
			}
		}
	}
	return nil
}
