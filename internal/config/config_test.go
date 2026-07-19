package config

import "testing"

func TestToolEnabledNoDenylist(t *testing.T) {
	cfg := Default()
	if !cfg.ToolEnabled("click") {
		t.Error("expected click to be enabled with empty denylist")
	}
	if !cfg.ToolEnabled("shutdown") {
		t.Error("expected shutdown to be enabled with empty denylist")
	}
}

func TestToolEnabledDenylist(t *testing.T) {
	cfg := Default()
	cfg.ToolDenylist = []string{"shutdown", "restart", "hibernate"}

	if !cfg.ToolEnabled("click") {
		t.Error("expected click to be enabled")
	}
	if !cfg.ToolEnabled("screenshot") {
		t.Error("expected screenshot to be enabled")
	}
	if cfg.ToolEnabled("shutdown") {
		t.Error("expected shutdown to be disabled")
	}
	if cfg.ToolEnabled("restart") {
		t.Error("expected restart to be disabled")
	}
	if cfg.ToolEnabled("hibernate") {
		t.Error("expected hibernate to be disabled")
	}
}

func TestToolEnabledCaseInsensitive(t *testing.T) {
	cfg := Default()
	cfg.ToolDenylist = []string{"Shutdown"}

	if cfg.ToolEnabled("shutdown") {
		t.Error("expected shutdown to be disabled (case-insensitive)")
	}
	if cfg.ToolEnabled("SHUTDOWN") {
		t.Error("expected SHUTDOWN to be disabled (case-insensitive)")
	}
	if cfg.ToolEnabled("Shutdown") {
		t.Error("expected Shutdown to be disabled (case-insensitive)")
	}
}

func TestToolEnabledEmptyName(t *testing.T) {
	cfg := Default()
	cfg.ToolDenylist = []string{"click"}

	if !cfg.ToolEnabled("") {
		t.Error("expected empty name to be enabled (not in denylist)")
	}
}

func TestToolEnabledNilDenylist(t *testing.T) {
	cfg := &Config{}
	if !cfg.ToolEnabled("click") {
		t.Error("expected click to be enabled with nil denylist")
	}
}

func TestDefaultChainAbort(t *testing.T) {
	cfg := Default()
	if !cfg.ChainAbortEnabled {
		t.Error("expected ChainAbortEnabled=true by default")
	}
	if cfg.ChainAbortKeys != "Ctrl+Shift+Escape" {
		t.Errorf("expected ChainAbortKeys='Ctrl+Shift+Escape', got %q", cfg.ChainAbortKeys)
	}
	if cfg.ChainAbortPollMs != 50 {
		t.Errorf("expected ChainAbortPollMs=50, got %d", cfg.ChainAbortPollMs)
	}
}

func TestDefaultWindowLock(t *testing.T) {
	cfg := Default()
	if cfg.WindowLockEnabled {
		t.Error("expected WindowLockEnabled=false by default")
	}
	if !cfg.WindowLockAutoFocus {
		t.Error("expected WindowLockAutoFocus=true by default")
	}
}

func TestConfigJSONRoundTrip(t *testing.T) {
	cfg := Default()
	cfg.ChainAbortKeys = "Ctrl+Alt+Delete"
	cfg.ChainAbortPollMs = 100
	cfg.WindowLockEnabled = true
	cfg.WindowLockAutoFocus = false

	data, err := cfg.SaveToBytes()
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := LoadFromBytes(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.ChainAbortKeys != "Ctrl+Alt+Delete" {
		t.Errorf("expected ChainAbortKeys='Ctrl+Alt+Delete', got %q", loaded.ChainAbortKeys)
	}
	if loaded.ChainAbortPollMs != 100 {
		t.Errorf("expected ChainAbortPollMs=100, got %d", loaded.ChainAbortPollMs)
	}
	if !loaded.WindowLockEnabled {
		t.Error("expected WindowLockEnabled=true after roundtrip")
	}
	if loaded.WindowLockAutoFocus {
		t.Error("expected WindowLockAutoFocus=false after roundtrip")
	}
}
