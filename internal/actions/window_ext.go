package actions

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	moveWindow       = user32.NewProc("MoveWindow")
	getWindowRect    = user32.NewProc("GetWindowRect")
	showWindowAsync  = user32.NewProc("ShowWindowAsync")
	isIconic         = user32.NewProc("IsIconic")
	isZoomed         = user32.NewProc("IsZoomed")
	findWindowW      = user32.NewProc("FindWindowW")
)

const (
	SW_MINIMIZE = 6
	SW_MAXIMIZE = 3
	SW_HIDE     = 0
	SW_CLOSE    = 0x10
)

type WindowRect struct {
	Left   int32 `json:"left"`
	Top    int32 `json:"top"`
	Right  int32 `json:"right"`
	Bottom int32 `json:"bottom"`
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

type WindowStateInfo struct {
	Handle    uintptr `json:"handle"`
	Title     string  `json:"title"`
	Visible   bool    `json:"visible"`
	Minimized bool    `json:"minimized"`
	Maximized bool    `json:"maximized"`
	Rect      *WindowRect `json:"rect,omitempty"`
}

func MoveWindowByHandle(hwnd uintptr, x, y, w, h int32) error {
	ret, _, _ := moveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 1)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func GetWindowRectByHandle(hwnd uintptr) (*WindowRect, error) {
	var r struct {
		Left, Top, Right, Bottom int32
	}
	ret, _, _ := getWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	if ret == 0 {
		return nil, syscall.GetLastError()
	}
	return &WindowRect{
		Left:   r.Left,
		Top:    r.Top,
		Right:  r.Right,
		Bottom: r.Bottom,
		Width:  r.Right - r.Left,
		Height: r.Bottom - r.Top,
	}, nil
}

func MinimizeWindow(hwnd uintptr) error {
	showWindowAsync.Call(hwnd, SW_MINIMIZE)
	return nil
}

func MaximizeWindow(hwnd uintptr) error {
	showWindowAsync.Call(hwnd, SW_MAXIMIZE)
	return nil
}

func RestoreWindow(hwnd uintptr) error {
	showWindowAsync.Call(hwnd, SW_RESTORE)
	return nil
}

func CloseWindow(hwnd uintptr) error {
	ret, _, _ := showWindowAsync.Call(hwnd, SW_CLOSE)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func GetWindowState(hwnd uintptr) (*WindowStateInfo, error) {
	title := getWindowTitle(hwnd)
	info := &WindowStateInfo{
		Handle: hwnd,
		Title:  title,
	}

	v, _, _ := isWindowVisible.Call(hwnd)
	info.Visible = v != 0

	v, _, _ = isIconic.Call(hwnd)
	info.Minimized = v != 0

	v, _, _ = isZoomed.Call(hwnd)
	info.Maximized = v != 0

	rect, err := GetWindowRectByHandle(hwnd)
	if err == nil {
		info.Rect = rect
	}

	return info, nil
}

func FindWindowByTitle(title string) uintptr {
	t := syscall.StringToUTF16Ptr(title)
	hwnd, _, _ := findWindowW.Call(0, uintptr(unsafe.Pointer(t)))
	return hwnd
}

func WaitForWindow(title string, timeoutMs int32) (uintptr, error) {
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for time.Now().Before(deadline) {
		hwnd := FindWindowByTitle(title)
		if hwnd != 0 {
			return hwnd, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return 0, syscall.ENOENT
}
