package actor

import (
	"context"
	"testing"
)

func TestPriceAccessByRole(t *testing.T) {
	bg := context.Background()
	if !MaySeePrices(bg) || !MaySetPrices(bg) {
		t.Fatal("a context without roles (seed/job) is unrestricted")
	}
	staff := WithRoles(bg, []string{"staff"})
	if MaySeePrices(staff) || MaySetPrices(staff) {
		t.Fatal("staff must neither see nor set prices")
	}
	mgr := WithRoles(bg, []string{"manager"})
	if !MaySeePrices(mgr) || MaySetPrices(mgr) {
		t.Fatal("manager sees prices but does not set them")
	}
	// A staff account whose assigned RBAC role is named "Akuntan".
	acct := WithRoles(bg, []string{"staff", "Akuntan"})
	if !MaySeePrices(acct) || !MaySetPrices(acct) || !HasAnyRole(acct, TradeDocRoles...) {
		t.Fatal("accountant role must have price and PI/CI access")
	}
}
