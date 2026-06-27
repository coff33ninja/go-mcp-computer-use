package actions

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	createToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	process32First           = kernel32.NewProc("Process32FirstW")
	process32Next            = kernel32.NewProc("Process32NextW")
	openProcess              = kernel32.NewProc("OpenProcess")
	terminateProcess         = kernel32.NewProc("TerminateProcess")
	closeHandle              = kernel32.NewProc("CloseHandle")
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	PROCESS_TERMINATE  = 0x0001
)

type PROCESSENTRY32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriorityClass   int32
	Flags           uint32
	ExeFile         [260]uint16
}

type ProcessInfo struct {
	PID       uint32 `json:"pid"`
	Name      string `json:"name"`
	Threads   uint32 `json:"threads"`
	ParentPID uint32 `json:"parent_pid"`
}

func ListProcesses() ([]ProcessInfo, error) {
	snapshot, _, _ := createToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == uintptr(^uint32(0)) {
		return nil, syscall.GetLastError()
	}
	defer closeHandle.Call(snapshot)

	var pe PROCESSENTRY32
	pe.Size = uint32(unsafe.Sizeof(pe))
	ret, _, _ := process32First.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	if ret == 0 {
		return nil, nil
	}

	var processes []ProcessInfo
	for ret != 0 {
		name := syscall.UTF16ToString(pe.ExeFile[:])
		processes = append(processes, ProcessInfo{
			PID:       pe.ProcessID,
			Name:      name,
			Threads:   pe.CntThreads,
			ParentPID: pe.ParentProcessID,
		})
		ret, _, _ = process32Next.Call(snapshot, uintptr(unsafe.Pointer(&pe)))
	}

	return processes, nil
}

func LaunchApp(path string) error {
	if path == "" {
		return fmt.Errorf("launch_app: empty path")
	}
	p := syscall.StringToUTF16Ptr(path)
	op := syscall.StringToUTF16Ptr("open")
	ret, _, _ := shellExecuteW.Call(0, uintptr(unsafe.Pointer(op)),
		uintptr(unsafe.Pointer(p)), 0, 0, SW_SHOW)
	if ret <= 32 {
		return syscall.GetLastError()
	}
	return nil
}

type KillResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func KillProcess(pid uint32) error {
	h, _, _ := openProcess.Call(PROCESS_TERMINATE, 0, uintptr(pid))
	if h == 0 {
		return syscall.GetLastError()
	}
	defer closeHandle.Call(h)
	ret, _, _ := terminateProcess.Call(h, 1)
	if ret == 0 {
		return syscall.GetLastError()
	}
	return nil
}
