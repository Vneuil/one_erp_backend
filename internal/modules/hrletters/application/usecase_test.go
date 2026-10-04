package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrletters/domain"
	hrmDomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/google/uuid"
)

type memRepo struct {
	letters []*domain.Letter
}

func (r *memRepo) Create(_ context.Context, l *domain.Letter) error {
	r.letters = append(r.letters, l)
	return nil
}
func (r *memRepo) Get(_ context.Context, id uuid.UUID) (*domain.Letter, error) {
	for _, l := range r.letters {
		if l.ID == id {
			c := *l
			return &c, nil
		}
	}
	return nil, nil
}
func (r *memRepo) Update(_ context.Context, l *domain.Letter, _ bool) error {
	for i, cur := range r.letters {
		if cur.ID == l.ID {
			c := *l
			r.letters[i] = &c
		}
	}
	return nil
}
func (r *memRepo) List(context.Context, domain.Filter) ([]domain.Letter, error) { return nil, nil }
func (r *memRepo) CountIssued(_ context.Context, typ string, year int) (int64, error) {
	var n int64
	for _, l := range r.letters {
		if l.Type == typ && l.Number != "" && strings.HasPrefix(l.Date, time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Format("2006")) {
			n++
		}
	}
	return n, nil
}
func (r *memRepo) ListByEmployee(_ context.Context, id uuid.UUID) ([]domain.Letter, error) {
	var out []domain.Letter
	for _, l := range r.letters {
		if l.EmployeeID != nil && *l.EmployeeID == id && l.Status != domain.StatusCancelled {
			out = append(out, *l)
		}
	}
	return out, nil
}
func (r *memRepo) ListIssuedContracts(context.Context) ([]domain.Letter, error) {
	var out []domain.Letter
	for _, l := range r.letters {
		if l.Type == domain.TypeContract && l.Status == domain.StatusIssued {
			out = append(out, *l)
		}
	}
	return out, nil
}

type memEmployees struct {
	byID map[uuid.UUID]*hrmDomain.Employee
}

func (m memEmployees) GetEmployeeByID(_ context.Context, id uuid.UUID) (*hrmDomain.Employee, error) {
	return m.byID[id], nil
}
func (m memEmployees) UpdateEmployee(context.Context, *hrmDomain.Employee) error { return nil }

type fixture struct {
	uc    *useCase
	repo  *memRepo
	emp   *hrmDomain.Employee
	other *hrmDomain.Employee
	boss  *hrmDomain.Employee
}

func newFixture() *fixture {
	mk := func(nip, name, dept, role string) *hrmDomain.Employee {
		e := &hrmDomain.Employee{NIP: nip, Name: name, Department: dept, Role: role, Status: "Active", ContractType: "PKWT"}
		e.ID = uuid.New()
		return e
	}
	f := &fixture{repo: &memRepo{}, emp: mk("EMP-001", "Budi Santoso", "Produksi", "Operator"), other: mk("EMP-002", "Sari Dewi", "Produksi", "Operator"), boss: mk("EMP-009", "Hendra", "Gudang", "Kepala Gudang")}
	f.uc = NewUseCase(f.repo, memEmployees{byID: map[uuid.UUID]*hrmDomain.Employee{f.emp.ID: f.emp, f.other.ID: f.other, f.boss.ID: f.boss}}).(*useCase)
	f.uc.now = func() time.Time { return time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC) }
	return f
}

func (f *fixture) base(t string) LetterInput {
	return LetterInput{Type: t, Date: "2026-03-20", CompanyName: "PT Contoh", City: "Jakarta", SignerName: "Rina HRD", SignerTitle: "HR Manager", EmployeeID: &f.emp.ID}
}

func TestNumberingIsPerTypeAndYearAndOnlyAtIssue(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	in := f.base(domain.TypeReprimand)
	in.Data.Violation = "Terlambat 5 kali"
	a, err := f.uc.Create(ctx, in)
	if err != nil || a.Number != "" || a.Status != domain.StatusDraft {
		t.Fatalf("a draft has no number: %+v err=%v", a, err)
	}
	b, _ := f.uc.Create(ctx, in)
	ib, err := f.uc.Issue(ctx, b.ID)
	if err != nil || ib.Number != "001/HRD-ST/III/2026" {
		t.Fatalf("the first issued letter takes 001 even if an earlier draft exists: %+v err=%v", ib, err)
	}
	ia, _ := f.uc.Issue(ctx, a.ID)
	if ia.Number != "002/HRD-ST/III/2026" {
		t.Fatalf("second: %s", ia.Number)
	}
	w := f.base(domain.TypeWarning)
	w.Level, w.Data.Violation = 1, "Mangkir"
	wl, _ := f.uc.Create(ctx, w)
	iw, _ := f.uc.Issue(ctx, wl.ID)
	if iw.Number != "001/HRD-SP/III/2026" {
		t.Fatalf("numbering restarts per type: %s", iw.Number)
	}
	if _, err := f.uc.Issue(ctx, b.ID); err == nil {
		t.Fatal("an issued letter cannot be issued again")
	}
	if LetterNumber(domain.TypeMemo, 12, "2026-12-05") != "012/HRD-MEMO/XII/2026" {
		t.Fatal("roman month and padding")
	}
}

func TestIssueNeedsASigner(t *testing.T) {
	f := newFixture()
	in := f.base(domain.TypeReprimand)
	in.SignerName, in.Data.Violation = "", "x"
	l, _ := f.uc.Create(context.Background(), in)
	if _, err := f.uc.Issue(context.Background(), l.ID); err == nil {
		t.Fatal("a letter needs a signer to be issued")
	}
}

func TestContractRulesAndEffectOnEmployee(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	in := f.base(domain.TypeContract)
	in.EffectiveDate, in.EndDate = "2026-04-01", "2027-03-31"
	in.Data = domain.LetterData{ContractType: "PKWT", Position: "Operator Mesin", Salary: 5_500_000, Workplace: "Pabrik Cikarang", ProbationMonth: 2}
	l, err := f.uc.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Perjanjian Kerja Waktu Tertentu", "Budi Santoso", "Rp 5.500.000", "1 April 2026", "31 Maret 2027", "masa percobaan selama 2"} {
		if !strings.Contains(l.Subject+l.Body, want) {
			t.Errorf("contract text lacks %q:\n%s", want, l.Body)
		}
	}
	bad := map[string]func(*LetterInput){
		"no end for PKWT":  func(i *LetterInput) { i.EndDate = "" },
		"end before start": func(i *LetterInput) { i.EndDate = "2026-03-01" },
		"over five years":  func(i *LetterInput) { i.EndDate = "2031-04-01" },
		"zero salary":      func(i *LetterInput) { i.Data.Salary = 0 },
		"unknown type":     func(i *LetterInput) { i.Data.ContractType = "Freelance" },
		"long probation":   func(i *LetterInput) { i.Data.ProbationMonth = 6 },
	}
	for name, mut := range bad {
		x := in
		mut(&x)
		if _, err := f.uc.Create(ctx, x); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	// PKWTT has no end date and its text says so.
	p := in
	p.Data.ContractType, p.EndDate = "PKWTT", "2030-01-01"
	pl, err := f.uc.Create(ctx, p)
	if err != nil || pl.EndDate != "" || !strings.Contains(pl.Body, "waktu tidak tertentu") {
		t.Fatalf("pkwtt: %+v err=%v", pl, err)
	}
	// Issued before its start date: not applied until it is in effect.
	il, err := f.uc.Issue(ctx, l.ID)
	if err != nil || il.Applied || f.emp.ContractType != "PKWT" {
		t.Fatalf("a future contract must not be applied yet: %+v", il)
	}
	if _, err := f.uc.Apply(ctx, l.ID); err == nil {
		t.Fatal("cannot apply before the effective date")
	}
	f.uc.now = func() time.Time { return time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC) }
	al, err := f.uc.Apply(ctx, l.ID)
	if err != nil || !al.Applied || al.AppliedAt == nil {
		t.Fatalf("apply on the start date: %+v err=%v", al, err)
	}
	if _, err := f.uc.Apply(ctx, l.ID); err == nil {
		t.Fatal("applying twice must fail")
	}
}

func TestWarningLevelValidityAndSummary(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	mk := func(level int, date string) *domain.Letter {
		in := f.base(domain.TypeWarning)
		in.Date, in.Level, in.Data.Violation = date, level, "Pelanggaran tata tertib"
		l, err := f.uc.Create(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.uc.Issue(ctx, l.ID); err != nil {
			t.Fatal(err)
		}
		return l
	}
	sp1 := mk(1, "2025-06-01") // valid 6 months => lapsed on 2025-12-01
	sp2 := mk(2, "2026-02-01") // valid until 2026-08-01
	got, _ := f.uc.Get(ctx, sp2.ID)
	if got.EndDate != "2026-08-01" || got.Subject != "Surat Peringatan II" || !strings.Contains(got.Body, "6 bulan") {
		t.Fatalf("SP2: %+v", got)
	}
	sum, err := f.uc.EmployeeSummary(ctx, f.emp.ID)
	if err != nil || len(sum.ActiveWarnings) != 1 || sum.ActiveWarnings[0].ID != sp2.ID || sum.HighestActiveLevel != 2 || len(sum.Letters) != 2 {
		t.Fatalf("summary: %+v err=%v", sum, err)
	}
	_ = sp1
	if sum.Unacknowledged != 2 {
		t.Fatalf("both warnings still await the employee's acknowledgement: %d", sum.Unacknowledged)
	}
	if _, err := f.uc.Acknowledge(ctx, sp2.ID); err != nil {
		t.Fatal(err)
	}
	if sum, _ = f.uc.EmployeeSummary(ctx, f.emp.ID); sum.Unacknowledged != 1 {
		t.Fatalf("after acknowledging: %d", sum.Unacknowledged)
	}
	bad := f.base(domain.TypeWarning)
	bad.Data.Violation = "x"
	for _, lvl := range []int{0, 4} {
		bad.Level = lvl
		if _, err := f.uc.Create(ctx, bad); err == nil {
			t.Errorf("level %d must be rejected", lvl)
		}
	}
	// SP3 text carries the final-warning wording.
	in := f.base(domain.TypeWarning)
	in.Level, in.Data.Violation = 3, "Mangkir 5 hari"
	l3, _ := f.uc.Create(ctx, in)
	if !strings.Contains(l3.Body, "peringatan terakhir") {
		t.Fatalf("SP3 body: %s", l3.Body)
	}
}

func TestMutationSnapshotsAndAppliesToEmployee(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	in := f.base(domain.TypeMutation)
	in.EffectiveDate = "2026-03-20"
	in.Data = domain.LetterData{ToDepartment: "Gudang", ToRole: "Staf Gudang", ToManagerID: f.boss.ID.String()}
	l, err := f.uc.Create(ctx, in)
	if err != nil || l.Data.FromDepartment != "Produksi" || l.Data.FromRole != "Operator" || !strings.Contains(l.Body, "Staf Gudang") {
		t.Fatalf("draft: %+v err=%v", l, err)
	}
	if _, err := f.uc.Create(ctx, func() LetterInput { x := in; x.Data = domain.LetterData{}; return x }()); err == nil {
		t.Fatal("a mutation must change something")
	}
	il, err := f.uc.Issue(ctx, l.ID)
	if err != nil || !il.Applied {
		t.Fatalf("issued on its effective date => applied: %+v err=%v", il, err)
	}
	if f.emp.Department != "Gudang" || f.emp.Role != "Staf Gudang" || f.emp.ManagerID == nil || *f.emp.ManagerID != f.boss.ID {
		t.Fatalf("employee record: %+v", f.emp)
	}
	if _, err := f.uc.Cancel(ctx, l.ID, "salah"); err == nil {
		t.Fatal("an applied letter cannot be cancelled")
	}
	// An employee cannot become their own supervisor: refused when drafting, so nothing is numbered.
	self := f.base(domain.TypeMutation)
	self.EffectiveDate = "2026-03-20"
	self.Data = domain.LetterData{ToManagerID: f.emp.ID.String()}
	if _, err := f.uc.Create(ctx, self); err == nil {
		t.Fatal("self-supervision must be refused")
	}
	sum, _ := f.uc.EmployeeSummary(ctx, f.emp.ID)
	if len(sum.Mutations) != 1 {
		t.Fatalf("mutation history: %+v", sum.Mutations)
	}
}

func TestTerminationSetsStatusOnTheLastWorkingDay(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	in := f.base(domain.TypeTermination)
	in.Data = domain.LetterData{TerminationReason: "berakhirnya perjanjian kerja waktu tertentu", LastWorkDay: "2026-03-31", Severance: 0, ServiceAward: 1_000_000, CompensationRights: 2_500_000}
	l, err := f.uc.Create(ctx, in)
	if err != nil || l.EffectiveDate != "2026-03-31" {
		t.Fatalf("draft: %+v err=%v", l, err)
	}
	for _, want := range []string{"31 Maret 2026", "uang penghargaan masa kerja Rp 1.000.000", "uang penggantian hak Rp 2.500.000"} {
		if !strings.Contains(l.Body, want) {
			t.Errorf("termination text lacks %q", want)
		}
	}
	il, _ := f.uc.Issue(ctx, l.ID)
	if il.Applied || f.emp.Status != "Active" {
		t.Fatal("still employed until the last working day")
	}
	f.uc.now = func() time.Time { return time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC) }
	if _, err := f.uc.Apply(ctx, l.ID); err != nil || f.emp.Status != "Terminated" {
		t.Fatalf("status after the last day: %s err=%v", f.emp.Status, err)
	}
	if _, err := f.uc.Create(ctx, func() LetterInput { x := in; x.Data.LastWorkDay = ""; return x }()); err == nil {
		t.Fatal("last working day is required")
	}
}

func TestOvertimeOrderMemoAndSummonsRules(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	spl := LetterInput{Type: domain.TypeOvertimeOrder, Date: "2026-03-20", SignerName: "Kepala Produksi", RecipientIDs: []uuid.UUID{f.emp.ID, f.other.ID},
		Data: domain.LetterData{WorkDate: "2026-03-21", StartTime: "17:00", EndTime: "21:00", Tasks: "Mengejar order PRD-001"}}
	l, err := f.uc.Create(ctx, spl)
	if err != nil || len(l.Recipients) != 2 || l.EffectiveDate != "2026-03-21" || !strings.Contains(l.Body, "Sabtu, 21 Maret 2026") || !strings.Contains(l.Body, "2. Sari Dewi") {
		t.Fatalf("spl: %+v err=%v", l, err)
	}
	for name, mut := range map[string]func(*LetterInput){
		"no employees":     func(i *LetterInput) { i.RecipientIDs = nil },
		"end before start": func(i *LetterInput) { i.Data.EndTime = "16:00" },
		"bad time":         func(i *LetterInput) { i.Data.StartTime = "5pm" },
		"no tasks":         func(i *LetterInput) { i.Data.Tasks = " " },
	} {
		x := spl
		mut(&x)
		if _, err := f.uc.Create(ctx, x); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	memo := LetterInput{Type: domain.TypeMemo, Date: "2026-03-20", SignerName: "HRD", Subject: "Libur Lebaran", Body: "Kantor libur 20-24 Maret."}
	if _, err := f.uc.Create(ctx, memo); err == nil {
		t.Fatal("a memo must be addressed to someone")
	}
	memo.Data.AudienceAll = true
	if m, err := f.uc.Create(ctx, memo); err != nil || m.Body != "Kantor libur 20-24 Maret." || m.Subject != "Libur Lebaran" {
		t.Fatalf("memo: %+v err=%v", m, err)
	}
	memo.Body = ""
	if _, err := f.uc.Create(ctx, memo); err == nil {
		t.Fatal("a memo has no standard text, so a body is required")
	}
	sum := f.base(domain.TypeSummons)
	sum.Data = domain.LetterData{MeetingDate: "2026-03-25", MeetingTime: "10:00", Place: "Ruang HRD", Reason: "klarifikasi ketidakhadiran"}
	sl, err := f.uc.Create(ctx, sum)
	if err != nil || !strings.Contains(sl.Body, "Rabu, 25 Maret 2026") || !strings.Contains(sl.Body, "Ruang HRD") {
		t.Fatalf("summons: %+v err=%v", sl, err)
	}
	if _, err := f.uc.Preview(ctx, spl); err != nil {
		t.Fatal(err)
	}
}

func TestDraftEditingAndCancelling(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	in := f.base(domain.TypeReprimand)
	in.Data.Violation = "Merokok di area gudang"
	l, _ := f.uc.Create(ctx, in)
	in.Data.Violation = "Merokok di area produksi"
	up, err := f.uc.Update(ctx, l.ID, in)
	if err != nil || up.ID != l.ID || !strings.Contains(up.Body, "area produksi") {
		t.Fatalf("update re-renders the body: %+v err=%v", up, err)
	}
	if _, err := f.uc.Issue(ctx, l.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Update(ctx, l.ID, in); err == nil {
		t.Fatal("an issued letter is not editable")
	}
	if _, err := f.uc.Cancel(ctx, l.ID, " "); err == nil {
		t.Fatal("cancelling an issued letter needs a reason")
	}
	c, err := f.uc.Cancel(ctx, l.ID, "Salah orang")
	if err != nil || c.Status != domain.StatusCancelled {
		t.Fatalf("cancel: %+v err=%v", c, err)
	}
	if _, err := f.uc.Cancel(ctx, l.ID, "lagi"); err == nil {
		t.Fatal("already cancelled")
	}
}

func TestExpiringContractsSkipsRenewedPermanentAndLeft(t *testing.T) {
	f := newFixture()
	ctx := context.Background()
	mk := func(emp *hrmDomain.Employee, typ, start, end string) {
		in := f.base(domain.TypeContract)
		in.EmployeeID = &emp.ID
		in.EffectiveDate, in.EndDate = start, end
		in.Data = domain.LetterData{ContractType: typ, Position: "Operator", Salary: 5_000_000}
		l, err := f.uc.Create(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.uc.Issue(ctx, l.ID); err != nil {
			t.Fatal(err)
		}
	}
	mk(f.emp, "PKWT", "2025-04-01", "2026-04-15")   // ends in 26 days, not renewed => expiring
	mk(f.other, "PKWT", "2025-01-01", "2026-03-10") // ended 10 days ago => expired
	mk(f.other, "PKWT", "2025-01-01", "2026-03-10")
	mk(f.boss, "PKWTT", "2024-01-01", "") // permanent: never listed
	list, err := f.uc.ExpiringContracts(ctx, 60)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %+v err=%v", list, err)
	}
	if list[0].EmployeeName != "Sari Dewi" || list[0].Status != "expired" || list[0].DaysLeft != -10 || list[1].Status != "expiring" || list[1].DaysLeft != 26 {
		t.Fatalf("order/status: %+v", list)
	}
	mk(f.emp, "PKWT", "2026-04-16", "2027-04-15") // renewal supersedes the old one
	f.other.Status = "Terminated"
	if list, _ = f.uc.ExpiringContracts(ctx, 60); len(list) != 0 {
		t.Fatalf("a renewed contract and a departed employee are not listed: %+v", list)
	}
}

func TestIndonesianFormatting(t *testing.T) {
	if idDate("2026-08-17") != "17 Agustus 2026" || idDayDate("2026-08-17") != "Senin, 17 Agustus 2026" || idDate("x") != "x" {
		t.Fatal("dates")
	}
	for in, want := range map[float64]string{0: "Rp 0", 999: "Rp 999", 1000: "Rp 1.000", 5500000: "Rp 5.500.000", 12345678: "Rp 12.345.678"} {
		if got := rupiah(in); got != want {
			t.Errorf("rupiah(%v) = %q, want %q", in, got, want)
		}
	}
}
