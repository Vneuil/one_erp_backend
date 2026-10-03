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
	Module    string `json:"module"`
	CanView   bool   `json:"canView"`
	CanManage bool   `json:"canManage"`
	// CanApprove allows approving/rejecting/paying/posting documents of this module.
	CanApprove bool `json:"canApprove"`
}

type SetPermissionsDTO struct {
	Permissions []PermissionDTO `json:"permissions"`
}

func ToPermissionDTOList(items []domain.Permission) []PermissionDTO {
	result := make([]PermissionDTO, len(items))
	for i, p := range items {
		result[i] = PermissionDTO{Module: p.Module, CanView: p.CanView, CanManage: p.CanManage, CanApprove: p.CanApprove}
	}
	return result
}
