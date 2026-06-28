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
