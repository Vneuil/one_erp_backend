package application

import (
	"testing"
	"time"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	"github.com/google/uuid"
)

func TestBuildHRDashboard(t *testing.T) {
	today := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	mk := func(nip, dept, status, join string) hrmdomain.Employee {
		e := hrmdomain.Employee{NIP: nip, Name: nip, Department: dept, Status: status, ContractType: "PKWTT Tetap", JoinDate: join}
		e.ID = uuid.New()
		return e
	}
	a, b, c, gone := mk("A", "Ops", "Active", "2026-10-01"), mk("B", "Ops", "Active", "2024-01-01"), mk("C", "Sales", "Active", "2024-01-01"), mk("D", "Ops", "Resigned", "2022-01-01")
	att := []hrmdomain.Attendance{
		{NIP: "A", Date: "2026-10-14", Status: "Late"},
		{NIP: "A", Date: "2026-10-14", Status: "On Time"}, // a second record the same day counts once
		{NIP: "B", Date: "2026-10-14", Status: "On Time"},
		{NIP: "B", Date: "2026-10-13", Status: "Late"},
		{NIP: "D", Date: "2026-10-14", Status: "On Time"}, // resigned: ignored for presence
	}
	soon := domain.EmployeeDocument{EmployeeID: a.ID, Title: "Kontrak", DocType: "Kontrak", ExpiresOn: "2026-10-20"}
	expired := domain.EmployeeDocument{EmployeeID: b.ID, Title: "SIM", DocType: "Lain", ExpiresOn: "2026-10-01"}
	far := domain.EmployeeDocument{EmployeeID: c.ID, Title: "KTP", DocType: "KTP", ExpiresOn: "2027-10-01"}

	d := BuildHRDashboard(DashboardInput{Today: today, Employees: []hrmdomain.Employee{a, b, c, gone}, Attendance: att, OvertimeMin: 150,
		Docs: []domain.EmployeeDocument{soon, expired, far}, PendingLeaves: 2, PendingCorr: 1, PendingAdv: 1})

	if d.Headcount != 3 || d.NewJoiners30d != 1 {
		t.Errorf("headcount=%d joiners=%d", d.Headcount, d.NewJoiners30d)
	}
	if d.PresentToday != 2 || d.LateToday != 1 || d.NotClockedInYet != 1 || d.LateIncidentsMTD != 2 {
		t.Errorf("attendance: %+v", d)
	}
	if d.OvertimeHoursMTD != 2.5 {
		t.Errorf("overtime = %v", d.OvertimeHoursMTD)
	}
	if d.Pending.Total != 4 {
		t.Errorf("pending = %+v", d.Pending)
	}
	if len(d.ExpiringDocuments) != 2 || d.ExpiringDocuments[0].Title != "SIM" || d.ExpiringDocuments[0].DaysLeft != -13 || d.ExpiringDocuments[1].DaysLeft != 6 {
		t.Errorf("expiring = %+v", d.ExpiringDocuments)
	}
	if d.ByDepartment[0].Name != "Ops" || d.ByDepartment[0].Count != 2 {
		t.Errorf("departments = %+v", d.ByDepartment)
	}
}
