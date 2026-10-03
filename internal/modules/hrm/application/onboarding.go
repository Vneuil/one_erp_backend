package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/hrm/domain"
	"github.com/google/uuid"
)

// MemberIdentity describes a company member for HR onboarding.
type MemberIdentity struct {
	Name     string
	Email    string
	Role     string // legacy account role: admin / manager / staff
	TenantID *uuid.UUID
}

// EnsureEmployeeForMember gives a company member an HR employee record, matched
// by email, if they do not have one. It reports whether it created one.
//
// What it knows is only what the account holds: name, email and role. The record
// is a starting point - salary is left at 0, tax status defaults to TK/0 - and
// HR completes it. It never overwrites an existing employee.
func EnsureEmployeeForMember(ctx context.Context, repo domain.HRMRepository, m MemberIdentity) (bool, error) {
	email := strings.TrimSpace(m.Email)
	if email == "" {
		return false, fmt.Errorf("member has no email")
	}
	name := strings.TrimSpace(m.Name)
	if name == "" {
		name = nameFromEmail(email)
	}

	department, title := "General", "Staff"
	switch strings.ToLower(m.Role) {
	case "admin":
		department, title = "Management", "Administrator"
	case "manager":
		title = "Manager"
	}

	emp := &domain.Employee{
		TenantID:       m.TenantID,
		Name:           name,
		Email:          email,
		Department:     department,
		Role:           title,
		ContractType:   "PKWTT Tetap",
		JoinDate:       time.Now().Format("2006-01-02"),
		Status:         "Active",
		LeaveQuotaDays: 12,
		PTKPStatus:     "TK/0",
	}
	return repo.CreateEmployeeIfEmailAbsent(ctx, emp)
}

// nameFromEmail turns "budi.santoso@x.com" into "Budi Santoso" for accounts that
// have no display name.
func nameFromEmail(email string) string {
	local := email
	if i := strings.Index(email, "@"); i > 0 {
		local = email[:i]
	}
	parts := strings.FieldsFunc(local, func(r rune) bool { return r == '.' || r == '_' || r == '-' || r == '+' })
	for i, p := range parts {
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	if len(parts) == 0 {
		return email
	}
	return strings.Join(parts, " ")
}
