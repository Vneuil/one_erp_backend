package application

import (
	"context"
	"strings"

	"github.com/divinecoid/one-backend/internal/modules/rbac/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/google/uuid"
)

// allModules is the canonical list of module keys permissions can be
// granted against - kept here (not derived dynamically) so the
// permission matrix has a stable, predictable shape for the frontend.
// Matches the moduleFromPath() segments used by modules/activitylog and
// this package's own Enforce middleware.
var allModules = []string{
	"sales", "pos", "procurement", "loyalty", "crm", "omnichannel", "marketplace",
	"inventory", "warehouse", "manufacturing", "product", "customers", "suppliers",
	"salesmen", "shipping-methods", "product-categories", "units-of-measure",
	"payment-terms", "customer-types", "tax-rates",
	"finance", "banking", "assets", "commission", "contracts",
	"hrm", "recruitment", "lms", "project", "goals", "support", "device",
	"collaboration", "docflow", "integration", "report",
	"users", "rbac", "activity-logs", "tenant", "company",
}

type RoleWithPermissionsDTO struct {
	Role        RoleResponseDTO `json:"role"`
	Permissions []PermissionDTO `json:"permissions"`
}

type RBACUseCase interface {
	ListRoles(ctx context.Context, companyID uuid.UUID) ([]RoleResponseDTO, error)
	CreateRole(ctx context.Context, companyID uuid.UUID, dto CreateRoleDTO) (*RoleResponseDTO, error)
	DeleteRole(ctx context.Context, id uuid.UUID) error

	GetRoleWithPermissions(ctx context.Context, id uuid.UUID) (*RoleWithPermissionsDTO, error)
	SetPermissions(ctx context.Context, roleID uuid.UUID, dto SetPermissionsDTO) ([]PermissionDTO, error)

	AvailableModules() []string
}

type rbacUseCase struct {
	repo domain.RBACRepository
}

func NewRBACUseCase(repo domain.RBACRepository) RBACUseCase {
	return &rbacUseCase{repo: repo}
}

func (uc *rbacUseCase) AvailableModules() []string {
	return allModules
}

// seedSystemRoles lazily creates the three built-in roles for a company
// the first time its roles are listed, so no separate hook is needed in
// the company-creation flow. Admin gets full access to every module;
// Manager gets view+manage everywhere except Users/RBAC/Settings;
// Staff gets view-only everywhere except Users/RBAC/Settings.
func (uc *rbacUseCase) seedSystemRoles(ctx context.Context, companyID uuid.UUID) error {
	restricted := map[string]bool{"users": true, "rbac": true, "tenant": true, "company": true}

	seed := []struct {
		name        string
		description string
		build       func(module string) PermissionDTO
	}{
		{"Admin", "Full access to every module", func(m string) PermissionDTO {
			return PermissionDTO{Module: m, CanView: true, CanManage: true, CanApprove: true}
		}},
		{"Manager", "View and manage business modules; no user/role administration", func(m string) PermissionDTO {
			if restricted[m] {
				return PermissionDTO{Module: m, CanView: true, CanManage: false}
			}
			return PermissionDTO{Module: m, CanView: true, CanManage: true, CanApprove: true}
		}},
		{"Staff", "View-only access to business modules", func(m string) PermissionDTO {
			if restricted[m] {
				return PermissionDTO{Module: m, CanView: false, CanManage: false}
			}
			return PermissionDTO{Module: m, CanView: true, CanManage: false}
		}},
	}

	for _, s := range seed {
		role := &domain.Role{CompanyID: companyID, Name: s.name, Description: s.description, IsSystem: true}
		if err := uc.repo.CreateRole(ctx, role); err != nil {
			return err
		}
		perms := make([]domain.Permission, 0, len(allModules))
		for _, m := range allModules {
			dto := s.build(m)
			perms = append(perms, domain.Permission{RoleID: role.ID, Module: dto.Module, CanView: dto.CanView, CanManage: dto.CanManage, CanApprove: dto.CanApprove})
		}
		if err := uc.repo.ReplacePermissions(ctx, role.ID, perms); err != nil {
			return err
		}
	}
	return nil
}

func (uc *rbacUseCase) ListRoles(ctx context.Context, companyID uuid.UUID) ([]RoleResponseDTO, error) {
	roles, err := uc.repo.ListRolesByCompany(ctx, companyID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list roles")
	}
	if len(roles) == 0 {
		if err := uc.seedSystemRoles(ctx, companyID); err != nil {
			return nil, apperrors.NewInternal(err, "Failed to seed default roles")
		}
		roles, err = uc.repo.ListRolesByCompany(ctx, companyID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to list roles")
		}
	}
	return ToRoleResponseList(roles), nil
}

func (uc *rbacUseCase) CreateRole(ctx context.Context, companyID uuid.UUID, dto CreateRoleDTO) (*RoleResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	if name == "" {
		return nil, apperrors.NewBadRequest("Role name is required")
	}
	role := &domain.Role{CompanyID: companyID, Name: name, Description: dto.Description}
	if err := uc.repo.CreateRole(ctx, role); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create role")
	}
	res := ToRoleResponse(role)
	return &res, nil
}

func (uc *rbacUseCase) DeleteRole(ctx context.Context, id uuid.UUID) error {
	role, err := uc.repo.GetRoleByID(ctx, id)
	if err != nil {
		return apperrors.NewInternal(err, "Failed to retrieve role")
	}
	if role == nil {
		return apperrors.NewNotFound("Role not found")
	}
	if role.IsSystem {
		return apperrors.NewBadRequest("Built-in roles cannot be deleted")
	}
	if err := uc.repo.DeleteRole(ctx, id); err != nil {
		return apperrors.NewInternal(err, "Failed to delete role")
	}
	return nil
}

func (uc *rbacUseCase) GetRoleWithPermissions(ctx context.Context, id uuid.UUID) (*RoleWithPermissionsDTO, error) {
	role, err := uc.repo.GetRoleByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve role")
	}
	if role == nil {
		return nil, apperrors.NewNotFound("Role not found")
	}
	perms, err := uc.repo.ListPermissionsByRole(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list permissions")
	}
	return &RoleWithPermissionsDTO{Role: ToRoleResponse(role), Permissions: ToPermissionDTOList(perms)}, nil
}

func (uc *rbacUseCase) SetPermissions(ctx context.Context, roleID uuid.UUID, dto SetPermissionsDTO) ([]PermissionDTO, error) {
	role, err := uc.repo.GetRoleByID(ctx, roleID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to retrieve role")
	}
	if role == nil {
		return nil, apperrors.NewNotFound("Role not found")
	}

	perms := make([]domain.Permission, 0, len(dto.Permissions))
	for _, p := range dto.Permissions {
		if p.Module == "" {
			continue
		}
		perms = append(perms, domain.Permission{Module: p.Module, CanView: p.CanView, CanManage: p.CanManage, CanApprove: p.CanApprove})
	}
	if err := uc.repo.ReplacePermissions(ctx, roleID, perms); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to save permissions")
	}

	saved, err := uc.repo.ListPermissionsByRole(ctx, roleID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list permissions")
	}
	return ToPermissionDTOList(saved), nil
}
