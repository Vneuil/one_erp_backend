package application

import (
	"context"
	"testing"
	"time"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/hrops/domain"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"github.com/divinecoid/one-backend/internal/shared/types"
	"github.com/google/uuid"
)

func TestDetectOvertimeMinutes(t *testing.T) {
	cases := map[int]int{0: 0, 480: 0, 510: 0, 511: 31, 600: 120}
	for work, want := range cases {
		if got := DetectOvertimeMinutes(work); got != want {
			t.Errorf("work %d: got %d want %d", work, got, want)
		}
	}
}

func TestClockStatusMatchesLiveClockInRule(t *testing.T) {
	in, _ := parseClock("2026-09-10", "08:59")
	if s, _ := clockStatus(in, ""); s != "On Time" {
		t.Fatalf("08:59 = %s", s)
	}
	in, _ = parseClock("2026-09-10", "09:00")
	if s, _ := clockStatus(in, ""); s != "Late" {
		t.Fatalf("09:00 = %s", s)
	}
	// A night shift starting at 22:00 makes 21:50 on time.
	in, _ = parseClock("2026-09-10", "21:50")
	if s, _ := clockStatus(in, "22:00"); s != "On Time" {
		t.Fatalf("21:50 vs 22:00 = %s", s)
	}
}

func TestSummarizeFeedback(t *testing.T) {
	s := summarizeFeedback([]ratings{
		{"peer", 4, 4, 4, 4, "good"},
		{"peer", 2, 4, 2, 4, ""},
		{"manager", 5, 5, 5, 5, ""},
	})
	if s.Count != 3 || s.Overall["communication"] != 3.67 || s.ByRelationship["peer"]["communication"] != 3 {
		t.Fatalf("bad summary: %+v", s)
	}
	if len(s.Comments) != 1 {
		t.Fatalf("comments: %v", s.Comments)
	}
}

// ---- fakes

type fakeRepo struct {
	domain.Repository
	corr     *domain.AttendanceCorrection
	fb       []domain.Feedback
	overtime []domain.OvertimeRecord
	notes    []domain.Notification
	assign   *domain.ShiftAssignment
	shift    *domain.Shift
}

func (f *fakeRepo) GetCorrection(context.Context, uuid.UUID) (*domain.AttendanceCorrection, error) {
	return f.corr, nil
}
func (f *fakeRepo) UpdateCorrection(context.Context, *domain.AttendanceCorrection) error { return nil }
func (f *fakeRepo) GetAssignment(context.Context, string, string) (*domain.ShiftAssignment, error) {
	return f.assign, nil
}
func (f *fakeRepo) GetShift(context.Context, uuid.UUID) (*domain.Shift, error) { return f.shift, nil }
func (f *fakeRepo) ListFeedback(context.Context, uuid.UUID, string) ([]domain.Feedback, error) {
	return f.fb, nil
}
func (f *fakeRepo) FeedbackExists(context.Context, uuid.UUID, string, string) (bool, error) {
	return false, nil
}
func (f *fakeRepo) CreateFeedback(_ context.Context, v *domain.Feedback) error {
	f.fb = append(f.fb, *v)
	return nil
}
func (f *fakeRepo) OvertimeExists(_ context.Context, nip, date, source string) (bool, error) {
	for _, o := range f.overtime {
		if o.NIP == nip && o.Date == date && o.Source == source {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeRepo) CreateOvertime(_ context.Context, v *domain.OvertimeRecord) error {
	f.overtime = append(f.overtime, *v)
	return nil
}
func (f *fakeRepo) CreateAnnouncement(context.Context, *domain.Announcement) error { return nil }
func (f *fakeRepo) CreateNotifications(_ context.Context, v []domain.Notification) error {
	f.notes = append(f.notes, v...)
	return nil
}

type fakeHRM struct {
	hrmdomain.HRMRepository
	att       *hrmdomain.Attendance
	recorded  *hrmdomain.Attendance
	updated   *hrmdomain.Attendance
	byPeriod  []hrmdomain.Attendance
	employees []hrmdomain.Employee
}

func (f *fakeHRM) FindAttendanceByNIPAndDate(context.Context, string, string) (*hrmdomain.Attendance, error) {
	return f.att, nil
}
func (f *fakeHRM) RecordAttendance(_ context.Context, a *hrmdomain.Attendance) error {
	f.recorded = a
	return nil
}
func (f *fakeHRM) UpdateAttendance(_ context.Context, a *hrmdomain.Attendance) error {
	f.updated = a
	return nil
}
func (f *fakeHRM) ListAttendanceByNIPAndPeriod(context.Context, string, string) ([]hrmdomain.Attendance, error) {
	return f.byPeriod, nil
}
func (f *fakeHRM) ListEmployees(context.Context, types.PaginationQuery) ([]hrmdomain.Employee, int64, error) {
	return f.employees, int64(len(f.employees)), nil
}
func (f *fakeHRM) GetEmployeeByID(_ context.Context, id uuid.UUID) (*hrmdomain.Employee, error) {
	for i := range f.employees {
		if f.employees[i].ID == id {
			return &f.employees[i], nil
		}
	}
	return nil, nil
}

func svc(r *fakeRepo, h *fakeHRM) *Service {
	s := NewService(r, h)
	s.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, zone) }
	return s
}

func TestApprovingCorrectionCreatesAttendanceAndBlocksSelfApproval(t *testing.T) {
	corr := &domain.AttendanceCorrection{NIP: "E1", EmployeeName: "Ani", Date: "2026-09-10", ClockIn: "08:30", ClockOut: "17:30",
		Status: "pending", RequestedByEmail: "ani@x.com"}
	corr.ID = uuid.New()
	hrm := &fakeHRM{}
	s := svc(&fakeRepo{corr: corr}, hrm)

	if _, err := s.DecideCorrection(actor.WithEmail(context.Background(), "ANI@x.com"), corr.ID, true); err == nil {
		t.Fatal("requester must not approve their own correction")
	}
	if hrm.recorded != nil || corr.Status != "pending" {
		t.Fatal("a blocked approval must not change anything")
	}

	if _, err := s.DecideCorrection(actor.WithEmail(context.Background(), "boss@x.com"), corr.ID, true); err != nil {
		t.Fatal(err)
	}
	a := hrm.recorded
	if a == nil || a.Method != "Correction" || a.Status != "On Time" || a.WorkMinutes != 540 {
		t.Fatalf("attendance not written correctly: %+v", a)
	}
	if corr.Status != "approved" || corr.DecidedBy != "boss@x.com" {
		t.Fatalf("status %s by %s", corr.Status, corr.DecidedBy)
	}
	if _, err := s.DecideCorrection(context.Background(), corr.ID, false); err == nil {
		t.Fatal("a decided request cannot be decided again")
	}
}

func TestCorrectionUpdatesExistingRecordUsingShiftStart(t *testing.T) {
	corr := &domain.AttendanceCorrection{NIP: "E1", Date: "2026-09-10", ClockIn: "21:55", Status: "pending", RequestedByEmail: "ani@x.com"}
	corr.ID = uuid.New()
	existing := &hrmdomain.Attendance{NIP: "E1", Date: "2026-09-10", Status: "Late"}
	hrm := &fakeHRM{att: existing}
	r := &fakeRepo{corr: corr, assign: &domain.ShiftAssignment{ShiftID: uuid.New()}, shift: &domain.Shift{StartTime: "22:00"}}
	if _, err := svc(r, hrm).DecideCorrection(actor.WithEmail(context.Background(), "boss@x.com"), corr.ID, true); err != nil {
		t.Fatal(err)
	}
	if hrm.updated != existing || existing.Status != "On Time" || existing.ClockOutAt != nil {
		t.Fatalf("existing record not updated: %+v", existing)
	}
}

func TestFeedbackSummaryHiddenBelowThreshold(t *testing.T) {
	subject := uuid.New()
	r := &fakeRepo{fb: []domain.Feedback{
		{Relationship: "peer", Communication: 5, Teamwork: 5, Leadership: 5, Reliability: 5},
		{Relationship: "manager", Communication: 1, Teamwork: 1, Leadership: 1, Reliability: 1},
		{Relationship: "self", Communication: 5, Teamwork: 5, Leadership: 5, Reliability: 5},
	}}
	got, err := svc(r, &fakeHRM{}).FeedbackSummaryFor(context.Background(), subject, "2026")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Overall) != 0 || got.Count != 3 {
		t.Fatalf("two non-self reviewers must not reveal averages: %+v", got)
	}
	r.fb = append(r.fb, domain.Feedback{Relationship: "peer", Communication: 3, Teamwork: 3, Leadership: 3, Reliability: 3})
	got, _ = svc(r, &fakeHRM{}).FeedbackSummaryFor(context.Background(), subject, "2026")
	if len(got.Overall) == 0 {
		t.Fatal("three reviewers should reveal the summary")
	}
}

func TestSelfRelationshipOnlyForSelf(t *testing.T) {
	emp := hrmdomain.Employee{Email: "ani@x.com"}
	emp.ID = uuid.New()
	s := svc(&fakeRepo{}, &fakeHRM{employees: []hrmdomain.Employee{emp}})
	in := FeedbackInput{SubjectID: emp.ID, Relationship: "peer", Period: "2026", Communication: 3, Teamwork: 3, Leadership: 3, Reliability: 3}
	if err := s.SubmitFeedback(context.Background(), Caller{Email: "ANI@x.com"}, in); err == nil {
		t.Fatal("reviewing yourself as a peer must be rejected")
	}
	in.Relationship = "self"
	if err := s.SubmitFeedback(context.Background(), Caller{Email: "bob@x.com"}, in); err == nil {
		t.Fatal("claiming to be self for someone else must be rejected")
	}
	if err := s.SubmitFeedback(context.Background(), Caller{Email: "ani@x.com"}, in); err != nil {
		t.Fatalf("legitimate self review: %v", err)
	}
	in.Communication = 6
	if err := s.SubmitFeedback(context.Background(), Caller{Email: "ani@x.com"}, in); err == nil {
		t.Fatal("ratings above 5 must be rejected")
	}
}

func TestDetectOvertimeIsIdempotent(t *testing.T) {
	hrm := &fakeHRM{byPeriod: []hrmdomain.Attendance{
		{NIP: "E1", Date: "2026-09-01", WorkMinutes: 600},
		{NIP: "E1", Date: "2026-09-02", WorkMinutes: 500}, // within threshold
	}}
	r := &fakeRepo{}
	s := svc(r, hrm)
	n, err := s.DetectOvertime(context.Background(), "2026-09")
	if err != nil || n != 1 {
		t.Fatalf("first run: %d %v", n, err)
	}
	if n, _ := s.DetectOvertime(context.Background(), "2026-09"); n != 0 {
		t.Fatalf("second run created %d duplicates", n)
	}
	if r.overtime[0].Minutes != 120 || r.overtime[0].Status != "pending" || r.overtime[0].Source != "auto" {
		t.Fatalf("bad record: %+v", r.overtime[0])
	}
}

func TestAnnouncementNotifiesTargetedEmployeesOnly(t *testing.T) {
	hrm := &fakeHRM{employees: []hrmdomain.Employee{
		{Email: "a@x.com", Department: "Ops", Status: "Active"},
		{Email: "b@x.com", Department: "Sales", Status: "Active"},
		{Email: "c@x.com", Department: "Ops", Status: "Inactive"},
		{Email: "", Department: "Ops", Status: "Active"},
	}}
	r := &fakeRepo{}
	_, n, err := svc(r, hrm).PublishAnnouncement(context.Background(), Caller{Email: "hr@x.com"}, AnnouncementInput{Title: "T", Body: "B", Department: "ops"})
	if err != nil || n != 1 || r.notes[0].RecipientEmail != "a@x.com" {
		t.Fatalf("notified %d: %+v err %v", n, r.notes, err)
	}
}

func TestPublicRequestsNeedOwnEmployeeRecordOrPrivilege(t *testing.T) {
	own := hrmdomain.Employee{NIP: "E1", Email: "ani@x.com", Name: "Ani"}
	other := hrmdomain.Employee{NIP: "E2", Email: "bob@x.com", Name: "Bob"}
	hrm := &fakeRepoHRM{employees: []hrmdomain.Employee{own, other}}
	s := NewService(&fakeRepo{}, hrm)
	s.now = func() time.Time { return time.Date(2026, 9, 30, 12, 0, 0, 0, zone) }
	if _, err := s.resolveEmployee(context.Background(), Caller{Email: "ani@x.com"}, "E2"); err == nil {
		t.Fatal("a normal user must not file for someone else")
	}
	if e, err := s.resolveEmployee(context.Background(), Caller{Email: "hr@x.com", Privileged: true}, "E2"); err != nil || e.NIP != "E2" {
		t.Fatalf("privileged caller should reach E2: %v %v", e, err)
	}
	if e, err := s.resolveEmployee(context.Background(), Caller{Email: "ANI@x.com"}, ""); err != nil || e.NIP != "E1" {
		t.Fatalf("own record by email: %v %v", e, err)
	}
	if _, err := s.resolveEmployee(context.Background(), Caller{Email: "stranger@x.com"}, ""); err == nil {
		t.Fatal("no employee record must be an error")
	}
}

type fakeRepoHRM struct {
	fakeHRM
	employees []hrmdomain.Employee
}

func (f *fakeRepoHRM) FindEmployeeByEmail(_ context.Context, email string) (*hrmdomain.Employee, error) {
	for i := range f.employees {
		if equalFold(f.employees[i].Email, email) {
			return &f.employees[i], nil
		}
	}
	return nil, nil
}
func (f *fakeRepoHRM) ListEmployees(context.Context, types.PaginationQuery) ([]hrmdomain.Employee, int64, error) {
	return f.employees, int64(len(f.employees)), nil
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 32
		}
		if 'A' <= y && y <= 'Z' {
			y += 32
		}
		if x != y {
			return false
		}
	}
	return true
}
