package actions

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var (
	getKeyboardLayoutNameW = user32.NewProc("GetKeyboardLayoutNameW")
)

type KeyboardLayoutInfo struct {
	Current string `json:"current"`
	Available []string `json:"available,omitempty"`
}

func GetKeyboardLayout() (*KeyboardLayoutInfo, error) {
	var buf [9]uint16
	ret, _, _ := getKeyboardLayoutNameW.Call(uintptr(unsafe.Pointer(&buf[0])))
	if ret == 0 {
		return nil, syscall.GetLastError()
	}
	current := syscall.UTF16ToString(buf[:])

	out, err := exec.Command("powershell", "-NoProfile", "-Command",
		`(Get-WinUserLanguageList).LanguageTag -join ','`).Output()
	available := []string{}
	if err == nil {
		s := strings.TrimSpace(string(out))
		if s != "" {
			available = strings.Split(s, ",")
		}
	}

	return &KeyboardLayoutInfo{
		Current:   current,
		Available: available,
	}, nil
}

func SetKeyboardLayout(lang string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(`$lang=New-WinUserLanguageList -Language %s; Set-WinUserLanguageList -LanguageList $lang.LanguageTag -Force`, lang))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set keyboard layout: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func GetScreenDPI() (int, error) {
	hdc := GetDesktopDC()
	if hdc == 0 {
		return 0, syscall.GetLastError()
	}
	defer ReleaseDesktopDC(hdc)

	dpiX, _, _ := gdi32.NewProc("GetDeviceCaps").Call(hdc, 88)
	return int(dpiX), nil
}
