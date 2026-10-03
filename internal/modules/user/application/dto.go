package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/user/domain"
	"github.com/google/uuid"
)

type CreateUserDTO struct {
	CompanyID *uuid.UUID      `json:"companyId"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Password  string          `json:"password"`
	Role      domain.UserRole `json:"role"`
}

type UpdateUserDTO struct {
	Name     *string          `json:"name"`
	Role     *domain.UserRole `json:"role"`
	IsActive *bool            `json:"isActive"`
}

type AssignRoleDTO struct {
	RoleID *uuid.UUID `json:"roleId"`
}

type UserResponseDTO struct {
	ID        uuid.UUID       `json:"id"`
	CompanyID *uuid.UUID      `json:"companyId,omitempty"`
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	Role      domain.UserRole `json:"role"`
	RoleID    *uuid.UUID      `json:"roleId,omitempty"`
	IsActive  bool            `json:"isActive"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

func ToUserResponse(user *domain.User) UserResponseDTO {
	return UserResponseDTO{
		ID:        user.ID,
		CompanyID: user.CompanyID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		RoleID:    user.RoleID,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
