package actions

import (
	"testing"
	"time"
)

func TestRecordAndReplicate_DurationZero(t *testing.T) {
	// duration=0 should return nil session (manual mode), not an error
	result, err := RecordAndReplicate(0, 1000, 1, 1)
	if err != nil {
		t.Fatalf("unexpected error for duration 0: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Session != nil {
		t.Fatal("expected nil session for manual mode")
	}
}

func TestRecordAndReplicate_DurationNegative(t *testing.T) {
	_, err := RecordAndReplicate(-1, 1000, 1, 1)
	if err == nil {
		t.Fatal("expected error for duration -1")
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

func TestSmartClickSteps_UIAPriority(t *testing.T) {
	ev := EnrichedEvent{
		Kind: "click",
		X:    100, Y: 200,
		UIAElement: map[string]any{
			"name":          "Submit",
			"automation_id": "btnSubmit",
			"control_type":  "Button",
		},
		OCRText: "Submit Form",
	}
	steps := smartClickSteps(ev)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "uia_invoke" {
		t.Errorf("expected uia_invoke, got %v", steps[0].Tool)
	}
	if steps[0].Args["name"] != "Submit" {
		t.Errorf("expected name=Submit, got %v", steps[0].Args["name"])
	}
}

func TestSmartClickSteps_OCRFallback(t *testing.T) {
	ev := EnrichedEvent{
		Kind:    "click",
		X:       100, Y: 200,
		OCRText: "Login",
		Window:  "Chrome",
	}
	steps := smartClickSteps(ev)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "find_text_and_click" {
		t.Errorf("expected find_text_and_click, got %v", steps[0].Tool)
	}
	if steps[0].Args["text"] != "Login" {
		t.Errorf("expected text=Login, got %v", steps[0].Args["text"])
	}
}

func TestSmartClickSteps_MLFallback(t *testing.T) {
	ev := EnrichedEvent{
		Kind: "click",
		X:    100, Y: 200,
		MLCoords: &MLPrediction{
			X:          150,
			Y:          250,
			Confidence: 0.8,
		},
	}
	steps := smartClickSteps(ev)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "click" {
		t.Errorf("expected click, got %v", steps[0].Tool)
	}
	if steps[0].Args["x"] != 150 || steps[0].Args["y"] != 250 {
		t.Errorf("expected ML coords (150,250), got (%v,%v)", steps[0].Args["x"], steps[0].Args["y"])
	}
}

func TestSmartClickSteps_RawCoordsFallback(t *testing.T) {
	ev := EnrichedEvent{
		Kind:   "click",
		X:      100, Y: 200,
		Button: "right",
	}
	steps := smartClickSteps(ev)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "click" {
		t.Errorf("expected click, got %v", steps[0].Tool)
	}
	if steps[0].Args["x"] != int32(100) || steps[0].Args["y"] != int32(200) {
		t.Errorf("expected raw coords (100,200), got (%v,%v)", steps[0].Args["x"], steps[0].Args["y"])
	}
	if steps[0].Args["button"] != "right" {
		t.Errorf("expected button=right, got %v", steps[0].Args["button"])
	}
}

func TestSmartClickSteps_LowMLConfidence(t *testing.T) {
	ev := EnrichedEvent{
		Kind: "click",
		X:    100, Y: 200,
		MLCoords: &MLPrediction{
			X:          150,
			Y:          250,
			Confidence: 0.3,
		},
	}
	steps := smartClickSteps(ev)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "click" {
		t.Errorf("expected click (raw coords), got %v", steps[0].Tool)
	}
	if steps[0].Args["x"] != int32(100) || steps[0].Args["y"] != int32(200) {
		t.Errorf("expected raw coords (100,200), got (%v,%v)", steps[0].Args["x"], steps[0].Args["y"])
	}
}

func TestEventsToSmartSteps_PreservesMouseMoves(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "move_mouse", X: 10, Y: 20},
		{Kind: "click", X: 100, Y: 200},
		{Kind: "move_mouse", X: 300, Y: 400},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
	if steps[0].Tool != "move_mouse" {
		t.Errorf("step 0: expected move_mouse, got %v", steps[0].Tool)
	}
	if steps[2].Tool != "move_mouse" {
		t.Errorf("step 2: expected move_mouse, got %v", steps[2].Tool)
	}
}

func TestEventsToSmartSteps_PreservesKeyEvents(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "key_down", KeyName: "W"},
		{Kind: "key_up", KeyName: "W"},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].Tool != "key_down" {
		t.Errorf("step 0: expected key_down, got %v", steps[0].Tool)
	}
	if steps[1].Tool != "key_up" {
		t.Errorf("step 1: expected key_up, got %v", steps[1].Tool)
	}
}

func TestEventsToSmartSteps_FocusWindow(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "focus", KeyName: "Notepad"},
		{Kind: "click", X: 100, Y: 200},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
	if steps[0].FocusWindow != "Notepad" {
		t.Errorf("expected FocusWindow=Notepad, got %v", steps[0].FocusWindow)
	}
}

func TestDistanceBetween(t *testing.T) {
	d := distanceBetween(0, 0, 3, 4)
	if d != 5.0 {
		t.Errorf("expected 5.0, got %f", d)
	}
}

// ── Text accumulation tests ──

func TestEnrichEvents_TextAccumulation(t *testing.T) {
	raw := []map[string]any{
		{"tool": "key_down", "args": map[string]any{"key": "h"}},
		{"tool": "key_up", "args": map[string]any{"key": "h"}},
		{"tool": "key_down", "args": map[string]any{"key": "e"}},
		{"tool": "key_up", "args": map[string]any{"key": "e"}},
		{"tool": "key_down", "args": map[string]any{"key": "l"}},
		{"tool": "key_up", "args": map[string]any{"key": "l"}},
		{"tool": "key_down", "args": map[string]any{"key": "l"}},
		{"tool": "key_up", "args": map[string]any{"key": "l"}},
		{"tool": "key_down", "args": map[string]any{"key": "o"}},
		{"tool": "key_up", "args": map[string]any{"key": "o"}},
	}
	meta := map[string]any{"window_start": "Notepad"}
	events := enrichEvents(raw, meta)

	// Should accumulate into one type event
	typeCount := 0
	for _, ev := range events {
		if ev.Kind == "type" {
			typeCount++
			if ev.Text != "hello" {
				t.Errorf("expected text 'hello', got '%s'", ev.Text)
			}
		}
	}
	if typeCount != 1 {
		t.Fatalf("expected 1 type event, got %d", typeCount)
	}
}

func TestEnrichEvents_ModifierCombo(t *testing.T) {
	raw := []map[string]any{
		{"tool": "key_down", "args": map[string]any{"key": "CTRL"}},
		{"tool": "key_down", "args": map[string]any{"key": "c"}},
		{"tool": "key_up", "args": map[string]any{"key": "c"}},
		{"tool": "key_up", "args": map[string]any{"key": "CTRL"}},
	}
	meta := map[string]any{"window_start": "test"}
	events := enrichEvents(raw, meta)

	// Should produce key_combo events with modifiers, not type
	for _, ev := range events {
		if ev.Kind == "type" {
			t.Error("modifier combo should not produce type event")
		}
		if ev.Kind == "key_combo" && ev.KeyName == "c" {
			// Should have CTRL in modifiers
			hasMod := false
			for _, m := range ev.Modifiers {
				if m == "CTRL" {
					hasMod = true
				}
			}
			if !hasMod {
				t.Errorf("expected CTRL in modifiers, got %v", ev.Modifiers)
			}
			return // found it
		}
	}
	t.Error("expected key_combo event for Ctrl+C")
}

func TestEnrichEvents_TextFlushOnFocus(t *testing.T) {
	raw := []map[string]any{
		{"tool": "key_down", "args": map[string]any{"key": "a"}},
		{"tool": "key_up", "args": map[string]any{"key": "a"}},
		{"tool": "key_down", "args": map[string]any{"key": "b"}},
		{"tool": "key_up", "args": map[string]any{"key": "b"}},
		{"tool": "_focus", "args": map[string]any{"window": "Chrome"}},
		{"tool": "key_down", "args": map[string]any{"key": "c"}},
		{"tool": "key_up", "args": map[string]any{"key": "c"}},
	}
	meta := map[string]any{"window_start": "Notepad"}
	events := enrichEvents(raw, meta)

	var texts []string
	for _, ev := range events {
		if ev.Kind == "type" {
			texts = append(texts, ev.Text)
		}
	}
	if len(texts) != 2 {
		t.Fatalf("expected 2 type events (flushed on focus), got %d", len(texts))
	}
	if texts[0] != "ab" {
		t.Errorf("expected first text 'ab', got '%s'", texts[0])
	}
	if texts[1] != "c" {
		t.Errorf("expected second text 'c', got '%s'", texts[1])
	}
}

func TestEnrichEvents_TextFlushOnClick(t *testing.T) {
	raw := []map[string]any{
		{"tool": "key_down", "args": map[string]any{"key": "x"}},
		{"tool": "key_up", "args": map[string]any{"key": "x"}},
		{"tool": "click", "args": map[string]any{"x": 100, "y": 200}},
		{"tool": "key_down", "args": map[string]any{"key": "y"}},
		{"tool": "key_up", "args": map[string]any{"key": "y"}},
	}
	meta := map[string]any{"window_start": "test"}
	events := enrichEvents(raw, meta)

	var texts []string
	for _, ev := range events {
		if ev.Kind == "type" {
			texts = append(texts, ev.Text)
		}
	}
	if len(texts) != 2 {
		t.Fatalf("expected 2 type events (flushed on click), got %d", len(texts))
	}
	if texts[0] != "x" || texts[1] != "y" {
		t.Errorf("expected ['x','y'], got %v", texts)
	}
}

func TestEventsToSmartSteps_TypeEvent(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "type", Text: "hello world"},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "type" {
		t.Errorf("expected type, got %v", steps[0].Tool)
	}
	if steps[0].Args["text"] != "hello world" {
		t.Errorf("expected 'hello world', got %v", steps[0].Args["text"])
	}
}

func TestEventsToSmartSteps_KeyCombo(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "key_combo", KeyName: "c", Modifiers: []string{"CTRL"}},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "key_press" {
		t.Errorf("expected key_press, got %v", steps[0].Tool)
	}
	keys, ok := steps[0].Args["keys"].([]string)
	if !ok {
		t.Fatalf("expected keys to be []string, got %T", steps[0].Args["keys"])
	}
	if len(keys) != 2 || keys[0] != "Ctrl" || keys[1] != "c" {
		t.Errorf("expected keys=['Ctrl','c'], got %v", keys)
	}
}

// ── VK reverse map tests ──

func TestVKToChar_Roundtrip(t *testing.T) {
	// Test that charToVK and vkToChar are consistent
	for r, cv := range charToVK {
		if cv.shift {
			continue // skip shifted chars, vkToChar only has lowercase
		}
		ch, ok := vkToChar[uint32(cv.vk)]
		if !ok {
			t.Errorf("vkToChar missing VK 0x%02X (char '%c')", cv.vk, r)
			continue
		}
		if ch != r {
			t.Errorf("vkToChar[0x%02X] = '%c', want '%c'", cv.vk, ch, r)
		}
	}
}

func TestIsModifierVK(t *testing.T) {
	tests := []struct{ vk uint32; want bool }{
		{0x10, true},  // Shift
		{0x11, true},  // Ctrl
		{0x12, true},  // Alt
		{0x5B, true},  // LWin
		{0x5C, true},  // RWin
		{0x41, false}, // A
		{0x0D, false}, // Enter
	}
	for _, tt := range tests {
		if got := isModifierVK(tt.vk); got != tt.want {
			t.Errorf("isModifierVK(0x%02X) = %v, want %v", tt.vk, got, tt.want)
		}
	}
}

func TestIsModifierName(t *testing.T) {
	tests := []struct{ name string; want bool }{
		{"CTRL", true}, {"CONTROL", true}, {"ALT", true},
		{"SHIFT", true}, {"LWIN", true}, {"RWIN", true},
		{"ENTER", false}, {"A", false}, {"SPACE", false},
	}
	for _, tt := range tests {
		if got := isModifierName(tt.name); got != tt.want {
			t.Errorf("isModifierName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestEnrichEvents_ShiftUppercase(t *testing.T) {
	raw := []map[string]any{
		{"tool": "key_down", "args": map[string]any{"key": "SHIFT"}},
		{"tool": "key_down", "args": map[string]any{"key": "h"}},
		{"tool": "key_up", "args": map[string]any{"key": "h"}},
		{"tool": "key_up", "args": map[string]any{"key": "SHIFT"}},
		{"tool": "key_down", "args": map[string]any{"key": "i"}},
		{"tool": "key_up", "args": map[string]any{"key": "i"}},
	}
	meta := map[string]any{"window_start": "test"}
	events := enrichEvents(raw, meta)

	for _, ev := range events {
		if ev.Kind == "type" {
			if ev.Text != "Hi" {
				t.Errorf("expected 'Hi' with shift, got '%s'", ev.Text)
			}
			return
		}
	}
	t.Error("expected type event with 'Hi'")
}

func TestEventsToSmartSteps_RightClick(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "click", Button: "right", X: 100, Y: 200},
	}
	steps := eventsToSmartSteps(events)
	// Should have: click(right) + wait(300ms) = 2 steps
	if len(steps) < 2 {
		t.Fatalf("expected at least 2 steps for right-click, got %d", len(steps))
	}
	if steps[0].Tool != "click" {
		t.Errorf("step 0: expected click, got %v", steps[0].Tool)
	}
	if steps[0].Args["button"] != "right" {
		t.Errorf("step 0: expected button=right, got %v", steps[0].Args["button"])
	}
	if steps[1].Tool != "wait" {
		t.Errorf("step 1: expected wait, got %v", steps[1].Tool)
	}
}

func TestEventsToSmartSteps_RightClickWithOCR(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "click", Button: "right", X: 100, Y: 200, OCRText: "Copy"},
	}
	steps := eventsToSmartSteps(events)
	// Should have: click(right) + wait + uia_find + find_text_and_click = 4 steps
	if len(steps) < 4 {
		t.Fatalf("expected at least 4 steps for right-click+OCR, got %d", len(steps))
	}
	if steps[0].Tool != "click" {
		t.Errorf("step 0: expected click, got %v", steps[0].Tool)
	}
	if steps[1].Tool != "wait" {
		t.Errorf("step 1: expected wait, got %v", steps[1].Tool)
	}
	if steps[2].Tool != "uia_find" {
		t.Errorf("step 2: expected uia_find, got %v", steps[2].Tool)
	}
	if steps[3].Tool != "find_text_and_click" {
		t.Errorf("step 3: expected find_text_and_click, got %v", steps[3].Tool)
	}
}

func TestEventsToSmartSteps_MiddleClick(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "click", Button: "middle", X: 300, Y: 400},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "click" {
		t.Errorf("expected click, got %v", steps[0].Tool)
	}
	if steps[0].Args["button"] != "middle" {
		t.Errorf("expected button=middle, got %v", steps[0].Args["button"])
	}
}

func TestEventsToSmartSteps_XButton1(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "click", Button: "xbutton1"},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].Tool != "key_press" {
		t.Errorf("expected key_press, got %v", steps[0].Tool)
	}
	keys := steps[0].Args["keys"].([]string)
	if len(keys) != 2 || keys[0] != "Alt" || keys[1] != "Left" {
		t.Errorf("expected keys=['Alt','Left'], got %v", keys)
	}
}

func TestEventsToSmartSteps_XButton2(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "click", Button: "xbutton2"},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	keys := steps[0].Args["keys"].([]string)
	if len(keys) != 2 || keys[0] != "Alt" || keys[1] != "Right" {
		t.Errorf("expected keys=['Alt','Right'], got %v", keys)
	}
}

func TestEventsToSmartSteps_KeyComboNoMods(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "key_combo", KeyName: "ENTER", Modifiers: nil},
	}
	steps := eventsToSmartSteps(events)
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	keys, ok := steps[0].Args["keys"].([]string)
	if !ok || len(keys) != 1 || keys[0] != "ENTER" {
		t.Errorf("expected keys=['ENTER'], got %v", steps[0].Args["keys"])
	}
}

// ── Window restoration chain tests ──

func TestReplicate_WindowRestoration(t *testing.T) {
	session := &RecordSession{
		Events: []EnrichedEvent{
			{Kind: "click", X: 100, Y: 200},
		},
		WindowsAtStart: []WindowStateInfo{
			{
				Title: "Notepad",
				Rect:  &WindowRect{Left: 10, Top: 20, Width: 800, Height: 600},
			},
			{
				Title:      "Chrome",
				Minimized:  true,
			},
			{
				Title:      "VSCode",
				Maximized: true,
				Rect:       &WindowRect{Left: 0, Top: 0, Width: 1920, Height: 1080},
			},
		},
	}

	result, err := Replicate(session, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	// Should have: find_window + move_window + focus (Notepad) +
	//              find_window + restore_window + focus (Chrome) +
	//              find_window + move_window + maximize_window + focus (VSCode) +
	//              click = 10 steps
	if len(result.Steps) < 10 {
		t.Fatalf("expected at least 10 steps, got %d", len(result.Steps))
	}

	// First step should be find_window for Notepad
	if result.Steps[0].Tool != "find_window" {
		t.Errorf("step 0: expected find_window, got %v", result.Steps[0].Tool)
	}
	if result.Steps[0].Capture != "win_0" {
		t.Errorf("step 0: expected capture=win_0, got %v", result.Steps[0].Capture)
	}

	// Second step should be move_window with variable reference
	if result.Steps[1].Tool != "move_window" {
		t.Errorf("step 1: expected move_window, got %v", result.Steps[1].Tool)
	}
	handle, _ := result.Steps[1].Args["handle"].(string)
	if handle != "${win_0.handle}" {
		t.Errorf("step 1: expected handle=${win_0.handle}, got %v", handle)
	}

	// Chrome should have restore_window (it was minimized)
	for _, step := range result.Steps {
		if step.Tool == "restore_window" {
			h, _ := step.Args["handle"].(string)
			if h == "${win_1.handle}" {
				return // found it
			}
		}
	}
	t.Error("expected restore_window step for Chrome")
}

func TestMergeDoubleClicks_MergesTwoRapidClicks(t *testing.T) {
	now := time.Now()
	events := []EnrichedEvent{
		{Kind: "click", Button: "left", X: 100, Y: 200, Timestamp: now},
		{Kind: "click", Button: "left", X: 100, Y: 200, Timestamp: now.Add(100 * time.Millisecond)},
	}
	merged := mergeDoubleClicks(events, 400)
	if len(merged) != 1 {
		t.Fatalf("expected 1 merged event, got %d", len(merged))
	}
	if merged[0].Kind != "double_click" {
		t.Errorf("expected double_click, got %v", merged[0].Kind)
	}
}

func TestMergeDoubleClicks_NoMergeIfTooSlow(t *testing.T) {
	now := time.Now()
	events := []EnrichedEvent{
		{Kind: "click", Button: "left", X: 100, Y: 200, Timestamp: now},
		{Kind: "click", Button: "left", X: 100, Y: 200, Timestamp: now.Add(500 * time.Millisecond)},
	}
	merged := mergeDoubleClicks(events, 400)
	if len(merged) != 2 {
		t.Fatalf("expected 2 unmerged events, got %d", len(merged))
	}
}

func TestMergeDoubleClicks_NoMergeIfFarApart(t *testing.T) {
	now := time.Now()
	events := []EnrichedEvent{
		{Kind: "click", Button: "left", X: 100, Y: 200, Timestamp: now},
		{Kind: "click", Button: "left", X: 300, Y: 400, Timestamp: now.Add(50 * time.Millisecond)},
	}
	merged := mergeDoubleClicks(events, 400)
	if len(merged) != 2 {
		t.Fatalf("expected 2 unmerged events, got %d", len(merged))
	}
}

func TestMergeDoubleClicks_IgnoresRightClick(t *testing.T) {
	now := time.Now()
	events := []EnrichedEvent{
		{Kind: "click", Button: "right", X: 100, Y: 200, Timestamp: now},
		{Kind: "click", Button: "left", X: 100, Y: 200, Timestamp: now.Add(50 * time.Millisecond)},
	}
	merged := mergeDoubleClicks(events, 400)
	if len(merged) != 2 {
		t.Fatalf("expected 2 unmerged events, got %d", len(merged))
	}
}

func TestDetectLongPress_ConvertsClickWithHighElapsedMs(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "click", Button: "left", X: 100, Y: 200, ElapsedMs: 800},
	}
	detected := detectLongPress(events, 500)
	if len(detected) != 1 {
		t.Fatalf("expected 1 event, got %d", len(detected))
	}
	if detected[0].Kind != "long_press" {
		t.Errorf("expected long_press, got %v", detected[0].Kind)
	}
}

func TestDetectLongPress_KeepsShortClick(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "click", Button: "left", X: 100, Y: 200, ElapsedMs: 200},
	}
	detected := detectLongPress(events, 500)
	if len(detected) != 1 {
		t.Fatalf("expected 1 event, got %d", len(detected))
	}
	if detected[0].Kind != "click" {
		t.Errorf("expected click, got %v", detected[0].Kind)
	}
}

func TestEventsToSmartSteps_LongPress(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "long_press", Button: "left", X: 100, Y: 200, ElapsedMs: 800},
	}
	steps := eventsToSmartSteps(events)
	// Should have: click (smart) + wait (hold duration) = at least 2
	if len(steps) < 2 {
		t.Fatalf("expected at least 2 steps for long_press, got %d", len(steps))
	}
	if steps[0].Tool != "click" {
		t.Errorf("step 0: expected click, got %v", steps[0].Tool)
	}
	waitStep := -1
	for i, s := range steps {
		if s.Tool == "wait" {
			waitStep = i
			break
		}
	}
	if waitStep == -1 {
		t.Fatal("expected a wait step after long_press click")
	}
}

func TestEventsToSmartSteps_DoubleClick(t *testing.T) {
	events := []EnrichedEvent{
		{Kind: "double_click", Button: "left", X: 100, Y: 200, OCRText: "file.txt"},
	}
	steps := eventsToSmartSteps(events)
	// Should have at least: smart click + double click
	if len(steps) < 2 {
		t.Fatalf("expected at least 2 steps for double_click, got %d", len(steps))
	}
	foundDoubleClick := false
	for _, s := range steps {
		if s.Tool == "click" {
			if clicks, ok := s.Args["clicks"].(int); ok && clicks == 2 {
				foundDoubleClick = true
			}
		}
	}
	if !foundDoubleClick {
		t.Error("expected a click step with clicks=2 for double_click event")
	}
}

func TestMenuCache_RoundTrip(t *testing.T) {
	// Clear any existing cache
	menuCacheMu.Lock()
	menuCache = map[menuCacheKey][]menuCacheEntry{}
	menuCacheMu.Unlock()

	cacheMenuItems("TestWindow", 100, 200, []string{"Copy", "Paste", "Cut"})
	entries := lookupMenuItems("TestWindow", 100, 200)
	if len(entries) != 3 {
		t.Fatalf("expected 3 cached entries, got %d", len(entries))
	}
	if entries[0].ItemText != "Copy" {
		t.Errorf("expected first item Copy, got %v", entries[0].ItemText)
	}
	if entries[1].ItemText != "Paste" {
		t.Errorf("expected second item Paste, got %v", entries[1].ItemText)
	}
}

func TestMenuCache_HitCountIncrements(t *testing.T) {
	menuCacheMu.Lock()
	menuCache = map[menuCacheKey][]menuCacheEntry{}
	menuCacheMu.Unlock()

	cacheMenuItems("W", 100, 200, []string{"Copy", "Paste"})
	cacheMenuItems("W", 100, 200, []string{"Copy", "Delete"})
	entries := lookupMenuItems("W", 100, 200)
	for _, e := range entries {
		if e.ItemText == "Copy" && e.HitCount != 2 {
			t.Errorf("expected Copy hit_count=2, got %d", e.HitCount)
		}
		if e.ItemText == "Paste" && e.HitCount != 1 {
			t.Errorf("expected Paste hit_count=1, got %d", e.HitCount)
		}
		if e.ItemText == "Delete" && e.HitCount != 1 {
			t.Errorf("expected Delete hit_count=1, got %d", e.HitCount)
		}
	}
}

func TestMenuCache_FuzzyPosition(t *testing.T) {
	menuCacheMu.Lock()
	menuCache = map[menuCacheKey][]menuCacheEntry{}
	menuCacheMu.Unlock()

	cacheMenuItems("W", 100, 200, []string{"Copy"})
	// Same bucket (50px grid) should match
	entries := lookupMenuItems("W", 110, 210)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry from fuzzy match, got %d", len(entries))
	}
}

func TestMenuCache_EmptyWindow(t *testing.T) {
	entries := lookupMenuItems("", 0, 0)
	if entries != nil {
		t.Errorf("expected nil for empty window, got %v", entries)
	}
}

func TestDoubleClickDetection_FullEnrichEvents(t *testing.T) {
	rawSteps := []map[string]any{
		{"tool": "click", "args": map[string]any{"button": "left", "x": int32(100), "y": int32(200), "elapsed_ms": int64(50)}},
		{"tool": "click", "args": map[string]any{"button": "left", "x": int32(100), "y": int32(200), "elapsed_ms": int64(30)}},
	}
	events := enrichEvents(rawSteps, map[string]any{})
	doubleClicks := 0
	for _, ev := range events {
		if ev.Kind == "double_click" {
			doubleClicks++
		}
	}
	if doubleClicks != 1 {
		t.Errorf("expected 1 double_click event, got %d (events=%d)", doubleClicks, len(events))
	}
}

func TestLongPressDetection_FullEnrichEvents(t *testing.T) {
	rawSteps := []map[string]any{
		{"tool": "click", "args": map[string]any{"button": "left", "x": int32(100), "y": int32(200), "elapsed_ms": int64(800)}},
	}
	events := enrichEvents(rawSteps, map[string]any{})
	longPresses := 0
	for _, ev := range events {
		if ev.Kind == "long_press" {
			longPresses++
		}
	}
	if longPresses != 1 {
		t.Errorf("expected 1 long_press event, got %d", longPresses)
	}
}

func TestLogEnrichPattern_DoubleClick(t *testing.T) {
	// Should not panic — OCR may fail in test env but function must not crash
	LogEnrichPattern("double_click", 100, 200, 0, "TestWindow", true)

	// Verify it was recorded in the adaptive engine
	coordIndex := Adaptive.coordIndex["double_click"]
	if coordIndex == nil {
		t.Fatal("expected coordIndex for double_click")
	}
	if len(coordIndex) == 0 {
		t.Fatal("expected at least 1 coord sample for double_click")
	}
}

func TestLogEnrichPattern_LongPress(t *testing.T) {
	LogEnrichPattern("long_press", 300, 400, 800, "TestWindow", true)

	coordIndex := Adaptive.coordIndex["long_press"]
	if coordIndex == nil {
		t.Fatal("expected coordIndex for long_press")
	}
	if len(coordIndex) == 0 {
		t.Fatal("expected at least 1 coord sample for long_press")
	}
}

func TestLogEnrichPattern_ContextMenu(t *testing.T) {
	LogEnrichPattern("context_menu", 150, 250, 0, "TestWindow", true)

	coordIndex := Adaptive.coordIndex["context_menu"]
	if coordIndex == nil {
		t.Fatal("expected coordIndex for context_menu")
	}
	if len(coordIndex) == 0 {
		t.Fatal("expected at least 1 coord sample for context_menu")
	}
}

func TestLogEnrichPattern_IgnoresUnknownKind(t *testing.T) {
	before := len(Adaptive.coordIndex)
	LogEnrichPattern("unknown_tool", 0, 0, 0, "", true)
	after := len(Adaptive.coordIndex)
	if after != before {
		t.Errorf("expected no new coordIndex entries, got %d new", after-before)
	}
}

func TestLogEnrichPatternsFromSession(t *testing.T) {
	session := &RecordSession{
		Events: []EnrichedEvent{
			{Kind: "double_click", X: 100, Y: 200, Window: "W1"},
			{Kind: "long_press", X: 300, Y: 400, ElapsedMs: 600, Window: "W1"},
			{Kind: "click", Button: "right", X: 150, Y: 250, Window: "W1"},
			{Kind: "type", Text: "hello"}, // should be ignored
		},
	}
	LogEnrichPatternsFromSession(session, true)

	for _, kind := range []string{"double_click", "long_press", "context_menu"} {
		if Adaptive.coordIndex[kind] == nil || len(Adaptive.coordIndex[kind]) == 0 {
			t.Errorf("expected coordIndex for %s", kind)
		}
	}
}

func TestLogEnrichPatternsFromSession_NilSession(t *testing.T) {
	// Should not panic
	LogEnrichPatternsFromSession(nil, true)
}

func TestDetectAndLogEnrichPatterns_DoubleClick(t *testing.T) {
	steps := []ChainStep{
		{Tool: "click", Args: map[string]any{"x": int32(100), "y": int32(200), "clicks": 2}},
	}
	detectAndLogEnrichPatterns(steps)
	if Adaptive.coordIndex["double_click"] == nil || len(Adaptive.coordIndex["double_click"]) == 0 {
		t.Error("expected double_click logged to ML from chain steps")
	}
}

func TestDetectAndLogEnrichPatterns_ContextMenu(t *testing.T) {
	steps := []ChainStep{
		{Tool: "click", Args: map[string]any{"x": int32(100), "y": int32(200), "button": "right"}},
		{Tool: "wait", Args: map[string]any{"ms": 350}},
		{Tool: "uia_find", Args: map[string]any{"control_type": "MenuItem"}},
	}
	detectAndLogEnrichPatterns(steps)
	if Adaptive.coordIndex["context_menu"] == nil || len(Adaptive.coordIndex["context_menu"]) == 0 {
		t.Error("expected context_menu logged to ML from chain steps")
	}
}

func TestDetectAndLogEnrichPatterns_LongPress(t *testing.T) {
	steps := []ChainStep{
		{Tool: "click", Args: map[string]any{"x": int32(100), "y": int32(200)}},
		{Tool: "wait", Args: map[string]any{"ms": 800}},
	}
	detectAndLogEnrichPatterns(steps)
	if Adaptive.coordIndex["long_press"] == nil || len(Adaptive.coordIndex["long_press"]) == 0 {
		t.Error("expected long_press logged to ML from chain steps")
	}
}

func TestExtractCoordsFromArgs_DoubleClick(t *testing.T) {
	argsJSON := `{"x":100,"y":200}`
	pts := extractCoordsFromArgs("double_click", argsJSON)
	if len(pts) != 1 || pts[0].x != 100 || pts[0].y != 200 {
		t.Errorf("expected [{100,200}], got %v", pts)
	}
}

func TestExtractCoordsFromArgs_LongPress(t *testing.T) {
	argsJSON := `{"x":300,"y":400,"elapsed_ms":800}`
	pts := extractCoordsFromArgs("long_press", argsJSON)
	if len(pts) != 1 || pts[0].x != 300 || pts[0].y != 400 {
		t.Errorf("expected [{300,400}], got %v", pts)
	}
}

func TestExtractCoordsFromArgs_ContextMenu(t *testing.T) {
	argsJSON := `{"x":150,"y":250}`
	pts := extractCoordsFromArgs("context_menu", argsJSON)
	if len(pts) != 1 || pts[0].x != 150 || pts[0].y != 250 {
		t.Errorf("expected [{150,250}], got %v", pts)
	}
}
