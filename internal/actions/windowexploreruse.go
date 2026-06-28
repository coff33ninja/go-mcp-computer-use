package actions

import (
	"fmt"
	"os"
	"strings"
)

func findExplorerWindow() uintptr {
	windows, err := ListWindows()
	if err != nil {
		return 0
	}
	for _, w := range windows {
		title := strings.ToLower(w.Title)
		if strings.Contains(title, "explorer") && len(w.Title) > 0 {
			return w.Handle
		}
	}
	return 0
}

func ExplorerFocus() error {
	hwnd := findExplorerWindow()
	if hwnd == 0 {
		return fmt.Errorf("no file explorer window found")
	}
	if err := FocusWindow(hwnd); err != nil {
		return fmt.Errorf("explorer focus: %w", err)
	}
	state, err := GetWindowState(hwnd)
	if err == nil && state != nil {
		clickX := (state.Rect.Left + state.Rect.Right) / 2
		clickY := state.Rect.Top + 10
		Click(ClickInput{X: clickX, Y: clickY})
		Wait(50)
	}
	return nil
}

func ExplorerOpenPath(path string) error {
	// Try to find existing explorer window to reuse
	hwnd := findExplorerWindow()
	if hwnd != 0 {
		// Explorer open — focus and navigate via Ctrl+L + type path + Enter
		if err := focusAndActivateWindow(hwnd); err != nil {
			// Fall through to launch new explorer
			return explorerLaunchNew(path)
		}
		if err := KeyPress([]string{"Ctrl", "L"}); err != nil {
			return explorerLaunchNew(path)
		}
		Wait(100)
		if err := TypeAndSubmit(path); err != nil {
			return explorerLaunchNew(path)
		}
		Wait(300)
		return nil
	}
	return explorerLaunchNew(path)
}

func explorerLaunchNew(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", path)
	}
	return OpenFileExplorer(path)
}

func ExplorerNavigateTo(path string) error {
	if err := ExplorerFocus(); err != nil {
		return err
	}
	if err := KeyPress([]string{"Ctrl", "L"}); err != nil {
		return fmt.Errorf("explorer navigate: %w", err)
	}
	Wait(100)
	return TypeAndSubmit(path)
}
