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
