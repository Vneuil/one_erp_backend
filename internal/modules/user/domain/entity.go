package domain

import (
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleManager UserRole = "manager"
	RoleStaff   UserRole = "staff"
)

// User represents a system user account
type User struct {
	types.BaseEntity
	CompanyID    *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	Name         string     `gorm:"type:varchar(255);not null" json:"name"`
	Email        string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	Role         UserRole   `gorm:"type:varchar(50);default:'staff';not null" json:"role"`
	// RoleID references a custom Role (see modules/rbac) with granular
	// per-module permissions. Nil means "no custom role assigned" - the
	// user is governed only by the legacy Role string above (admin has
	// full access; manager/staff are unrestricted today, matching current
	// behavior exactly, so assigning custom roles is strictly opt-in and
	// never locks out an existing account on deploy).
	RoleID   *uuid.UUID `gorm:"type:uuid;index" json:"roleId,omitempty"`
	IsActive bool       `gorm:"default:true" json:"isActive"`
}

func (User) TableName() string {
	return "users"
}
