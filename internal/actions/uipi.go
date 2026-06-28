package actions

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi32            = windows.NewLazySystemDLL("advapi32.dll")
	openProcessToken    = advapi32.NewProc("OpenProcessToken")
	getTokenInformation = advapi32.NewProc("GetTokenInformation")
)

const (
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	TOKEN_QUERY                       = 0x0008
	TokenElevation                    = 20
)

func isForegroundElevated() (bool, error) {
	hwnd, _, _ := getForegroundWindow.Call()
	if hwnd == 0 {
		return false, nil
	}

	var pid uint32
	getWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return false, nil
	}

	hProcess, _, _ := openProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if hProcess == 0 {
		if syscall.GetLastError() == windows.ERROR_ACCESS_DENIED {
			return true, nil
		}
		return false, nil
	}
	defer closeHandle.Call(hProcess)

	var hToken uintptr
	r, _, _ := openProcessToken.Call(hProcess, TOKEN_QUERY, uintptr(unsafe.Pointer(&hToken)))
	if r == 0 {
		return false, nil
	}
	defer closeHandle.Call(hToken)

	var elevated uint32
	var retLen uint32
	r, _, _ = getTokenInformation.Call(hToken, TokenElevation,
		uintptr(unsafe.Pointer(&elevated)), uintptr(unsafe.Sizeof(elevated)),
		uintptr(unsafe.Pointer(&retLen)))
	if r == 0 {
		return false, nil
	}
	return elevated != 0, nil
}

func isProcessElevated(pid uint32) (bool, error) {
	hProcess, _, _ := openProcess.Call(PROCESS_QUERY_LIMITED_INFORMATION, 0, uintptr(pid))
	if hProcess == 0 {
		if syscall.GetLastError() == windows.ERROR_ACCESS_DENIED {
			return true, nil
		}
		return false, nil
	}
	defer closeHandle.Call(hProcess)

	var hToken uintptr
	r, _, _ := openProcessToken.Call(hProcess, TOKEN_QUERY, uintptr(unsafe.Pointer(&hToken)))
	if r == 0 {
		return false, nil
	}
	defer closeHandle.Call(hToken)

	var elevated uint32
	var retLen uint32
	r, _, _ = getTokenInformation.Call(hToken, TokenElevation,
		uintptr(unsafe.Pointer(&elevated)), uintptr(unsafe.Sizeof(elevated)),
		uintptr(unsafe.Pointer(&retLen)))
	if r == 0 {
		return false, nil
	}
	return elevated != 0, nil
}

func isSelfElevated() (bool, error) {
	return isProcessElevated(uint32(syscall.Getpid()))
}

func warnElevated() error {
	targetElevated, err := isForegroundElevated()
	if err != nil {
		return nil
	}
	if !targetElevated {
		return nil
	}
	selfElevated, err := isSelfElevated()
	if err != nil {
		return nil
	}
	if selfElevated {
		return nil
	}
	return fmt.Errorf("foreground window is elevated (admin). Input from non-elevated MCP server is blocked by Windows UIPI. Run mcp-server.exe as Administrator or target a non-elevated window")
}
