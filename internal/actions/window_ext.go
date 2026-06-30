package actions

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	moveWindow          = user32.NewProc("MoveWindow")
	getWindowRect       = user32.NewProc("GetWindowRect")
	showWindowAsync     = user32.NewProc("ShowWindowAsync")
	postMessageW        = user32.NewProc("PostMessageW")
	isIconic            = user32.NewProc("IsIconic")
	isZoomed            = user32.NewProc("IsZoomed")
	findWindowW         = user32.NewProc("FindWindowW")
	monitorFromWindow   = user32.NewProc("MonitorFromWindow")
)

const (
	SW_MINIMIZE = 6
	SW_MAXIMIZE = 3
	SW_HIDE     = 0
	WM_CLOSE    = 0x0010
	WS_CAPTION  = 0x00C00000
)

type WindowRect struct {
	Left   int32 `json:"left"`
	Top    int32 `json:"top"`
	Right  int32 `json:"right"`
	Bottom int32 `json:"bottom"`
	Width  int32 `json:"width"`
	Height int32 `json:"height"`
}

type MONITORINFO struct {
	Size    uint32
	Monitor struct {
		Left, Top, Right, Bottom int32
	}
	WorkArea struct {
		Left, Top, Right, Bottom int32
	}
	Flags uint32
}

type WindowStateInfo struct {
	Handle     uintptr     `json:"handle"`
	Title      string      `json:"title"`
	Visible    bool        `json:"visible"`
	Minimized  bool        `json:"minimized"`
	Maximized  bool        `json:"maximized"`
	Fullscreen bool        `json:"fullscreen"`
	Rect       *WindowRect `json:"rect,omitempty"`
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
	ret, _, _ := postMessageW.Call(hwnd, WM_CLOSE, 0, 0)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func isFullscreen(hwnd uintptr) bool {
	rect, err := GetWindowRectByHandle(hwnd)
	if err != nil {
		return false
	}

	style, _, _ := getWindowLongW.Call(hwnd, uintptr(^uint32(15)))
	hasCaption := style&WS_CAPTION != 0
	if hasCaption {
		return false
	}

	hmon, _, _ := monitorFromWindow.Call(hwnd, MONITOR_DEFAULTTONEAREST)
	if hmon == 0 {
		return false
	}

	var mi MONITORINFO
	mi.Size = uint32(unsafe.Sizeof(mi))
	ret, _, _ := getMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi)))
	if ret == 0 {
		return false
	}

	mw := mi.Monitor.Right - mi.Monitor.Left
	mh := mi.Monitor.Bottom - mi.Monitor.Top
	ww := rect.Right - rect.Left
	wh := rect.Bottom - rect.Top

	return ww >= mw && wh >= mh
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

	info.Fullscreen = isFullscreen(hwnd)

	return info, nil
}

func FindWindowByTitle(title string) uintptr {
	t := syscall.StringToUTF16Ptr(title)
	hwnd, _, _ := findWindowW.Call(0, uintptr(unsafe.Pointer(t)))
	return hwnd
}

func WaitForWindow(title string, timeoutMs int32) (uintptr, error) {
	if title == "" {
		return 0, fmt.Errorf("wait_for_window: empty title")
	}
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
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
