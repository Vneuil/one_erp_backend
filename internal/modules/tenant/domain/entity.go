package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// TenantDatabaseStatus tracks the provisioning lifecycle of a tenant database
type TenantDatabaseStatus string

const (
	TenantDBStatusProvisioning TenantDatabaseStatus = "provisioning"
	TenantDBStatusMigrating    TenantDatabaseStatus = "migrating"
	TenantDBStatusReady        TenantDatabaseStatus = "ready"
	TenantDBStatusFailed       TenantDatabaseStatus = "failed"
)

// TenantDatabase is the control-plane registry entry mapping a company to its
// dedicated PostgreSQL database. Credentials are NOT stored here - for now the
// tenant database is reached using the same shared Postgres user/password as
// the central database (see internal/foundation/config DatabaseConfig).
//
// TODO(phase2+): once tenant databases need isolated credentials, introduce a
// secrets manager integration instead of reusing the central DB user/password.
type TenantDatabase struct {
	types.BaseEntity
	CompanyID     uuid.UUID            `gorm:"type:uuid;uniqueIndex;not null" json:"companyId"`
	DatabaseHost  string               `gorm:"type:varchar(255);not null" json:"databaseHost"`
	DatabasePort  int                  `gorm:"not null" json:"databasePort"`
	DatabaseName  string               `gorm:"type:varchar(100);not null" json:"databaseName"`
	Status        TenantDatabaseStatus `gorm:"type:varchar(20);not null;default:'provisioning'" json:"status"`
	SchemaVersion string               `gorm:"type:varchar(50)" json:"schemaVersion"`
}

func (TenantDatabase) TableName() string {
	return "tenant_databases"
}

// CompanyMembershipStatus tracks a user's membership status within a company
type CompanyMembershipStatus string

const (
	MembershipStatusActive    CompanyMembershipStatus = "active"
	MembershipStatusInvited   CompanyMembershipStatus = "invited"
	MembershipStatusSuspended CompanyMembershipStatus = "suspended"
)

// CompanyMembership is the additive N:M relationship between users and companies.
// It exists alongside the existing single User.CompanyID (kept as the user's
// default/last-active company for backward compatibility).
type CompanyMembership struct {
	types.BaseEntity
	UserID    uuid.UUID               `gorm:"type:uuid;not null;uniqueIndex:idx_company_membership_user_company" json:"userId"`
	CompanyID uuid.UUID               `gorm:"type:uuid;not null;uniqueIndex:idx_company_membership_user_company" json:"companyId"`
	Role      string                  `gorm:"type:varchar(50);not null" json:"role"`
	Status    CompanyMembershipStatus `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
}

func (CompanyMembership) TableName() string {
	return "company_users"
}

// TenantDemoPing is the proof-of-concept table migrated into each tenant's
// own database to demonstrate per-tenant data isolation end-to-end.
type TenantDemoPing struct {
	types.BaseEntity
}

func (TenantDemoPing) TableName() string {
	return "tenant_demo_pings"
}

type TenantDatabaseRepository interface {
	Create(ctx context.Context, td *TenantDatabase) error
	GetByCompanyID(ctx context.Context, companyID uuid.UUID) (*TenantDatabase, error)
	Update(ctx context.Context, td *TenantDatabase) error
}

type CompanyMembershipRepository interface {
	Create(ctx context.Context, m *CompanyMembership) error
	GetByUserAndCompany(ctx context.Context, userID, companyID uuid.UUID) (*CompanyMembership, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]CompanyMembership, error)
}
