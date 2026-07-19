package actions

import (
	"testing"
	"time"
)

func TestParseHotkeyString_SingleKey(t *testing.T) {
	codes := ParseHotkeyString("Escape")
	if len(codes) != 1 {
		t.Fatalf("expected 1 code, got %d", len(codes))
	}
	if codes[0] != 0x1B {
		t.Errorf("expected VK_ESCAPE=0x1B, got 0x%X", codes[0])
	}
}

func TestParseHotkeyString_CtrlShiftEscape(t *testing.T) {
	codes := ParseHotkeyString("Ctrl+Shift+Escape")
	if len(codes) != 3 {
		t.Fatalf("expected 3 codes, got %d", len(codes))
	}
	// Ctrl=0x11, Shift=0x10, Escape=0x1B
	expected := []uint32{0x11, 0x10, 0x1B}
	for i, want := range expected {
		if codes[i] != want {
			t.Errorf("code[%d]: expected 0x%X, got 0x%X", i, want, codes[i])
		}
	}
}

func TestParseHotkeyString_AltF4(t *testing.T) {
	codes := ParseHotkeyString("Alt+F4")
	if len(codes) != 2 {
		t.Fatalf("expected 2 codes, got %d", len(codes))
	}
	if codes[0] != 0x12 {
		t.Errorf("expected VK_ALT=0x12, got 0x%X", codes[0])
	}
	if codes[1] != 0x73 {
		t.Errorf("expected VK_F4=0x73, got 0x%X", codes[1])
	}
}

func TestParseHotkeyString_CaseInsensitive(t *testing.T) {
	codes := ParseHotkeyString("ctrl+shift+escape")
	if len(codes) != 3 {
		t.Fatalf("expected 3 codes, got %d", len(codes))
	}
	expected := []uint32{0x11, 0x10, 0x1B}
	for i, want := range expected {
		if codes[i] != want {
			t.Errorf("code[%d]: expected 0x%X, got 0x%X", i, want, codes[i])
		}
	}
}

func TestParseHotkeyString_WithSpaces(t *testing.T) {
	codes := ParseHotkeyString("Ctrl + Shift + Escape")
	if len(codes) != 3 {
		t.Fatalf("expected 3 codes, got %d", len(codes))
	}
}

func TestParseHotkeyString_EmptyFallsBack(t *testing.T) {
	codes := ParseHotkeyString("")
	// Should fall back to Ctrl+Shift+Escape
	if len(codes) != 3 {
		t.Fatalf("expected fallback to 3 codes, got %d", len(codes))
	}
	expected := []uint32{0x11, 0x10, 0x1B}
	for i, want := range expected {
		if codes[i] != want {
			t.Errorf("code[%d]: expected 0x%X, got 0x%X", i, want, codes[i])
		}
	}
}

func TestParseHotkeyString_UnknownKeyFallsBack(t *testing.T) {
	codes := ParseHotkeyString("Xyzzy+Blah")
	// Both unknown, falls back to Ctrl+Shift+Escape
	if len(codes) != 3 {
		t.Fatalf("expected fallback to 3 codes, got %d", len(codes))
	}
}

func TestParseHotkeyString_F12(t *testing.T) {
	codes := ParseHotkeyString("Ctrl+F12")
	if len(codes) != 2 {
		t.Fatalf("expected 2 codes, got %d", len(codes))
	}
	if codes[1] != 0x7B {
		t.Errorf("expected VK_F12=0x7B, got 0x%X", codes[1])
	}
}

func TestAbortChannelLifecycle(t *testing.T) {
	// Fresh state: not aborted
	ResetAbortChannel()
	if IsChainAborted() {
		t.Error("expected not aborted initially")
	}

	// Get channel, should be readable and not closed
	ch := GetAbortChannel()
	if ch == nil {
		t.Fatal("expected non-nil channel")
	}
	select {
	case <-ch:
		t.Error("channel should not be closed yet")
	default:
	}

	// Close it (simulate abort trigger)
	ResetAbortChannel()
	// After reset, new channel is created
	ch2 := GetAbortChannel()
	select {
	case <-ch2:
		t.Error("new channel should not be closed")
	default:
	}
}

func TestAbortChannelCloseSignal(t *testing.T) {
	ResetAbortChannel()
	ch := GetAbortChannel()

	// Close the channel to signal abort
	close(abortCh)

	if !IsChainAborted() {
		t.Error("expected aborted after channel close")
	}

	// GetAbortChannel should create a fresh one
	ch2 := GetAbortChannel()
	if ch2 == ch {
		t.Error("expected new channel after close")
	}
	if IsChainAborted() {
		t.Error("expected not aborted with fresh channel")
	}
}

func TestStartStopAbortPoller(t *testing.T) {
	codes := []uint32{0x11, 0x10, 0x1B} // Ctrl+Shift+Escape
	StopAbortPoller()
	StartAbortPoller(codes, 50)

	// Should be running (no panic on double start)
	StartAbortPoller(codes, 50)

	// Stop should be clean
	StopAbortPoller()
	StopAbortPoller() // double stop is safe
}

func TestAbortPollerResetsOnStart(t *testing.T) {
	ResetAbortChannel()
	// Simulate an abort
	abortCh = make(chan struct{})
	close(abortCh)

	if !IsChainAborted() {
		t.Error("expected aborted before restart")
	}

	// Starting poller should reset
	codes := []uint32{0x11, 0x10, 0x1B}
	StartAbortPoller(codes, 50)

	// Give it a moment to settle
	time.Sleep(20 * time.Millisecond)

	// The poller should have reset the channel
	// (it calls GetAbortChannel which resets closed channels)
	StopAbortPoller()
}

func TestIsScreenTool(t *testing.T) {
	screenTools := []string{
		"click", "move_mouse", "scroll", "drag", "hover",
		"type", "type_and_submit", "select_all_and_type",
		"key_press", "key_down", "key_up",
		"screenshot", "screenshot_element", "ocr", "ocr_window", "ocr_active_window",
		"find_text_and_click", "click_menu_item",
		"uia_find", "uia_get_text", "uia_invoke", "uia_set_text",
		"uia_get_element_at_point", "uia_get_all_elements",
		"find_image", "find_all_images", "onnx_detect", "find_ui_element",
		"browser_focus_url_bar", "browser_new_tab", "browser_navigate", "browser_search",
		"dismiss_all_menus", "keylogger_start",
		"wait_for_ui_element", "layout_validate", "template_store", "template_find",
	}
	nonScreenTools := []string{
		"get_cursor_position", "get_screen_size", "get_system_info",
		"get_battery", "get_volume", "set_volume",
		"list_windows", "find_window", "get_window_state",
		"ping", "get_network_info", "get_disk_usage",
		"memory_set", "memory_get", "memory_search",
		"chain", "chain_abort", "set_window_lock", "clear_window_lock",
		"set_config", "get_active_window",
	}

	for _, name := range screenTools {
		if !IsScreenTool(name) {
			t.Errorf("expected %q to be a screen tool", name)
		}
	}
	for _, name := range nonScreenTools {
		if IsScreenTool(name) {
			t.Errorf("expected %q to NOT be a screen tool", name)
		}
	}
}
