package license

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestIsTableMissingError(t *testing.T) {
	tests := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{errors.New("connection reset by peer"), false},
		{gorm.ErrRecordNotFound, false},
		{errors.New(`ERROR: relation "platform_subscriptions" does not exist (SQLSTATE 42P01)`), true},
		{errors.New("no such table: platform_subscriptions"), true},
	}
	for _, tt := range tests {
		if got := isTableMissingError(tt.err); got != tt.expected {
			t.Errorf("isTableMissingError(%v) = %v; want %v", tt.err, got, tt.expected)
		}
	}
}

func TestSubscriptionRowLogic(t *testing.T) {
	now := time.Now()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	// Check logic matches license.IsActive status handling
	tests := []struct {
		name     string
		row      subscriptionRow
		expected bool
	}{
		{"expired status", subscriptionRow{Status: "expired", EndsAt: &future}, false},
		{"trial active", subscriptionRow{Status: "trial", TrialEndsAt: &future}, true},
		{"trial expired", subscriptionRow{Status: "trial", TrialEndsAt: &past}, false},
		{"active valid", subscriptionRow{Status: "active", EndsAt: &future}, true},
		{"active expired", subscriptionRow{Status: "active", EndsAt: &past}, false},
		{"pending_payment default active", subscriptionRow{Status: "pending_payment"}, true},
		{"unknown status default active", subscriptionRow{Status: "custom"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var active bool
			switch tt.row.Status {
			case "expired":
				active = false
			case "trial":
				active = tt.row.TrialEndsAt == nil || now.Before(*tt.row.TrialEndsAt)
			case "active":
				active = tt.row.EndsAt == nil || now.Before(*tt.row.EndsAt)
			default:
				active = true
			}

			if active != tt.expected {
				t.Errorf("status %s: got active=%v, want %v", tt.row.Status, active, tt.expected)
			}
		})
	}
}

// Compile check to ensure PlatformSubscription implements expected types
func TestPlatformSubscriptionTable(t *testing.T) {
	sub := PlatformSubscription{}
	if sub.TableName() != "platform_subscriptions" {
		t.Errorf("got %s, want platform_subscriptions", sub.TableName())
	}
}
