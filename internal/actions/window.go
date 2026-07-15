package actions

import (
	"encoding/json"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	enumWindows              = user32.NewProc("EnumWindows")
	getWindowTextW           = user32.NewProc("GetWindowTextW")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	isWindowVisible          = user32.NewProc("IsWindowVisible")
	setForegroundWindow      = user32.NewProc("SetForegroundWindow")
	showWindow               = user32.NewProc("ShowWindow")
	getWindowLongW           = user32.NewProc("GetWindowLongW")
	bringWindowToTop         = user32.NewProc("BringWindowToTop")
	switchToThisWindow       = user32.NewProc("SwitchToThisWindow")
)

const (
	WS_EX_APPWINDOW  = 0x00040000
	WS_EX_TOOLWINDOW = 0x00000080
	SW_RESTORE       = 9
)

type WindowInfo struct {
	Handle uintptr `json:"handle"`
	Title  string  `json:"title"`
	PID    uint32  `json:"pid"`
	X      int32   `json:"x,omitempty"`
	Y      int32   `json:"y,omitempty"`
	Width  int32   `json:"width,omitempty"`
	Height int32   `json:"height,omitempty"`
}

type windowCallback func(hwnd uintptr) bool

var enumCallback windowCallback

func windowEnumProc(hwnd uintptr, lparam uintptr) uintptr {
	if enumCallback(hwnd) {
		return 1
	}
	return 0
}

func getWindowTitle(hwnd uintptr) string {
	buf := make([]uint16, 512)
	ret, _, _ := getWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:ret])
}

func getWindowPID(hwnd uintptr) uint32 {
	var pid uint32
	getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}

func isAppWindow(hwnd uintptr) bool {
	style, _, _ := getWindowLongW.Call(hwnd, uintptr(^uint32(19)))
	if style&WS_EX_APPWINDOW != 0 {
		return true
	}
	if style&WS_EX_TOOLWINDOW != 0 {
		return false
	}
	v, _, _ := isWindowVisible.Call(hwnd)
	return v != 0
}

func ListWindows() ([]WindowInfo, error) {
	var windows []WindowInfo

	callback := func(hwnd uintptr) bool {
		if !isAppWindow(hwnd) {
			return true
		}
		title := getWindowTitle(hwnd)
		if title == "" {
			return true
		}
		wi := WindowInfo{
			Handle: hwnd,
			Title:  title,
			PID:    getWindowPID(hwnd),
		}
		if rect, err := GetWindowRectByHandle(hwnd); err == nil && rect != nil {
			wi.X = rect.Left
			wi.Y = rect.Top
			wi.Width = rect.Width
			wi.Height = rect.Height
		}
		windows = append(windows, wi)
		return true
	}
	enumCallback = callback

	cb := syscall.NewCallback(windowEnumProc)
	enumWindows.Call(cb, 0)

	return windows, nil
}

func trySetForeground(handle uintptr) bool {
	windowThread, _, _ := getWindowThreadProcessId.Call(handle, 0)
	currentThread, _, _ := getCurrentThreadId.Call()
	if windowThread != currentThread {
		attachThreadInput.Call(currentThread, windowThread, 1)
		ret, _, _ := setForegroundWindow.Call(handle)
		attachThreadInput.Call(currentThread, windowThread, 0)
		return ret != 0
	}
	ret, _, _ := setForegroundWindow.Call(handle)
	return ret != 0
}

func isForeground(handle uintptr) bool {
	fg, _, _ := getForegroundWindow.Call()
	return fg == handle
}

func FocusWindow(handle uintptr) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]uintptr{"handle": handle})
		LogToolCall("focus_window", string(b), err)
		Adaptive.RecordResult("focus_window", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("focus_window", string(b), err == nil)
	}()

	// If already foreground, still restore in case minimized
	showWindow.Call(handle, SW_RESTORE)
	if isForeground(handle) {
		return nil
	}

	// Attempt 1: SetForegroundWindow with AttachThreadInput
	trySetForeground(handle)
	showWindow.Call(handle, SW_RESTORE)
	if isForeground(handle) {
		return nil
	}

	// Attempt 2: BringWindowToTop + SW_SHOW
	bringWindowToTop.Call(handle)
	showWindow.Call(handle, SW_SHOW)
	if isForeground(handle) {
		return nil
	}

	// Attempt 3: SwitchToThisWindow (bypasses foreground lock)
	switchToThisWindow.Call(handle, 1)
	showWindow.Call(handle, SW_RESTORE)
	if isForeground(handle) {
		return nil
	}

	// Attempt 4: Retry SetForegroundWindow one more time after delay
	Wait(100)
	trySetForeground(handle)
	showWindow.Call(handle, SW_RESTORE)
	if isForeground(handle) {
		return nil
	}

	return fmt.Errorf("focus_window: failed to bring window to foreground after all attempts")
}
