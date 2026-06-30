package actions

import (
	"encoding/json"
	"syscall"
	"time"
	"unsafe"
)

var (
	enumWindows         = user32.NewProc("EnumWindows")
	getWindowTextW      = user32.NewProc("GetWindowTextW")
	getWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	isWindowVisible     = user32.NewProc("IsWindowVisible")
	setForegroundWindow = user32.NewProc("SetForegroundWindow")
	showWindow          = user32.NewProc("ShowWindow")
	getWindowLongW      = user32.NewProc("GetWindowLongW")
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
		windows = append(windows, WindowInfo{
			Handle: hwnd,
			Title:  title,
			PID:    getWindowPID(hwnd),
		})
		return true
	}
	enumCallback = callback

	cb := syscall.NewCallback(windowEnumProc)
	enumWindows.Call(cb, 0)

	return windows, nil
}

func FocusWindow(handle uintptr) (err error) {
	start := time.Now()
	defer func() {
		b, _ := json.Marshal(map[string]uintptr{"handle": handle})
		LogToolCall("focus_window", string(b), err)
		Adaptive.RecordResult("focus_window", float64(time.Since(start).Milliseconds()), err == nil)
		Adaptive.LearnFromCommand("focus_window", string(b), err == nil)
	}()
	windowThread, _, _ := getWindowThreadProcessId.Call(handle, 0)
	currentThread, _, _ := getCurrentThreadId.Call()
	if windowThread != currentThread {
		attachThreadInput.Call(currentThread, windowThread, 1)
		setForegroundWindow.Call(handle)
		attachThreadInput.Call(currentThread, windowThread, 0)
	} else {
		setForegroundWindow.Call(handle)
	}
	showWindow.Call(handle, SW_RESTORE)
	return
}
