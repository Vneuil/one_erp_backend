package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// ActivityLog is a generic audit-trail row captured at the HTTP layer for
// every mutating request (POST/PUT/PATCH/DELETE) that reached a tenant-
// scoped handler and succeeded. Recording at the HTTP layer (rather than
// inside each of the ~30 modules' usecases individually) trades per-field
// diffing for coverage: every module gets a real audit trail for free,
// without touching its own code.
type ActivityLog struct {
	types.BaseEntity
	CompanyID  *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	TenantID   *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	UserID     uuid.UUID  `gorm:"type:uuid;index" json:"userId"`
	UserEmail  string     `gorm:"type:varchar(255)" json:"userEmail"`
	Method     string     `gorm:"type:varchar(10);not null" json:"method"`
	Path       string     `gorm:"type:varchar(255);not null;index" json:"path"`
	Module     string     `gorm:"type:varchar(100);index" json:"module"`
	StatusCode int        `gorm:"not null" json:"statusCode"`
	IPAddress  string     `gorm:"type:varchar(64)" json:"ipAddress"`
}

func (ActivityLog) TableName() string {
	return "activity_logs"
}

type ActivityLogRepository interface {
	Create(ctx context.Context, log *ActivityLog) error
	List(ctx context.Context, query types.PaginationQuery) ([]ActivityLog, int64, error)
}
