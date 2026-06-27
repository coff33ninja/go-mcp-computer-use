package actions

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	powrprof         = windows.NewLazySystemDLL("powrprof.dll")
	setSuspendState  = powrprof.NewProc("SetSuspendState")

	exitWindowsEx    = user32.NewProc("ExitWindowsEx")
)

const (
	EWX_SHUTDOWN  = 0x00000001
	EWX_REBOOT    = 0x00000002
	EWX_FORCE     = 0x00000004
	EWX_POWEROFF  = 0x00000008
)

func GetUptime() (time.Duration, error) {
	tick, _, _ := getTickCount64.Call()
	if tick == 0 {
		return 0, syscall.GetLastError()
	}
	return time.Duration(tick) * time.Millisecond, nil
}

func Shutdown() error {
	ret, _, _ := exitWindowsEx.Call(EWX_SHUTDOWN|EWX_FORCE, 0)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func Restart() error {
	ret, _, _ := exitWindowsEx.Call(EWX_REBOOT|EWX_FORCE, 0)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}

func Sleep() error {
	ret, _, _ := setSuspendState.Call(0, 0, 1)
	if ret == 0 {
		return fmt.Errorf("sleep: %w", syscall.GetLastError())
	}
	return nil
}

func Hibernate() error {
	ret, _, _ := setSuspendState.Call(1, 0, 1)
	if ret == 0 {
		return fmt.Errorf("hibernate: %w", syscall.GetLastError())
	}
	return nil
}

type DiskUsage struct {
	Path       string `json:"path"`
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
	UsagePct   float64 `json:"usage_percent"`
}

var (
	getDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
	getLogicalDrives    = kernel32.NewProc("GetLogicalDrives")
	getDriveTypeW       = kernel32.NewProc("GetDriveTypeW")
)

const (
	DRIVE_UNKNOWN    = 0
	DRIVE_NO_ROOT_DIR = 1
	DRIVE_REMOVABLE  = 2
	DRIVE_FIXED      = 3
	DRIVE_REMOTE     = 4
	DRIVE_CDROM      = 5
	DRIVE_RAMDISK    = 6
)

func GetDiskUsage() ([]DiskUsage, error) {
	mask, _, _ := getLogicalDrives.Call()
	var result []DiskUsage
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		drive := string(rune('A'+i)) + ":\\"
		root := syscall.StringToUTF16Ptr(drive)
		dt, _, _ := getDriveTypeW.Call(uintptr(unsafe.Pointer(root)))
		if dt != DRIVE_FIXED && dt != DRIVE_RAMDISK {
			continue
		}
		var free, total, totalFree uint64
		ret, _, _ := getDiskFreeSpaceExW.Call(
			uintptr(unsafe.Pointer(root)),
			uintptr(unsafe.Pointer(&free)),
			uintptr(unsafe.Pointer(&total)),
			uintptr(unsafe.Pointer(&totalFree)),
		)
		if ret == 0 {
			continue
		}
		used := total - free
		pct := float64(used) / float64(total) * 100
		result = append(result, DiskUsage{
			Path:       drive,
			TotalBytes: total,
			FreeBytes:  free,
			UsedBytes:  used,
			UsagePct:   pct,
		})
	}
	return result, nil
}

func OpenFileExplorer(path string) error {
	verb := syscall.StringToUTF16Ptr("explore")
	p := syscall.StringToUTF16Ptr(path)
	ret, _, _ := shellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(p)), 0, 0, SW_SHOW)
	if ret <= 32 {
		return syscall.GetLastError()
	}
	return nil
}

func OpenFileLocation(path string) error {
	verb := syscall.StringToUTF16Ptr("open")
	operation := syscall.StringToUTF16Ptr("explorer")
	params := syscall.StringToUTF16Ptr(fmt.Sprintf("/select,\"%s\"", path))
	ret, _, _ := shellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(operation)), uintptr(unsafe.Pointer(params)), 0, SW_SHOW)
	if ret <= 32 {
		return syscall.GetLastError()
	}
	return nil
}
