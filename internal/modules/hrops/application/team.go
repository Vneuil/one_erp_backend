package application

import (
	"context"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

// TeamLeave is a pending leave request as a line manager sees it.
type TeamLeave struct {
	ID           uuid.UUID `json:"id"`
	EmployeeID   uuid.UUID `json:"employeeId"`
	EmployeeName string    `json:"employeeName"`
	Type         string    `json:"type"`
	StartDate    string    `json:"startDate"`
	EndDate      string    `json:"endDate"`
	TotalDays    int       `json:"totalDays"`
	Reason       string    `json:"reason"`
}

// LeaveBridge lets the manager flow reach leave requests without this package
// depending on the leave module.
type LeaveBridge interface {
	Pending(ctx context.Context) ([]TeamLeave, error)
	Owner(ctx context.Context, id uuid.UUID) (uuid.UUID, error)
	Decide(ctx context.Context, id uuid.UUID, approve bool, by string) error
}

// WithLeave enables leave requests in the manager flow.
func (s *Service) WithLeave(l LeaveBridge) *Service {
	s.leaves = l
	return s
}

// Descendants returns everyone who reports to rootID, directly or through
// other managers. The root is not included, and cycles cannot loop.
func Descendants(emps []hrmdomain.Employee, rootID uuid.UUID) map[uuid.UUID]hrmdomain.Employee {
	children := map[uuid.UUID][]hrmdomain.Employee{}
	for _, e := range emps {
		if e.ManagerID != nil {
			children[*e.ManagerID] = append(children[*e.ManagerID], e)
		}
	}
	out := map[uuid.UUID]hrmdomain.Employee{}
	queue := []uuid.UUID{rootID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range children[cur] {
			if _, seen := out[c.ID]; seen || c.ID == rootID {
				continue
			}
			out[c.ID] = c
			queue = append(queue, c.ID)
		}
	}
	return out
}

// team is the set of people the caller manages, by id and by NIP.
type team struct {
	byID  map[uuid.UUID]hrmdomain.Employee
	byNIP map[string]hrmdomain.Employee
}

func (s *Service) teamOf(ctx context.Context, c Caller) (*team, error) {
	own, err := s.hrm.FindEmployeeByEmail(ctx, c.Email)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to look up your employee record")
	}
	t := &team{byID: map[uuid.UUID]hrmdomain.Employee{}, byNIP: map[string]hrmdomain.Employee{}}
	if own == nil {
		return t, nil
	}
	all, _, err := s.hrm.ListEmployees(ctx, types.PaginationQuery{Page: 1, PerPage: 5000})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load the organization")
	}
	t.byID = Descendants(all, own.ID)
	for _, e := range t.byID {
		t.byNIP[e.NIP] = e
	}
	return t, nil
}

// TeamRequests are the pending requests of the caller's reports.
type TeamRequests struct {
	Leaves       []TeamLeave                   `json:"leaves"`
	Corrections  []domain.AttendanceCorrection `json:"corrections"`
	Overtime     []domain.OvertimeRecord       `json:"overtime"`
	ShiftChanges []domain.ShiftChangeRequest   `json:"shiftChanges"`
	Advances     []AdvanceView                 `json:"advances"`
	Total        int                           `json:"total"`
}

func (s *Service) TeamRequests(ctx context.Context, c Caller) (*TeamRequests, error) {
	t, err := s.teamOf(ctx, c)
	if err != nil {
		return nil, err
	}
	out := &TeamRequests{Leaves: []TeamLeave{}, Corrections: []domain.AttendanceCorrection{}, Overtime: []domain.OvertimeRecord{},
		ShiftChanges: []domain.ShiftChangeRequest{}, Advances: []AdvanceView{}}
	if len(t.byID) == 0 {
		return out, nil
	}
	if s.leaves != nil {
		list, err := s.leaves.Pending(ctx)
		if err != nil {
			return nil, err
		}
		for _, l := range list {
			if _, ok := t.byID[l.EmployeeID]; ok {
				out.Leaves = append(out.Leaves, l)
			}
		}
	}
	corr, err := s.repo.ListCorrections(ctx, "pending", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list corrections")
	}
	for _, v := range corr {
		if _, ok := t.byNIP[v.NIP]; ok {
			out.Corrections = append(out.Corrections, v)
		}
	}
	ot, err := s.repo.ListOvertime(ctx, "", "pending", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list overtime")
	}
	for _, v := range ot {
		if _, ok := t.byNIP[v.NIP]; ok {
			out.Overtime = append(out.Overtime, v)
		}
	}
	sc, err := s.repo.ListShiftChanges(ctx, "pending", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to list shift changes")
	}
	for _, v := range sc {
		if _, ok := t.byNIP[v.NIP]; ok {
			out.ShiftChanges = append(out.ShiftChanges, v)
		}
	}
	adv, err := s.ListAdvances(ctx, "", "pending")
	if err != nil {
		return nil, err
	}
	for _, v := range adv {
		if _, ok := t.byNIP[v.NIP]; ok {
			out.Advances = append(out.Advances, v)
		}
	}
	out.Total = len(out.Leaves) + len(out.Corrections) + len(out.Overtime) + len(out.ShiftChanges) + len(out.Advances)
	return out, nil
}

// DecideAsManager approves or rejects a request on behalf of the requester's
// line manager (or anyone above them). It needs no HR approval right, but the
// request must belong to someone in the caller's own reporting line, and the
// usual rule against approving your own request still applies.
func (s *Service) DecideAsManager(ctx context.Context, c Caller, kind string, id uuid.UUID, approve bool) error {
	t, err := s.teamOf(ctx, c)
	if err != nil {
		return err
	}
	deny := apperrors.NewForbidden("Only the requester's line manager can decide this request here")
	inTeamNIP := func(nip string) bool { _, ok := t.byNIP[nip]; return ok }
	switch kind {
	case "leave":
		if s.leaves == nil {
			return apperrors.NewNotFound("Leave requests are not available")
		}
		owner, err := s.leaves.Owner(ctx, id)
		if err != nil {
			return err
		}
		if _, ok := t.byID[owner]; !ok {
			return deny
		}
		return s.leaves.Decide(ctx, id, approve, c.Email)
	case "correction":
		v, err := s.repo.GetCorrection(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load correction")
		}
		if v == nil {
			return apperrors.NewNotFound("Correction request not found")
		}
		if !inTeamNIP(v.NIP) {
			return deny
		}
		_, err = s.DecideCorrection(ctx, id, approve)
		return err
	case "overtime":
		v, err := s.repo.GetOvertime(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load overtime")
		}
		if v == nil {
			return apperrors.NewNotFound("Overtime record not found")
		}
		if !inTeamNIP(v.NIP) {
			return deny
		}
		_, err = s.DecideOvertime(ctx, id, approve)
		return err
	case "shift-change":
		v, err := s.repo.GetShiftChange(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load shift change")
		}
		if v == nil {
			return apperrors.NewNotFound("Shift change request not found")
		}
		if !inTeamNIP(v.NIP) {
			return deny
		}
		_, err = s.DecideShiftChange(ctx, id, approve)
		return err
	case "cash-advance":
		v, err := s.repo.GetAdvance(ctx, id)
		if err != nil {
			return apperrors.NewInternal(err, "Failed to load cash advance")
		}
		if v == nil {
			return apperrors.NewNotFound("Cash advance not found")
		}
		if !inTeamNIP(v.NIP) {
			return deny
		}
		_, err = s.DecideAdvance(ctx, id, approve)
		return err
	}
	return apperrors.NewBadRequest("Unknown request type")
}
