package application

import (
	"context"
	"testing"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
)

type captureRepo struct {
	domain.HRMRepository
	got     *domain.Employee
	created bool
}

func (c *captureRepo) CreateEmployeeIfEmailAbsent(_ context.Context, e *domain.Employee) (bool, error) {
	c.got = e
	return c.created, nil
}

func TestEnsureEmployeeForMemberBuildsSensibleRecord(t *testing.T) {
	cases := []struct {
		role, dept, title string
	}{
		{"admin", "Management", "Administrator"},
		{"manager", "General", "Manager"},
		{"staff", "General", "Staff"},
		{"", "General", "Staff"},
	}
	for _, c := range cases {
		repo := &captureRepo{created: true}
		created, err := EnsureEmployeeForMember(context.Background(), repo, MemberIdentity{Name: "Ani Wijaya", Email: " Ani@X.com ", Role: c.role})
		if err != nil || !created {
			t.Fatalf("role %q: created=%v err=%v", c.role, created, err)
		}
		e := repo.got
		if e.Department != c.dept || e.Role != c.title || e.Email != "Ani@X.com" || e.Name != "Ani Wijaya" {
			t.Errorf("role %q: unexpected record %+v", c.role, e)
		}
		if e.Status != "Active" || e.LeaveQuotaDays != 12 || e.PTKPStatus != "TK/0" || e.BaseSalary != 0 || e.NIP != "" {
			t.Errorf("role %q: defaults wrong (salary must stay 0, NIP left for the repo): %+v", c.role, e)
		}
	}
}

func TestEnsureEmployeeForMemberNameFallbackAndValidation(t *testing.T) {
	repo := &captureRepo{}
	if _, err := EnsureEmployeeForMember(context.Background(), repo, MemberIdentity{Email: "budi.santoso@x.com"}); err != nil {
		t.Fatal(err)
	}
	if repo.got.Name != "Budi Santoso" {
		t.Fatalf("name derived from email = %q", repo.got.Name)
	}
	if _, err := EnsureEmployeeForMember(context.Background(), repo, MemberIdentity{Name: "No Email"}); err == nil {
		t.Fatal("a member without an email cannot be matched and must be rejected")
	}
}
