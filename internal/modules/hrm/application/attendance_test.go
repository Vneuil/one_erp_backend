package application

import (
	"context"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/google/uuid"
)

type fakeHRMRepo struct {
	domain.HRMRepository
	att  []*domain.Attendance
	locs []domain.AttendanceLocation
}

func (f *fakeHRMRepo) RecordAttendance(_ context.Context, a *domain.Attendance) error {
	f.att = append(f.att, a)
	return nil
}
func (f *fakeHRMRepo) FindAttendanceByNIPAndDate(_ context.Context, nip, date string) (*domain.Attendance, error) {
	for i := len(f.att) - 1; i >= 0; i-- {
		if f.att[i].NIP == nip && f.att[i].Date == date {
			return f.att[i], nil
		}
	}
	return nil, nil
}
func (f *fakeHRMRepo) UpdateAttendance(context.Context, *domain.Attendance) error { return nil }
func (f *fakeHRMRepo) ListAttendanceLocations(_ context.Context, _ bool) ([]domain.AttendanceLocation, error) {
	return f.locs, nil
}

func newTestUC(repo *fakeHRMRepo, at time.Time) *hrmUseCase {
	return &hrmUseCase{repo: repo, now: func() time.Time { return at }}
}

func wib(h, m int) time.Time { return time.Date(2026, 9, 29, h, m, 0, 0, attendanceZone) }

func f64(v float64) *float64 { return &v }

func TestClockInLateAndDuplicate(t *testing.T) {
	repo := &fakeHRMRepo{}
	ctx := context.Background()

	res, err := newTestUC(repo, wib(8, 55)).RecordClockIn(ctx, ClockInDTO{NIP: "E1", EmployeeName: "A"})
	if err != nil || res.Status != "On Time" || res.Location != "Unspecified" {
		t.Fatalf("on-time clock-in: %+v err=%v", res, err)
	}
	if _, err := newTestUC(repo, wib(9, 30)).RecordClockIn(ctx, ClockInDTO{NIP: "E1", EmployeeName: "A"}); err == nil {
		t.Fatal("expected duplicate clock-in to be rejected")
	}
	res, err = newTestUC(repo, wib(9, 0)).RecordClockIn(ctx, ClockInDTO{NIP: "E2", EmployeeName: "B"})
	if err != nil || res.Status != "Late" {
		t.Fatalf("09:00 should be Late: %+v err=%v", res, err)
	}
}

func TestClockInJudgedInWIBNotUTC(t *testing.T) {
	// 01:30 UTC is 08:30 WIB -> On Time, even though the server clock says 01:30.
	utc := time.Date(2026, 9, 29, 1, 30, 0, 0, time.UTC)
	res, err := newTestUC(&fakeHRMRepo{}, utc).RecordClockIn(context.Background(), ClockInDTO{NIP: "E1", EmployeeName: "A"})
	if err != nil || res.Status != "On Time" || res.ClockIn != "08:30 WIB" {
		t.Fatalf("unexpected: %+v err=%v", res, err)
	}
}

func TestGeofence(t *testing.T) {
	office := domain.AttendanceLocation{Name: "HQ", Latitude: -6.2000, Longitude: 106.8166, RadiusMeters: 100, IsActive: true}
	office.ID = uuid.New()
	repo := &fakeHRMRepo{locs: []domain.AttendanceLocation{office}}
	ctx := context.Background()
	uc := newTestUC(repo, wib(8, 0))

	if _, err := uc.RecordClockIn(ctx, ClockInDTO{NIP: "E1", EmployeeName: "A"}); err == nil {
		t.Fatal("coordinates must be required when locations are configured")
	}
	if _, err := uc.RecordClockIn(ctx, ClockInDTO{NIP: "E1", EmployeeName: "A", Latitude: f64(-6.2100), Longitude: f64(106.8166)}); err == nil {
		t.Fatal("expected ~1.1 km away to be rejected")
	}
	res, err := uc.RecordClockIn(ctx, ClockInDTO{NIP: "E1", EmployeeName: "A", Latitude: f64(-6.20005), Longitude: f64(106.8166)})
	if err != nil || res.Location != "HQ" {
		t.Fatalf("expected match to HQ: %+v err=%v", res, err)
	}
}

func TestClockOut(t *testing.T) {
	repo := &fakeHRMRepo{}
	ctx := context.Background()

	if _, err := newTestUC(repo, wib(17, 0)).RecordClockOut(ctx, ClockOutDTO{NIP: "E1"}); err == nil {
		t.Fatal("clock-out without clock-in must fail")
	}
	if _, err := newTestUC(repo, wib(8, 0)).RecordClockIn(ctx, ClockInDTO{NIP: "E1", EmployeeName: "A"}); err != nil {
		t.Fatal(err)
	}
	res, err := newTestUC(repo, wib(17, 30)).RecordClockOut(ctx, ClockOutDTO{NIP: "E1"})
	if err != nil || res.ClockOut != "17:30 WIB" || res.WorkMinutes != 570 {
		t.Fatalf("unexpected clock-out: %+v err=%v", res, err)
	}
	if _, err := newTestUC(repo, wib(18, 0)).RecordClockOut(ctx, ClockOutDTO{NIP: "E1"}); err == nil {
		t.Fatal("second clock-out must be rejected")
	}
}
