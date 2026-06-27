package actions

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	winmm          = syscall.NewLazyDLL("winmm.dll")
	waveOutGetVol  = winmm.NewProc("waveOutGetVolume")
	waveOutSetVol  = winmm.NewProc("waveOutSetVolume")

	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getComputerNameW   = kernel32.NewProc("GetComputerNameW")
	globalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	globalAlloc        = kernel32.NewProc("GlobalAlloc")
	globalLock         = kernel32.NewProc("GlobalLock")
	globalUnlock       = kernel32.NewProc("GlobalUnlock")

	shell32       = syscall.NewLazyDLL("shell32.dll")
	shellExecuteW = shell32.NewProc("ShellExecuteW")

	openClipboard      = user32.NewProc("OpenClipboard")
	closeClipboard     = user32.NewProc("CloseClipboard")
	getClipboardData   = user32.NewProc("GetClipboardData")
	setClipboardData   = user32.NewProc("SetClipboardData")
	emptyClipboard     = user32.NewProc("EmptyClipboard")
	getForegroundWindow = user32.NewProc("GetForegroundWindow")
)

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
	SW_SHOW        = 5
)

type SystemInfo struct {
	Hostname   string `json:"hostname"`
	OS         string `json:"os"`
	TotalRAMMB uint64 `json:"total_ram_mb"`
	FreeRAMMB  uint64 `json:"free_ram_mb"`
}

type MEMORYSTATUSEX struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

type ActiveWindowInfo struct {
	Handle uintptr `json:"handle"`
	Title  string  `json:"title"`
	PID    uint32  `json:"pid"`
}

func GetVolume() (uint32, error) {
	var vol uint32
	ret, _, _ := waveOutGetVol.Call(uintptr(^uint32(0)), uintptr(unsafe.Pointer(&vol)))
	if ret != 0 {
		return 0, syscall.Errno(ret)
	}
	left := vol & 0xFFFF
	right := (vol >> 16) & 0xFFFF
	avg := (left + right) / 2
	return (avg * 100) / 0xFFFF, nil
}

func SetVolume(pct uint32) error {
	if pct > 100 {
		pct = 100
	}
	val := (pct * 0xFFFF) / 100
	vol := val | (val << 16)
	ret, _, _ := waveOutSetVol.Call(uintptr(^uint32(0)), uintptr(vol))
	if ret != 0 {
		return syscall.Errno(ret)
	}
	return nil
}

func SetMute(mute bool) error {
	if mute {
		return SetVolume(0)
	}
	return SetVolume(50)
}

func GetSystemInfo() (*SystemInfo, error) {
	buf := make([]uint16, 64)
	n := uint32(len(buf))
	ret, _, _ := getComputerNameW.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	hostname := ""
	if ret != 0 {
		hostname = syscall.UTF16ToString(buf[:n])
	}

	var mem MEMORYSTATUSEX
	mem.Length = uint32(unsafe.Sizeof(mem))
	globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))

	return &SystemInfo{
		Hostname:   hostname,
		OS:         "Windows",
		TotalRAMMB: mem.TotalPhys / 1024 / 1024,
		FreeRAMMB:  mem.AvailPhys / 1024 / 1024,
	}, nil
}

func GetActiveWindowInfo() (*ActiveWindowInfo, error) {
	hwnd, _, _ := getForegroundWindow.Call()
	if hwnd == 0 {
		return &ActiveWindowInfo{}, nil
	}
	title := getWindowTitle(hwnd)
	pid := getWindowPID(hwnd)
	return &ActiveWindowInfo{
		Handle: hwnd,
		Title:  title,
		PID:    pid,
	}, nil
}

func openClipboardWithRetry() error {
	for i := 0; i < 5; i++ {
		ret, _, _ := openClipboard.Call(0)
		if ret != 0 {
			return nil
		}
		Wait(100)
	}
	return fmt.Errorf("clipboard is held by another application")
}

func GetClipboardText() (string, error) {
	var result string
	err := WithTimeout(func() error {
		if err := openClipboardWithRetry(); err != nil {
			return err
		}
		defer closeClipboard.Call()

		h, _, _ := getClipboardData.Call(CF_UNICODETEXT)
		if h == 0 {
			return nil
		}
		p, _, _ := globalLock.Call(h)
		if p == 0 {
			return nil
		}
		defer globalUnlock.Call(h)

		result = syscall.UTF16ToString((*[4096]uint16)(unsafe.Pointer(p))[:])
		return nil
	})
	return result, err
}

func SetClipboardText(text string) error {
	return WithTimeout(func() error {
		if err := openClipboardWithRetry(); err != nil {
			return err
		}
		defer closeClipboard.Call()
		emptyClipboard.Call()

		utf16 := syscall.StringToUTF16(text)
		size := uintptr(len(utf16) * 2)
		h, _, _ := globalAlloc.Call(GMEM_MOVEABLE, size)
		if h == 0 {
			return syscall.GetLastError()
		}
		p, _, _ := globalLock.Call(h)
		if p == 0 {
			return syscall.GetLastError()
		}
		copy((*[1 << 20]uint16)(unsafe.Pointer(p))[:], utf16)
		globalUnlock.Call(h)

		setClipboardData.Call(CF_UNICODETEXT, h)
		return nil
	})
}

func OpenURL(url string) error {
	u := syscall.StringToUTF16Ptr(url)
	op := syscall.StringToUTF16Ptr("open")
	ret, _, _ := shellExecuteW.Call(0, uintptr(unsafe.Pointer(op)),
		uintptr(unsafe.Pointer(u)), 0, 0, SW_SHOW)
	if ret <= 32 {
		return syscall.GetLastError()
	}
	return nil
}
