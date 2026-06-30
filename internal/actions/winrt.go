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

	// Discovered via scripts\discover-winrt-iids.ps1 2026-06-30, verified Win11 26200
	IID_IBitmapDecoderStatics2 = &windows.GUID{
		Data1: 0x50BA68EA, Data2: 0x99A1, Data3: 0x40C4,
		Data4: [8]byte{0x80, 0xD9, 0xAE, 0xF0, 0xDA, 0xFA, 0x6C, 0x3F},
	}
	IID_ISoftwareBitmap = &windows.GUID{
		Data1: 0x689E0708, Data2: 0x7EEF, Data3: 0x483F,
		Data4: [8]byte{0x96, 0x3F, 0xDA, 0x93, 0x88, 0x18, 0xE0, 0x73},
	}
	IID_IBitmapDecoder = &windows.GUID{
		Data1: 0xACEF22BA, Data2: 0x1D74, Data3: 0x4C91,
		Data4: [8]byte{0x9D, 0xFC, 0x96, 0x20, 0x74, 0x52, 0x33, 0xE6},
	}
	IID_ILanguage = &windows.GUID{
		Data1: 0xEA79A752, Data2: 0xF7C2, Data3: 0x4265,
		Data4: [8]byte{0xB1, 0xBD, 0xC4, 0xDE, 0xC4, 0xE4, 0xF0, 0x80},
	}
	IID_IRandomAccessStreamWithContentType = &windows.GUID{
		Data1: 0xCC254827, Data2: 0x4B3D, Data3: 0x438F,
		Data4: [8]byte{0x92, 0x32, 0x10, 0xC7, 0x6B, 0xC7, 0xE0, 0x38},
	}
	IID_IBitmapFrame = &windows.GUID{
		Data1: 0x72A49A1C, Data2: 0x8081, Data3: 0x438D,
		Data4: [8]byte{0x91, 0xBC, 0x94, 0xEC, 0xFC, 0x81, 0x85, 0xC6},
	}
	IID_IRandomAccessStreamStatics = &windows.GUID{
		Data1: 0x524CEDCF, Data2: 0x6E29, Data3: 0x4CE5,
		Data4: [8]byte{0x95, 0x73, 0x6B, 0x75, 0x3D, 0xB6, 0x6C, 0x3A},
	}

	// ── Storage & Streams (extended) ──
	IID_IStorageFileStatics2 = &windows.GUID{
		Data1: 0x5C76A781, Data2: 0x212E, Data3: 0x4AF9,
		Data4: [8]byte{0x8F, 0x04, 0x74, 0x0C, 0xAE, 0x10, 0x89, 0x74},
	}
	IID_IRandomAccessStream = &windows.GUID{
		Data1: 0x905A0FE1, Data2: 0xBC53, Data3: 0x11DF,
		Data4: [8]byte{0x8C, 0x49, 0x00, 0x1E, 0x4F, 0xC6, 0x86, 0xDA},
	}
	IID_IInputStream = &windows.GUID{
		Data1: 0x905A0FE2, Data2: 0xBC53, Data3: 0x11DF,
		Data4: [8]byte{0x8C, 0x49, 0x00, 0x1E, 0x4F, 0xC6, 0x86, 0xDA},
	}
	IID_IOutputStream = &windows.GUID{
		Data1: 0x905A0FE6, Data2: 0xBC53, Data3: 0x11DF,
		Data4: [8]byte{0x8C, 0x49, 0x00, 0x1E, 0x4F, 0xC6, 0x86, 0xDA},
	}
	IID_IDataWriter = &windows.GUID{
		Data1: 0x64B89265, Data2: 0xD341, Data3: 0x4922,
		Data4: [8]byte{0xB3, 0x8A, 0xDD, 0x4A, 0xF8, 0x80, 0x8C, 0x4E},
	}
	IID_IDataReader = &windows.GUID{
		Data1: 0xE2B50029, Data2: 0xB4C1, Data3: 0x4314,
		Data4: [8]byte{0xA4, 0xB8, 0xFB, 0x81, 0x3A, 0x2F, 0x27, 0x5E},
	}
	IID_IFileOpenPicker = &windows.GUID{
		Data1: 0x8CEB6CD2, Data2: 0xB446, Data3: 0x46F7,
		Data4: [8]byte{0xB2, 0x65, 0x90, 0xF8, 0xE5, 0x5A, 0xD6, 0x50},
	}
	IID_IFileSavePicker = &windows.GUID{
		Data1: 0x0EC313A2, Data2: 0xD24B, Data3: 0x449A,
		Data4: [8]byte{0x81, 0x97, 0xE8, 0x91, 0x04, 0xFD, 0x42, 0xCC},
	}
	IID_IFileOpenPickerStatics = &windows.GUID{
		Data1: 0x6821573B, Data2: 0x2F02, Data3: 0x4833,
		Data4: [8]byte{0x96, 0xD4, 0xAB, 0xBF, 0xAD, 0x72, 0xB6, 0x7B},
	}
	IID_IFileSavePickerStatics = &windows.GUID{
		Data1: 0x28E3CF9E, Data2: 0x961C, Data3: 0x5E2C,
		Data4: [8]byte{0xAE, 0xD7, 0xE6, 0x47, 0x37, 0xF4, 0xCE, 0x37},
	}

	// ── Bitmap / Imaging (extended) ──
	IID_IBitmapEncoder = &windows.GUID{
		Data1: 0x2BC468E3, Data2: 0xE1F8, Data3: 0x4B54,
		Data4: [8]byte{0x95, 0xE8, 0x32, 0x91, 0x95, 0x51, 0xCE, 0x62},
	}
	IID_IBitmapEncoderStatics = &windows.GUID{
		Data1: 0xA74356A7, Data2: 0xA4E4, Data3: 0x4EB9,
		Data4: [8]byte{0x8E, 0x40, 0x56, 0x4D, 0xE7, 0xE1, 0xCC, 0xB2},
	}

	// ── Globalization (extended) ──
	IID_ILanguageStatics = &windows.GUID{
		Data1: 0xB23CD557, Data2: 0x0865, Data3: 0x46D4,
		Data4: [8]byte{0x89, 0xB8, 0xD5, 0x9B, 0xE8, 0x99, 0x0F, 0x0D},
	}

	// ── Devices & Audio ──
	IID_IDeviceInformation = &windows.GUID{
		Data1: 0xABA0FB95, Data2: 0x4398, Data3: 0x489D,
		Data4: [8]byte{0x8E, 0x44, 0xE6, 0x13, 0x09, 0x27, 0x01, 0x1F},
	}
	IID_IDeviceInformationStatics = &windows.GUID{
		Data1: 0xC17F100E, Data2: 0x3A46, Data3: 0x4A78,
		Data4: [8]byte{0x80, 0x13, 0x76, 0x9D, 0xC9, 0xB9, 0x73, 0x90},
	}
	IID_IMediaDeviceStatics = &windows.GUID{
		Data1: 0xAA2D9A40, Data2: 0x909F, Data3: 0x4BBA,
		Data4: [8]byte{0xBF, 0x8B, 0x0C, 0x0D, 0x29, 0x6F, 0x14, 0xF0},
	}

	// ── Power / System ──
	IID_IPowerManagerStatics = &windows.GUID{
		Data1: 0x1394825D, Data2: 0x62CE, Data3: 0x4364,
		Data4: [8]byte{0x98, 0xD5, 0xAA, 0x28, 0xC7, 0xFB, 0xD1, 0x5B},
	}
	IID_ILauncherStatics = &windows.GUID{
		Data1: 0x277151C3, Data2: 0x9E3E, Data3: 0x42F6,
		Data4: [8]byte{0x91, 0xA4, 0x5D, 0xFD, 0xEB, 0x23, 0x24, 0x51},
	}
	IID_IUserProfilePersonalizationSettings = &windows.GUID{
		Data1: 0x8CEDDAB4, Data2: 0x7998, Data3: 0x46D5,
		Data4: [8]byte{0x8D, 0xD3, 0x18, 0x4F, 0x1C, 0x5F, 0x9A, 0xB9},
	}
	IID_IUserProfilePersonalizationSettingsStatics = &windows.GUID{
		Data1: 0x91ACB841, Data2: 0x5037, Data3: 0x454B,
		Data4: [8]byte{0x98, 0x83, 0xBB, 0x77, 0x2D, 0x08, 0xDD, 0x16},
	}
	IID_IUserInformationStatics = &windows.GUID{
		Data1: 0x77F3A910, Data2: 0x48FA, Data3: 0x489C,
		Data4: [8]byte{0x93, 0x4E, 0x2A, 0xE8, 0x5B, 0xA8, 0xF7, 0x72},
	}
	IID_IProcessDiagnosticInfo = &windows.GUID{
		Data1: 0xE830B04B, Data2: 0x300E, Data3: 0x4EE6,
		Data4: [8]byte{0xA0, 0xAB, 0x5B, 0x5F, 0x52, 0x31, 0xB4, 0x34},
	}
	IID_IProcessDiagnosticInfoStatics = &windows.GUID{
		Data1: 0x2F41B260, Data2: 0xB49F, Data3: 0x428C,
		Data4: [8]byte{0xAA, 0x0E, 0x84, 0x74, 0x4F, 0x49, 0xCA, 0x95},
	}

	// ── Display ──
	IID_IDisplayInformation = &windows.GUID{
		Data1: 0xBED112AE, Data2: 0xADC3, Data3: 0x4DC9,
		Data4: [8]byte{0xAE, 0x65, 0x85, 0x1F, 0x4D, 0x7D, 0x47, 0x99},
	}
	IID_IDisplayInformationStatics = &windows.GUID{
		Data1: 0xC6A02A6C, Data2: 0xD452, Data3: 0x44DC,
		Data4: [8]byte{0xBA, 0x07, 0x96, 0xF3, 0xC6, 0xAD, 0xF9, 0xD1},
	}

	// ── Notifications ──
	IID_IToastNotification = &windows.GUID{
		Data1: 0x997E2675, Data2: 0x059E, Data3: 0x4E60,
		Data4: [8]byte{0x8B, 0x06, 0x17, 0x60, 0x91, 0x7C, 0x8B, 0x80},
	}
	IID_IToastNotificationManagerStatics = &windows.GUID{
		Data1: 0x50AC103F, Data2: 0xD235, Data3: 0x4598,
		Data4: [8]byte{0xBB, 0xEF, 0x98, 0xFE, 0x4D, 0x1A, 0x3A, 0xD4},
	}

	// ── Clipboard ──
	IID_IClipboardStatics = &windows.GUID{
		Data1: 0xC627E291, Data2: 0x34E2, Data3: 0x4963,
		Data4: [8]byte{0x8E, 0xED, 0x93, 0xCB, 0xB0, 0xEA, 0x3D, 0x70},
	}
	IID_IDataPackage = &windows.GUID{
		Data1: 0x61EBF5C7, Data2: 0xEFEA, Data3: 0x4346,
		Data4: [8]byte{0x95, 0x54, 0x98, 0x1D, 0x7E, 0x19, 0x8F, 0xFE},
	}

	// ── Media Control ──
	IID_IGlobalSystemMediaTransportControlsSessionManager = &windows.GUID{
		Data1: 0xCACE8EAC, Data2: 0xE86E, Data3: 0x504A,
		Data4: [8]byte{0xAB, 0x31, 0x5F, 0xF8, 0xFF, 0x1B, 0xCE, 0x49},
	}
	IID_IGlobalSystemMediaTransportControlsSessionManagerStatics = &windows.GUID{
		Data1: 0x2050C4EE, Data2: 0x11A0, Data3: 0x57DE,
		Data4: [8]byte{0xAE, 0xD7, 0xC9, 0x7C, 0x70, 0x33, 0x82, 0x45},
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

// ── WinRT COM infrastructure vtable indices ──
// Verified 2026-06-30 — Win11 26200 (24H2), SDK 10.0.26200.0
// IUnknown:
//   0 = QueryInterface (= qei)
//   1 = AddRef
//   2 = Release (= comRelease)
// IAsyncInfo:
//   7 = get_Status
//   8 = get_ErrorCode
// IAsyncOperation<T>:
//   8 = GetResults (= getAsyncObj)

func qei(obj unsafe.Pointer, iid *windows.GUID) (unsafe.Pointer, error) {
	if obj == nil {
		return nil, fmt.Errorf("QI on nil")
	}
	var result unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(obj, 0), uintptr(obj), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&result))) // 0 = QueryInterface
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
		r, _, _ := syscall.SyscallN(vtblMethod(info, 7), uintptr(info), uintptr(unsafe.Pointer(&status))) // 7 = get_Status
		if r != 0 {
			return fmt.Errorf("get_Status 0x%X", r)
		}
		switch status {
		case AsyncStatusCompleted:
			return nil
		case AsyncStatusError:
			var errCode uint32
			syscall.SyscallN(vtblMethod(info, 8), uintptr(info), uintptr(unsafe.Pointer(&errCode))) // 8 = get_ErrorCode
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
	r, _, _ := syscall.SyscallN(vtblMethod(op, 8), uintptr(op), uintptr(unsafe.Pointer(&result))) // 8 = GetResults
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
