package actor

import (
	"context"
	"strings"
)

// TradeDocRoles are the role names (legacy account role or assigned RBAC role
// name) allowed to create and read price documents - the PI/CI menu: superadmin
// (Ko Steven), admin (Mba Nadin) and the accountants.
var TradeDocRoles = []string{"admin", "superadmin", "owner", "finance", "akuntan", "accountant", "accounting"}

// HasAnyRole reports whether the caller carries one of the given role names.
func HasAnyRole(ctx context.Context, names ...string) bool {
	for _, have := range RolesFrom(ctx) {
		for _, want := range names {
			if strings.EqualFold(have, want) {
				return true
			}
		}
	}
	return false
}

// MaySeePrices reports whether selling prices may be shown to the caller. Staff
// only receive the work instruction (what, how much, when) and never the prices.
// A context without roles (seeds, background jobs, tests) is not a staff request
// and is allowed.
func MaySeePrices(ctx context.Context) bool {
	if len(RolesFrom(ctx)) == 0 {
		return true
	}
	return HasAnyRole(ctx, append([]string{"manager"}, TradeDocRoles...)...)
}

// MaySetPrices reports whether the caller may enter or change selling prices.
func MaySetPrices(ctx context.Context) bool {
	if len(RolesFrom(ctx)) == 0 {
		return true
	}
	return HasAnyRole(ctx, TradeDocRoles...)
}
