package actions

import (
	"fmt"
	"strings"
)

var browserPatterns = []struct {
	hint     string
	titles   []string
	urlBarFn func() error
	newTabFn func() error
}{
	{
		hint:   "firefox",
		titles: []string{"Mozilla Firefox", "Firefox"},
		// Firefox with Multi-Account Containers: Ctrl+T bypasses container picker
		newTabFn: func() error { return KeyPress([]string{"Ctrl", "T"}) },
		// After new tab, focus is already in URL bar for Firefox
		urlBarFn: func() error { return KeyPress([]string{"Ctrl", "T"}) },
	},
	{
		hint:   "chrome",
		titles: []string{"Google Chrome", "Chrome", " - Google Chrome"},
		newTabFn: func() error { return KeyPress([]string{"Ctrl", "T"}) },
		// Chrome/Edge: Ctrl+L focuses the URL bar directly
		urlBarFn: func() error { return KeyPress([]string{"Ctrl", "L"}) },
	},
	{
		hint:   "edge",
		titles: []string{"Microsoft Edge", "Edge", " - Microsoft Edge", " - Edge"},
		newTabFn: func() error { return KeyPress([]string{"Ctrl", "T"}) },
		urlBarFn: func() error { return KeyPress([]string{"Ctrl", "L"}) },
	},
	{
		hint:   "brave",
		titles: []string{"Brave", " - Brave"},
		newTabFn: func() error { return KeyPress([]string{"Ctrl", "T"}) },
		urlBarFn: func() error { return KeyPress([]string{"Ctrl", "L"}) },
	},
	{
		hint:   "opera",
		titles: []string{"Opera", " - Opera"},
		newTabFn: func() error { return KeyPress([]string{"Ctrl", "T"}) },
		urlBarFn: func() error { return KeyPress([]string{"Ctrl", "L"}) },
	},
}

func browserPattern(hint string) (int, bool) {
	lower := strings.ToLower(hint)
	for i, bp := range browserPatterns {
		if strings.Contains(lower, bp.hint) {
			return i, true
		}
	}
	return 0, false
}

func findBrowserWindow(hint string) uintptr {
	lower := strings.ToLower(hint)
	// Try exact window title first
	if idx, ok := browserPattern(lower); ok {
		for _, title := range browserPatterns[idx].titles {
			if hwnd := FindWindowByTitle(title); hwnd != 0 {
				return hwnd
			}
		}
	}
	// Fallback: search by substring through all windows
	windows, err := ListWindows()
	if err != nil {
		return 0
	}
	lowerHint := lower
	for _, w := range windows {
		t := strings.ToLower(w.Title)
		if strings.Contains(t, lowerHint) {
			return w.Handle
		}
		// Also check common browser keywords
		for _, kw := range []string{"firefox", "chrome", "edge", "brave", "opera", "browser", " - "} {
			if strings.Contains(t, kw) {
				return w.Handle
			}
		}
	}
	return 0
}

func focusAndActivateWindow(hwnd uintptr) error {
	if err := FocusWindow(hwnd); err != nil {
		return err
	}
	state, err := GetWindowState(hwnd)
	if err == nil && state != nil {
		clickX := (state.Rect.Left + state.Rect.Right) / 2
		clickY := state.Rect.Top + 10
		Click(ClickInput{X: clickX, Y: clickY})
		Wait(100)
	}
	return nil
}

func BrowserFocusURLBar(browserHint string) error {
	hwnd := findBrowserWindow(browserHint)
	if hwnd == 0 {
		return fmt.Errorf("browser window not found: %s", browserHint)
	}
	if err := focusAndActivateWindow(hwnd); err != nil {
		return fmt.Errorf("browser focus: %w", err)
	}
	idx, known := browserPattern(browserHint)
	if known && browserPatterns[idx].urlBarFn != nil {
		if err := browserPatterns[idx].urlBarFn(); err != nil {
			return fmt.Errorf("browser focus url bar: %w", err)
		}
	} else {
		// Unknown browser: try Ctrl+L first, then Ctrl+T as fallback
		if err := KeyPress([]string{"Ctrl", "L"}); err != nil {
			return fmt.Errorf("browser focus url bar: %w", err)
		}
	}
	Wait(200)
	return nil
}

func BrowserNewTab(browserHint string) error {
	hwnd := findBrowserWindow(browserHint)
	if hwnd == 0 {
		return fmt.Errorf("browser window not found: %s", browserHint)
	}
	if err := focusAndActivateWindow(hwnd); err != nil {
		return fmt.Errorf("browser focus: %w", err)
	}
	idx, known := browserPattern(browserHint)
	if known && browserPatterns[idx].newTabFn != nil {
		if err := browserPatterns[idx].newTabFn(); err != nil {
			return fmt.Errorf("browser new tab: %w", err)
		}
	} else {
		if err := KeyPress([]string{"Ctrl", "T"}); err != nil {
			return fmt.Errorf("browser new tab: %w", err)
		}
	}
	Wait(300)
	return nil
}

func BrowserNavigate(browserHint, url string) error {
	if err := BrowserNewTab(browserHint); err != nil {
		return err
	}
	// For Firefox, new tab (Ctrl+T) already focuses URL bar
	// For Chrome/Edge, need Ctrl+L after new tab
	idx, known := browserPattern(browserHint)
	if known && browserPatterns[idx].hint != "firefox" {
		KeyPress([]string{"Ctrl", "L"})
		Wait(100)
	}
	return TypeAndSubmit(url)
}

func BrowserSearch(browserHint, query string) error {
	return BrowserNavigate(browserHint, query)
}
