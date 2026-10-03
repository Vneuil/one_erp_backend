package application

import (
	"context"
	"strconv"
	"time"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/leave/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/inbox"
	"github.com/divinecoid/one-backend/internal/shared/sod"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

const (
	dateLayout = "2006-01-02"
	// annualLeaveType is the only leave type that consumes the employee's
	// annual leave quota (Employee.LeaveQuotaDays); sick leave and special
	// permission do not.
	annualLeaveType = "Cuti Tahunan"
)

type LeaveUseCase interface {
	CreateLeaveRequest(ctx context.Context, dto CreateLeaveRequestDTO) (*LeaveRequestResponseDTO, error)
	ListLeaveRequests(ctx context.Context, query types.PaginationQuery) ([]LeaveRequestResponseDTO, types.PaginationMeta, error)
	ApproveLeaveRequest(ctx context.Context, id uuid.UUID, dto ApproveLeaveRequestDTO) (*LeaveRequestResponseDTO, error)
	RejectLeaveRequest(ctx context.Context, id uuid.UUID, dto RejectLeaveRequestDTO) (*LeaveRequestResponseDTO, error)

	SeedInitialData(ctx context.Context) error
}

type leaveUseCase struct {
	repo    domain.LeaveRepository
	hrmRepo hrmdomain.HRMRepository
	inbox   inbox.Sender // optional
}

// NewLeaveUseCase wires the leave repository together with the HRM
// repository, so leave requests reference real employees and approvals can
// be checked against - and deducted from - that employee's leave balance.
func NewLeaveUseCase(repo domain.LeaveRepository, hrmRepo hrmdomain.HRMRepository, opts ...Option) LeaveUseCase {
	uc := &leaveUseCase{repo: repo, hrmRepo: hrmRepo}
	for _, o := range opts {
		o(uc)
	}
	return uc
}

// Option customises optional collaborators of the leave use case.
type Option func(*leaveUseCase)

// WithInbox sends notifications: to the manager when a request is filed, and
// to the employee when it is decided.
func WithInbox(s inbox.Sender) Option { return func(uc *leaveUseCase) { uc.inbox = s } }

// notifyManager tells the employee's direct manager a request is waiting.
func (uc *leaveUseCase) notifyManager(ctx context.Context, emp *hrmdomain.Employee, what string) {
	if uc.inbox == nil || emp == nil || emp.ManagerID == nil {
		return
	}
	if mgr, err := uc.hrmRepo.GetEmployeeByID(ctx, *emp.ManagerID); err == nil && mgr != nil {
		uc.inbox(ctx, mgr.Email, "Cuti baru", emp.Name+" mengajukan "+what, "/hrm/team-approvals")
	}
}

func (uc *leaveUseCase) notifyDecision(ctx context.Context, employeeID uuid.UUID, verdict string) {
	if uc.inbox == nil || employeeID == uuid.Nil {
		return
	}
	if emp, err := uc.hrmRepo.GetEmployeeByID(ctx, employeeID); err == nil && emp != nil {
		uc.inbox(ctx, emp.Email, "Cuti "+verdict, "Pengajuan cuti Anda telah "+verdict+".", "/hrm/leaves")
	}
}

func (uc *leaveUseCase) CreateLeaveRequest(ctx context.Context, dto CreateLeaveRequestDTO) (*LeaveRequestResponseDTO, error) {
	if dto.EmployeeID == uuid.Nil {
		return nil, apperrors.NewBadRequest("Employee is required")
	}
	if dto.StartDate == "" || dto.EndDate == "" {
		return nil, apperrors.NewBadRequest("Start Date and End Date are required")
	}
	leaveType := dto.Type
	if leaveType == "" {
		return nil, apperrors.NewBadRequest("Leave Type is required")
	}
	start, err := time.Parse(dateLayout, dto.StartDate)
	if err != nil {
		return nil, apperrors.NewBadRequest("Start Date must be in YYYY-MM-DD format")
	}
	end, err := time.Parse(dateLayout, dto.EndDate)
	if err != nil {
		return nil, apperrors.NewBadRequest("End Date must be in YYYY-MM-DD format")
	}
	if end.Before(start) {
		return nil, apperrors.NewBadRequest("End Date cannot be before Start Date")
	}
	// TotalDays is optional: it defaults to the inclusive calendar span. A
	// smaller explicit value is allowed (weekends/holidays excluded); a larger
	// one is not.
	calendarDays := int(end.Sub(start).Hours()/24) + 1
	totalDays := dto.TotalDays
	if totalDays < 0 {
		return nil, apperrors.NewBadRequest("Total Days cannot be negative")
	}
	if totalDays == 0 {
		totalDays = calendarDays
	}
	if totalDays > calendarDays {
		return nil, apperrors.NewBadRequest("Total Days (" + strconv.Itoa(totalDays) + ") exceeds the selected date range (" + strconv.Itoa(calendarDays) + " day(s))")
	}
	dto.TotalDays = totalDays

	employee, err := uc.hrmRepo.GetEmployeeByID(ctx, dto.EmployeeID)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get employee")
	}
	if employee == nil {
		return nil, apperrors.NewNotFound("Employee not found")
	}

	overlapping, err := uc.repo.FindOverlapping(ctx, employee.ID, dto.StartDate, dto.EndDate)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to check overlapping leave requests")
	}
	if len(overlapping) > 0 {
		return nil, apperrors.NewConflict("This employee already has a pending or approved leave request overlapping " +
			overlapping[0].StartDate + " to " + overlapping[0].EndDate)
	}

	req := &domain.LeaveRequest{
		EmployeeID:   employee.ID,
		EmployeeName: employee.Name,
		Department:   employee.Department,
		Type:         leaveType,
		StartDate:    dto.StartDate,
		EndDate:      dto.EndDate,
		TotalDays:    dto.TotalDays,
		Reason:       dto.Reason,
		Status:       "pending",
	}

	if err := uc.repo.Create(ctx, req); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to create leave request")
	}

	uc.notifyManager(ctx, employee, "cuti "+dto.StartDate+" s/d "+dto.EndDate)
	return ToLeaveRequestResponse(req), nil
}

func (uc *leaveUseCase) ListLeaveRequests(ctx context.Context, query types.PaginationQuery) ([]LeaveRequestResponseDTO, types.PaginationMeta, error) {
	query.SetDefaults()
	items, total, err := uc.repo.List(ctx, query)
	if err != nil {
		return nil, types.PaginationMeta{}, apperrors.NewInternal(err, "Failed to list leave requests")
	}

	meta := types.NewPaginationMeta(total, query.Page, query.PerPage)
	return ToLeaveRequestResponseList(items), meta, nil
}

func (uc *leaveUseCase) ApproveLeaveRequest(ctx context.Context, id uuid.UUID, dto ApproveLeaveRequestDTO) (*LeaveRequestResponseDTO, error) {
	req, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get leave request")
	}
	if req == nil {
		return nil, apperrors.NewNotFound("Leave request not found")
	}
	if req.Status != "pending" {
		return nil, apperrors.NewBadRequest("Only pending leave requests can be approved")
	}

	// The requester's employee record is needed for the self-approval check
	// and, for annual leave, the quota check.
	var employee *hrmdomain.Employee
	if req.EmployeeID != uuid.Nil {
		employee, err = uc.hrmRepo.GetEmployeeByID(ctx, req.EmployeeID)
		if err != nil {
			return nil, apperrors.NewInternal(err, "Failed to get employee")
		}
		if employee == nil {
			return nil, apperrors.NewNotFound("Employee not found")
		}
		if err := sod.ForbidSelfApproval(ctx, employee.Email); err != nil {
			return nil, err
		}
	}

	// Only annual leave draws down the annual quota; the balance is checked
	// before approval so a request can't be approved past the allowance.
	if req.Type == annualLeaveType && employee != nil {
		remaining := employee.LeaveQuotaDays - employee.LeaveUsedDays
		if req.TotalDays > remaining {
			return nil, apperrors.NewBadRequest("Insufficient leave balance: employee has " +
				strconv.Itoa(remaining) + " day(s) remaining but requested " + strconv.Itoa(req.TotalDays) + " day(s)")
		}
	}

	req.Status = "approved"
	req.ApprovedBy = dto.ApprovedBy
	if err := uc.repo.Update(ctx, req); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to approve leave request")
	}

	if req.Type == annualLeaveType && employee != nil {
		employee.LeaveUsedDays += req.TotalDays
		if err := uc.hrmRepo.UpdateEmployee(ctx, employee); err != nil {
			// Roll the request back so the quota and the request never disagree.
			req.Status = "pending"
			req.ApprovedBy = ""
			_ = uc.repo.Update(ctx, req)
			return nil, apperrors.NewInternal(err, "Failed to update employee leave balance")
		}
	}

	uc.notifyDecision(ctx, req.EmployeeID, "disetujui")
	return ToLeaveRequestResponse(req), nil
}

func (uc *leaveUseCase) RejectLeaveRequest(ctx context.Context, id uuid.UUID, dto RejectLeaveRequestDTO) (*LeaveRequestResponseDTO, error) {
	req, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to get leave request")
	}
	if req == nil {
		return nil, apperrors.NewNotFound("Leave request not found")
	}

	if req.Status != "pending" {
		return nil, apperrors.NewBadRequest("Only pending leave requests can be rejected")
	}

	req.Status = "rejected"
	req.ApprovedBy = dto.ApprovedBy

	if err := uc.repo.Update(ctx, req); err != nil {
		return nil, apperrors.NewInternal(err, "Failed to reject leave request")
	}

	uc.notifyDecision(ctx, req.EmployeeID, "ditolak")
	return ToLeaveRequestResponse(req), nil
}

func (uc *leaveUseCase) SeedInitialData(ctx context.Context) error {
	// No seed data required for the leave module.
	return nil
}
