package application

import (
	"context"
	"sort"
	"time"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	apperrors "github.com/divinecoid/one-backend/internal/shared/errors"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

type CountRow struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ExpiringDoc struct {
	DocumentID   uuid.UUID `json:"documentId"`
	EmployeeID   uuid.UUID `json:"employeeId"`
	EmployeeName string    `json:"employeeName"`
	Title        string    `json:"title"`
	DocType      string    `json:"docType"`
	ExpiresOn    string    `json:"expiresOn"`
	DaysLeft     int       `json:"daysLeft"` // negative once expired
}

// HRDashboard is the executive picture of the workforce today.
type HRDashboard struct {
	AsOf string `json:"asOf"`

	Headcount     int        `json:"headcount"` // active people
	ByStatus      []CountRow `json:"byStatus"`
	ByDepartment  []CountRow `json:"byDepartment"`
	ByContract    []CountRow `json:"byContract"`
	NewJoiners30d int        `json:"newJoiners30d"`

	PresentToday     int `json:"presentToday"`
	LateToday        int `json:"lateToday"`
	NotClockedInYet  int `json:"notClockedInYet"`
	LateIncidentsMTD int `json:"lateIncidentsMonth"`

	OvertimeHoursMTD float64 `json:"overtimeHoursMonth"`

	Pending struct {
		Leaves       int `json:"leaves"`
		Corrections  int `json:"corrections"`
		Overtime     int `json:"overtime"`
		ShiftChanges int `json:"shiftChanges"`
		Advances     int `json:"advances"`
		Total        int `json:"total"`
	} `json:"pending"`

	ExpiringDocuments []ExpiringDoc `json:"expiringDocuments"`
}

// DashboardInput is everything BuildHRDashboard needs, already loaded.
type DashboardInput struct {
	Today         time.Time
	Employees     []hrmdomain.Employee
	Attendance    []hrmdomain.Attendance // the current month
	OvertimeMin   int                    // approved overtime minutes this month
	Docs          []domain.EmployeeDocument
	PendingLeaves int
	PendingCorr   int
	PendingOT     int
	PendingShift  int
	PendingAdv    int
}

func sortedRows(m map[string]int) []CountRow {
	out := make([]CountRow, 0, len(m))
	for k, v := range m {
		out = append(out, CountRow{Name: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// BuildHRDashboard is pure: all the arithmetic is testable without a database.
func BuildHRDashboard(in DashboardInput) HRDashboard {
	today := in.Today.Format("2006-01-02")
	d := HRDashboard{AsOf: today}
	status, dept, contract := map[string]int{}, map[string]int{}, map[string]int{}
	byID := map[uuid.UUID]hrmdomain.Employee{}
	active := map[string]bool{} // NIP of active employees
	cutoff := in.Today.AddDate(0, 0, -30).Format("2006-01-02")
	for _, e := range in.Employees {
		byID[e.ID] = e
		status[e.Status]++
		if e.Status == "Resigned" {
			continue
		}
		d.Headcount++
		active[e.NIP] = true
		dept[e.Department]++
		contract[e.ContractType]++
		if e.JoinDate >= cutoff && e.JoinDate <= today {
			d.NewJoiners30d++
		}
	}
	d.ByStatus, d.ByDepartment, d.ByContract = sortedRows(status), sortedRows(dept), sortedRows(contract)

	seen := map[string]bool{}
	for _, a := range in.Attendance {
		if a.Status == "Late" {
			d.LateIncidentsMTD++
		}
		if a.Date == today && active[a.NIP] && !seen[a.NIP] {
			seen[a.NIP] = true
			d.PresentToday++
			if a.Status == "Late" {
				d.LateToday++
			}
		}
	}
	d.NotClockedInYet = d.Headcount - d.PresentToday
	if d.NotClockedInYet < 0 {
		d.NotClockedInYet = 0
	}
	d.OvertimeHoursMTD = float64(in.OvertimeMin) / 60
	d.OvertimeHoursMTD = float64(int(d.OvertimeHoursMTD*100+0.5)) / 100

	d.Pending.Leaves, d.Pending.Corrections, d.Pending.Overtime = in.PendingLeaves, in.PendingCorr, in.PendingOT
	d.Pending.ShiftChanges, d.Pending.Advances = in.PendingShift, in.PendingAdv
	d.Pending.Total = in.PendingLeaves + in.PendingCorr + in.PendingOT + in.PendingShift + in.PendingAdv

	d.ExpiringDocuments = []ExpiringDoc{}
	for _, doc := range in.Docs {
		exp, err := time.ParseInLocation("2006-01-02", doc.ExpiresOn, in.Today.Location())
		if err != nil {
			continue
		}
		base, _ := time.ParseInLocation("2006-01-02", today, in.Today.Location())
		left := int(exp.Sub(base).Hours() / 24)
		if left > 30 {
			continue
		}
		e := byID[doc.EmployeeID]
		d.ExpiringDocuments = append(d.ExpiringDocuments, ExpiringDoc{DocumentID: doc.ID, EmployeeID: doc.EmployeeID, EmployeeName: e.Name,
			Title: doc.Title, DocType: doc.DocType, ExpiresOn: doc.ExpiresOn, DaysLeft: left})
	}
	sort.SliceStable(d.ExpiringDocuments, func(i, j int) bool { return d.ExpiringDocuments[i].DaysLeft < d.ExpiringDocuments[j].DaysLeft })
	if len(d.ExpiringDocuments) > 20 {
		d.ExpiringDocuments = d.ExpiringDocuments[:20]
	}
	return d
}

// Dashboard gathers the figures for HR and managers.
func (s *Service) Dashboard(ctx context.Context, c Caller) (*HRDashboard, error) {
	if !c.Privileged {
		return nil, apperrors.NewForbidden("The HR dashboard is for HR and managers")
	}
	now := s.now().In(zone)
	period := now.Format("2006-01")
	emps, _, err := s.hrm.ListEmployees(ctx, types.PaginationQuery{Page: 1, PerPage: 5000})
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load employees")
	}
	att, err := s.hrm.ListAttendanceByNIPAndPeriod(ctx, "", period)
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load attendance")
	}
	ot, err := s.repo.ListOvertime(ctx, period, "approved", "")
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load overtime")
	}
	otMin := 0
	for _, r := range ot {
		otMin += r.Minutes
	}
	docs, err := s.repo.ExpiringDocuments(ctx, now.AddDate(0, 0, 30).Format("2006-01-02"))
	if err != nil {
		return nil, apperrors.NewInternal(err, "Failed to load documents")
	}
	in := DashboardInput{Today: now, Employees: emps, Attendance: att, OvertimeMin: otMin, Docs: docs}
	if s.leaves != nil {
		l, err := s.leaves.Pending(ctx)
		if err != nil {
			return nil, err
		}
		in.PendingLeaves = len(l)
	}
	count := func(n int, err error, what string) (int, error) {
		if err != nil {
			return 0, apperrors.NewInternal(err, "Failed to count "+what)
		}
		return n, nil
	}
	corr, err := s.repo.ListCorrections(ctx, "pending", "")
	if in.PendingCorr, err = count(len(corr), err, "corrections"); err != nil {
		return nil, err
	}
	pot, err := s.repo.ListOvertime(ctx, "", "pending", "")
	if in.PendingOT, err = count(len(pot), err, "overtime"); err != nil {
		return nil, err
	}
	sc, err := s.repo.ListShiftChanges(ctx, "pending", "")
	if in.PendingShift, err = count(len(sc), err, "shift changes"); err != nil {
		return nil, err
	}
	adv, err := s.repo.ListAdvances(ctx, "", "pending")
	if in.PendingAdv, err = count(len(adv), err, "cash advances"); err != nil {
		return nil, err
	}
	d := BuildHRDashboard(in)
	return &d, nil
}
