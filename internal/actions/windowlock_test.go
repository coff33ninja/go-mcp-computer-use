package actions

import (
	"testing"
)

func TestWindowLockLifecycle(t *testing.T) {
	// Clear any existing lock
	ClearWindowLock()

	handle, title, pid, locked := GetWindowLock()
	if locked {
		t.Error("expected not locked initially")
	}
	if handle != 0 {
		t.Errorf("expected 0 handle, got %d", handle)
	}
	if title != "" {
		t.Errorf("expected empty title, got %q", title)
	}
	if pid != 0 {
		t.Errorf("expected 0 pid, got %d", pid)
	}
}

func TestWindowLockInvalidHandle(t *testing.T) {
	ClearWindowLock()
	err := SetWindowLock(0)
	if err == nil {
		t.Error("expected error for handle 0")
	}

	err = SetWindowLock(999999)
	if err == nil {
		t.Error("expected error for non-existent window handle")
	}
}

func TestGetWindowLockTitle(t *testing.T) {
	ClearWindowLock()
	if got := GetWindowLockTitle(); got != "" {
		t.Errorf("expected empty title, got %q", got)
	}
}

func TestGetWindowLockPID(t *testing.T) {
	ClearWindowLock()
	if got := GetWindowLockPID(); got != 0 {
		t.Errorf("expected 0 pid, got %d", got)
	}
}

func TestWindowLockClearIsIdempotent(t *testing.T) {
	ClearWindowLock()
	ClearWindowLock() // should not panic
}

func TestVerifyWindowLockNoLock(t *testing.T) {
	ClearWindowLock()
	// No lock = always valid (nothing to check)
	valid, _ := VerifyWindowLock(false)
	if !valid {
		t.Error("expected valid when no lock is set")
	}
}
