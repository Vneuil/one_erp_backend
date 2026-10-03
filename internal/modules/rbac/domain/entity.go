package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// Role is a company-defined authorization group with granular, per-module
// permissions (see Permission below). Lives in the control-plane database
// alongside modules/user, since a Role is meaningless without the Users it
// governs and Users themselves are control-plane, not tenant, data.
type Role struct {
	types.BaseEntity
	CompanyID   uuid.UUID `gorm:"type:uuid;not null;index" json:"companyId"`
	Name        string    `gorm:"type:varchar(150);not null" json:"name"`
	Description string    `gorm:"type:varchar(255)" json:"description,omitempty"`
	// IsSystem marks the three built-in roles (Admin/Manager/Staff) seeded
	// for every company - not deletable, though their permissions can
	// still be edited like any other role.
	IsSystem bool `gorm:"default:false" json:"isSystem"`
}

func (Role) TableName() string {
	return "roles"
}

// Permission is one (role, module) grant: whether the role can view
// (GET) and/or manage (POST/PUT/PATCH/DELETE) that module's endpoints.
// One row per module the role has ANY access to - a module with no row
// for a given role is fully denied once that role is assigned to a user
// (see rbac.Enforce middleware).
type Permission struct {
	types.BaseEntity
	RoleID    uuid.UUID `gorm:"type:uuid;not null;index:idx_permission_role_module,unique" json:"roleId"`
	Module    string    `gorm:"type:varchar(100);not null;index:idx_permission_role_module,unique" json:"module"`
	CanView   bool      `gorm:"default:false" json:"canView"`
	CanManage bool      `gorm:"default:false" json:"canManage"`
	// CanApprove is a separate, stronger grant than CanManage: it allows the
	// role to approve/reject/pay/post documents in this module (see
	// ApprovalGate). Managing a module (creating and editing records) does not
	// imply being allowed to approve them.
	CanApprove bool `gorm:"default:false" json:"canApprove"`
}

func (Permission) TableName() string {
	return "role_permissions"
}

type RBACRepository interface {
	CreateRole(ctx context.Context, r *Role) error
	GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error)
	ListRolesByCompany(ctx context.Context, companyID uuid.UUID) ([]Role, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error

	ReplacePermissions(ctx context.Context, roleID uuid.UUID, perms []Permission) error
	ListPermissionsByRole(ctx context.Context, roleID uuid.UUID) ([]Permission, error)
	GetPermission(ctx context.Context, roleID uuid.UUID, module string) (*Permission, error)
}
