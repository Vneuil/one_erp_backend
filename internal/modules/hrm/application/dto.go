package application

import (
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/google/uuid"
)

type CreateEmployeeDTO struct {
	NIP          string  `json:"nip"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Phone        string  `json:"phone"`
	Department   string  `json:"department"`
	Role         string  `json:"role"`
	ContractType string  `json:"contractType"`
	JoinDate     string  `json:"joinDate"`
	BaseSalary   float64 `json:"baseSalary"`
	Status       string  `json:"status"`
	// PTKPStatus (TK/0..TK/3, K/0..K/3) defaults to TK/0; HasNPWP defaults to true.
	PTKPStatus string `json:"ptkpStatus"`
	HasNPWP    *bool  `json:"hasNpwp"`
}

type EmployeeResponseDTO struct {
	ID             uuid.UUID  `json:"id"`
	NIP            string     `json:"nip"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Phone          string     `json:"phone"`
	Department     string     `json:"department"`
	Role           string     `json:"role"`
	ContractType   string     `json:"contractType"`
	JoinDate       string     `json:"joinDate"`
	Status         string     `json:"status"`
	BaseSalary     float64    `json:"baseSalary"`
	LeaveQuotaDays int        `json:"leaveQuotaDays"`
	LeaveUsedDays  int        `json:"leaveUsedDays"`
	LeaveBalance   int        `json:"leaveBalance"`
	PTKPStatus     string     `json:"ptkpStatus"`
	HasNPWP        bool       `json:"hasNpwp"`
	ManagerID      *uuid.UUID `json:"managerId,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type ClockInDTO struct {
	EmployeeName string `json:"employeeName"`
	NIP          string `json:"nip"`
	Location     string `json:"location"`
	Method       string `json:"method"`
	// Latitude/Longitude are required when the company has configured
	// attendance locations (geofence); PhotoRef is an optional verification photo.
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	PhotoRef  string   `json:"photoRef,omitempty"`
}

type ClockOutDTO struct {
	NIP       string   `json:"nip"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	PhotoRef  string   `json:"photoRef,omitempty"`
}

type CreateAttendanceLocationDTO struct {
	Name         string  `json:"name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	RadiusMeters int     `json:"radiusMeters"`
}

type AttendanceLocationResponseDTO struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	RadiusMeters int       `json:"radiusMeters"`
	IsActive     bool      `json:"isActive"`
}

type AttendanceResponseDTO struct {
	ID           uuid.UUID `json:"id"`
	EmployeeName string    `json:"employeeName"`
	NIP          string    `json:"nip"`
	Date         string    `json:"date"`
	ClockIn      string    `json:"clockIn"`
	ClockOut     string    `json:"clockOut"`
	Location     string    `json:"location"`
	Method       string    `json:"method"`
	Status       string    `json:"status"`
	WorkMinutes  int       `json:"workMinutes"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	PhotoRef     string    `json:"photoRef,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

func ToEmployeeResponse(e *domain.Employee) *EmployeeResponseDTO {
	if e == nil {
		return nil
	}
	return &EmployeeResponseDTO{
		ID:             e.ID,
		NIP:            e.NIP,
		Name:           e.Name,
		Email:          e.Email,
		Phone:          e.Phone,
		Department:     e.Department,
		Role:           e.Role,
		ContractType:   e.ContractType,
		JoinDate:       e.JoinDate,
		Status:         e.Status,
		BaseSalary:     e.BaseSalary,
		LeaveQuotaDays: e.LeaveQuotaDays,
		LeaveUsedDays:  e.LeaveUsedDays,
		LeaveBalance:   e.LeaveQuotaDays - e.LeaveUsedDays,
		PTKPStatus:     e.PTKPStatus,
		HasNPWP:        !e.NoNPWP,
		ManagerID:      e.ManagerID,
		CreatedAt:      e.CreatedAt,
	}
}

func ToEmployeeResponseList(employees []domain.Employee) []EmployeeResponseDTO {
	result := make([]EmployeeResponseDTO, len(employees))
	for i, e := range employees {
		result[i] = *ToEmployeeResponse(&e)
	}
	return result
}

func ToAttendanceResponse(a *domain.Attendance) *AttendanceResponseDTO {
	if a == nil {
		return nil
	}
	return &AttendanceResponseDTO{
		ID:           a.ID,
		EmployeeName: a.EmployeeName,
		NIP:          a.NIP,
		Date:         a.Date,
		ClockIn:      a.ClockIn,
		ClockOut:     a.ClockOut,
		Location:     a.Location,
		Method:       a.Method,
		Status:       a.Status,
		WorkMinutes:  a.WorkMinutes,
		Latitude:     a.Latitude,
		Longitude:    a.Longitude,
		PhotoRef:     a.PhotoRef,
		CreatedAt:    a.CreatedAt,
	}
}

func ToAttendanceResponseList(records []domain.Attendance) []AttendanceResponseDTO {
	result := make([]AttendanceResponseDTO, len(records))
	for i, a := range records {
		result[i] = *ToAttendanceResponse(&a)
	}
	return result
}
