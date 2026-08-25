package actions

import (
	"sync"
	"testing"
)

func TestComputeSignalLevel(t *testing.T) {
	tests := []struct {
		detCount   int
		category   string
		taskPrompt string
		want       int
	}{
		{0, "", "", 0},
		{0, "click", "", 0},
		{1, "", "", 1},
		{5, "", "", 1},
		{1, "click", "", 2},
		{1, "", "find button", 2},
		{3, "navigate", "go to settings", 2},
		{0, "click", "click the button", 0},
	}
	for _, tc := range tests {
		got := computeSignalLevel(tc.detCount, tc.category, tc.taskPrompt)
		if got != tc.want {
			t.Errorf("computeSignalLevel(%d, %q, %q) = %d, want %d",
				tc.detCount, tc.category, tc.taskPrompt, got, tc.want)
		}
	}
}

func TestJoinWhere(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{nil, "1=1"},
		{[]string{}, "1=1"},
		{[]string{"source = 'raw'"}, "source = 'raw'"},
		{[]string{"a = 1", "b = 2"}, "a = 1 AND b = 2"},
		{[]string{"x", "y", "z"}, "x AND y AND z"},
	}
	for _, tc := range tests {
		got := joinWhere(tc.input)
		if got != tc.want {
			t.Errorf("joinWhere(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestStopRetentionPruner_NoopWhenNotStarted(t *testing.T) {
	retentionStop = nil
	retentionOnce = sync.Once{}

	StopRetentionPruner()

	if retentionStop != nil {
		t.Error("retentionStop should remain nil when no pruner is running")
	}
}

func TestStopRetentionPruner_StopsGoroutine(t *testing.T) {
	retentionStop = nil
	retentionOnce = sync.Once{}

	StartRetentionPruner(30)

	if retentionStop == nil {
		t.Fatal("retentionStop should be set after StartRetentionPruner")
	}

	StopRetentionPruner()

	if retentionStop != nil {
		t.Error("retentionStop should be nil after StopRetentionPruner")
	}
}

func TestStartRetentionPruner_NoopWhenZeroDays(t *testing.T) {
	retentionStop = nil
	retentionOnce = sync.Once{}

	StartRetentionPruner(0)

	// Should not have started the goroutine
	if retentionStop != nil {
		t.Error("StartRetentionPruner(0) should not start pruner")
	}
}

func TestStartRetentionPruner_SkipsWhenDisabled(t *testing.T) {
	retentionStop = nil
	retentionOnce = sync.Once{}
	ActiveConfig = nil

	StartRetentionPruner(7)

	if retentionStop == nil {
		t.Error("StartRetentionPruner should start goroutine regardless of ActiveConfig")
	}

	StopRetentionPruner()
}
