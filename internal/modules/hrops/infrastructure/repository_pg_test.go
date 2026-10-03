package infrastructure

import (
	"context"
	"os"
	"testing"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	hrminfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when HROPS_TEST_DSN is set (CI/dev: point it at a
// scratch database). The GORM column-naming quirks (NIP -> n_ip) and upsert
// conflicts can only be checked against the real thing.
func TestRepositoryAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("HROPS_TEST_DSN")
	if dsn == "" {
		t.Skip("HROPS_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&hrmdomain.Employee{}, &hrmdomain.Attendance{}, &domain.Shift{}, &domain.ShiftAssignment{},
		&domain.OvertimeRecord{}, &domain.Feedback{}, &domain.Notification{}, &domain.Announcement{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewRepository(db)

	// Shift assignment upsert: same employee+date twice leaves one row with the latest shift.
	s1, s2 := &domain.Shift{Name: "Pagi", StartTime: "07:00", EndTime: "15:00", IsActive: true}, &domain.Shift{Name: "Malam", StartTime: "22:00", EndTime: "06:00", IsActive: true}
	if err := repo.CreateShift(ctx, s1); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateShift(ctx, s2); err != nil {
		t.Fatal(err)
	}
	for _, sh := range []uuid.UUID{s1.ID, s2.ID} {
		if err := repo.UpsertAssignment(ctx, &domain.ShiftAssignment{NIP: "E1", Date: "2026-09-10", ShiftID: sh}); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}
	list, err := repo.ListAssignments(ctx, "2026-09", "E1")
	if err != nil || len(list) != 1 || list[0].ShiftID != s2.ID {
		t.Fatalf("assignments: %+v %v", list, err)
	}

	// Approved overtime feeds payroll.
	for i, st := range []string{"approved", "pending", "approved"} {
		if err := repo.CreateOvertime(ctx, &domain.OvertimeRecord{NIP: "E1", Date: "2026-09-0" + string(rune('1'+i)), Minutes: 60 * (i + 1), Source: "manual", Status: st}); err != nil {
			t.Fatal(err)
		}
	}
	mins, err := NewOvertimeSource(db).ApprovedOvertimeMinutes(ctx, "E1", "2026-09")
	if err != nil || len(mins) != 2 {
		t.Fatalf("approved minutes: %v %v", mins, err)
	}
	if ok, _ := repo.OvertimeExists(ctx, "E1", "2026-09-01", "manual"); !ok {
		t.Fatal("OvertimeExists should find the record")
	}

	// Employee lookup by email is case-insensitive; attendance by period filters by month and NIP.
	hrm := hrminfra.NewHRMRepository(db)
	if err := hrm.CreateEmployee(ctx, &hrmdomain.Employee{NIP: "E1", Name: "Ani", Email: "Ani@X.com", Department: "Ops", Role: "Staff"}); err != nil {
		t.Fatal(err)
	}
	if e, err := hrm.FindEmployeeByEmail(ctx, "ani@x.COM"); err != nil || e == nil || e.NIP != "E1" {
		t.Fatalf("find by email: %v %v", e, err)
	}
	for _, a := range []hrmdomain.Attendance{{NIP: "E1", Date: "2026-09-02", EmployeeName: "Ani"}, {NIP: "E1", Date: "2026-10-01", EmployeeName: "Ani"}, {NIP: "E2", Date: "2026-09-02", EmployeeName: "Bo"}} {
		a := a
		if err := hrm.RecordAttendance(ctx, &a); err != nil {
			t.Fatal(err)
		}
	}
	if recs, err := hrm.ListAttendanceByNIPAndPeriod(ctx, "E1", "2026-09"); err != nil || len(recs) != 1 {
		t.Fatalf("by nip+period: %v %v", recs, err)
	}
	if recs, _ := hrm.ListAttendanceByNIPAndPeriod(ctx, "", "2026-09"); len(recs) != 2 {
		t.Fatalf("all in period: %d", len(recs))
	}

	// Feedback uniqueness check and notifications inbox.
	subject := uuid.New()
	_ = repo.CreateFeedback(ctx, &domain.Feedback{SubjectID: subject, ReviewerEmail: "Rev@x.com", Relationship: "peer", Period: "2026"})
	if ok, _ := repo.FeedbackExists(ctx, subject, "rev@X.com", "2026"); !ok {
		t.Fatal("feedback duplicate check must be case-insensitive")
	}
	_ = repo.CreateNotifications(ctx, []domain.Notification{{RecipientEmail: "Ani@x.com", Title: "a"}, {RecipientEmail: "ani@x.com", Title: "b"}, {RecipientEmail: "bo@x.com", Title: "c"}})
	if n, _ := repo.CountUnread(ctx, "ANI@x.com"); n != 2 {
		t.Fatalf("unread = %d", n)
	}
	items, _ := repo.ListNotifications(ctx, "ani@x.com", true)
	if ok, _ := repo.MarkNotificationRead(ctx, "bo@x.com", items[0].ID); ok {
		t.Fatal("must not mark someone else's notification")
	}
	if ok, _ := repo.MarkNotificationRead(ctx, "ani@x.com", items[0].ID); !ok {
		t.Fatal("own notification should be marked")
	}
	if n, _ := repo.CountUnread(ctx, "ani@x.com"); n != 1 {
		t.Fatalf("unread after one read = %d", n)
	}
}

func TestExtrasAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("HROPS_TEST_DSN")
	if dsn == "" {
		t.Skip("HROPS_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.ReimbursementEvidence{}, &domain.CanteenItem{}, &domain.CanteenOrder{}, &domain.VisitStop{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewRepository(db)
	src := NewOvertimeSource(db)

	item := &domain.CanteenItem{Name: "Nasi", Price: 20_000, IsActive: true}
	if err := repo.CreateCanteenItem(ctx, item); err != nil {
		t.Fatal(err)
	}
	mk := func(nip, name, date, status string, amt float64) {
		if err := repo.CreateCanteenOrder(ctx, &domain.CanteenOrder{NIP: nip, EmployeeName: name, ItemID: item.ID, ItemName: "Nasi", Quantity: 1, UnitPrice: amt, Amount: amt, Date: date, Status: status}); err != nil {
			t.Fatal(err)
		}
	}
	mk("E1", "Ani", "2026-09-02", "ordered", 20_000)
	mk("E1", "Ani", "2026-09-03", "ordered", 25_000)
	mk("E1", "Ani", "2026-09-04", "cancelled", 99_000) // cancelled orders never count
	mk("E1", "Ani", "2026-10-01", "ordered", 7_000)    // another month
	mk("E2", "Bob", "2026-09-02", "ordered", 20_000)

	if amt, err := src.CanteenAmountFor(ctx, "E1", "2026-09"); err != nil || amt != 45_000 {
		t.Fatalf("payroll amount for E1: %v %v", amt, err)
	}
	totals, err := repo.CanteenTotals(ctx, "2026-09")
	if err != nil || len(totals) != 2 || totals[0].NIP != "E1" || totals[0].Amount != 45_000 || totals[0].Orders != 2 || totals[0].EmployeeName != "Ani" {
		t.Fatalf("totals: %+v %v", totals, err)
	}
	if list, _ := repo.ListCanteenOrders(ctx, "E1", "2026-09"); len(list) != 3 {
		t.Fatalf("E1 September orders: %d", len(list))
	}

	// Visits list in sequence order per employee/date, and updates persist atomically.
	stops := []domain.VisitStop{
		{NIP: "E1", Date: "2026-09-30", Seq: 2, CustomerName: "B", Latitude: -6.2, Longitude: 106.8, RadiusMeters: 100, Status: "planned"},
		{NIP: "E1", Date: "2026-09-30", Seq: 1, CustomerName: "A", Latitude: -6.3, Longitude: 106.9, RadiusMeters: 100, Status: "planned"},
		{NIP: "E2", Date: "2026-09-30", Seq: 1, CustomerName: "C", Latitude: -6.4, Longitude: 106.7, RadiusMeters: 100, Status: "planned"},
	}
	if err := repo.CreateStops(ctx, stops); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ListStops(ctx, "E1", "2026-09-30")
	if err != nil || len(got) != 2 || got[0].CustomerName != "A" || got[1].CustomerName != "B" {
		t.Fatalf("stop order: %+v %v", got, err)
	}
	if all, _ := repo.ListStops(ctx, "", "2026-09-30"); len(all) != 3 {
		t.Fatalf("all employees: %d", len(all))
	}
	got[0].Status, got[1].Seq = "visited", 1
	got[0].Seq = 2
	if err := repo.UpdateStops(ctx, got); err != nil {
		t.Fatal(err)
	}
	again, _ := repo.ListStops(ctx, "E1", "2026-09-30")
	if again[0].CustomerName != "B" || again[1].Status != "visited" {
		t.Fatalf("after update: %+v", again)
	}
}
