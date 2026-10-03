package domain

import (
	"context"

	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type LeaveRequest struct {
	types.BaseEntity
	CompanyID *uuid.UUID `gorm:"type:uuid;index" json:"companyId,omitempty"`
	// TenantID scopes this leave request to one business unit within the
	// company (see modules/workspace). Nil means the company's default tenant.
	TenantID *uuid.UUID `gorm:"type:uuid;index" json:"tenantId,omitempty"`
	// EmployeeID links this request to a real HRM employee record (see
	// modules/hrm). EmployeeName/Department are denormalized copies filled
	// in server-side from that record, never trusted from the client.
	EmployeeID   uuid.UUID `gorm:"type:uuid;not null;index" json:"employeeId"`
	EmployeeName string    `gorm:"type:varchar(255);not null" json:"employeeName"`
	Department   string    `gorm:"type:varchar(100);not null" json:"department"`
	Type         string    `gorm:"type:varchar(100);not null" json:"type"`
	StartDate    string    `gorm:"type:varchar(50);not null" json:"startDate"`
	EndDate      string    `gorm:"type:varchar(50);not null" json:"endDate"`
	TotalDays    int       `gorm:"type:int;default:0" json:"totalDays"`
	Reason       string    `gorm:"type:text" json:"reason"`
	Status       string    `gorm:"type:varchar(50);default:'pending'" json:"status"`
	ApprovedBy   string    `gorm:"type:varchar(255)" json:"approvedBy"`
}

func (LeaveRequest) TableName() string {
	return "leave_requests"
}

type LeaveRepository interface {
	Create(ctx context.Context, req *LeaveRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*LeaveRequest, error)
	List(ctx context.Context, query types.PaginationQuery) ([]LeaveRequest, int64, error)
	Update(ctx context.Context, req *LeaveRequest) error
	Count(ctx context.Context) (int64, error)
	// FindOverlapping returns the employee's pending or approved requests whose
	// date range intersects [start, end] (YYYY-MM-DD, inclusive).
	FindOverlapping(ctx context.Context, employeeID uuid.UUID, start, end string) ([]LeaveRequest, error)
}
