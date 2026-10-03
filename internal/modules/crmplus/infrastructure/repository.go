package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/divinecoid/one-backend/internal/foundation/tenantctx"
	"github.com/divinecoid/one-backend/internal/modules/crmplus/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) domain.Repository { return &repository{db: db} }

func (r *repository) q(ctx context.Context, model any) *gorm.DB {
	return tenantctx.Scope(ctx, r.db.WithContext(ctx).Model(model))
}

func (r *repository) create(ctx context.Context, v any, tenant **uuid.UUID) error {
	if tenant != nil {
		tenantctx.SetTenantID(ctx, tenant)
	}
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *repository) save(ctx context.Context, v any) error {
	return r.db.WithContext(ctx).Save(v).Error
}

func getByID[T any](ctx context.Context, r *repository, id uuid.UUID) (*T, error) {
	var v T
	err := r.q(ctx, &v).Where("id = ?", id).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func deleteByID[T any](ctx context.Context, r *repository, id uuid.UUID) (bool, error) {
	var v T
	res := r.q(ctx, &v).Where("id = ?", id).Delete(&v)
	return res.RowsAffected > 0, res.Error
}

// --- interactions

func (r *repository) CreateInteraction(ctx context.Context, v *domain.Interaction) error {
	return r.create(ctx, v, &v.TenantID)
}

func (r *repository) ListInteractions(ctx context.Context, parentType string, parentID uuid.UUID) ([]domain.Interaction, error) {
	var out []domain.Interaction
	return out, r.q(ctx, &domain.Interaction{}).Where("parent_type = ? AND parent_id = ?", parentType, parentID).Order("occurred_at desc").Find(&out).Error
}

func (r *repository) LastInteractionByParent(ctx context.Context, parentType string) (map[uuid.UUID]time.Time, error) {
	type row struct {
		ParentID uuid.UUID
		Last     time.Time
	}
	var rows []row
	err := r.q(ctx, &domain.Interaction{}).Select("parent_id, MAX(occurred_at) AS last").Where("parent_type = ?", parentType).Group("parent_id").Scan(&rows).Error
	out := make(map[uuid.UUID]time.Time, len(rows))
	for _, x := range rows {
		out[x.ParentID] = x.Last
	}
	return out, err
}

// --- tasks

func (r *repository) CreateTask(ctx context.Context, v *domain.SalesTask) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetTask(ctx context.Context, id uuid.UUID) (*domain.SalesTask, error) {
	return getByID[domain.SalesTask](ctx, r, id)
}
func (r *repository) UpdateTask(ctx context.Context, v *domain.SalesTask) error {
	return r.save(ctx, v)
}
func (r *repository) ListTasks(ctx context.Context, status, assignee, parentType string, parentID *uuid.UUID) ([]domain.SalesTask, error) {
	var out []domain.SalesTask
	db := r.q(ctx, &domain.SalesTask{})
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if assignee != "" {
		db = db.Where("LOWER(assignee_email) = LOWER(?)", assignee)
	}
	if parentType != "" && parentID != nil {
		db = db.Where("parent_type = ? AND parent_id = ?", parentType, *parentID)
	}
	return out, db.Order("due_date asc, created_at asc").Find(&out).Error
}

// --- documents

func (r *repository) CreateDocument(ctx context.Context, v *domain.Document) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) ListDocuments(ctx context.Context, parentType string, parentID uuid.UUID) ([]domain.Document, error) {
	var out []domain.Document
	return out, r.q(ctx, &domain.Document{}).Where("parent_type = ? AND parent_id = ?", parentType, parentID).Order("created_at desc").Find(&out).Error
}
func (r *repository) GetDocument(ctx context.Context, id uuid.UUID) (*domain.Document, error) {
	return getByID[domain.Document](ctx, r, id)
}
func (r *repository) DeleteDocument(ctx context.Context, id uuid.UUID) (bool, error) {
	return deleteByID[domain.Document](ctx, r, id)
}

// --- tags

func (r *repository) CreateTag(ctx context.Context, v *domain.Tag) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) ListTags(ctx context.Context) ([]domain.Tag, error) {
	var out []domain.Tag
	return out, r.q(ctx, &domain.Tag{}).Order("name asc").Find(&out).Error
}
func (r *repository) GetTag(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	return getByID[domain.Tag](ctx, r, id)
}
func (r *repository) DeleteTag(ctx context.Context, id uuid.UUID) (bool, error) {
	if err := r.db.WithContext(ctx).Where("tag_id = ?", id).Delete(&domain.TagAssignment{}).Error; err != nil {
		return false, err
	}
	return deleteByID[domain.Tag](ctx, r, id)
}
func (r *repository) AssignTag(ctx context.Context, v *domain.TagAssignment) error {
	// Assigning twice is a no-op thanks to the unique index.
	return r.db.WithContext(ctx).Where(domain.TagAssignment{TagID: v.TagID, ParentType: v.ParentType, ParentID: v.ParentID}).FirstOrCreate(v).Error
}
func (r *repository) UnassignTag(ctx context.Context, tagID uuid.UUID, parentType string, parentID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("tag_id = ? AND parent_type = ? AND parent_id = ?", tagID, parentType, parentID).Delete(&domain.TagAssignment{}).Error
}
func (r *repository) TagsFor(ctx context.Context, parentType string, parentID uuid.UUID) ([]domain.Tag, error) {
	var out []domain.Tag
	return out, r.db.WithContext(ctx).Model(&domain.Tag{}).
		Joins("JOIN crm_tag_assignments a ON a.tag_id = crm_tags.id AND a.deleted_at IS NULL").
		Where("a.parent_type = ? AND a.parent_id = ?", parentType, parentID).Order("crm_tags.name asc").Find(&out).Error
}
func (r *repository) ParentsWithTag(ctx context.Context, tagID uuid.UUID, parentType string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	return ids, r.db.WithContext(ctx).Model(&domain.TagAssignment{}).Where("tag_id = ? AND parent_type = ?", tagID, parentType).Pluck("parent_id", &ids).Error
}

// --- contacts

func (r *repository) CreateContact(ctx context.Context, v *domain.Contact) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetContact(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	return getByID[domain.Contact](ctx, r, id)
}
func (r *repository) UpdateContact(ctx context.Context, v *domain.Contact) error {
	return r.save(ctx, v)
}
func (r *repository) DeleteContact(ctx context.Context, id uuid.UUID) (bool, error) {
	return deleteByID[domain.Contact](ctx, r, id)
}
func (r *repository) ListContacts(ctx context.Context, companyName string, leadID *uuid.UUID) ([]domain.Contact, error) {
	var out []domain.Contact
	db := r.q(ctx, &domain.Contact{})
	if companyName != "" {
		db = db.Where("LOWER(company_name) = LOWER(?)", companyName)
	}
	if leadID != nil {
		db = db.Where("lead_id = ?", *leadID)
	}
	return out, db.Order("is_primary desc, name asc").Find(&out).Error
}

// --- stages

func (r *repository) CreateStage(ctx context.Context, v *domain.PipelineStage) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetStage(ctx context.Context, id uuid.UUID) (*domain.PipelineStage, error) {
	return getByID[domain.PipelineStage](ctx, r, id)
}
func (r *repository) UpdateStage(ctx context.Context, v *domain.PipelineStage) error {
	return r.save(ctx, v)
}
func (r *repository) DeleteStage(ctx context.Context, id uuid.UUID) error {
	_, err := deleteByID[domain.PipelineStage](ctx, r, id)
	return err
}
func (r *repository) ListStages(ctx context.Context, activeOnly bool) ([]domain.PipelineStage, error) {
	var out []domain.PipelineStage
	db := r.q(ctx, &domain.PipelineStage{})
	if activeOnly {
		db = db.Where("is_active = ?", true)
	}
	return out, db.Order("position asc").Find(&out).Error
}
func (r *repository) CountStages(ctx context.Context) (int64, error) {
	var n int64
	return n, r.q(ctx, &domain.PipelineStage{}).Count(&n).Error
}

// --- lead forms

func (r *repository) CreateLeadForm(ctx context.Context, v *domain.LeadForm) error {
	return r.create(ctx, v, &v.TenantID)
}
func (r *repository) GetLeadFormByKey(ctx context.Context, key string) (*domain.LeadForm, error) {
	var v domain.LeadForm
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &v, err
}
func (r *repository) GetLeadForm(ctx context.Context, id uuid.UUID) (*domain.LeadForm, error) {
	return getByID[domain.LeadForm](ctx, r, id)
}
func (r *repository) UpdateLeadForm(ctx context.Context, v *domain.LeadForm) error {
	return r.save(ctx, v)
}
func (r *repository) DeleteLeadForm(ctx context.Context, id uuid.UUID) error {
	_, err := deleteByID[domain.LeadForm](ctx, r, id)
	return err
}
func (r *repository) ListLeadForms(ctx context.Context) ([]domain.LeadForm, error) {
	var out []domain.LeadForm
	return out, r.q(ctx, &domain.LeadForm{}).Order("created_at desc").Find(&out).Error
}
func (r *repository) IncrementSubmissions(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.LeadForm{}).Where("id = ?", id).UpdateColumn("submissions", gorm.Expr("submissions + 1")).Error
}

// --- control-plane registry

type formRegistry struct{ db *gorm.DB }

// NewFormRegistry binds the registry to the CONTROL-PLANE database.
func NewFormRegistry(controlPlaneDB *gorm.DB) domain.FormRegistry {
	return &formRegistry{db: controlPlaneDB}
}

func (r *formRegistry) Upsert(ctx context.Context, key string, companyID uuid.UUID, tenantID *uuid.UUID) error {
	var existing domain.LeadFormIndex
	err := r.db.WithContext(ctx).Where("form_key = ?", key).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.WithContext(ctx).Create(&domain.LeadFormIndex{FormKey: key, CompanyID: companyID, TenantID: tenantID}).Error
	}
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&domain.LeadFormIndex{}).Where("id = ?", existing.ID).Updates(map[string]any{"company_id": companyID, "tenant_id": tenantID}).Error
}

func (r *formRegistry) Lookup(ctx context.Context, key string) (*domain.LeadFormIndex, error) {
	var v domain.LeadFormIndex
	err := r.db.WithContext(ctx).Where("form_key = ?", key).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &v, err
}

func (r *formRegistry) Delete(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Where("form_key = ?", key).Delete(&domain.LeadFormIndex{}).Error
}
