// Package whatsappregistry holds a single control-plane-scoped table:
// WhatsAppNumberIndex, mapping a WhatsApp Cloud API phone_number_id to the
// company (and tenant, if any) that owns it.
//
// This is a DELIBERATE EXCEPTION to the rest of this codebase's rule that
// business/credential data lives in each tenant's own database
// (internal/foundation/tenantctx, and every other module's domain/entity.go
// comments). It has to live in the control-plane DB instead because inbound
// WhatsApp webhook deliveries (POST /omnichannel/webhooks/whatsapp) carry no
// JWT/session and no company identifier of their own - only a
// phone_number_id inside the payload. The webhook handler must resolve
// "which company does this phone_number_id belong to" BEFORE it can call
// internal/foundation/tenant.Manager.GetDB(ctx, companyID) to reach that
// company's own tenant database (where the real ChannelConnection record and
// the Conversation/Message rows live - see internal/modules/omnichannel).
// A per-tenant DB cannot answer "which tenant is this" about itself, so a
// small control-plane-wide lookup index is unavoidable, mirroring how
// internal/modules/tenant/infrastructure's tenant_databases table (company ID
// -> physical DB name) already has to live in the control-plane DB for the
// same structural reason.
package whatsappregistry

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WhatsAppNumberIndex is one phone_number_id -> company/tenant mapping.
// Lives in the control-plane DB (see package doc above), NOT in any tenant
// database, and is therefore not tenant-scoped via tenantctx like every
// other entity in this codebase - there is no "active tenant" at the point
// this table is read (webhook delivery, pre-auth).
type WhatsAppNumberIndex struct {
	types.BaseEntity
	PhoneNumberID string     `gorm:"type:varchar(100);uniqueIndex;not null"`
	CompanyID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	TenantID      *uuid.UUID `gorm:"type:uuid;index"`
}

func (WhatsAppNumberIndex) TableName() string {
	return "whatsapp_number_index"
}

// Repository is the persistence port for the control-plane registry.
type Repository interface {
	// Upsert creates or replaces the mapping for one phone_number_id.
	Upsert(ctx context.Context, phoneNumberID string, companyID uuid.UUID, tenantID *uuid.UUID) error
	// GetByPhoneNumberID looks up the owning company/tenant for a
	// phone_number_id. Returns (nil, nil) when none is registered.
	GetByPhoneNumberID(ctx context.Context, phoneNumberID string) (*WhatsAppNumberIndex, error)
	// Delete removes the mapping for one phone_number_id (called on
	// disconnect so stale entries never route webhooks to the wrong tenant).
	Delete(ctx context.Context, phoneNumberID string) error
}

type repository struct {
	db *gorm.DB // the CONTROL-PLANE db, never a tenant db
}

// NewRepository builds the registry repository bound to the control-plane
// database (the same *gorm.DB passed to company.NewModule/tenant.NewModule
// in cmd/server/main.go - never a per-tenant connection from
// tenant.Manager.GetDB).
func NewRepository(controlPlaneDB *gorm.DB) Repository {
	return &repository{db: controlPlaneDB}
}

func (r *repository) Upsert(ctx context.Context, phoneNumberID string, companyID uuid.UUID, tenantID *uuid.UUID) error {
	var existing WhatsAppNumberIndex
	err := r.db.WithContext(ctx).Where("phone_number_id = ?", phoneNumberID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(&WhatsAppNumberIndex{
				PhoneNumberID: phoneNumberID,
				CompanyID:     companyID,
				TenantID:      tenantID,
			}).Error
		}
		return err
	}
	return r.db.WithContext(ctx).Model(&WhatsAppNumberIndex{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"company_id": companyID,
		"tenant_id":  tenantID,
	}).Error
}

func (r *repository) GetByPhoneNumberID(ctx context.Context, phoneNumberID string) (*WhatsAppNumberIndex, error) {
	if phoneNumberID == "" {
		return nil, nil
	}
	var idx WhatsAppNumberIndex
	err := r.db.WithContext(ctx).Where("phone_number_id = ?", phoneNumberID).First(&idx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &idx, nil
}

func (r *repository) Delete(ctx context.Context, phoneNumberID string) error {
	return r.db.WithContext(ctx).Where("phone_number_id = ?", phoneNumberID).Delete(&WhatsAppNumberIndex{}).Error
}
