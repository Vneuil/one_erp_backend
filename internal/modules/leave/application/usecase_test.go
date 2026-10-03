package application

import (
	"context"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"testing"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/leave/domain"
	"github.com/google/uuid"
)

type fakeLeaveRepo struct {
	domain.LeaveRepository
	items []*domain.LeaveRequest
}

func (f *fakeLeaveRepo) Create(_ context.Context, r *domain.LeaveRequest) error {
	r.ID = uuid.New()
	f.items = append(f.items, r)
	return nil
}
func (f *fakeLeaveRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.LeaveRequest, error) {
	for _, r := range f.items {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, nil
}
func (f *fakeLeaveRepo) Update(context.Context, *domain.LeaveRequest) error { return nil }
func (f *fakeLeaveRepo) FindOverlapping(_ context.Context, emp uuid.UUID, start, end string) ([]domain.LeaveRequest, error) {
	var out []domain.LeaveRequest
	for _, r := range f.items {
		if r.EmployeeID == emp && (r.Status == "pending" || r.Status == "approved") && r.StartDate <= end && r.EndDate >= start {
			out = append(out, *r)
		}
	}
	return out, nil
}

type fakeHRM struct {
	hrmdomain.HRMRepository
	emp *hrmdomain.Employee
}

func (f *fakeHRM) GetEmployeeByID(context.Context, uuid.UUID) (*hrmdomain.Employee, error) {
	return f.emp, nil
}
func (f *fakeHRM) UpdateEmployee(context.Context, *hrmdomain.Employee) error { return nil }

func setup(quota, used int) (*fakeLeaveRepo, *hrmdomain.Employee, LeaveUseCase) {
	emp := &hrmdomain.Employee{Name: "Ani", Department: "Ops", LeaveQuotaDays: quota, LeaveUsedDays: used}
	emp.ID = uuid.New()
	repo := &fakeLeaveRepo{}
	return repo, emp, NewLeaveUseCase(repo, &fakeHRM{emp: emp})
}

func TestCreateValidatesDatesAndDefaultsTotalDays(t *testing.T) {
	_, emp, uc := setup(12, 0)
	ctx := context.Background()
	base := CreateLeaveRequestDTO{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "2026-10-05", EndDate: "2026-10-07"}

	res, err := uc.CreateLeaveRequest(ctx, base)
	if err != nil || res.TotalDays != 3 {
		t.Fatalf("expected TotalDays defaulted to 3, got %+v err=%v", res, err)
	}

	bad := []CreateLeaveRequestDTO{
		{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "2026-11-10", EndDate: "2026-11-01"},               // end before start
		{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "10/11/2026", EndDate: "12/11/2026"},               // bad format
		{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "2026-11-01", EndDate: "2026-11-02", TotalDays: 5}, // more days than range
	}
	for i, dto := range bad {
		if _, err := uc.CreateLeaveRequest(ctx, dto); err == nil {
			t.Errorf("bad case %d should have been rejected", i)
		}
	}
}

func TestCreateRejectsOverlap(t *testing.T) {
	_, emp, uc := setup(12, 0)
	ctx := context.Background()
	if _, err := uc.CreateLeaveRequest(ctx, CreateLeaveRequestDTO{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "2026-10-05", EndDate: "2026-10-07"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CreateLeaveRequest(ctx, CreateLeaveRequestDTO{EmployeeID: emp.ID, Type: "Cuti Sakit", StartDate: "2026-10-07", EndDate: "2026-10-09"}); err == nil {
		t.Fatal("overlapping request should be rejected")
	}
	if _, err := uc.CreateLeaveRequest(ctx, CreateLeaveRequestDTO{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "2026-10-08", EndDate: "2026-10-09"}); err != nil {
		t.Fatalf("adjacent, non-overlapping request should be accepted: %v", err)
	}
}

func TestOnlyAnnualLeaveConsumesQuota(t *testing.T) {
	_, emp, uc := setup(12, 11) // 1 day left
	ctx := context.Background()

	sick, _ := uc.CreateLeaveRequest(ctx, CreateLeaveRequestDTO{EmployeeID: emp.ID, Type: "Cuti Sakit", StartDate: "2026-10-05", EndDate: "2026-10-07"})
	if _, err := uc.ApproveLeaveRequest(ctx, sick.ID, ApproveLeaveRequestDTO{ApprovedBy: "HR"}); err != nil {
		t.Fatalf("sick leave must not need annual quota: %v", err)
	}
	if emp.LeaveUsedDays != 11 {
		t.Fatalf("sick leave changed annual usage to %d", emp.LeaveUsedDays)
	}

	annual, _ := uc.CreateLeaveRequest(ctx, CreateLeaveRequestDTO{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "2026-11-02", EndDate: "2026-11-04"})
	if _, err := uc.ApproveLeaveRequest(ctx, annual.ID, ApproveLeaveRequestDTO{ApprovedBy: "HR"}); err == nil {
		t.Fatal("3 annual days with 1 remaining must be rejected")
	}
}

func TestRejectOnlyPending(t *testing.T) {
	_, emp, uc := setup(12, 0)
	ctx := context.Background()
	req, _ := uc.CreateLeaveRequest(ctx, CreateLeaveRequestDTO{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "2026-10-05", EndDate: "2026-10-05"})
	if _, err := uc.ApproveLeaveRequest(ctx, req.ID, ApproveLeaveRequestDTO{ApprovedBy: "HR"}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.RejectLeaveRequest(ctx, req.ID, RejectLeaveRequestDTO{ApprovedBy: "HR"}); err == nil {
		t.Fatal("an approved request must not be rejectable (its quota would never be restored)")
	}
}

func TestCannotApproveOwnLeave(t *testing.T) {
	_, emp, uc := setup(12, 0)
	emp.Email = "ani@x.com"
	req, err := uc.CreateLeaveRequest(context.Background(), CreateLeaveRequestDTO{EmployeeID: emp.ID, Type: "Cuti Tahunan", StartDate: "2026-10-05", EndDate: "2026-10-05"})
	if err != nil {
		t.Fatal(err)
	}

	own := actor.WithEmail(context.Background(), "ANI@x.com")
	if _, err := uc.ApproveLeaveRequest(own, req.ID, ApproveLeaveRequestDTO{ApprovedBy: "Ani"}); err == nil {
		t.Fatal("an employee must not approve their own leave")
	}
	// The request must still be pending and the quota untouched.
	if emp.LeaveUsedDays != 0 {
		t.Fatalf("quota changed to %d by a blocked approval", emp.LeaveUsedDays)
	}

	other := actor.WithEmail(context.Background(), "hr@x.com")
	if _, err := uc.ApproveLeaveRequest(other, req.ID, ApproveLeaveRequestDTO{ApprovedBy: "HR"}); err != nil {
		t.Fatalf("another approver must be able to approve: %v", err)
	}
}
