package application

import (
	"context"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type HRMUseCase interface {
	CreateEmployee(ctx context.Context, dto CreateEmployeeDTO) (*EmployeeResponseDTO, error)
	UpdateEmployee(ctx context.Context, id uuid.UUID, dto UpdateEmployeeDTO) (*EmployeeResponseDTO, error)
	DeleteEmployee(ctx context.Context, id uuid.UUID) error
	Organization(ctx context.Context) (*Organization, error)
	GetEmployeeByID(ctx context.Context, id uuid.UUID) (*EmployeeResponseDTO, error)
	ListEmployees(ctx context.Context, query types.PaginationQuery) ([]EmployeeResponseDTO, types.PaginationMeta, error)

	RecordClockIn(ctx context.Context, dto ClockInDTO) (*AttendanceResponseDTO, error)
	RecordClockOut(ctx context.Context, dto ClockOutDTO) (*AttendanceResponseDTO, error)
	ListAttendance(ctx context.Context, query types.PaginationQuery) ([]AttendanceResponseDTO, types.PaginationMeta, error)

	CreateAttendanceLocation(ctx context.Context, dto CreateAttendanceLocationDTO) (*AttendanceLocationResponseDTO, error)
	ListAttendanceLocations(ctx context.Context) ([]AttendanceLocationResponseDTO, error)
	DeleteAttendanceLocation(ctx context.Context, id uuid.UUID) error

	SeedInitialData(ctx context.Context) error
}

type hrmUseCase struct {
	repo domain.HRMRepository
	now  func() time.Time // injectable clock for tests
}

func NewHRMUseCase(repo domain.HRMRepository) HRMUseCase {
	return &hrmUseCase{repo: repo, now: time.Now}
}

func (uc *hrmUseCase) CreateEmployee(ctx context.Context, dto CreateEmployeeDTO) (*EmployeeResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	nip := strings.TrimSpace(dto.NIP)

	if name == "" || nip == "" {
		return nil, apperrors.NewBadRequest("Employee Name and NIP are required")
	}

	dept := dto.Department
	if dept == "" {
		dept = "General"
	}
	role := dto.Role
	if role == "" {
		role = "Staff"
	}
	contractType := dto.ContractType
	if contractType == "" {
		contractType = "PKWTT Tetap"
	}
	joinDate := dto.JoinDate
	if joinDate == "" {
		joinDate = time.Now().Format("2006-01-02")
	}
	status := dto.Status
	if status == "" {
		status = "Active"
	}

	ptkp := strings.ToUpper(strings.TrimSpace(dto.PTKPStatus))
	if ptkp == "" {
		ptkp = "TK/0"
	}
	if !validPTKPStatus(ptkp) {
		return nil, apperrors.NewBadRequest("PTKP status must be one of TK/0-TK/3 or K/0-K/3")
	}

	emp := &domain.Employee{
		NIP:            nip,
		Name:           name,
		Email:          dto.Email,
		Phone:          dto.Phone,
		Department:     dept,
		Role:           role,
		ContractType:   contractType,
		JoinDate:       joinDate,
		Status:         status,
		BaseSalary:     dto.BaseSalary,
		LeaveQuotaDays: 12,
		PTKPStatus:     ptkp,
		NoNPWP:         dto.HasNPWP != nil && !*dto.HasNPWP,
	}

	if err := uc.repo.CreateEmployee(ctx, emp); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create employee")
	}

	return ToEmployeeResponse(emp), nil
}

func (uc *hrmUseCase) GetEmployeeByID(ctx context.Context, id uuid.UUID) (*EmployeeResponseDTO, error) {
	emp, err := uc.repo.GetEmployeeByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get employee")
	}
	if emp == nil {
		return nil, apperrors.NewNotFound("Employee not found")
	}
	return ToEmployeeResponse(emp), nil
}

func (uc *hrmUseCase) ListEmployees(ctx context.Context, query types.PaginationQuery) ([]EmployeeResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	employees, total, err := uc.repo.ListEmployees(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list employees")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToEmployeeResponseList(employees), meta, nil
}

func (uc *hrmUseCase) ListAttendance(ctx context.Context, query types.PaginationQuery) ([]AttendanceResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	records, total, err := uc.repo.ListAttendance(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list attendance")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToAttendanceResponseList(records), meta, nil
}

func (uc *hrmUseCase) SeedInitialData(ctx context.Context) error {
	count, err := uc.repo.CountEmployees(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	employees := []CreateEmployeeDTO{
		{
			NIP:          "EMP-001",
			Name:         "Nicholas Tantra",
			Email:        "nicholas@one-erp.com",
			Phone:        "+62 812-3456-7890",
			Department:   "Management",
			Role:         "Managing Director",
			ContractType: "PKWTT Tetap",
			JoinDate:     "2024-01-15",
			Status:       "Active",
			BaseSalary:   25000000,
		},
		{
			NIP:          "EMP-002",
			Name:         "Budi Santoso",
			Email:        "budi.santoso@one-erp.com",
			Phone:        "+62 813-2233-4455",
			Department:   "Warehouse & Logistics",
			Role:         "Head of Warehouse Cikarang",
			ContractType: "PKWTT Tetap",
			JoinDate:     "2024-03-01",
			Status:       "Active",
			BaseSalary:   12500000,
		},
		{
			NIP:          "EMP-003",
			Name:         "Dewi Lestari",
			Email:        "dewi.lestari@one-erp.com",
			Phone:        "+62 811-9988-7766",
			Department:   "Finance & Accounting",
			Role:         "Senior Tax & General Ledger",
			ContractType: "PKWTT Tetap",
			JoinDate:     "2024-06-10",
			Status:       "Active",
			BaseSalary:   11000000,
		},
		{
			NIP:          "EMP-004",
			Name:         "Rizky Ramadhan",
			Email:        "rizky.r@one-erp.com",
			Phone:        "+62 818-4455-6677",
			Department:   "Sales & Marketing",
			Role:         "Key Account Executive B2B",
			ContractType: "PKWT Kontrak",
			JoinDate:     "2025-02-01",
			Status:       "Active",
			BaseSalary:   8500000,
		},
	}

	for _, item := range employees {
		_, _ = uc.CreateEmployee(ctx, item)
	}

	_ = uc.repo.RecordAttendance(context.Background(), &domain.Attendance{
		EmployeeName: "Nicholas Tantra",
		NIP:          "EMP-001",
		Date:         time.Now().Format("2006-01-02"),
		ClockIn:      "07:54 WIB",
		ClockOut:     "--:--",
		Location:     "Head Office Cikarang (Radius 12m)",
		Method:       "GPS Geofence Mobile",
		Status:       "On Time",
	})
	_ = uc.repo.RecordAttendance(context.Background(), &domain.Attendance{
		EmployeeName: "Budi Santoso",
		NIP:          "EMP-002",
		Date:         time.Now().Format("2006-01-02"),
		ClockIn:      "07:48 WIB",
		ClockOut:     "--:--",
		Location:     "Gudang Utama Cikarang Dock 2",
		Method:       "Terminal Fingerprint NFC",
		Status:       "On Time",
	})

	return nil
}

// validPTKPStatus reports whether s is a PPh 21 tax status: TK/0-TK/3 or K/0-K/3.
func validPTKPStatus(s string) bool {
	switch s {
	case "TK/0", "TK/1", "TK/2", "TK/3", "K/0", "K/1", "K/2", "K/3":
		return true
	}
	return false
}
