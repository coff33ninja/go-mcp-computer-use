package actions

import (
	"testing"
	"time"
)

func TestNormalizeChainSteps_LegacyFocus(t *testing.T) {
	raw := []any{
		map[string]any{"tool": "wait", "args": map[string]any{"ms": float64(50)}},
		map[string]any{"tool": "_focus", "args": map[string]any{"window": "Notepad"}},
		map[string]any{"tool": "focus_window_by_title", "args": map[string]any{"window": "Firefox"}},
	}
	steps := normalizeChainSteps(raw)
	if len(steps) != 3 {
		t.Fatalf("steps=%d", len(steps))
	}
	if steps[0].Tool != "wait" {
		t.Fatalf("step0 tool=%q", steps[0].Tool)
	}
	if steps[1].FocusWindow != "Notepad" || steps[1].Tool != "" {
		t.Fatalf("legacy _focus should become FocusWindow-only, got tool=%q focus=%q", steps[1].Tool, steps[1].FocusWindow)
	}
	if steps[2].Tool != "focus_window_by_title" {
		t.Fatalf("step2 tool=%q", steps[2].Tool)
	}
}

func TestKlEventsToSteps_EmitsRealFocusTool(t *testing.T) {
	steps := klEventsToSteps([]recordedEvent{
		{kind: "focus", keyName: "OpenCode", timestamp: time.Now()},
	})
	if len(steps) != 1 {
		t.Fatalf("steps=%d", len(steps))
	}
	tool, _ := steps[0]["tool"].(string)
	if tool != "focus_window_by_title" {
		t.Fatalf("tool=%q want focus_window_by_title", tool)
	}
	if tool == "_focus" {
		t.Fatal("legacy _focus must not be emitted")
	}
}

func TestToolDispatchHasKeyloggerFocusAliases(t *testing.T) {
	for _, name := range []string{
		"focus_window", "focus_window_by_title", "_focus",
		"keylogger_start", "keylogger_stop", "keylogger_status",
		"replicate", "record_and_replicate",
	} {
		if _, ok := toolDispatch[name]; !ok {
			t.Errorf("toolDispatch missing %q — keylogger/replicate chains will fail", name)
		}
	}
}

func TestEventsToSmartSteps_FocusHasTool(t *testing.T) {
	steps := eventsToSmartSteps([]EnrichedEvent{
		{Kind: "focus", KeyName: "Notepad"},
	})
	if len(steps) != 1 {
		t.Fatalf("steps=%d", len(steps))
	}
	if steps[0].Tool != "focus_window_by_title" {
		t.Fatalf("tool=%q", steps[0].Tool)
	}
	if steps[0].FocusWindow != "Notepad" {
		t.Fatalf("FocusWindow=%q", steps[0].FocusWindow)
	}
}
