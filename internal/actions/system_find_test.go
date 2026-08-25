package actions

import (
	"testing"
	"time"
)

func TestSystemFindStats_InitialZero(t *testing.T) {
	systemFindMu.Lock()
	systemFindLastUsed = time.Time{}
	SystemFindCount = 0
	systemFindMu.Unlock()

	lastUsed, count := SystemFindStats()
	if count != 0 {
		t.Errorf("SystemFindStats() count = %d, want 0", count)
	}
	if !lastUsed.IsZero() {
		t.Errorf("SystemFindStats() last_used = %v, want zero time", lastUsed)
	}
}

func TestSystemFindStats_ReturnsCurrentValues(t *testing.T) {
	systemFindMu.Lock()
	systemFindLastUsed = time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	SystemFindCount = 42
	systemFindMu.Unlock()

	lastUsed, count := SystemFindStats()
	if count != 42 {
		t.Errorf("SystemFindStats() count = %d, want 42", count)
	}
	if lastUsed.Year() != 2026 || lastUsed.Month() != 8 || lastUsed.Day() != 22 {
		t.Errorf("SystemFindStats() last_used = %v, want 2026-08-22", lastUsed)
	}
}

func TestSystemFindStats_ReflectsIncrement(t *testing.T) {
	systemFindMu.Lock()
	systemFindLastUsed = time.Time{}
	SystemFindCount = 0
	systemFindMu.Unlock()

	// Simulate a successful find
	systemFindMu.Lock()
	systemFindLastUsed = time.Now()
	SystemFindCount++
	systemFindMu.Unlock()

	lastUsed, count := SystemFindStats()
	if count != 1 {
		t.Errorf("SystemFindStats() count = %d, want 1", count)
	}
	if lastUsed.IsZero() {
		t.Error("SystemFindStats() last_used should not be zero after increment")
	}
}
