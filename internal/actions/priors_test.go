package actions

import (
	"testing"
)

func TestNormalizeWindowTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "__empty__"},
		{"  ", "__empty__"},
		{"Notepad", "notepad"},
		{"Notepad - Untitled", "notepad"},
		{"Chrome", "chrome"},
		{"Firefox", "firefox"},
		{"Code - main.go", "code"},
		{"Settings", "settings"},
		{"Terminal", "terminal"},
		{"Calculator", "calculator"},
		{"MyCustomApp v1.0", "mycustomapp v1.0"},
		{"A very long title that exceeds forty characters and should be truncated", "a very long title that exceeds forty cha"},
	}
	for _, tc := range tests {
		got := normalizeWindowTitle(tc.input)
		if got != tc.want {
			t.Errorf("normalizeWindowTitle(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, min, max, want float64
	}{
		{0.5, 0.0, 1.0, 0.5},
		{-0.1, 0.0, 1.0, 0.0},
		{1.5, 0.0, 1.0, 1.0},
		{0.0, 0.0, 1.0, 0.0},
		{1.0, 0.0, 1.0, 1.0},
		{5.0, -5.0, 5.0, 5.0},
	}
	for _, tc := range tests {
		got := clamp(tc.v, tc.min, tc.max)
		if got != tc.want {
			t.Errorf("clamp(%v, %v, %v) = %v, want %v", tc.v, tc.min, tc.max, got, tc.want)
		}
	}
}

func TestJoinNames(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{}, ""},
		{[]string{"person"}, `"person"`},
		{[]string{"person", "car"}, `"person", "car"`},
		{[]string{"a", "b", "c"}, `"a", "b", "c"`},
	}
	for _, tc := range tests {
		got := joinNames(tc.input)
		if got != tc.want {
			t.Errorf("joinNames(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func seedPriors(priors []ElementPrior) {
	elementPriors.mu.Lock()
	elementPriors.priors = priors
	elementPriors.mu.Unlock()
}

func TestElementKnownConfidently(t *testing.T) {
	seedPriors([]ElementPrior{
		{Class: "person", WindowTitle: "chrome", SampleCount: 20, AvgX: 0.5, AvgY: 0.5, Frequency: 0.9},
		{Class: "button", WindowTitle: "chrome", SampleCount: 2, AvgX: 0.1, AvgY: 0.1, Frequency: 0.9},
	})

	// Known: class+window match, enough samples, in-tolerance location.
	if !ElementKnownConfidently("Chrome", "person", 0.52, 0.48, 0.9, 5, 0.15) {
		t.Error("expected 'person' at (0.52,0.48) to be known confidently")
	}
	// Out-of-tolerance location -> not known (element moved).
	if ElementKnownConfidently("Chrome", "person", 0.9, 0.9, 0.9, 5, 0.15) {
		t.Error("expected out-of-tolerance location to NOT be known")
	}
	// Below minSample -> not known yet.
	if ElementKnownConfidently("Chrome", "button", 0.1, 0.1, 0.9, 5, 0.15) {
		t.Error("expected low-sample prior to NOT be known")
	}
	// Different window -> not known.
	if ElementKnownConfidently("explorer", "person", 0.5, 0.5, 0.9, 5, 0.15) {
		t.Error("expected different window to NOT be known")
	}
}

func TestIsDesktopWindow(t *testing.T) {
	desktop := []string{"Program Manager", "", "  ", "Shell_TrayWnd", "Windows Shell Experience"}
	app := []string{"Chrome - GitHub", "Code", "Settings", "Terminal", "Notepad - notes.txt", "File Explorer"}

	for _, title := range desktop {
		if !isDesktopWindow(title) {
			t.Errorf("expected %q to be treated as desktop/wallpaper", title)
		}
	}
	// NOTE: real app titles must NOT be flagged as desktop.
	for _, title := range app {
		if isDesktopWindow(title) {
			t.Errorf("expected %q to NOT be treated as desktop/wallpaper", title)
		}
	}
}
