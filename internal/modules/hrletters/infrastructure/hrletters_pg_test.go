package infrastructure

import (
	"context"
	"os"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/hrletters/application"
	"github.com/divinecoid/one-backend/internal/modules/hrletters/domain"
	hrmDomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	hrmInfra "github.com/divinecoid/one-backend/internal/modules/hrm/infrastructure"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Runs against a real Postgres when HRLETTERS_TEST_DSN is set (point it at an EMPTY scratch database).
func TestHRLettersAgainstPostgres(t *testing.T) {
	dsn := os.Getenv("HRLETTERS_TEST_DSN")
	if dsn == "" {
		t.Skip("HRLETTERS_TEST_DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Letter{}, &domain.Recipient{}, &hrmDomain.Employee{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	hrm := hrmInfra.NewHRMRepository(db)
	uc := application.NewUseCase(NewRepository(db), hrm)

	mk := func(nip, name, dept, role string) *hrmDomain.Employee {
		e := &hrmDomain.Employee{NIP: nip, Name: name, Department: dept, Role: role, Status: "Active", ContractType: "PKWT", BaseSalary: 5_000_000}
		e.ID = uuid.New()
		if err := hrm.CreateEmployee(ctx, e); err != nil {
			t.Fatal(err)
		}
		return e
	}
	budi, sari, hendra := mk("EMP-001", "Budi", "Produksi", "Operator"), mk("EMP-002", "Sari", "Produksi", "Operator"), mk("EMP-009", "Hendra", "Gudang", "Kepala Gudang")
	base := func(typ string, emp *hrmDomain.Employee) application.LetterInput {
		return application.LetterInput{Type: typ, Date: "2026-03-01", CompanyName: "PT Contoh", SignerName: "Rina", SignerTitle: "HR", EmployeeID: &emp.ID}
	}

	// Round trip of the type-specific JSON data and the computed dates.
	in := base(domain.TypeMutation, budi)
	in.EffectiveDate = "2026-03-01"
	in.Data = domain.LetterData{ToDepartment: "Gudang", ToRole: "Staf Gudang", ToManagerID: hendra.ID.String()}
	l, err := uc.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	got, err := uc.Get(ctx, l.ID)
	if err != nil || got.Data.ToRole != "Staf Gudang" || got.Data.FromDepartment != "Produksi" || got.EffectiveDate != "2026-03-01" || got.Status != "draft" {
		t.Fatalf("round trip: %+v err=%v", got, err)
	}
	issued, err := uc.Issue(ctx, l.ID)
	if err != nil || issued.Number != "001/HRD-MUT/III/2026" || !issued.Applied {
		t.Fatalf("issue: %+v err=%v", issued, err)
	}
	emp, _ := hrm.GetEmployeeByID(ctx, budi.ID)
	if emp.Department != "Gudang" || emp.Role != "Staf Gudang" || emp.ManagerID == nil || *emp.ManagerID != hendra.ID {
		t.Fatalf("employee record after the mutation: %+v", emp)
	}

	// Recipients are stored and replaced on edit.
	spl := application.LetterInput{Type: domain.TypeOvertimeOrder, Date: "2026-03-02", SignerName: "Kepala", RecipientIDs: []uuid.UUID{budi.ID, sari.ID},
		Data: domain.LetterData{WorkDate: "2026-03-03", StartTime: "17:00", EndTime: "20:00", Tasks: "Kejar order"}}
	d, err := uc.Create(ctx, spl)
	if err != nil {
		t.Fatal(err)
	}
	if g, _ := uc.Get(ctx, d.ID); len(g.Recipients) != 2 {
		t.Fatalf("recipients: %+v", g.Recipients)
	}
	spl.RecipientIDs = []uuid.UUID{sari.ID}
	if up, err := uc.Update(ctx, d.ID, spl); err != nil || len(up.Recipients) != 1 {
		t.Fatalf("update: %+v err=%v", up, err)
	}
	if g, _ := uc.Get(ctx, d.ID); len(g.Recipients) != 1 || g.Recipients[0].Name != "Sari" {
		t.Fatalf("recipients after edit: %+v", g.Recipients)
	}

	// Numbering counts only numbered letters, per type and year.
	w := base(domain.TypeWarning, sari)
	w.Level, w.Data.Violation = 1, "Mangkir"
	wl, _ := uc.Create(ctx, w)
	if iw, err := uc.Issue(ctx, wl.ID); err != nil || iw.Number != "001/HRD-SP/III/2026" || iw.EndDate != "2026-09-01" {
		t.Fatalf("warning: %+v err=%v", iw, err)
	}
	wl2, _ := uc.Create(ctx, w)
	if iw, _ := uc.Issue(ctx, wl2.ID); iw.Number != "002/HRD-SP/III/2026" {
		t.Fatalf("second warning: %s", iw.Number)
	}
	if _, err := uc.Cancel(ctx, wl2.ID, "salah ketik"); err != nil {
		t.Fatal(err)
	}
	wl3, _ := uc.Create(ctx, w)
	if iw, _ := uc.Issue(ctx, wl3.ID); iw.Number != "003/HRD-SP/III/2026" {
		t.Fatalf("a cancelled letter keeps its number: %s", iw.Number)
	}

	// Listing and filters.
	all, err := uc.List(ctx, domain.Filter{})
	if err != nil || len(all) != 5 {
		t.Fatalf("list: %d err=%v", len(all), err)
	}
	if only, _ := uc.List(ctx, domain.Filter{Type: domain.TypeWarning, Status: domain.StatusIssued}); len(only) != 2 {
		t.Fatalf("issued warnings: %d", len(only))
	}
	if byEmp, _ := uc.List(ctx, domain.Filter{EmployeeID: &sari.ID}); len(byEmp) != 3 {
		t.Fatalf("by employee: %d", len(byEmp))
	}
	if found, err := uc.List(ctx, domain.Filter{Search: "003/HRD-SP"}); err != nil || len(found) != 1 {
		t.Fatalf("search by number: %d err=%v", len(found), err)
	}
	if found, err := uc.List(ctx, domain.Filter{Search: "emp-002"}); err != nil || len(found) != 3 {
		t.Fatalf("search by NIP (case-insensitive): %d err=%v", len(found), err)
	}

	// Contract: PKWT ending soon shows up, and a termination on the last day marks the employee.
	c := base(domain.TypeContract, sari)
	c.Date, c.EffectiveDate, c.EndDate = "2025-03-01", "2025-03-01", "2026-03-31"
	c.Data = domain.LetterData{ContractType: "PKWT", Position: "Operator", Salary: 5_000_000}
	cl, err := uc.Create(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Issue(ctx, cl.ID); err != nil {
		t.Fatal(err)
	}
	if e, _ := hrm.GetEmployeeByID(ctx, sari.ID); e.ContractType != "PKWT" {
		t.Fatalf("contract type: %s", e.ContractType)
	}
	// The warnings were issued for March and ran for 6 months, so against the real clock they have lapsed;
	// the cancelled one is not listed at all.
	sum, err := uc.EmployeeSummary(ctx, sari.ID)
	if err != nil || len(sum.Letters) != 3 || len(sum.ActiveWarnings) != 0 || sum.HighestActiveLevel != 0 {
		t.Fatalf("summary: %d letters, %d active warnings, err=%v", len(sum.Letters), len(sum.ActiveWarnings), err)
	}
	if _, err := uc.ExpiringContracts(ctx, 3650); err != nil {
		t.Fatal(err)
	}
}
