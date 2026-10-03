package application

import (
	"testing"
	"time"

	"github.com/divinecoid/one-backend/internal/modules/device/domain"
)

func TestEffectiveStatusDowngradesStaleOnlineDevices(t *testing.T) {
	now := time.Now()
	recent, stale := now.Add(-2*time.Minute), now.Add(-30*time.Minute)

	cases := []struct {
		name string
		dev  domain.Device
		want string
	}{
		{"online with a recent heartbeat", domain.Device{Status: "online", LastHeartbeat: &recent}, "online"},
		{"online but silent for 30 minutes", domain.Device{Status: "online", LastHeartbeat: &stale}, "offline"},
		{"online that never sent a heartbeat", domain.Device{Status: "online"}, "offline"},
		{"explicit offline stays offline", domain.Device{Status: "offline", LastHeartbeat: &recent}, "offline"},
		{"syncing is reported as stored", domain.Device{Status: "syncing", LastHeartbeat: &stale}, "syncing"},
	}
	for _, c := range cases {
		d := c.dev
		if got := effectiveStatus(&d, now); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
