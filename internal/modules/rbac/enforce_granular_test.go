package rbac

import (
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/rbac/domain"
)

func TestIsSubmitPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/api/v1/sales/orders/123/submit", true},
		{"/api/v1/sales/orders/123/submit/", true},
		{"/api/v1/procurement/requests/abc-def/submit", true},
		{"/api/v1/procurement/orders/abc-def/submit", true},
		{"/api/v1/sales/orders", false},
		{"/api/v1/sales/orders/submit/extra", false},
		{"/api/v1/procurement/orders/123/approve", false},
	}

	for _, c := range cases {
		if got := isSubmitPath(c.path); got != c.want {
			t.Errorf("isSubmitPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestGranularPermissionsDefaults(t *testing.T) {
	perm := domain.Permission{
		Module:        "sales",
		CanView:       true,
		CanCreate:     true,
		CanEdit:       true,
		CanDelete:     false,
		CanSubmit:     true,
		CanApprove:    false,
		CanViewPrices: true,
		CanEditPrices: false,
	}

	if !perm.CanView || !perm.CanCreate || !perm.CanEdit || !perm.CanSubmit || !perm.CanViewPrices {
		t.Errorf("expected view, create, edit, submit, and view prices to be true")
	}
	if perm.CanDelete || perm.CanApprove || perm.CanEditPrices {
		t.Errorf("expected delete, approve, and edit prices to be false")
	}
}
