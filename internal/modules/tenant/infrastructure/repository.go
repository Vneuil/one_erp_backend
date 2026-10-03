package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/modules/tenant/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type tenantDatabaseRepository struct {
	db *gorm.DB
}

func NewTenantDatabaseRepository(db *gorm.DB) domain.TenantDatabaseRepository {
	return &tenantDatabaseRepository{db: db}
}

func (r *tenantDatabaseRepository) Create(ctx context.Context, td *domain.TenantDatabase) error {
	return r.db.WithContext(ctx).Create(td).Error
}

func (r *tenantDatabaseRepository) GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*domain.TenantDatabase, error) {
	var td domain.TenantDatabase
	err := r.db.WithContext(ctx).Where("company_id = ?", companyID).First(&td).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &td, nil
}

func (r *tenantDatabaseRepository) Update(ctx context.Context, td *domain.TenantDatabase) error {
	return r.db.WithContext(ctx).Save(td).Error
}

type companyMembershipRepository struct {
	db *gorm.DB
}

func NewCompanyMembershipRepository(db *gorm.DB) domain.CompanyMembershipRepository {
	return &companyMembershipRepository{db: db}
}

func (r *companyMembershipRepository) Create(ctx context.Context, m *domain.CompanyMembership) error {
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *companyMembershipRepository) GetByUserAndCompany(ctx context.Context, userID, companyID uuid.UUID) (*domain.CompanyMembership, error) {
	var m domain.CompanyMembership
	err := r.db.WithContext(ctx).Where("user_id = ? AND company_id = ?", userID, companyID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *companyMembershipRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]domain.CompanyMembership, error) {
	var memberships []domain.CompanyMembership
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at asc").Find(&memberships).Error
	return memberships, err
}
