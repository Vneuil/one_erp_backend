package infrastructure

import (
	"context"
	"errors"

	"github.com/divinecoid/one-backend/internal/modules/rbac/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type rbacRepository struct {
	db *gorm.DB
}

func NewRBACRepository(db *gorm.DB) domain.RBACRepository {
	return &rbacRepository{db: db}
}

func (r *rbacRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *rbacRepository) GetRoleByID(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	var role domain.Role
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *rbacRepository) ListRolesByCompany(ctx context.Context, companyID uuid.UUID) ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.WithContext(ctx).Where("company_id = ?", companyID).Order("is_system desc, name asc").Find(&roles).Error
	return roles, err
}

func (r *rbacRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&domain.Permission{}).Error; err != nil {
			return err
		}
		return tx.Delete(&domain.Role{}, id).Error
	})
}

func (r *rbacRepository) ReplacePermissions(ctx context.Context, roleID uuid.UUID, perms []domain.Permission) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&domain.Permission{}).Error; err != nil {
			return err
		}
		if len(perms) == 0 {
			return nil
		}
		for i := range perms {
			perms[i].RoleID = roleID
		}
		return tx.Create(&perms).Error
	})
}

func (r *rbacRepository) ListPermissionsByRole(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	var perms []domain.Permission
	err := r.db.WithContext(ctx).Where("role_id = ?", roleID).Order("module asc").Find(&perms).Error
	return perms, err
}

func (r *rbacRepository) GetPermission(ctx context.Context, roleID uuid.UUID, module string) (*domain.Permission, error) {
	var perm domain.Permission
	err := r.db.WithContext(ctx).Where("role_id = ? AND module = ?", roleID, module).First(&perm).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &perm, nil
}
