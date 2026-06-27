//go:build windows

package actions

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ── COM GUIDs ──
var (
	CLSID_CUIAutomation = &windows.GUID{
		Data1: 0xff48dba4, Data2: 0x60ef, Data3: 0x4201,
		Data4: [8]byte{0xaa, 0x87, 0x54, 0x10, 0x3e, 0xef, 0x59, 0x4e},
	}
	IID_IUIAutomation = &windows.GUID{
		Data1: 0x30cbe57d, Data2: 0xd9d0, Data3: 0x452a,
		Data4: [8]byte{0xab, 0x13, 0x7a, 0xc5, 0xac, 0x48, 0x25, 0xee},
	}
	IID_IUIAutomationElement = &windows.GUID{
		Data1: 0xd22108aa, Data2: 0x8ac5, Data3: 0x49a5,
		Data4: [8]byte{0x83, 0x7b, 0x37, 0xbb, 0xb3, 0xd7, 0x59, 0x1e},
	}
)

// ── Constants ──
const (
	COINIT_MULTITHREADED = 0
	CLSCTX_INPROC_SERVER = 1
	S_OK                 = 0

	UIA_NamePropertyId            = 30005
	UIA_ControlTypePropertyId     = 30003
	UIA_ProcessIdPropertyId       = 30020
	UIA_IsEnabledPropertyId       = 30062
	UIA_BoundingRectanglePropertyId = 30031
	UIA_AutomationIdPropertyId    = 30011

	UIA_ButtonControlType   = 50000
	UIA_EditControlType     = 50004
	UIA_WindowControlType   = 50032
	UIA_PaneControlType     = 50033

	InvokePatternId = 10000
	ValuePatternId  = 10002

	TreeScope_Descendants = 4
	TreeScope_Children    = 2

	VT_I4    = 3
	VT_R8    = 5
	VT_BSTR  = 8
	VT_BOOL  = 11
	VT_ARRAY = 0x2000
)

var (
	modole32    = windows.NewLazySystemDLL("ole32.dll")
	modoleaut32 = windows.NewLazySystemDLL("oleaut32.dll")

	procCoCreateInstance     = modole32.NewProc("CoCreateInstance")
	procCoInitializeEx       = modole32.NewProc("CoInitializeEx")
	procSysFreeString        = modoleaut32.NewProc("SysFreeString")
	procSysAllocString       = modoleaut32.NewProc("SysAllocString")
	procSafeArrayGetElement  = modoleaut32.NewProc("SafeArrayGetElement")
)

// ── Variant ──
type VARIANT struct {
	VT         uint16
	wReserved1 uint16
	wReserved2 uint16
	wReserved3 uint16
	data       [8]byte
}

// ── Bounding rectangle ──
type UIA_RECT struct {
	Left, Top, Right, Bottom float64
}

// ── COM utilities ──
func vtblMethod(iface unsafe.Pointer, idx int) uintptr {
	return *(*uintptr)(unsafe.Add(*(*unsafe.Pointer)(iface), uintptr(idx)*8))
}

func comRelease(p unsafe.Pointer) {
	if p != nil {
		syscall.SyscallN(vtblMethod(p, 2), uintptr(p))
	}
}

func bstrAlloc(s string) uintptr {
	if s == "" {
		return 0
	}
	u, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return 0
	}
	r, _, _ := procSysAllocString.Call(uintptr(unsafe.Pointer(u)))
	return r
}

func bstrFree(p uintptr) {
	if p != 0 {
		procSysFreeString.Call(p)
	}
}

func bstrToGo(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return windows.UTF16PtrToString((*uint16)(p))
}

func varString(s string) *VARIANT {
	b := bstrAlloc(s)
	return &VARIANT{VT: VT_BSTR, data: *(*[8]byte)(unsafe.Pointer(&b))}
}

func varInt(v int32) *VARIANT {
	return &VARIANT{VT: VT_I4, data: *(*[8]byte)(unsafe.Pointer(&v))}
}

func varFree(v *VARIANT) {
	if v != nil && v.VT == VT_BSTR {
		bstrFree(*(*uintptr)(unsafe.Pointer(&v.data)))
	}
}

// ── IUIAutomation vtable indices (after IUnknown: QueryInterface=0,AddRef=1,Release=2) ──
// Verified:
//   5 = GetRootElement ✓
//  21 = CreateTrueCondition ✓
//  23 = CreatePropertyCondition ✓
//   6 = ElementFromHandle ✓

type uiaAuto struct {
	p unsafe.Pointer
}

func newUIA() (*uiaAuto, error) {
	var p unsafe.Pointer
	r, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(CLSID_CUIAutomation)),
		0, CLSCTX_INPROC_SERVER,
		uintptr(unsafe.Pointer(IID_IUIAutomation)),
		uintptr(unsafe.Pointer(&p)),
	)
	if r != S_OK {
		return nil, fmt.Errorf("CoCreateInstance IUIAutomation: 0x%X", r)
	}
	return &uiaAuto{p: p}, nil
}

func (a *uiaAuto) release() { comRelease(a.p) }

func (a *uiaAuto) getRootElement() (*uiaElement, error) {
	var e unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(a.p, 5), uintptr(a.p), uintptr(unsafe.Pointer(&e)))
	if r != S_OK {
		return nil, fmt.Errorf("GetRootElement: 0x%X", r)
	}
	return &uiaElement{p: e}, nil
}

func (a *uiaAuto) elementFromHandle(hwnd uintptr) (*uiaElement, error) {
	var e unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(a.p, 6), uintptr(a.p), hwnd, uintptr(unsafe.Pointer(&e)))
	if r != S_OK {
		return nil, fmt.Errorf("ElementFromHandle: 0x%X", r)
	}
	return &uiaElement{p: e}, nil
}

func (a *uiaAuto) createPropertyCondition(id int32, v *VARIANT) (*uiaCondition, error) {
	var c unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(a.p, 23), uintptr(a.p), uintptr(id),
		uintptr(unsafe.Pointer(v)), uintptr(unsafe.Pointer(&c)))
	if r != S_OK {
		return nil, fmt.Errorf("CreatePropertyCondition(%d): 0x%X", id, r)
	}
	return &uiaCondition{p: c}, nil
}

func (a *uiaAuto) createTrueCondition() (*uiaCondition, error) {
	var c unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(a.p, 21), uintptr(a.p), uintptr(unsafe.Pointer(&c)))
	if r != S_OK {
		return nil, fmt.Errorf("CreateTrueCondition: 0x%X", r)
	}
	return &uiaCondition{p: c}, nil
}

// ── IUIAutomationCondition ──
type uiaCondition struct {
	p unsafe.Pointer
}

func (c *uiaCondition) release() { comRelease(c.p) }

// ── IUIAutomationElement vtable indices (after IUnknown) ──
// Verified:
//   5 = FindFirst ✓
//   6 = FindAll ✓
//  10 = GetCurrentPropertyValue ✓

type uiaElement struct {
	p unsafe.Pointer
}

func (e *uiaElement) release() { comRelease(e.p) }

func (e *uiaElement) findFirst(scope int, cond uintptr) (*uiaElement, error) {
	var found unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(e.p, 5), uintptr(e.p), uintptr(scope), cond,
		uintptr(unsafe.Pointer(&found)))
	if r != S_OK {
		return nil, fmt.Errorf("FindFirst: 0x%X", r)
	}
	if found == nil {
		return nil, nil
	}
	return &uiaElement{p: found}, nil
}

func (e *uiaElement) findAll(scope int, cond uintptr) (*uiaElementArray, error) {
	var arr unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(e.p, 6), uintptr(e.p), uintptr(scope), cond,
		uintptr(unsafe.Pointer(&arr)))
	if r != S_OK {
		return nil, fmt.Errorf("FindAll: 0x%X", r)
	}
	return &uiaElementArray{p: arr}, nil
}

func (e *uiaElement) getPropValue(propId int32) (*VARIANT, error) {
	var v VARIANT
	r, _, _ := syscall.SyscallN(vtblMethod(e.p, 10), uintptr(e.p), uintptr(propId),
		uintptr(unsafe.Pointer(&v)))
	if r != S_OK {
		return nil, fmt.Errorf("GetCurrentPropertyValue(%d): 0x%X", propId, r)
	}
	return &v, nil
}

func (e *uiaElement) getName() string {
	v, err := e.getPropValue(UIA_NamePropertyId)
	if err != nil {
		return ""
	}
	if v.VT != VT_BSTR {
		return ""
	}
	return bstrToGo(*(*unsafe.Pointer)(unsafe.Pointer(&v.data)))
}

func (e *uiaElement) getPid() int {
	v, err := e.getPropValue(UIA_ProcessIdPropertyId)
	if err != nil || v.VT != VT_I4 {
		return 0
	}
	return int(*(*int32)(unsafe.Pointer(&v.data)))
}

func (e *uiaElement) isEnabled() bool {
	v, err := e.getPropValue(UIA_IsEnabledPropertyId)
	if err != nil || v.VT != VT_BOOL {
		return false
	}
	return *(*int16)(unsafe.Pointer(&v.data)) != 0
}

func safeArrayGetDouble(psa uintptr, idx int32) (float64, error) {
	var val float64
	r, _, _ := procSafeArrayGetElement.Call(psa, uintptr(unsafe.Pointer(&idx)), uintptr(unsafe.Pointer(&val)))
	if r != S_OK {
		return 0, fmt.Errorf("SafeArrayGetElement(%d): 0x%X", idx, r)
	}
	return val, nil
}

type uiaNativeRect struct {
	Left, Top, Right, Bottom uint32
}

func (e *uiaElement) getBoundingRect() (UIA_RECT, error) {
	// Try direct property getter first (vtbl index 43 = get_CurrentBoundingRectangle)
	var nr uiaNativeRect
	hr, _, _ := syscall.SyscallN(vtblMethod(e.p, 43), uintptr(e.p), uintptr(unsafe.Pointer(&nr)))
	if hr == S_OK {
		return UIA_RECT{
			Left:   float64(nr.Left),
			Top:    float64(nr.Top),
			Right:  float64(nr.Right),
			Bottom: float64(nr.Bottom),
		}, nil
	}

	// Fallback: GetCurrentPropertyValue (vtbl index 10) with safe array
	v, err := e.getPropValue(UIA_BoundingRectanglePropertyId)
	if err != nil {
		return UIA_RECT{}, err
	}
	vtActual := v.VT
	if vtActual != (VT_R8 | VT_ARRAY) {
		return UIA_RECT{}, fmt.Errorf("unexpected vt for BoundingRect: %d (expected %d)", vtActual, VT_R8|VT_ARRAY)
	}
	psa := *(*uintptr)(unsafe.Pointer(&v.data))
	if psa == 0 {
		return UIA_RECT{}, fmt.Errorf("null safe array pointer")
	}

	var r UIA_RECT
	if r.Left, err = safeArrayGetDouble(psa, 0); err != nil {
		return UIA_RECT{}, err
	}
	if r.Top, err = safeArrayGetDouble(psa, 1); err != nil {
		return UIA_RECT{}, err
	}
	if r.Right, err = safeArrayGetDouble(psa, 2); err != nil {
		return UIA_RECT{}, err
	}
	if r.Bottom, err = safeArrayGetDouble(psa, 3); err != nil {
		return UIA_RECT{}, err
	}
	return r, nil
}

func controlTypeName(id int) string {
	switch id {
	case 50000:
		return "Button"
	case 50001:
		return "Calendar"
	case 50002:
		return "CheckBox"
	case 50003:
		return "ComboBox"
	case 50004:
		return "Edit"
	case 50005:
		return "Hyperlink"
	case 50006:
		return "Image"
	case 50007:
		return "ListItem"
	case 50008:
		return "List"
	case 50009:
		return "Menu"
	case 50010:
		return "MenuBar"
	case 50011:
		return "MenuItem"
	case 50012:
		return "ProgressBar"
	case 50013:
		return "RadioButton"
	case 50014:
		return "ScrollBar"
	case 50015:
		return "Slider"
	case 50016:
		return "Spinner"
	case 50017:
		return "StatusBar"
	case 50018:
		return "Tab"
	case 50019:
		return "TabItem"
	case 50020:
		return "Text"
	case 50021:
		return "ToolBar"
	case 50022:
		return "ToolTip"
	case 50023:
		return "Tree"
	case 50024:
		return "TreeItem"
	case 50025:
		return "Custom"
	case 50026:
		return "Group"
	case 50027:
		return "Thumb"
	case 50028:
		return "DataGrid"
	case 50029:
		return "DataItem"
	case 50030:
		return "Document"
	case 50031:
		return "SplitButton"
	case 50032:
		return "Window"
	case 50033:
		return "Pane"
	case 50034:
		return "Header"
	case 50035:
		return "HeaderItem"
	case 50036:
		return "Table"
	case 50037:
		return "TitleBar"
	case 50038:
		return "Separator"
	case 50039:
		return "SemanticZoom"
	case 50040:
		return "AppBar"
	default:
		return fmt.Sprintf("ControlType_%d", id)
	}
}

func (e *uiaElement) getAutomationId() string {
	v, err := e.getPropValue(UIA_AutomationIdPropertyId)
	if err != nil || v.VT != VT_BSTR {
		return ""
	}
	return bstrToGo(*(*unsafe.Pointer)(unsafe.Pointer(&v.data)))
}

func (e *uiaElement) getControlTypeId() int {
	v, err := e.getPropValue(UIA_ControlTypePropertyId)
	if err != nil || v.VT != VT_I4 {
		return 0
	}
	return int(*(*int32)(unsafe.Pointer(&v.data)))
}

func (e *uiaElement) toElement() UIAElement {
	el := UIAElement{
		Name:         e.getName(),
		AutomationID: e.getAutomationId(),
		ControlType:  controlTypeName(e.getControlTypeId()),
		ProcessID:    e.getPid(),
		IsEnabled:    e.isEnabled(),
	}
	if r, err := e.getBoundingRect(); err == nil {
		el.X = r.Left
		el.Y = r.Top
		el.Width = r.Right - r.Left
		el.Height = r.Bottom - r.Top
	}
	return el
}

// ── IUIAutomationElement patterns (index 16 = GetCurrentPattern) ──

func (e *uiaElement) getCurrentPattern(patternId int32) (unsafe.Pointer, error) {
	var p unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(e.p, 16), uintptr(e.p), uintptr(patternId),
		uintptr(unsafe.Pointer(&p)))
	if r != S_OK {
		return nil, fmt.Errorf("GetCurrentPattern(%d): 0x%X", patternId, r)
	}
	if p == nil {
		return nil, fmt.Errorf("GetCurrentPattern(%d): nil pattern (not supported)", patternId)
	}
	return p, nil
}

// ── IUIAutomationValuePattern (get_Value=3, SetValue=4) ──

func (e *uiaElement) getValue() (string, error) {
	unk, err := e.getCurrentPattern(ValuePatternId)
	if err != nil {
		return "", fmt.Errorf("getValuePattern: %w", err)
	}
	defer comRelease(unk)

	var bstr unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(unk, 3), uintptr(unk), uintptr(unsafe.Pointer(&bstr)))
	if r != S_OK {
		return "", fmt.Errorf("get_Value: 0x%X", r)
	}
	defer bstrFree(uintptr(bstr))
	return bstrToGo(bstr), nil
}

func (e *uiaElement) setValue(val string) error {
	unk, err := e.getCurrentPattern(ValuePatternId)
	if err != nil {
		return fmt.Errorf("getValuePattern: %w", err)
	}
	defer comRelease(unk)

	b := bstrAlloc(val)
	if b == 0 {
		return fmt.Errorf("bstrAlloc failed")
	}
	defer bstrFree(b)

	r, _, _ := syscall.SyscallN(vtblMethod(unk, 4), uintptr(unk), b)
	if r != S_OK {
		return fmt.Errorf("SetValue: 0x%X", r)
	}
	return nil
}

// ── IUIAutomationInvokePattern (Invoke=3) ──

func (e *uiaElement) invoke() error {
	unk, err := e.getCurrentPattern(InvokePatternId)
	if err != nil {
		return fmt.Errorf("getInvokePattern: %w", err)
	}
	defer comRelease(unk)

	r, _, _ := syscall.SyscallN(vtblMethod(unk, 3), uintptr(unk))
	if r != S_OK {
		return fmt.Errorf("Invoke: 0x%X", r)
	}
	return nil
}

// ── IUIAutomationElementArray vtable ──
//   3 = Length, 4 = GetElement

type uiaElementArray struct {
	p unsafe.Pointer
}

func (a *uiaElementArray) release() { comRelease(a.p) }

func (a *uiaElementArray) length() int {
	var l int32
	r, _, _ := syscall.SyscallN(vtblMethod(a.p, 3), uintptr(a.p), uintptr(unsafe.Pointer(&l)))
	if r != S_OK {
		return 0
	}
	return int(l)
}

func (a *uiaElementArray) get(idx int) (*uiaElement, error) {
	var e unsafe.Pointer
	r, _, _ := syscall.SyscallN(vtblMethod(a.p, 4), uintptr(a.p), uintptr(idx),
		uintptr(unsafe.Pointer(&e)))
	if r != S_OK {
		return nil, fmt.Errorf("arrGet(%d): 0x%X", idx, r)
	}
	return &uiaElement{p: e}, nil
}
