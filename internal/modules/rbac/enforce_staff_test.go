package rbac

import (
	"testing"

	userDomain "github.com/divinecoid/one-backend/internal/modules/user/domain"
)

func TestStaffBlockedFromPayroll(t *testing.T) {
	cases := []struct {
		path string
		role userDomain.UserRole
		want bool
	}{
		{"/api/v1/payroll/entries", userDomain.RoleStaff, true},
		{"/api/v1/payroll", userDomain.RoleStaff, true},
		{"/api/v1/payroll/entries/abc/payslip", userDomain.RoleStaff, true},
		{"/api/v1/payroll/entries", userDomain.RoleManager, false},
		{"/api/v1/payroll/entries", userDomain.RoleAdmin, false},
		{"/api/v1/payrollx", userDomain.RoleStaff, false},
		{"/api/v1/hr-self/payslips", userDomain.RoleStaff, false},
		{"/api/v1/leaves", userDomain.RoleStaff, false},
	}
	for _, c := range cases {
		if got := staffBlockedFromPath(c.path, c.role); got != c.want {
			t.Errorf("%s as %s: got %v want %v", c.path, c.role, got, c.want)
		}
	}
}
