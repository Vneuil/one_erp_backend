package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/leave/domain"
	"github.com/google/uuid"
)

type CreateLeaveRequestDTO struct {
	// EmployeeID must reference a real employee (see modules/hrm). It is
	// either picked from a dropdown on the frontend, or server-locked to
	// the caller's own employee record for self-service leave requests.
	EmployeeID uuid.UUID `json:"employeeId"`
	Type       string    `json:"type"`
	StartDate  string    `json:"startDate"`
	EndDate    string    `json:"endDate"`
	TotalDays  int       `json:"totalDays"`
	Reason     string    `json:"reason"`
}

type ApproveLeaveRequestDTO struct {
	ApprovedBy string `json:"approvedBy"`
}

type RejectLeaveRequestDTO struct {
	ApprovedBy string `json:"approvedBy"`
}

type LeaveRequestResponseDTO struct {
	ID           uuid.UUID `json:"id"`
	EmployeeID   uuid.UUID `json:"employeeId"`
	EmployeeName string    `json:"employeeName"`
	Department   string    `json:"department"`
	Type         string    `json:"type"`
	StartDate    string    `json:"startDate"`
	EndDate      string    `json:"endDate"`
	TotalDays    int       `json:"totalDays"`
	Reason       string    `json:"reason"`
	Status       string    `json:"status"`
	ApprovedBy   string    `json:"approvedBy"`
	CreatedAt    time.Time `json:"createdAt"`
}

func ToLeaveRequestResponse(r *domain.LeaveRequest) *LeaveRequestResponseDTO {
	if r == nil {
		return nil
	}
	return &LeaveRequestResponseDTO{
		ID:           r.ID,
		EmployeeID:   r.EmployeeID,
		EmployeeName: r.EmployeeName,
		Department:   r.Department,
		Type:         r.Type,
		StartDate:    r.StartDate,
		EndDate:      r.EndDate,
		TotalDays:    r.TotalDays,
		Reason:       r.Reason,
		Status:       r.Status,
		ApprovedBy:   r.ApprovedBy,
		CreatedAt:    r.CreatedAt,
	}
}

func ToLeaveRequestResponseList(items []domain.LeaveRequest) []LeaveRequestResponseDTO {
	result := make([]LeaveRequestResponseDTO, len(items))
	for i, item := range items {
		result[i] = *ToLeaveRequestResponse(&item)
	}
	return result
}
