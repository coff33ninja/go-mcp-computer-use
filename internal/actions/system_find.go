package actions

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var findableApps = []struct {
	hint    string
	titles  []string
}{
	{hint: "firefox", titles: []string{"Mozilla Firefox", "Firefox"}},
	{hint: "chrome", titles: []string{"Google Chrome", "Chrome", " - Google Chrome"}},
	{hint: "edge", titles: []string{"Microsoft Edge", "Edge", " - Edge"}},
	{hint: "brave", titles: []string{"Brave", " - Brave"}},
	{hint: "opera", titles: []string{"Opera", " - Opera"}},
	{hint: "notepad", titles: []string{"Notepad", " - Notepad"}},
	{hint: "code", titles: []string{"Visual Studio Code", "VS Code"}},
	{hint: "explorer", titles: []string{"File Explorer", "Windows Explorer"}},
}

var (
	systemFindLastUsed time.Time
	systemFindMu       sync.Mutex
	SystemFindCount    int
)

func isFindableApp(windowTitle string) bool {
	lower := strings.ToLower(windowTitle)
	for _, app := range findableApps {
		for _, t := range app.titles {
			if strings.Contains(lower, strings.ToLower(t)) {
				return true
			}
		}
	}
	return false
}

func SystemFindText(text, windowTitle string) (found bool, x, y int32, err error) {
	if text == "" {
		return false, 0, 0, fmt.Errorf("system_find_text: empty text")
	}
	if windowTitle == "" {
		return false, 0, 0, nil
	}
	if !isFindableApp(windowTitle) {
		return false, 0, 0, nil
	}

	start := time.Now()

	hwnd := FindWindowByTitle(windowTitle)
	if hwnd == 0 {
		return false, 0, 0, nil
	}
	if err := focusAndActivateWindow(hwnd); err != nil {
		return false, 0, 0, nil
	}
	Wait(200)

	if time.Since(start) > 5*time.Second {
		return false, 0, 0, nil
	}

	if err := KeyPress([]string{"Ctrl", "F"}); err != nil {
		return false, 0, 0, nil
	}
	Wait(300)

	for _, r := range text {
		sendCharWithVK(r)
	}
	Wait(200)

	if err := KeyPress([]string{"ENTER"}); err != nil {
		return false, 0, 0, nil
	}
	Wait(500)

	result, ocrErr := OCRWindow(hwnd, "")
	if ocrErr != nil {
		KeyPress([]string{"ESCAPE"})
		return false, 0, 0, nil
	}

	lowerText := strings.ToLower(text)
	for _, word := range result.Words {
		if strings.Contains(strings.ToLower(word.Text), lowerText) {
			KeyPress([]string{"ESCAPE"})
			Wait(100)
			systemFindMu.Lock()
			systemFindLastUsed = time.Now()
			SystemFindCount++
			systemFindMu.Unlock()
			return true, int32(word.X + word.W/2), int32(word.Y + word.H/2), nil
		}
	}
	for _, line := range result.Lines {
		if strings.Contains(strings.ToLower(line.Text), lowerText) {
			KeyPress([]string{"ESCAPE"})
			Wait(100)
			systemFindMu.Lock()
			systemFindLastUsed = time.Now()
			SystemFindCount++
			systemFindMu.Unlock()
			return true, int32(line.X + line.W/2), int32(line.Y + line.H/2), nil
		}
	}

	KeyPress([]string{"ESCAPE"})
	Wait(100)
	return false, 0, 0, nil
}

func SystemFindTextAndClick(text, windowTitle string) (found bool, x, y int32, err error) {
	found, x, y, err = SystemFindText(text, windowTitle)
	if err != nil || !found {
		return found, x, y, err
	}
	if clickErr := Click(ClickInput{X: x, Y: y, Button: "left", Clicks: 1}); clickErr != nil {
		return found, x, y, clickErr
	}
	return found, x, y, nil
}

func SystemFindStats() (lastUsed time.Time, count int) {
	systemFindMu.Lock()
	defer systemFindMu.Unlock()
	return systemFindLastUsed, SystemFindCount
}

func DetectActiveWindowCategory() string {
	info, err := GetActiveWindowInfo()
	if err != nil || info.Handle == 0 {
		return "unknown"
	}
	title := strings.ToLower(info.Title)
	for _, app := range findableApps {
		for _, t := range app.titles {
			if strings.Contains(title, strings.ToLower(t)) {
				return app.hint
			}
		}
	}
	if strings.Contains(title, "powershell") || strings.Contains(title, "cmd") || strings.Contains(title, "terminal") {
		return "terminal"
	}
	return "unknown"
}
