package actions

import (
	"fmt"
	"strings"
	"time"
)

func FindTextAndClick(text string, regionX, regionY, regionW, regionH *int32) error {
	var result *OCRResult
	var err error

	if regionW != nil && regionH != nil {
		x := int32(0)
		y := int32(0)
		if regionX != nil { x = *regionX }
		if regionY != nil { y = *regionY }
		result, err = OCRRegion(x, y, *regionW, *regionH)
	} else {
		result, err = OCRScreen()
	}
	if err != nil {
		return fmt.Errorf("find_text_and_click ocr: %w", err)
	}

	lowerText := strings.ToLower(text)
	for _, word := range result.Words {
		if strings.Contains(strings.ToLower(word.Text), lowerText) {
			cx := int32(word.X + word.W/2)
			cy := int32(word.Y + word.H/2)
			return Click(ClickInput{X: cx, Y: cy, Button: "left", Clicks: 1})
		}
	}
	for _, line := range result.Lines {
		if strings.Contains(strings.ToLower(line.Text), lowerText) {
			cx := int32(line.X + line.W/2)
			cy := int32(line.Y + line.H/2)
			return Click(ClickInput{X: cx, Y: cy, Button: "left", Clicks: 1})
		}
	}

	return fmt.Errorf("find_text_and_click: text %q not found on screen", text)
}

func TypeAndSubmit(text string) error {
	if err := TypeText(text); err != nil {
		return fmt.Errorf("type_and_submit type: %w", err)
	}
	return KeyPress([]string{"Enter"})
}

func LaunchAndWait(path, windowTitle string, timeoutMs int32) (uintptr, error) {
	if err := LaunchApp(path); err != nil {
		return 0, fmt.Errorf("launch_and_while: %w", err)
	}
	hwnd, err := WaitForWindow(windowTitle, timeoutMs)
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
	return CaptureRegion(state.Rect.Left, state.Rect.Top, state.Rect.Width, state.Rect.Height)
}

func Hover(x, y int32) error {
	if err := ValidateClickCoord(x, y); err != nil {
		return err
	}
	if err := MoveMouse(x, y); err != nil {
		return fmt.Errorf("hover move: %w", err)
	}
	Wait(300)
	return nil
}

func WaitForText(text string, timeoutMs int32) (*OCRResult, error) {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		result, err := OCRScreen()
		if err != nil {
			return nil, fmt.Errorf("wait_for_text ocr: %w", err)
		}
		lowerText := strings.ToLower(text)
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
		Wait(500)
	}
	return nil, fmt.Errorf("wait_for_text: text %q not found within %dms", text, timeoutMs)
}

func SelectAllAndType(text string) error {
	if err := KeyPress([]string{"Ctrl", "a"}); err != nil {
		return fmt.Errorf("select_all_and_type key: %w", err)
	}
	Wait(100)
	return TypeText(text)
}

func ClickMenuItem(windowTitle, menuItemText string) error {
	hwnd := FindWindowByTitle(windowTitle)
	if hwnd == 0 {
		return fmt.Errorf("click_menu_item: window %q not found", windowTitle)
	}
	state, err := GetWindowState(hwnd)
	if err != nil {
		return fmt.Errorf("click_menu_item state: %w", err)
	}
	if state.Rect == nil {
		return fmt.Errorf("click_menu_item: window %q has no position", windowTitle)
	}

	result, err := OCRRegion(state.Rect.Left, state.Rect.Top, state.Rect.Width, state.Rect.Height)
	if err != nil {
		return fmt.Errorf("click_menu_item ocr: %w", err)
	}

	return clickFirstMatch(result, menuItemText, state.Rect.Left, state.Rect.Top)
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
