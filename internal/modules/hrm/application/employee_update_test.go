package application

import (
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
)

func TestApplyEmployeeUpdate(t *testing.T) {
	emp := &domain.Employee{NIP: "E1", Name: "Ani", Department: "Ops", BaseSalary: 5_000_000, PTKPStatus: "TK/0"}
	name, sal, ptkp, no := "Ani Putri", 6_000_000.0, "k/1", false
	if err := ApplyEmployeeUpdate(emp, UpdateEmployeeDTO{Name: &name, BaseSalary: &sal, PTKPStatus: &ptkp, HasNPWP: &no}); err != nil {
		t.Fatal(err)
	}
	if emp.Name != "Ani Putri" || emp.BaseSalary != 6_000_000 || emp.PTKPStatus != "K/1" || !emp.NoNPWP {
		t.Fatalf("got %+v", emp)
	}
	if emp.NIP != "E1" || emp.Department != "Ops" {
		t.Fatalf("untouched fields changed: %+v", emp)
	}
	empty, neg, bad := " ", -1.0, "XX"
	for _, dto := range []UpdateEmployeeDTO{{Name: &empty}, {BaseSalary: &neg}, {PTKPStatus: &bad}, {Department: &empty}} {
		if err := ApplyEmployeeUpdate(emp, dto); err == nil {
			t.Fatalf("expected an error for %+v", dto)
		}
	}
}
