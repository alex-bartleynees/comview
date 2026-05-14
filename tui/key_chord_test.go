package tui

import (
	"testing"
	"time"
)

func TestKeyChordStateClearsExpiredPendingKeys(t *testing.T) {
	var state keyChordState
	now := time.Now()
	state.Set("g", now.Add(-pendingKeyTimeout-time.Millisecond))

	state.ClearExpired(now)

	if state.Pending() != "" {
		t.Fatalf("pending keys = %q, want empty", state.Pending())
	}
}

func TestKeyChordStateKeepsFreshPendingKeys(t *testing.T) {
	var state keyChordState
	now := time.Now()
	state.Set("g", now.Add(-pendingKeyTimeout+time.Millisecond))

	state.ClearExpired(now)

	if state.Pending() != "g" {
		t.Fatalf("pending keys = %q, want g", state.Pending())
	}
}
