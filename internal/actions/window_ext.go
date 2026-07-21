package actions

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	moveWindow           = user32.NewProc("MoveWindow")
	getWindowRect        = user32.NewProc("GetWindowRect")
	getWindow            = user32.NewProc("GetWindow")
	getDesktopWindow     = user32.NewProc("GetDesktopWindow")
	showWindowAsync      = user32.NewProc("ShowWindowAsync")
	postMessageW         = user32.NewProc("PostMessageW")
	isIconic             = user32.NewProc("IsIconic")
	isZoomed             = user32.NewProc("IsZoomed")
	findWindowW          = user32.NewProc("FindWindowW")
	monitorFromWindow    = user32.NewProc("MonitorFromWindow")
)

const (
	SW_MINIMIZE = 6
	SW_MAXIMIZE = 3
	SW_HIDE     = 0
	WM_CLOSE    = 0x0010
	WS_CAPTION  = 0x00C00000
	GW_HWNDNEXT = 2
	GW_HWNDPREV = 3
	GW_CHILD    = 5
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
	Handle      uintptr     `json:"handle"`
	Title       string      `json:"title"`
	Visible     bool        `json:"visible"`
	Minimized   bool        `json:"minimized"`
	Maximized   bool        `json:"maximized"`
	Fullscreen  bool        `json:"fullscreen"`
	Foreground  bool        `json:"foreground"`
	ZOrder      int         `json:"z_order"`
	Rect        *WindowRect `json:"rect,omitempty"`
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

	fg, _, _ := getForegroundWindow.Call()
	info.Foreground = fg == hwnd

	rect, err := GetWindowRectByHandle(hwnd)
	if err == nil {
		info.Rect = rect
	}

	info.Fullscreen = isFullscreen(hwnd)
	info.ZOrder = GetWindowZOrder(hwnd)

	return info, nil
}

// GetWindowZOrder returns the Z-order position for the given window handle.
// 0 = topmost, higher values = deeper in the stack (behind other windows).
// Uses GetDesktopWindow + GW_CHILD to get the true topmost top-level window,
// then walks GW_HWNDNEXT through all siblings. Only windows with WS_VISIBLE
// are counted, so invisible helper windows don't skew the position.
// A window with visible=true, foreground=false, z_order>0 is behind others.
func GetWindowZOrder(hwnd uintptr) int {
	desk, _, _ := getDesktopWindow.Call()
	if desk == 0 {
		return -1
	}
	top, _, _ := getWindow.Call(desk, GW_CHILD)
	if top == 0 {
		return -1
	}
	pos := 0
	cur := top
	for cur != 0 {
		if cur == hwnd {
			return pos
		}
		v, _, _ := isWindowVisible.Call(cur)
		if v != 0 {
			pos++
		}
		cur, _, _ = getWindow.Call(cur, GW_HWNDNEXT)
	}
	return -1
}

func FindWindowByTitle(title string) uintptr {
	t := syscall.StringToUTF16Ptr(title)
	hwnd, _, _ := findWindowW.Call(0, uintptr(unsafe.Pointer(t)))
	if hwnd != 0 {
		return hwnd
	}

	lowerTitle := strings.ToLower(title)
	var found uintptr
	enumCallbackOld := enumCallback
	defer func() { enumCallback = enumCallbackOld }()

	enumCallback = func(h uintptr) bool {
		wt := getWindowTitle(h)
		if strings.Contains(strings.ToLower(wt), lowerTitle) {
			found = h
			return false
		}
		return true
	}
	enumWindows.Call(syscall.NewCallback(windowEnumProc), 0)
	return found
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
