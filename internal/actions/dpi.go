package actions

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shcore                = windows.NewLazySystemDLL("shcore.dll")
	getDpiForMonitor      = shcore.NewProc("GetDpiForMonitor")
	setProcessDpiAwareness = shcore.NewProc("SetProcessDpiAwareness")
)

const (
	MONITOR_DEFAULTTONEAREST = 2
)

func SetDPIAware() {
	setProcessDpiAwareness.Call(2) // PROCESS_PER_MONITOR_DPI_AWARE
}

type MonitorDPI struct {
	Name string `json:"name"`
	DPI  int    `json:"dpi"`
	ScalePercent int `json:"scale_percent"`
}

var monitorDPICallback func(string, int)

func monitorDPIEnumProc(hmonitor uintptr, hdc uintptr, rect uintptr, lparam uintptr) uintptr {
	var mi MONITORINFOEX
	mi.Size = uint32(unsafe.Sizeof(mi))
	getMonitorInfoW.Call(hmonitor, uintptr(unsafe.Pointer(&mi)))

	var dpiX, dpiY uint32
	getDpiForMonitor.Call(hmonitor, 0, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))

	name := syscall.UTF16ToString(mi.DeviceName[:])
	if dpiX > 0 && monitorDPICallback != nil {
		monitorDPICallback(name, int(dpiX))
	}
	return 1
}

func ListMonitorDPIs() ([]MonitorDPI, error) {
	var monitors []MonitorDPI
	monitorDPICallback = func(name string, dpi int) {
		scale := (dpi * 100) / 96
		monitors = append(monitors, MonitorDPI{
			Name:         name,
			DPI:          dpi,
			ScalePercent: scale,
		})
	}

	cb := syscall.NewCallback(monitorDPIEnumProc)
	enumDisplayMonitors.Call(0, 0, cb, 0)

	if len(monitors) == 0 {
		return nil, fmt.Errorf("no monitors found")
	}
	return monitors, nil
}

func GetDPIScaleForPoint(x, y int32) (int, error) {
	monitor, _, _ := user32.NewProc("MonitorFromPoint").Call(
		uintptr(x&0xFFFF)|(uintptr(y)<<16),
		MONITOR_DEFAULTTONEAREST,
	)
	if monitor == 0 {
		return 96, nil
	}

	var dpiX, dpiY uint32
	ret, _, _ := getDpiForMonitor.Call(monitor, 0, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
	if ret == 0 && dpiX > 0 {
		return int(dpiX), nil
	}
	return 96, nil
}

func ScaleCoordinate(x, y int32) (int32, int32) {
	dpi, err := GetDPIScaleForPoint(x, y)
	if err != nil {
		return x, y
	}
	if dpi == 96 {
		return x, y
	}
	scale := float64(dpi) / 96.0
	return int32(float64(x) / scale), int32(float64(y) / scale)
}

func GetDPIScaleForWindow(hwnd uintptr) float64 {
	rect, err := GetWindowRectByHandle(hwnd)
	if err != nil {
		return 1.0
	}
	cx := rect.Left + rect.Width/2
	cy := rect.Top + rect.Height/2
	dpi, err := GetDPIScaleForPoint(cx, cy)
	if err != nil {
		return 1.0
	}
	return float64(dpi) / 96.0
}

type NormalizedElement struct {
	Class      string  `json:"class"`
	Confidence float64 `json:"confidence"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	W          float64 `json:"w"`
	H          float64 `json:"h"`
}

type WindowNormalizer struct {
	WindowRect WindowRect
	DPIScale   float64
}

func NewWindowNormalizer(hwnd uintptr) (*WindowNormalizer, error) {
	rect, err := GetWindowRectByHandle(hwnd)
	if err != nil {
		return nil, fmt.Errorf("window normalizer: %w", err)
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return nil, fmt.Errorf("window normalizer: invalid dimensions %dx%d", rect.Width, rect.Height)
	}
	return &WindowNormalizer{
		WindowRect: *rect,
		DPIScale:   GetDPIScaleForWindow(hwnd),
	}, nil
}

func (wn *WindowNormalizer) Normalize(x, y, w, h int32) NormalizedElement {
	return NormalizedElement{
		X: float64(x-wn.WindowRect.Left) / float64(wn.WindowRect.Width),
		Y: float64(y-wn.WindowRect.Top) / float64(wn.WindowRect.Height),
		W: float64(w) / float64(wn.WindowRect.Width),
		H: float64(h) / float64(wn.WindowRect.Height),
	}
}

func (wn *WindowNormalizer) Denormalize(elem NormalizedElement) (int32, int32, int32, int32) {
	x := int32(elem.X*float64(wn.WindowRect.Width) + float64(wn.WindowRect.Left))
	y := int32(elem.Y*float64(wn.WindowRect.Height) + float64(wn.WindowRect.Top))
	w := int32(elem.W * float64(wn.WindowRect.Width))
	h := int32(elem.H * float64(wn.WindowRect.Height))
	return x, y, w, h
}

func (wn *WindowNormalizer) NormalizeElement(elem DetectedElement) NormalizedElement {
	n := wn.Normalize(elem.X, elem.Y, elem.W, elem.H)
	n.Class = elem.Class
	n.Confidence = elem.Confidence
	return n
}

func (wn *WindowNormalizer) DenormalizeElement(elem NormalizedElement) DetectedElement {
	x, y, w, h := wn.Denormalize(elem)
	return DetectedElement{
		Class:      elem.Class,
		Confidence: elem.Confidence,
		X:          x, Y: y, W: w, H: h,
	}
}

func (wn *WindowNormalizer) ProportionalRegion(leftFrac, topFrac, rightFrac, bottomFrac float64) (int32, int32, int32, int32) {
	x := wn.WindowRect.Left + int32(leftFrac*float64(wn.WindowRect.Width))
	y := wn.WindowRect.Top + int32(topFrac*float64(wn.WindowRect.Height))
	r := wn.WindowRect.Left + int32(rightFrac*float64(wn.WindowRect.Width))
	b := wn.WindowRect.Top + int32(bottomFrac*float64(wn.WindowRect.Height))
	return x, y, r - x, b - y
}
