package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/divinecoid/one-backend/internal/foundation/companyctx"
	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/docflow/domain"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Forms

type formsRepository struct {
	db *gorm.DB
}

func NewFormsRepository(db *gorm.DB) domain.FormsRepository {
	return &formsRepository{db: db}
}

func (r *formsRepository) CreateForm(ctx context.Context, f *domain.Form) error {
	companyctx.SetCompanyID(ctx, &f.CompanyID)
	tenantctx.SetTenantID(ctx, &f.TenantID)
	return r.db.WithContext(ctx).Create(f).Error
}

func (r *formsRepository) GetFormByID(ctx context.Context, id uuid.UUID) (*domain.Form, error) {
	var f domain.Form
	err := companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&f).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *formsRepository) ListForms(ctx context.Context, query types.PaginationQuery) ([]domain.Form, int64, error) {
	var forms []domain.Form
	var total int64

	db := companyctx.Scope(ctx, tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Form{})))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ? OR created_by ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&forms).Error
	return forms, total, err
}

func (r *formsRepository) UpdateForm(ctx context.Context, f *domain.Form) error {
	return r.db.WithContext(ctx).Save(f).Error
}

func (r *formsRepository) CountForms(ctx context.Context) (int64, error) {
	var total int64
	err := companyctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Form{})).Count(&total).Error
	return total, err
}

func (r *formsRepository) CreateResponse(ctx context.Context, resp *domain.FormResponse) error {
	return r.db.WithContext(ctx).Create(resp).Error
}

func (r *formsRepository) ListResponsesByForm(ctx context.Context, formID uuid.UUID) ([]domain.FormResponse, error) {
	var items []domain.FormResponse
	err := r.db.WithContext(ctx).Where("form_id = ?", formID).Order("created_at desc").Find(&items).Error
	return items, err
}

// Signatures

type signaturesRepository struct {
	db *gorm.DB
}

func NewSignaturesRepository(db *gorm.DB) domain.SignaturesRepository {
	return &signaturesRepository{db: db}
}

func (r *signaturesRepository) CreateDocument(ctx context.Context, d *domain.SignatureDocument) error {
	companyctx.SetCompanyID(ctx, &d.CompanyID)
	tenantctx.SetTenantID(ctx, &d.TenantID)
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *signaturesRepository) GetDocumentByID(ctx context.Context, id uuid.UUID) (*domain.SignatureDocument, error) {
	var d domain.SignatureDocument
	err := companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&d).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *signaturesRepository) ListDocuments(ctx context.Context, query types.PaginationQuery) ([]domain.SignatureDocument, int64, error) {
	var docs []domain.SignatureDocument
	var total int64

	db := companyctx.Scope(ctx, tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.SignatureDocument{})))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ? OR file_name ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("created_at desc").Offset(offset).Limit(query.PerPage).Find(&docs).Error
	return docs, total, err
}

func (r *signaturesRepository) UpdateDocument(ctx context.Context, d *domain.SignatureDocument) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *signaturesRepository) CountDocuments(ctx context.Context) (int64, error) {
	var total int64
	err := companyctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.SignatureDocument{})).Count(&total).Error
	return total, err
}

func (r *signaturesRepository) CreateRequest(ctx context.Context, req *domain.SignatureRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *signaturesRepository) GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.SignatureRequest, error) {
	var req domain.SignatureRequest
	// SignatureRequest has no company_id column of its own, so scope it via
	// its parent document's company_id - otherwise any authenticated user on
	// the platform could fetch/sign/decline another company's signature
	// request by guessing its UUID.
	err := companyctx.Scope(ctx, r.db.WithContext(ctx).
		Joins("JOIN docflow_signature_documents ON docflow_signature_documents.id = docflow_signature_requests.document_id")).
		Where("docflow_signature_requests.id = ?", id).
		First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *signaturesRepository) ListRequestsByDocument(ctx context.Context, documentID uuid.UUID) ([]domain.SignatureRequest, error) {
	var items []domain.SignatureRequest
	err := r.db.WithContext(ctx).Where("document_id = ?", documentID).Order("order_index asc").Find(&items).Error
	return items, err
}

func (r *signaturesRepository) UpdateRequest(ctx context.Context, req *domain.SignatureRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

// Meetings

type meetingsRepository struct {
	db *gorm.DB
}

func NewMeetingsRepository(db *gorm.DB) domain.MeetingsRepository {
	return &meetingsRepository{db: db}
}

func (r *meetingsRepository) CreateMeeting(ctx context.Context, m *domain.Meeting) error {
	companyctx.SetCompanyID(ctx, &m.CompanyID)
	tenantctx.SetTenantID(ctx, &m.TenantID)
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *meetingsRepository) GetMeetingByID(ctx context.Context, id uuid.UUID) (*domain.Meeting, error) {
	var m domain.Meeting
	err := companyctx.Scope(ctx, r.db.WithContext(ctx)).Where("id = ?", id).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *meetingsRepository) ListMeetings(ctx context.Context, query types.PaginationQuery) ([]domain.Meeting, int64, error) {
	var items []domain.Meeting
	var total int64

	db := companyctx.Scope(ctx, tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Meeting{})))
	if query.Search != "" {
		pattern := fmt.Sprintf("%%%s%%", query.Search)
		db = db.Where("title ILIKE ? OR organizer_name ILIKE ?", pattern, pattern)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.PerPage
	err := db.Order("scheduled_at desc").Offset(offset).Limit(query.PerPage).Find(&items).Error
	return items, total, err
}

func (r *meetingsRepository) UpdateMeeting(ctx context.Context, m *domain.Meeting) error {
	return r.db.WithContext(ctx).Save(m).Error
}

func (r *meetingsRepository) CountMeetings(ctx context.Context) (int64, error) {
	var total int64
	err := companyctx.Scope(ctx, r.db.WithContext(ctx).Model(&domain.Meeting{})).Count(&total).Error
	return total, err
}

func (r *meetingsRepository) CreateNote(ctx context.Context, n *domain.MeetingNote) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *meetingsRepository) ListNotesByMeeting(ctx context.Context, meetingID uuid.UUID) ([]domain.MeetingNote, error) {
	var items []domain.MeetingNote
	err := r.db.WithContext(ctx).Where("meeting_id = ?", meetingID).Order("created_at desc").Find(&items).Error
	return items, err
}
