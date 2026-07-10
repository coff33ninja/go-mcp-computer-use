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
