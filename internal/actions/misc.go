package actions

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ntdll           = windows.NewLazySystemDLL("ntdll.dll")
	delayExecution  = ntdll.NewProc("NtDelayExecution")

	getSystemPowerStatus = kernel32.NewProc("GetSystemPowerStatus")
)

const (
	batteryFlagHigh   = 0x01
	batteryFlagLow    = 0x02
	batteryFlagCritical = 0x04
	batteryFlagCharging  = 0x08
	batteryFlagNoBattery = 0x80
)

type SYSTEM_POWER_STATUS struct {
	ACLineStatus   byte
	BatteryFlag    byte
	BatteryLifePercent byte
	Reserved1      byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

type BatteryStatus struct {
	OnBattery  bool   `json:"on_battery"`
	Charging   bool   `json:"charging"`
	Percentage int    `json:"percentage,omitempty"`
	NoBattery  bool   `json:"no_battery"`
}

type DisplayInfo struct {
	Name     string `json:"name"`
	Width    int32  `json:"width"`
	Height   int32  `json:"height"`
	PositionX int32 `json:"position_x"`
	PositionY int32 `json:"position_y"`
	Primary  bool   `json:"primary"`
}

type DisplayMode struct {
	Name        string `json:"name"`
	Width       int32  `json:"width"`
	Height      int32  `json:"height"`
	RefreshRate int32  `json:"refresh_rate"`
	BitsPerPel  int32  `json:"bits_per_pel"`
}

type DEVMODEW struct {
	DeviceName      [32]uint16
	SpecVersion     uint16
	DriverVersion   uint16
	Size            uint16
	DriverExtra     uint16
	Fields          uint32
	Orientation     int16
	PaperSize       int16
	PaperLength     int16
	PaperWidth      int16
	Scale           int16
	Copies          int16
	DefaultSource   int16
	PrintQuality    int16
	Color           int16
	Duplex          int16
	YResolution     int16
	TTOption        int16
	Collate         int16
	FormName        [32]uint16
	LogPixels       uint16
	BitsPerPel      uint32
	PelsWidth       uint32
	PelsHeight      uint32
	DisplayFlags    uint32
	DisplayFrequency uint32
}

var (
	enumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	getMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
	enumDisplaySettingsW = user32.NewProc("EnumDisplaySettingsW")
)

type MONITORINFOEX struct {
	Size    uint32
	Monitor struct {
		Left, Top, Right, Bottom int32
	}
	WorkArea struct {
		Left, Top, Right, Bottom int32
	}
	Flags    uint32
	DeviceName [32]uint16
}

var monitorCallback func(DisplayInfo) bool

func monitorEnumProc(hmonitor uintptr, hdc uintptr, rect uintptr, lparam uintptr) uintptr {
	var mi MONITORINFOEX
	mi.Size = uint32(unsafe.Sizeof(mi))
	getMonitorInfoW.Call(hmonitor, uintptr(unsafe.Pointer(&mi)))
	if mi.Flags&1 != 0 {
		info := DisplayInfo{
			Name:      syscall.UTF16ToString(mi.DeviceName[:]),
			Width:     mi.Monitor.Right - mi.Monitor.Left,
			Height:    mi.Monitor.Bottom - mi.Monitor.Top,
			PositionX: mi.Monitor.Left,
			PositionY: mi.Monitor.Top,
			Primary:   mi.Flags&1 != 0,
		}
		if monitorCallback(info) {
			return 1
		}
	}
	return 1
}

func ListDisplays() ([]DisplayInfo, error) {
	var displays []DisplayInfo
	callback := func(info DisplayInfo) bool {
		displays = append(displays, info)
		return true
	}
	monitorCallback = callback

	cb := syscall.NewCallback(monitorEnumProc)
	enumDisplayMonitors.Call(0, 0, cb, 0)

	return displays, nil
}

func GetDisplayModes(deviceName string) ([]DisplayMode, error) {
	dn := syscall.StringToUTF16Ptr(deviceName)
	var modes []DisplayMode
	for i := uint32(0); ; i++ {
		var dm DEVMODEW
		dm.Size = uint16(unsafe.Sizeof(dm))
		ret, _, _ := enumDisplaySettingsW.Call(
			uintptr(unsafe.Pointer(dn)), uintptr(i), uintptr(unsafe.Pointer(&dm)))
		if ret == 0 {
			break
		}
		modes = append(modes, DisplayMode{
			Name:        deviceName,
			Width:       int32(dm.PelsWidth),
			Height:      int32(dm.PelsHeight),
			RefreshRate: int32(dm.DisplayFrequency),
			BitsPerPel:  int32(dm.BitsPerPel),
		})
	}
	if modes == nil {
		return nil, fmt.Errorf("no display modes found for %s", deviceName)
	}
	return modes, nil
}

func GetBattery() (*BatteryStatus, error) {
	var sps SYSTEM_POWER_STATUS
	ret, _, _ := getSystemPowerStatus.Call(uintptr(unsafe.Pointer(&sps)))
	if ret == 0 {
		return &BatteryStatus{NoBattery: true}, nil
	}

	status := &BatteryStatus{
		NoBattery: sps.BatteryFlag&batteryFlagNoBattery != 0,
		OnBattery: sps.ACLineStatus == 0,
		Charging:  sps.BatteryFlag&batteryFlagCharging != 0,
	}
	if sps.BatteryLifePercent <= 100 {
		status.Percentage = int(sps.BatteryLifePercent)
	}

	return status, nil
}

func Wait(ms int32) {
	if ms <= 0 {
		return
	}
	duration := time.Duration(ms) * time.Millisecond
	du := -(int64(duration) * 10000)
	delayExecution.Call(0, uintptr(unsafe.Pointer(&du)))
}

func ShowNotification(title, message string) error {
	ti := syscall.StringToUTF16Ptr(title)
	msg := syscall.StringToUTF16Ptr(message)

	user32.NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(msg)),
		uintptr(unsafe.Pointer(ti)), 0)
	return nil
}

var lockWorkStation = user32.NewProc("LockWorkStation")

func LockWorkstation() error {
	lockWorkStation.Call()
	return nil
}

func GetPixelColor(x, y int32) (string, error) {
	if err := ValidateClickCoord(x, y); err != nil {
		return "", err
	}
	hdc := GetDesktopDC()
	if hdc == 0 {
		return "", syscall.GetLastError()
	}
	defer ReleaseDesktopDC(hdc)

	px, _, _ := gdi32.NewProc("GetPixel").Call(hdc, uintptr(x), uintptr(y))
	if px == uintptr(^uint32(0)) {
		return "", syscall.GetLastError()
	}
	r := px & 0xFF
	g := (px >> 8) & 0xFF
	b := (px >> 16) & 0xFF
	return fmt.Sprintf("#%02x%02x%02x", r, g, b), nil
}
