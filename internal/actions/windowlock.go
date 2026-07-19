package actions

import (
	"fmt"
	"sync"
)

var (
	lockMu     sync.RWMutex
	lockHandle uintptr
	lockTitle  string
	lockPid    uint32
)

var isWindowProc = user32.NewProc("IsWindow")

// SetWindowLock stores the window as the active target.
// Verifies the window exists before locking.
func SetWindowLock(handle uintptr) error {
	if handle == 0 {
		return fmt.Errorf("window_lock: invalid handle 0")
	}

	ret, _, _ := isWindowProc.Call(handle)
	if ret == 0 {
		return fmt.Errorf("window_lock: window %d does not exist", handle)
	}

	lockMu.Lock()
	defer lockMu.Unlock()
	lockHandle = handle
	lockTitle = getWindowTitle(handle)
	lockPid = getWindowPID(handle)
	return nil
}

// ClearWindowLock releases the lock.
func ClearWindowLock() {
	lockMu.Lock()
	defer lockMu.Unlock()
	lockHandle = 0
	lockTitle = ""
	lockPid = 0
}

// GetWindowLock returns current lock state.
func GetWindowLock() (handle uintptr, title string, pid uint32, locked bool) {
	lockMu.RLock()
	defer lockMu.RUnlock()
	return lockHandle, lockTitle, lockPid, lockHandle != 0
}

// GetWindowLockTitle returns the locked window title.
func GetWindowLockTitle() string {
	lockMu.RLock()
	defer lockMu.RUnlock()
	return lockTitle
}

// GetWindowLockPID returns the locked window PID.
func GetWindowLockPID() uint32 {
	lockMu.RLock()
	defer lockMu.RUnlock()
	return lockPid
}

// VerifyWindowLock checks if the locked window is still valid and foreground.
// If autoFocus is true and window exists but is not foreground, auto-focuses it.
func VerifyWindowLock(autoFocus bool) (bool, string) {
	lockMu.RLock()
	h := lockHandle
	lockMu.RUnlock()

	if h == 0 {
		return true, ""
	}

	ret, _, _ := isWindowProc.Call(h)
	if ret == 0 {
		ClearWindowLock()
		return false, "locked window no longer exists"
	}

	if isForeground(h) {
		return true, ""
	}

	if autoFocus {
		if err := FocusWindow(h); err != nil {
			return false, fmt.Sprintf("auto-focus failed: %v", err)
		}
		return true, "auto-focused back to locked window"
	}

	fg, _, _ := getForegroundWindow.Call()
	return false, fmt.Sprintf("locked window lost foreground (now: %d)", fg)
}

// Screen-touching tools that require window lock verification.
var screenTools = map[string]bool{
	"click": true, "move_mouse": true, "scroll": true, "drag": true,
	"hover": true, "type": true, "type_and_submit": true,
	"select_all_and_type": true, "key_press": true, "key_down": true,
	"key_up": true, "screenshot": true, "screenshot_element": true,
	"ocr": true, "ocr_window": true, "ocr_active_window": true,
	"find_text_and_click": true, "click_menu_item": true,
	"uia_find": true, "uia_get_text": true, "uia_invoke": true,
	"uia_set_text": true, "uia_get_element_at_point": true,
	"uia_get_all_elements": true, "find_image": true, "find_all_images": true,
	"onnx_detect": true, "find_ui_element": true,
	"browser_focus_url_bar": true, "browser_new_tab": true,
	"browser_navigate": true, "browser_search": true,
	"dismiss_all_menus": true, "keylogger_start": true,
	"wait_for_ui_element": true, "layout_validate": true,
	"template_store": true, "template_find": true,
}

// IsScreenTool returns true if the tool touches the screen.
func IsScreenTool(name string) bool {
	return screenTools[name]
}
