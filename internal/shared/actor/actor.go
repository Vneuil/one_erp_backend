// Package actor carries the authenticated caller's identity through a
// request context, so use cases can make ownership decisions (for example
// "you cannot approve your own request") without every method taking an
// extra parameter.
package actor

import (
	"context"
	"strings"
)

type emailKey struct{}
type rolesKey struct{}

// WithEmail returns ctx carrying the caller's email address.
func WithEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, emailKey{}, email)
}

// EmailFrom returns the caller's email, or "" when the context carries none
// (background jobs, seeds, tests).
func EmailFrom(ctx context.Context) string {
	email, _ := ctx.Value(emailKey{}).(string)
	return email
}

// WithRoles returns ctx carrying the caller's effective role names (legacy role
// and/or assigned RBAC role name).
func WithRoles(ctx context.Context, roles []string) context.Context {
	seen := make(map[string]bool, len(roles))
	clean := make([]string, 0, len(roles))
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		key := strings.ToLower(r)
		if !seen[key] {
			seen[key] = true
			clean = append(clean, r)
		}
	}
	return context.WithValue(ctx, rolesKey{}, clean)
}

// RolesFrom returns the caller's effective role names stored on ctx.
func RolesFrom(ctx context.Context) []string {
	roles, _ := ctx.Value(rolesKey{}).([]string)
	return roles
}
