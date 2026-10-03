package application

import (
	"context"
	"github.com/divinecoid/one-backend/internal/shared/actor"
	"testing"

	hrmdomain "github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/divinecoid/one-backend/internal/modules/payroll/domain"
	"github.com/google/uuid"
)

type fakePayrollRepo struct {
	domain.PayrollRepository
	entries []*domain.PayrollEntry
}

func (f *fakePayrollRepo) CreateEntry(_ context.Context, e *domain.PayrollEntry) error {
	f.entries = append(f.entries, e)
	return nil
}
func (f *fakePayrollRepo) ListEntriesByPeriodAndStatus(_ context.Context, period, status string) ([]domain.PayrollEntry, error) {
	var out []domain.PayrollEntry
	for _, e := range f.entries {
		if e.Period == period && e.Status == status {
			out = append(out, *e)
		}
	}
	return out, nil
}
func (f *fakePayrollRepo) GetEntryByID(_ context.Context, id uuid.UUID) (*domain.PayrollEntry, error) {
	for _, e := range f.entries {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, nil
}
func (f *fakePayrollRepo) UpdateEntry(_ context.Context, e *domain.PayrollEntry) error {
	for i, x := range f.entries {
		if x.ID == e.ID {
			f.entries[i] = e
		}
	}
	return nil
}

func (f *fakePayrollRepo) GetPolicy(context.Context) (*domain.PayrollPolicy, error) {
	p := domain.DefaultPayrollPolicy()
	return &p, nil
}

type fakeHRM struct {
	hrmdomain.HRMRepository
	emp *hrmdomain.Employee
}

func (f *fakeHRM) GetEmployeeByID(context.Context, uuid.UUID) (*hrmdomain.Employee, error) {
	return f.emp, nil
}

func newEmployee(ptkp string, noNPWP bool) *hrmdomain.Employee {
	e := &hrmdomain.Employee{NIP: "E1", Name: "Ani", Department: "Ops", BaseSalary: 10_000_000, PTKPStatus: ptkp, NoNPWP: noNPWP}
	e.ID = uuid.New()
	return e
}

func TestAutoTaxUsesEmployeePTKPAtCreateAndRefreshesOnCalculate(t *testing.T) {
	emp := newEmployee("K/1", false)
	repo := &fakePayrollRepo{}
	uc := NewPayrollUseCase(repo, &fakeHRM{emp: emp}, nil)
	ctx := context.Background()

	// A manually supplied DeductionTax must be ignored in auto mode.
	res, err := uc.CreateEntry(ctx, CreatePayrollEntryDTO{Period: "2026-09", EmployeeID: &emp.ID, TaxMethod: "auto", DeductionTax: 999})
	if err != nil {
		t.Fatal(err)
	}
	if res.DeductionTax != 212_500 || res.PTKPStatus != "K/1" || res.TakeHomePay != 10_000_000-212_500 {
		t.Fatalf("create: %+v", res)
	}

	// Overtime added while still a draft: calculating the period re-estimates the tax.
	repo.entries[0].OvertimePay = 5_000_000 // gross 15M/month -> 180M/yr
	if _, err := uc.CalculatePeriod(ctx, "2026-09"); err != nil {
		t.Fatal(err)
	}
	// 180M - 6M job expense - 63M PTKP = 111M PKP -> 3M + 51M*15% = 10.65M/yr -> 887,500/month
	if got := repo.entries[0].DeductionTax; got != 887_500 {
		t.Fatalf("recalculated tax = %.0f, want 887500", got)
	}
	if want := 15_000_000.0 - 887_500; repo.entries[0].TakeHomePay != want {
		t.Fatalf("take-home = %.0f, want %.0f", repo.entries[0].TakeHomePay, want)
	}
}

func TestManualTaxIsUntouched(t *testing.T) {
	emp := newEmployee("TK/0", false)
	repo := &fakePayrollRepo{}
	uc := NewPayrollUseCase(repo, &fakeHRM{emp: emp}, nil)

	res, err := uc.CreateEntry(context.Background(), CreatePayrollEntryDTO{Period: "2026-09", EmployeeID: &emp.ID, DeductionTax: 123_456})
	if err != nil {
		t.Fatal(err)
	}
	if res.TaxMethod != "manual" || res.DeductionTax != 123_456 {
		t.Fatalf("manual entry changed: %+v", res)
	}
	if _, err := uc.CalculatePeriod(context.Background(), "2026-09"); err != nil {
		t.Fatal(err)
	}
	if repo.entries[0].DeductionTax != 123_456 {
		t.Fatalf("calculate overwrote a manual tax: %.0f", repo.entries[0].DeductionTax)
	}
}

func TestNoNPWPSurchargeAndInvalidMethod(t *testing.T) {
	emp := newEmployee("TK/0", true)
	uc := NewPayrollUseCase(&fakePayrollRepo{}, &fakeHRM{emp: emp}, nil)

	res, err := uc.CreateEntry(context.Background(), CreatePayrollEntryDTO{Period: "2026-09", EmployeeID: &emp.ID, TaxMethod: "auto"})
	if err != nil || res.DeductionTax != 300_000 {
		t.Fatalf("no-NPWP tax: %+v err=%v", res, err)
	}
	if _, err := uc.CreateEntry(context.Background(), CreatePayrollEntryDTO{Period: "2026-09", EmployeeID: &emp.ID, TaxMethod: "ter"}); err == nil {
		t.Fatal("expected invalid taxMethod to be rejected")
	}
}

func TestCannotApproveOrPayOwnPayroll(t *testing.T) {
	emp := newEmployee("TK/0", false)
	emp.Email = "ani@x.com"
	repo := &fakePayrollRepo{}
	uc := NewPayrollUseCase(repo, &fakeHRM{emp: emp}, nil)

	created, err := uc.CreateEntry(context.Background(), CreatePayrollEntryDTO{Period: "2026-09", EmployeeID: &emp.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.CalculatePeriod(context.Background(), "2026-09"); err != nil {
		t.Fatal(err)
	}
	repo.entries[0].ID = created.ID // fake repo lookup below relies on this

	own := actor.WithEmail(context.Background(), "ani@x.com")
	if _, err := uc.UpdateStatus(own, created.ID, "approved"); err == nil {
		t.Fatal("an employee must not approve their own payroll")
	}
}
