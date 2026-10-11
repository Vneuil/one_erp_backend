package application

import (
	"context"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/fingerprint/domain"
	hrmDomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/google/uuid"
)

func at(s string) time.Time {
	t, err := ParseTime(s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseATTLOGHandlesTabsSplitDatesAndBadLines(t *testing.T) {
	body := "12\t2026-03-10 07:55:01\t0\t1\t0\r\n" +
		"13\t2026-03-10\t08:02:10\t1\t15\n" +
		"\n" +
		"x\tnot-a-time\t0\t1\n" +
		"14\n"
	ps, problems := ParseATTLOG(body)
	if len(ps) != 2 || len(problems) != 2 {
		t.Fatalf("punches %+v problems %v", ps, problems)
	}
	if ps[0].PIN != "12" || ps[0].At.In(Zone).Format("15:04:05") != "07:55:01" || ps[0].Verify != 1 {
		t.Fatalf("first: %+v", ps[0])
	}
	if ps[1].PIN != "13" || ps[1].At.In(Zone).Format("15:04") != "08:02" || ps[1].State != 1 || ps[1].Verify != 15 {
		t.Fatalf("second (split date/time): %+v", ps[1])
	}
}

func TestMergeDayFirstAndLastPunchAndLateness(t *testing.T) {
	p := []time.Time{at("2026-03-10 17:05:00"), at("2026-03-10 08:50:00"), at("2026-03-10 08:51:00")}
	a, changed := MergeDay(nil, p, "EMP-1", "Budi", "2026-03-10", "Pintu")
	if !changed || a.ClockIn != "08:50 WIB" || a.ClockOut != "17:05 WIB" || a.Status != "On Time" || a.WorkMinutes != 495 || a.Method != "Fingerprint" {
		t.Fatalf("%+v", a)
	}
	late, _ := MergeDay(nil, []time.Time{at("2026-03-10 09:00:00")}, "EMP-1", "Budi", "2026-03-10", "")
	if late.Status != "Late" || late.ClockOutAt != nil || late.ClockOut != "--:--" {
		t.Fatalf("a single punch is a clock-in only: %+v", late)
	}
	// A double tap minutes apart is not a clock-out.
	dbl, _ := MergeDay(nil, []time.Time{at("2026-03-10 08:00:00"), at("2026-03-10 08:02:00")}, "EMP-1", "Budi", "2026-03-10", "")
	if dbl.ClockOutAt != nil {
		t.Fatalf("double tap treated as clock-out: %+v", dbl)
	}
}

func TestMergeDayWidensExistingRecordOnly(t *testing.T) {
	in := at("2026-03-10 07:30:00")
	existing := &hrmDomain.Attendance{NIP: "EMP-1", Date: "2026-03-10", ClockIn: "07:30 WIB", ClockInAt: &in, ClockOut: "--:--", Method: "GPS Mobile", Status: "On Time"}
	a, _ := MergeDay(existing, []time.Time{at("2026-03-10 08:10:00"), at("2026-03-10 16:00:00")}, "EMP-1", "Budi", "2026-03-10", "")
	if a.ClockIn != "07:30 WIB" || a.ClockOut != "16:00 WIB" || a.Method != "GPS Mobile" {
		t.Fatalf("a later punch must not replace the earlier GPS clock-in: %+v", a)
	}
	if _, changed := MergeDay(a, []time.Time{at("2026-03-10 12:00:00")}, "EMP-1", "Budi", "2026-03-10", ""); changed {
		t.Fatal("a punch inside the existing span changes nothing")
	}
}

type memRepo struct {
	domain.Repository
	punches   []domain.Punch
	maps      map[string]domain.PinMap
	employees map[string]string
	att       map[string]*hrmDomain.Attendance
}

func newMem() *memRepo {
	return &memRepo{maps: map[string]domain.PinMap{}, employees: map[string]string{"EMP-1": "Budi", "EMP-2": "Sari"}, att: map[string]*hrmDomain.Attendance{}}
}
func (r *memRepo) GetPinMap(_ context.Context, pin string) (*domain.PinMap, error) {
	if m, ok := r.maps[pin]; ok {
		return &m, nil
	}
	return nil, nil
}
func (r *memRepo) SavePinMap(_ context.Context, m *domain.PinMap) error {
	r.maps[m.PIN] = *m
	return nil
}
func (r *memRepo) EmployeeName(_ context.Context, nip string) (string, bool, error) {
	n, ok := r.employees[nip]
	return n, ok, nil
}
func (r *memRepo) InsertPunches(_ context.Context, ps []domain.Punch) (int, error) {
	n := 0
	for _, p := range ps {
		dup := false
		for _, q := range r.punches {
			if q.DeviceSerial == p.DeviceSerial && q.PIN == p.PIN && q.PunchedAt.Equal(p.PunchedAt) {
				dup = true
			}
		}
		if !dup {
			r.punches = append(r.punches, p)
			n++
		}
	}
	return n, nil
}
func (r *memRepo) PunchesForDay(_ context.Context, nip, date string) ([]domain.Punch, error) {
	var out []domain.Punch
	for _, p := range r.punches {
		if p.NIP == nip && p.Date == date {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *memRepo) UnresolvedPunches(context.Context) ([]domain.Punch, error) {
	var out []domain.Punch
	for _, p := range r.punches {
		if !p.Resolved {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *memRepo) UpdatePunch(_ context.Context, p *domain.Punch) error {
	for i := range r.punches {
		if r.punches[i].ID == p.ID {
			r.punches[i] = *p
		}
	}
	return nil
}
func (r *memRepo) FindAttendance(_ context.Context, nip, date string) (*hrmDomain.Attendance, error) {
	if a, ok := r.att[nip+date]; ok {
		cp := *a
		return &cp, nil
	}
	return nil, nil
}
func (r *memRepo) SaveAttendance(_ context.Context, a *hrmDomain.Attendance) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	cp := *a
	r.att[a.NIP+a.Date] = &cp
	return nil
}
func (r *memRepo) GetDevice(context.Context, string) (*domain.Device, error)     { return nil, nil }
func (r *memRepo) TouchDevice(context.Context, string, string, *time.Time) error { return nil }

func TestIngestIsIdempotentAndResolvesLater(t *testing.T) {
	repo := newMem()
	s := NewService(repo, nil)
	ctx := context.Background()
	rows := []RawPunch{
		{PIN: "EMP-1", At: at("2026-03-10 08:30:00")}, // PIN equals a NIP: resolves without a mapping
		{PIN: "EMP-1", At: at("2026-03-10 17:30:00")},
		{PIN: "77", At: at("2026-03-10 08:45:00")}, // unknown PIN
	}
	res, err := s.Ingest(ctx, nil, "SN1", rows)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stored != 3 || res.Unresolved != 1 || res.DaysSynced != 1 {
		t.Fatalf("first ingest: %+v", res)
	}
	a := repo.att["EMP-12026-03-10"]
	if a == nil || a.ClockIn != "08:30 WIB" || a.ClockOut != "17:30 WIB" {
		t.Fatalf("attendance: %+v", a)
	}
	// The machine re-sends its whole log: nothing is double counted.
	res, _ = s.Ingest(ctx, nil, "SN1", rows)
	if res.Stored != 0 || res.Duplicates != 3 || res.DaysSynced != 0 {
		t.Fatalf("re-send: %+v", res)
	}
	// Mapping PIN 77 to Sari resolves the waiting punch and creates her day.
	_, rep, err := s.MapPIN(ctx, PinMapInput{PIN: "77", NIP: "EMP-2"})
	if err != nil || rep.Stored != 1 || rep.DaysSynced != 1 {
		t.Fatalf("map: %v %+v", err, rep)
	}
	if b := repo.att["EMP-22026-03-10"]; b == nil || b.EmployeeName != "Sari" || b.ClockIn != "08:45 WIB" {
		t.Fatalf("Sari's day: %+v", b)
	}
	if _, _, err := s.MapPIN(ctx, PinMapInput{PIN: "9", NIP: "NOPE"}); err == nil {
		t.Fatal("mapping to an unknown NIP must fail")
	}
}

func TestTokenMatching(t *testing.T) {
	h := HashToken("abc")
	if !TokenMatches("abc", h) || TokenMatches("abd", h) || TokenMatches("", h) {
		t.Fatal("token comparison is wrong")
	}
}
