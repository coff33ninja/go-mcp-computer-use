package actions

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type FindTextOpts struct {
	Text     string
	Language string
	RegionX  *int32
	RegionY  *int32
	RegionW  *int32
	RegionH  *int32
	MaxScrolls    int32
	ScrollClicks  int32
	ScrollDown    bool
	WindowTitle   string
	SkipMemory    bool
	SkipSystemFind bool
}

func FindTextAndClick(opts FindTextOpts) (clickX, clickY int32, err error) {
	if opts.Text == "" {
		return 0, 0, fmt.Errorf("find_text_and_click: empty text")
	}

	maxScrolls := opts.MaxScrolls
	if maxScrolls < 0 {
		maxScrolls = 0
	}
	scrollClicks := opts.ScrollClicks
	if scrollClicks == 0 {
		scrollClicks = 5
	}

	windowTitle := opts.WindowTitle
	if windowTitle == "" {
		if info, cerr := GetActiveWindowInfo(); cerr == nil {
			windowTitle = info.Title
		}
	}

	// Capture foreground window z-order for layered window matching
	fgZOrder := 0
	if fgHwnd := ForegroundWindowHandle(); fgHwnd != 0 {
		fgZOrder = GetWindowZOrder(fgHwnd)
	}

	if !opts.SkipMemory {
		if loc := FindTextLocationMatch(opts.Text, windowTitle, fgZOrder); loc != nil {
			cx, cy := loc.X+loc.W/2, loc.Y+loc.H/2
			if ValidateClickCoord(cx, cy) == nil {
				if clickErr := Click(ClickInput{X: cx, Y: cy, Button: "left", Clicks: 1}); clickErr == nil {
					return cx, cy, nil
				}
			}
		}
		if loc := FindTextLocationAnyMatch(opts.Text, fgZOrder); loc != nil {
			cx, cy := loc.X+loc.W/2, loc.Y+loc.H/2
			if ValidateClickCoord(cx, cy) == nil {
				if clickErr := Click(ClickInput{X: cx, Y: cy, Button: "left", Clicks: 1}); clickErr == nil {
					return cx, cy, nil
				}
			}
		}
	}

	if !opts.SkipSystemFind && windowTitle != "" {
		if found, fx, fy, sysErr := SystemFindTextAndClick(opts.Text, windowTitle); sysErr == nil && found {
			StoreTextLocation(opts.Text, windowTitle, fx, fy, 10, 10, int32(fgZOrder))
			return fx, fy, nil
		}
	}

	var captureX, captureY int32

	doSearch := func() (*OCRResult, error) {
		captureX, captureY = 0, 0
		if opts.RegionW != nil && opts.RegionH != nil {
			x := int32(0)
			y := int32(0)
			if opts.RegionX != nil { x = *opts.RegionX }
			if opts.RegionY != nil { y = *opts.RegionY }
			w, h := *opts.RegionW, *opts.RegionH
			if w < 300 || h < 300 {
				if info, cerr := GetActiveWindowInfo(); cerr == nil && info.Handle != 0 {
					r, e := OCRProportionalWindowRegion(info.Handle, 0.05, 0.05, 0.95, 0.95, opts.Language)
					if e == nil {
						captureX, captureY = x, y
						return r, nil
					}
				}
			}
			captureX, captureY = x, y
			return OCRRegion(x, y, w, h, opts.Language)
		}
		// Try window-specific OCR first when title is provided (focused, correct coords)
		if windowTitle != "" {
			if hwnd := FindWindowByTitle(windowTitle); hwnd != 0 {
				if rect, rerr := GetWindowRectByHandle(hwnd); rerr == nil && rect.Width > 0 && rect.Height > 0 {
					if r, e := OCRWindow(hwnd, opts.Language); e == nil && len(r.Words) > 0 {
						captureX, captureY = rect.Left, rect.Top
						return r, nil
					}
				}
			}
		}
		// Fall back to full-screen OCR
		bounds := VirtualScreenBounds()
		captureX, captureY = bounds.X, bounds.Y
		return OCRScreen(opts.Language)
	}

	findInResult := func(result *OCRResult) (int32, int32, bool) {
		lowerText := strings.ToLower(opts.Text)
		for _, word := range result.Words {
			if strings.Contains(strings.ToLower(word.Text), lowerText) {
				return int32(word.X + word.W/2), int32(word.Y + word.H/2), true
			}
		}
		for _, line := range result.Lines {
			if strings.Contains(strings.ToLower(line.Text), lowerText) {
				return int32(line.X + line.W/2), int32(line.Y + line.H/2), true
			}
		}
		return 0, 0, false
	}

	var lastResult *OCRResult
	for attempt := int32(0); attempt <= maxScrolls; attempt++ {
		if attempt > 0 {
			scrollDir := scrollClicks
			if !opts.ScrollDown {
				scrollDir = -scrollClicks
			}
			Scroll(scrollDir, false)
			Wait(300)
		}
		result, ocrErr := doSearch()
		if ocrErr != nil {
			return 0, 0, fmt.Errorf("find_text_and_click ocr: %w", ocrErr)
		}
		lastResult = result
		if cx, cy, found := findInResult(result); found {
			vx, vy := cx+captureX, cy+captureY
			StoreTextLocation(opts.Text, windowTitle, vx, vy, 10, 10, int32(fgZOrder))
			return vx, vy, Click(ClickInput{X: vx, Y: vy, Button: "left", Clicks: 1})
		}
	}

	visible := make([]string, 0, min(len(lastResult.Lines), 15))
	for _, line := range lastResult.Lines {
		t := strings.TrimSpace(line.Text)
		if t != "" {
			visible = append(visible, t)
		}
		if len(visible) >= 15 {
			break
		}
	}
	if len(visible) > 0 {
		return 0, 0, fmt.Errorf("find_text_and_click: text %q not found after %d scroll attempts. Visible text: %q", opts.Text, maxScrolls, strings.Join(visible, " | "))
	}
	return 0, 0, fmt.Errorf("find_text_and_click: text %q not found after %d scroll attempts (no text detected via OCR)", opts.Text, maxScrolls)
}

func TypeAndSubmit(text string) error {
	if text == "" {
		return fmt.Errorf("type_and_submit: empty text")
	}
	if err := warnElevated(); err != nil {
		return err
	}
	for _, r := range text {
		sendCharWithVK(r)
	}
	// Small pause so the application can process the typed text before Enter
	Wait(50)
	return KeyPress([]string{"ENTER"})
}

func LaunchAndWait(path, windowTitle string, timeoutMs int32) (hwnd uintptr, err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]any{
			"path": path, "window_title": windowTitle, "timeout_ms": timeoutMs,
		})
		LogToolCall("launch_and_wait", string(b), err)
		Adaptive.RecordResult("launch_and_wait", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("launch_and_wait", string(b), err == nil)
	}()
	if err = LaunchApp(path); err != nil {
		return 0, fmt.Errorf("launch_and_wait: %w", err)
	}
	hwnd, err = WaitForWindow(windowTitle, timeoutMs)
	if err != nil {
		return 0, fmt.Errorf("launch_and_wait: %w", err)
	}
	return hwnd, nil
}

func ScreenshotElement(handle uintptr) (string, error) {
	state, err := GetWindowState(handle)
	if err != nil {
		return "", fmt.Errorf("screenshot_element state: %w", err)
	}
	if state.Rect == nil {
		return "", fmt.Errorf("screenshot_element: window has no position info")
	}
	sw, sh := ScreenSize()
	x, y, w, h := state.Rect.Left, state.Rect.Top, state.Rect.Width, state.Rect.Height
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > sw {
		w = sw - x
	}
	if y+h > sh {
		h = sh - y
	}
	if w <= 0 || h <= 0 {
		return "", fmt.Errorf("screenshot_element: window not visible on screen (clamped to %dx%d)", w, h)
	}
	return CaptureRegion(x, y, w, h)
}

func Hover(x, y int32) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]int32{"x": x, "y": y})
		LogToolCall("hover", string(b), err)
		Adaptive.RecordResult("hover", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("hover", string(b), err == nil)
	}()
	if err = ValidateClickCoord(x, y); err != nil {
		return
	}
	if err = MoveMouse(x, y); err != nil {
		err = fmt.Errorf("hover move: %w", err)
		return
	}
	Wait(300)
	return
}

func WaitForText(text string, timeoutMs int32, language string) (*OCRResult, error) {
	return WaitForTextScroll(text, timeoutMs, language, 0, 5, true)
}

func WaitForTextScroll(text string, timeoutMs int32, language string, maxScrolls, scrollClicks int32, scrollDown bool) (*OCRResult, error) {
	if text == "" {
		return nil, fmt.Errorf("wait_for_text: empty text")
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	if maxScrolls < 0 {
		maxScrolls = 0
	}
	if scrollClicks == 0 {
		scrollClicks = 5
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	lowerText := strings.ToLower(text)
	scrollAttempts := int32(0)
	for time.Now().Before(deadline) {
		result, err := OCRScreen(language)
		if err != nil {
			return nil, fmt.Errorf("wait_for_text ocr: %w", err)
		}
		for _, word := range result.Words {
			if strings.Contains(strings.ToLower(word.Text), lowerText) {
				return result, nil
			}
		}
		for _, line := range result.Lines {
			if strings.Contains(strings.ToLower(line.Text), lowerText) {
				return result, nil
			}
		}
		if scrollAttempts < maxScrolls {
			scrollDir := scrollClicks
			if !scrollDown {
				scrollDir = -scrollClicks
			}
			Scroll(scrollDir, false)
			scrollAttempts++
			Wait(300)
		} else {
			Wait(500)
		}
	}
	return nil, fmt.Errorf("wait_for_text: text %q not found within %dms (scrolled %d times)", text, timeoutMs, scrollAttempts)
}

func SelectAllAndType(text string) error {
	if text == "" {
		return fmt.Errorf("select_all_and_type: empty text")
	}
	if err := warnElevated(); err != nil {
		return err
	}
	// Use VK codes for Ctrl+A rather than KEYEVENTF_UNICODE (0x01 via VK_PACKET
	// doesn't trigger select-all in most applications)
	sendVK(0x11, true)  // VK_CONTROL down
	sendVK(0x41, true)  // VK_A down
	sendVK(0x41, false) // VK_A up
	sendVK(0x11, false) // VK_CONTROL up
	Wait(100)
	return TypeText(text)
}

func ClickMenuItem(handle uintptr, menuItemText, language string) error {
	if handle == 0 {
		return fmt.Errorf("click_menu_item: handle is 0")
	}
	state, err := GetWindowState(handle)
	if err != nil {
		return fmt.Errorf("click_menu_item state: %w", err)
	}
	if state.Rect == nil {
		return fmt.Errorf("click_menu_item: window has no position")
	}

	result, err := OCRRegion(state.Rect.Left, state.Rect.Top, state.Rect.Width, state.Rect.Height, language)
	if err != nil {
		return fmt.Errorf("click_menu_item ocr: %w", err)
	}

	return clickFirstMatch(result, menuItemText, state.Rect.Left, state.Rect.Top)
}

func ClickMenuItemByTitle(windowTitle, menuItemText, language string) error {
	hwnd := FindWindowByTitle(windowTitle)
	if hwnd == 0 {
		return fmt.Errorf("click_menu_item: window %q not found", windowTitle)
	}
	return ClickMenuItem(hwnd, menuItemText, language)
}

func clickFirstMatch(result *OCRResult, text string, offsetX, offsetY int32) error {
	lowerText := strings.ToLower(text)
	for _, word := range result.Words {
		if strings.Contains(strings.ToLower(word.Text), lowerText) {
			return Click(ClickInput{
				X: offsetX + int32(word.X+word.W/2),
				Y: offsetY + int32(word.Y+word.H/2),
				Button: "left", Clicks: 1,
			})
		}
	}
	for _, line := range result.Lines {
		if strings.Contains(strings.ToLower(line.Text), lowerText) {
			return Click(ClickInput{
				X: offsetX + int32(line.X+line.W/2),
				Y: offsetY + int32(line.Y+line.H/2),
				Button: "left", Clicks: 1,
			})
		}
	}
	return fmt.Errorf("text %q not found in OCR result", text)
}

func FocusWindowByTitle(title string) error {
	hwnd := FindWindowByTitle(title)
	if hwnd == 0 {
		return fmt.Errorf("window not found: %s", title)
	}
	return focusAndActivateWindow(hwnd)
}

type ScrollSearchOpts struct {
	MaxScrolls   int32
	ScrollClicks int32
	ScrollDown   bool
	Language     string
}

func ScrollUntilFound(opts ScrollSearchOpts, searchFn func(*OCRResult) bool) (*OCRResult, error) {
	maxScrolls := opts.MaxScrolls
	if maxScrolls <= 0 {
		maxScrolls = 0
	}
	scrollClicks := opts.ScrollClicks
	if scrollClicks == 0 {
		scrollClicks = 5
	}

	var lastResult *OCRResult
	for attempt := int32(0); attempt <= maxScrolls; attempt++ {
		if attempt > 0 {
			scrollDir := scrollClicks
			if !opts.ScrollDown {
				scrollDir = -scrollClicks
			}
			Scroll(scrollDir, false)
			Wait(300)
		}
		result, err := OCRScreen(opts.Language)
		if err != nil {
			return nil, fmt.Errorf("scroll_search ocr: %w", err)
		}
		lastResult = result
		if searchFn(result) {
			return result, nil
		}
	}
	return lastResult, nil
}
