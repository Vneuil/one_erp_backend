package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/rbac/domain"
	"github.com/google/uuid"
)

type CreateRoleDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleResponseDTO struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	IsSystem    bool      `json:"isSystem"`
	CreatedAt   time.Time `json:"createdAt"`
}

func ToRoleResponse(r *domain.Role) RoleResponseDTO {
	return RoleResponseDTO{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		CreatedAt:   r.CreatedAt,
	}
}

func ToRoleResponseList(items []domain.Role) []RoleResponseDTO {
	result := make([]RoleResponseDTO, len(items))
	for i, r := range items {
		result[i] = ToRoleResponse(&r)
	}
	return result
}

type PermissionDTO struct {
	Module        string `json:"module"`
	CanView       bool   `json:"canView"`
	CanCreate     bool   `json:"canCreate"`
	CanEdit       bool   `json:"canEdit"`
	CanDelete     bool   `json:"canDelete"`
	CanSubmit     bool   `json:"canSubmit"`
	CanManage     bool   `json:"canManage"`
	CanApprove    bool   `json:"canApprove"`
	CanViewPrices bool   `json:"canViewPrices"`
	CanEditPrices bool   `json:"canEditPrices"`
}

type SetPermissionsDTO struct {
	Permissions []PermissionDTO `json:"permissions"`
}

func ToPermissionDTOList(items []domain.Permission) []PermissionDTO {
	result := make([]PermissionDTO, len(items))
	for i, p := range items {
		canManage := p.CanManage || (p.CanCreate && p.CanEdit)
		result[i] = PermissionDTO{
			Module:        p.Module,
			CanView:       p.CanView,
			CanCreate:     p.CanCreate || p.CanManage,
			CanEdit:       p.CanEdit || p.CanManage,
			CanDelete:     p.CanDelete,
			CanSubmit:     p.CanSubmit || p.CanManage,
			CanManage:     canManage,
			CanApprove:    p.CanApprove,
			CanViewPrices: p.CanViewPrices,
			CanEditPrices: p.CanEditPrices,
		}
	}
	return result
}
