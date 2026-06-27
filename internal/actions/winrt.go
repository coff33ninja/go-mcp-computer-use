package actions

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	RO_INIT_SINGLETHREADED  = 0
	RO_INIT_MULTITHREADED   = 1
	RO_INIT_APARTMENTTHREADED = 2

	AsyncStatusStarted   = 0
	AsyncStatusCompleted = 1
	AsyncStatusCanceled  = 2
	AsyncStatusError     = 3

	FileAccessModeRead = 0
)

var (
	IID_IInspectable = &windows.GUID{
		Data1: 0xAF86E2E0, Data2: 0xB12D, Data3: 0x4c6a,
		Data4: [8]byte{0x9C, 0x5A, 0xD7, 0xAA, 0x65, 0x10, 0x1E, 0x90},
	}
	IID_IActivationFactory = &windows.GUID{
		Data1: 0x00000035, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
	IID_IAsyncInfo = &windows.GUID{
		Data1: 0x00000036, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}

	IID_IStorageFileStatics = &windows.GUID{
		Data1: 0x5984C710, Data2: 0xDAF2, Data3: 0x43C8,
		Data4: [8]byte{0x8B, 0xB4, 0xA4, 0xD3, 0xEA, 0xCF, 0xD0, 0x3F},
	}
	IID_IStorageFile = &windows.GUID{
		Data1: 0xFA3F6186, Data2: 0x4214, Data3: 0x428C,
		Data4: [8]byte{0xA6, 0x4C, 0x14, 0xC9, 0xAC, 0x73, 0x15, 0xEA},
	}
	IID_IBitmapDecoderStatics = &windows.GUID{
		Data1: 0x438CCB26, Data2: 0xBCEF, Data3: 0x4E95,
		Data4: [8]byte{0xBA, 0xD6, 0x23, 0xA8, 0x22, 0xE5, 0x8D, 0x01},
	}
	IID_IBitmapFrameWithSoftwareBitmap = &windows.GUID{
		Data1: 0xFE287C9A, Data2: 0x420C, Data3: 0x4963,
		Data4: [8]byte{0x87, 0xAD, 0x69, 0x14, 0x36, 0xE0, 0x83, 0x83},
	}
	IID_IOcrEngineStatics = &windows.GUID{
		Data1: 0x5BFFA85A, Data2: 0x3384, Data3: 0x3540,
		Data4: [8]byte{0x99, 0x40, 0x69, 0x91, 0x20, 0xD4, 0x28, 0xA8},
	}
	IID_IOcrEngine = &windows.GUID{
		Data1: 0x5A14BC41, Data2: 0x5B76, Data3: 0x3140,
		Data4: [8]byte{0xB6, 0x80, 0x88, 0x25, 0x56, 0x26, 0x83, 0xAC},
	}
	IID_IOcrResult = &windows.GUID{
		Data1: 0x9BD235B2, Data2: 0x175B, Data3: 0x3D6A,
		Data4: [8]byte{0x92, 0xE2, 0x38, 0x8C, 0x20, 0x6E, 0x2F, 0x63},
	}
	IID_IOcrLine = &windows.GUID{
		Data1: 0x0043A16F, Data2: 0xE31F, Data3: 0x3A24,
		Data4: [8]byte{0x89, 0x9C, 0xD4, 0x44, 0xBD, 0x08, 0x81, 0x24},
	}
	IID_IOcrWord = &windows.GUID{
		Data1: 0x3C2A477A, Data2: 0x5CD9, Data3: 0x3525,
		Data4: [8]byte{0xBA, 0x2A, 0x23, 0xD1, 0xE0, 0xA6, 0x8A, 0x1D},
	}
	IID_ILanguageFactory = &windows.GUID{
		Data1: 0x9B0252AC, Data2: 0x0C27, Data3: 0x44F8,
		Data4: [8]byte{0xB7, 0x92, 0x97, 0x93, 0xFB, 0x66, 0xC6, 0x3E},
	}
)

var (
	modCombase                    = windows.NewLazySystemDLL("combase.dll")
	procRoInitialize              = modCombase.NewProc("RoInitialize")
	procRoUninitialize            = modCombase.NewProc("RoUninitialize")
	procRoGetActivationFactory    = modCombase.NewProc("RoGetActivationFactory")
	procWindowsCreateString       = modCombase.NewProc("WindowsCreateString")
	procWindowsDeleteString       = modCombase.NewProc("WindowsDeleteString")
	procWindowsGetStringRawBuffer = modCombase.NewProc("WindowsGetStringRawBuffer")
)

type HSTRING uintptr

func roInitialize(initType uint32) error {
	r, _, _ := procRoInitialize.Call(uintptr(initType))
	if r != 0 && r != 1 { // 0=S_OK, 1=S_FALSE (already initialized)
		// 0x80010106 = RPC_E_CHANGED_MODE (already initialized in another mode)
		if r != 0x80010106 {
			return fmt.Errorf("RoInitialize 0x%X", r)
		}
	}
	return nil
}

func roUninitialize() {
	procRoUninitialize.Call()
}

func roGetActivationFactory(classID HSTRING, iid *windows.GUID) (unsafe.Pointer, error) {
	var factory unsafe.Pointer
	r, _, _ := procRoGetActivationFactory.Call(
		uintptr(classID),
		uintptr(unsafe.Pointer(iid)),
		uintptr(unsafe.Pointer(&factory)),
	)
	if r != 0 {
		return nil, fmt.Errorf("RoGetActivationFactory 0x%X", r)
	}
	return factory, nil
}

func windowsCreateString(s string) (HSTRING, error) {
	var h HSTRING
	u, err := syscall.UTF16FromString(s)
	if err != nil {
		return 0, err
	}
	// Exclude null terminator
	r, _, _ := procWindowsCreateString.Call(
		uintptr(unsafe.Pointer(&u[0])),
		uintptr(len(u)-1),
		uintptr(unsafe.Pointer(&h)),
	)
	if r != 0 {
		return 0, fmt.Errorf("WindowsCreateString 0x%X", r)
	}
	return h, nil
}

func windowsDeleteString(h HSTRING) error {
	if h == 0 {
		return nil
	}
	r, _, _ := procWindowsDeleteString.Call(uintptr(h))
	if r != 0 {
		return fmt.Errorf("WindowsDeleteString 0x%X", r)
	}
	return nil
}

func hstringToString(h HSTRING) (string, error) {
	if h == 0 {
		return "", nil
	}
	var length uint32
	bufRaw, _, _ := procWindowsGetStringRawBuffer.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&length)),
	)
	if bufRaw == 0 {
		return "", fmt.Errorf("WindowsGetStringRawBuffer returned nil")
	}
	if length == 0 {
		return "", nil
	}
	// SAFETY: bufRaw is a pointer into the HSTRING's internal buffer,
	// which stays alive as long as we hold the HSTRING handle.
	return syscall.UTF16ToString(unsafe.Slice((*uint16)(*(*unsafe.Pointer)(unsafe.Pointer(&bufRaw))), length)), nil
}

func qei(obj unsafe.Pointer, iid *windows.GUID) (unsafe.Pointer, error) {
	if obj == nil {
		return nil, fmt.Errorf("QI on nil")
	}
	var result unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(obj, 0), uintptr(obj), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&result)))
	if r != 0 {
		return nil, fmt.Errorf("QI 0x%X", r)
	}
	return result, nil
}

func waitForAsync(op unsafe.Pointer, timeout time.Duration) error {
	info, err := qei(op, IID_IAsyncInfo)
	if err != nil {
		return fmt.Errorf("QI IAsyncInfo: %w", err)
	}
	defer comRelease(info)

	deadline := time.Now().Add(timeout)
	for {
		var status int32
		r, _, _ := syscall.SyscallN(vtblMethod(info, 7), uintptr(info), uintptr(unsafe.Pointer(&status)))
		if r != 0 {
			return fmt.Errorf("get_Status 0x%X", r)
		}
		switch status {
		case AsyncStatusCompleted:
			return nil
		case AsyncStatusError:
			var errCode uint32
			syscall.SyscallN(vtblMethod(info, 8), uintptr(info), uintptr(unsafe.Pointer(&errCode)))
			return fmt.Errorf("async error 0x%X", errCode)
		case AsyncStatusCanceled:
			return fmt.Errorf("async cancelled")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("async timeout")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func getAsyncObj(op unsafe.Pointer, timeout time.Duration) (unsafe.Pointer, error) {
	if err := waitForAsync(op, timeout); err != nil {
		return nil, err
	}
	var result unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(op, 8), uintptr(op), uintptr(unsafe.Pointer(&result)))
	if r != 0 {
		return nil, fmt.Errorf("GetResults 0x%X", r)
	}
	return result, nil
}

func callStringGetter(obj unsafe.Pointer, idx int) (string, error) {
	var h HSTRING
	r, _, _ := syscall.SyscallN(vtblMethod(obj, idx), uintptr(obj), uintptr(unsafe.Pointer(&h)))
	if r != 0 {
		return "", fmt.Errorf("getter[%d] 0x%X", idx, r)
	}
	defer windowsDeleteString(h)
	return hstringToString(h)
}

func callObjectGetter(obj unsafe.Pointer, idx int) (unsafe.Pointer, error) {
	var result unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(obj, idx), uintptr(obj), uintptr(unsafe.Pointer(&result)))
	if r != 0 {
		return nil, fmt.Errorf("getter[%d] 0x%X", idx, r)
	}
	return result, nil
}
