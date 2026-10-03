package rbac

import (
	"net/http"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/rbac/domain"
	userDomain "github.com/divinecoid/one-backend/internal/modules/user/domain"
	"github.com/google/uuid"
)

func TestApprovalModuleFor(t *testing.T) {
	id := "3f1c2b1e-0000-4000-8000-000000000001"
	approve := []struct{ method, path, module string }{
		{http.MethodPost, "/api/v1/leaves/" + id + "/approve", "hrm"},
		{http.MethodPost, "/api/v1/leaves/" + id + "/reject", "hrm"},
		{http.MethodPost, "/api/v1/reimbursements/" + id + "/mark-paid", "hrm"},
		{http.MethodPut, "/api/v1/payroll/entries/" + id + "/status", "hrm"},
		{http.MethodPost, "/api/v1/payroll/entries/calculate", "hrm"},
		{http.MethodPut, "/api/v1/kpi/" + id + "/status", "hrm"},
		{http.MethodPost, "/api/v1/commission/records/" + id + "/approve", "commission"},
		{http.MethodPost, "/api/v1/sales/orders/" + id + "/approve", "sales"},
		{http.MethodPost, "/api/v1/procurement/orders/" + id + "/reject", "procurement"},
		{http.MethodPost, "/api/v1/procurement/requests/" + id + "/approve", "procurement"},
		{http.MethodPost, "/api/v1/inventory/opname/" + id + "/finalize", "inventory"},
		{http.MethodPost, "/api/v1/finance/journal-entries/" + id + "/post", "finance"},
		{http.MethodPost, "/api/v1/finance/petty-cash/" + id + "/transactions/" + id + "/approve", "finance"},
		{http.MethodPost, "/api/v1/manufacturing/production-orders/" + id + "/release", "manufacturing"},
		{http.MethodPost, "/api/v1/recruitment/candidates/" + id + "/hire", "recruitment"},
		{http.MethodPost, "/api/v1/leaves/" + id + "/approve/", "hrm"}, // trailing slash
	}
	for _, c := range approve {
		if m, ok := approvalModuleFor(c.method, c.path); !ok || m != c.module {
			t.Errorf("%s %s: got (%q,%v), want (%q,true)", c.method, c.path, m, ok, c.module)
		}
	}

	notApproval := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/leaves/"},                              // creating a request is not approving
		{http.MethodGet, "/api/v1/leaves/" + id + "/approve"},             // wrong method
		{http.MethodPost, "/api/v1/approval/requests/" + id + "/approve"}, // Approval Center has its own role match
		{http.MethodPost, "/api/v1/sales/invoices/" + id + "/payments"},
		{http.MethodPost, "/api/v1/leaves/" + id + "/approve/extra"},
	}
	for _, c := range notApproval {
		if m, ok := approvalModuleFor(c.method, c.path); ok {
			t.Errorf("%s %s should not be an approval action (matched %q)", c.method, c.path, m)
		}
	}
}

func TestMayApprove(t *testing.T) {
	role := uuid.New()
	user := func(r userDomain.UserRole, custom bool) *userDomain.User {
		u := &userDomain.User{Role: r}
		if custom {
			u.RoleID = &role
		}
		return u
	}
	cases := []struct {
		name    string
		jwtRole string
		user    *userDomain.User
		perm    *domain.Permission
		want    bool
	}{
		{"JWT admin always", "admin", nil, nil, true},
		{"legacy admin", "staff", user(userDomain.RoleAdmin, false), nil, true},
		{"legacy manager", "manager", user(userDomain.RoleManager, false), nil, true},
		{"legacy staff denied", "staff", user(userDomain.RoleStaff, false), nil, false},
		{"unknown user denied", "manager", nil, nil, false},
		{"custom role with CanApprove", "staff", user(userDomain.RoleStaff, true), &domain.Permission{CanApprove: true}, true},
		{"custom role, manage only", "manager", user(userDomain.RoleManager, true), &domain.Permission{CanView: true, CanManage: true}, false},
		{"custom role, no permission row", "manager", user(userDomain.RoleManager, true), nil, false},
	}
	for _, c := range cases {
		if got := mayApprove(c.jwtRole, c.user, c.perm); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestModuleFromPathAliases(t *testing.T) {
	cases := map[string]string{
		"/api/v1/leaves/":                   "hrm",
		"/api/v1/reimbursements/abc":        "hrm",
		"/api/v1/payroll/entries":           "hrm",
		"/api/v1/products":                  "product",
		"/api/v1/reports/executive-summary": "report",
		"/api/v1/projects/1":                "project",
		"/api/v1/finance/accounts":          "finance",
		"/api/v1/sales/orders":              "sales",
	}
	for path, want := range cases {
		if got := moduleFromPath(path); got != want {
			t.Errorf("moduleFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}
