package actions

import "golang.org/x/sys/windows"

var user32 = windows.NewLazySystemDLL("user32.dll")

var (
	getSystemMetrics = user32.NewProc("GetSystemMetrics")
	getDC            = user32.NewProc("GetDC")
	releaseDC        = user32.NewProc("ReleaseDC")
	sendInput        = user32.NewProc("SendInput")
	getCursorPos     = user32.NewProc("GetCursorPos")
	setCursorPos     = user32.NewProc("SetCursorPos")
)

func ScreenSize() (int32, int32) {
	w, _, _ := getSystemMetrics.Call(78) // SM_CXVIRTUALSCREEN
	h, _, _ := getSystemMetrics.Call(79) // SM_CYVIRTUALSCREEN
	return int32(w), int32(h)
}

func GetDesktopDC() uintptr {
	hdc, _, _ := getDC.Call(0)
	return hdc
}

func ReleaseDesktopDC(hdc uintptr) {
	releaseDC.Call(0, hdc)
}
