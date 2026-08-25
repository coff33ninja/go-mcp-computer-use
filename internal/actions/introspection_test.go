package actions

import (
	"testing"
	"time"
)

func TestTaskIsActive_InitiallyFalse(t *testing.T) {
	taskMu.Lock()
	taskActive = false
	taskMu.Unlock()

	if TaskIsActive() {
		t.Error("TaskIsActive() = true, want false when no task is active")
	}
}

func TestTaskIsActive_TrueDuringTask(t *testing.T) {
	taskMu.Lock()
	taskActive = true
	taskID = 999
	taskDesc = "test task"
	taskStart = time.Now()
	taskMu.Unlock()

	t.Cleanup(func() {
		taskMu.Lock()
		taskActive = false
		taskMu.Unlock()
	})

	if !TaskIsActive() {
		t.Error("TaskIsActive() = false, want true when task is active")
	}
}

func TestTaskIsActive_FalseAfterReset(t *testing.T) {
	taskMu.Lock()
	taskActive = true
	taskMu.Unlock()

	t.Cleanup(func() {
		taskMu.Lock()
		taskActive = false
		taskMu.Unlock()
	})

	if !TaskIsActive() {
		t.Fatal("TaskIsActive() should be true before reset")
	}

	taskMu.Lock()
	taskActive = false
	taskMu.Unlock()

	if TaskIsActive() {
		t.Error("TaskIsActive() = true, want false after reset")
	}
}
