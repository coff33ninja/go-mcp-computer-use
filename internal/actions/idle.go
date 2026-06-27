package actions

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	getTickCount64   = kernel32.NewProc("GetTickCount64")
	getLastInputInfo = user32.NewProc("GetLastInputInfo")
)

type LASTINPUTINFO struct {
	CbSize uint32
	DwTime uint32
}

func GetIdleTime() (time.Duration, error) {
	lpi := LASTINPUTINFO{CbSize: uint32(unsafe.Sizeof(LASTINPUTINFO{}))}
	ret, _, _ := getLastInputInfo.Call(uintptr(unsafe.Pointer(&lpi)))
	if ret == 0 {
		return 0, syscall.GetLastError()
	}
	tick, _, _ := getTickCount64.Call()
	currentTick := uint32(tick & 0xFFFFFFFF)
	idleMs := currentTick - lpi.DwTime
	return time.Duration(idleMs) * time.Millisecond, nil
}
