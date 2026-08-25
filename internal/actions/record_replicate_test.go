package actions

import (
	"testing"
)

func TestRecordAndReplicate_DurationZero(t *testing.T) {
	_, err := RecordAndReplicate(0, 1000, 1, 1)
	if err == nil {
		t.Fatal("expected error for duration 0")
	}
}

func TestRecordAndReplicate_DurationExceeds60(t *testing.T) {
	_, err := RecordAndReplicate(61, 1000, 1, 1)
	if err == nil {
		t.Fatal("expected error for duration 61")
	}
}

func TestApplySlowdownChain(t *testing.T) {
	steps := []ChainStep{
		{Tool: "wait", Args: map[string]any{"ms": 100}},
		{Tool: "click", Args: map[string]any{"x": 100, "y": 200}},
		{Tool: "wait", Args: map[string]any{"ms": 50}},
	}
	result := applySlowdownChain(steps, 3)
	if len(result) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(result))
	}
	wait1 := result[0].Args["ms"].(int)
	if wait1 != 300 {
		t.Errorf("expected wait 300ms, got %d", wait1)
	}
	if result[1].Tool != "click" {
		t.Errorf("click step modified unexpectedly")
	}
	wait2 := result[2].Args["ms"].(int)
	if wait2 != 150 {
		t.Errorf("expected wait 150ms, got %d", wait2)
	}
}

func TestApplySlowdownChain_OriginalUnmodified(t *testing.T) {
	steps := []ChainStep{
		{Tool: "wait", Args: map[string]any{"ms": 100}},
	}
	applySlowdownChain(steps, 5)
	if steps[0].Args["ms"].(int) != 100 {
		t.Error("original steps should not be modified")
	}
}

func TestDuplicateChainSteps(t *testing.T) {
	steps := []ChainStep{
		{Tool: "click", Args: map[string]any{"x": 1, "y": 2}},
		{Tool: "wait", Args: map[string]any{"ms": 50}},
	}
	result := duplicateChainSteps(steps, 3)
	if len(result) != 6 {
		t.Fatalf("expected 6 steps, got %d", len(result))
	}
	for i := 0; i < 3; i++ {
		idx := i * 2
		if result[idx].Tool != "click" {
			t.Errorf("step %d: expected click, got %v", idx, result[idx].Tool)
		}
		if result[idx+1].Tool != "wait" {
			t.Errorf("step %d: expected wait, got %v", idx+1, result[idx+1].Tool)
		}
	}
}

func TestDuplicateChainSteps_OriginalUnmodified(t *testing.T) {
	steps := []ChainStep{
		{Tool: "click", Args: map[string]any{"x": 1}},
	}
	duplicateChainSteps(steps, 2)
	if len(steps) != 1 {
		t.Error("original slice should not be modified")
	}
}
