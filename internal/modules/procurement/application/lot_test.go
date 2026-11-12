package application

import (
	"testing"
	"time"
)

func TestGenerateLotNumber(t *testing.T) {
	d := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	if got := GenerateLotNumber(d, 37, 2); got != "LOT-261112-0037-02" {
		t.Fatalf("got %s", got)
	}
}
