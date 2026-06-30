# COM & WinRT Patterns

All COM/WinRT calls use raw vtable dispatch via `syscall.SyscallN` — no CGO, no go-ole, no external COM libraries. Two native COM subsystems are used:

| Technology | DLL | Files | Purpose |
|-----------|-----|-------|---------|
| **WinRT COM** | `combase.dll` | `winrt.go`, `ocr_com.go`, `ocr.go` | OCR, OcrLanguages enumeration |
| **UIA COM** | `ole32.dll`, `oleaut32.dll` | `uia_com.go`, `uia.go` | UI Automation tree traversal |

Audio devices and brightness use **PowerShell scripts** (not Go COM calls), documented in [§11](#11-powershell-only-features-audio--brightness). OCR falls back to PowerShell if native COM fails, documented in [§9](#9-fallback-chain).

---

## 1. Threading Model

Two separate apartment models, both **MTA** (Multithreaded Apartment):

| Initializer | Call | Used by | Pattern |
|------------|------|---------|---------|
| `CoInitializeEx` | `ole32.dll` | UIA (`uia.go`) | MTA only |
| `RoInitialize` | `combase.dll` | WinRT OCR, OcrLanguages (`ocr_com.go`) | MTA only |

Both use `sync.Once` for per-process, single-threaded initialization:

```go
// uia.go — COM
var comOnce sync.Once
func ensureCOM() {
    comOnce.Do(func() {
        hr, _, _ := procCoInitializeEx.Call(0, COINIT_MULTITHREADED)
        // S_OK=0, S_FALSE=1 (already initialized)
        if hr != S_OK && hr != 1 {
            comInitErr = fmt.Errorf("CoInitializeEx: 0x%X", hr)
        }
    })
}

// ocr_com.go — WinRT
var roOnce sync.Once
func ensureRo() error {
    roOnce.Do(func() {
        err := roInitialize(RO_INIT_MULTITHREADED)
        // S_OK=0, S_FALSE=1, 0x80010106=RPC_E_CHANGED_MODE
    })
    return roErr
}
```

**Constraint**: Both `RoInitialize` and `CoInitializeEx` must agree on MTA. Previously this was a bug — `RO_INIT_SINGLETHREADED` vs `COINIT_MULTITHREADED` caused `RPC_E_CHANGED_MODE` (0x80010106) when OCR ran after UIA. Fixed by aligning both to MTA (CHANGELOG v0.2.5).

---

## 2. COM vtable Dispatch

All COM calls use a single helper to resolve method pointers from an interface vtable:

```go
func vtblMethod(iface unsafe.Pointer, idx int) uintptr {
    return *(*uintptr)(unsafe.Add(*(*unsafe.Pointer)(iface), uintptr(idx)*8))
}
```

**Layout**: Every COM interface starts with a vtable pointer `→ []uintptr`.  
**IUnknown** occupies indices 0-2: `QueryInterface=0`, `AddRef=1`, `Release=2`.  
**Application methods** start at index 3+.

```
IUnknown vtable:
  [0] QueryInterface(this, riid, ppv)
  [1] AddRef(this)
  [2] Release(this)

IUIAutomation vtable:
  [0-2] IUnknown
  [3]   ...
  [5]   GetRootElement(this, ppv)         ← verified
  [6]   ElementFromHandle(this, hwnd, ppv)← verified
  [21]  CreateTrueCondition(this, ppv)     ← verified
  [23]  CreatePropertyCondition(this, id, var, ppv) ← verified
  [25]  CreateAndCondition(this, c1, c2, ppv) ← used for compound conditions
```

### Verified vtable indices (uia_com.go)

Source file: `internal/actions/uia_com.go` — verified 2026-06-30, Win11 26200 (24H2), SDK 10.0.26200.0

| Interface | Index | Method | IUnknown base |
|-----------|-------|--------|---------------|
| `IUIAutomation` | 5 | `GetRootElement` | +3 |
| `IUIAutomation` | 6 | `ElementFromHandle` | +3 |
| `IUIAutomation` | 21 | `CreateTrueCondition` | +3 |
| `IUIAutomation` | 23 | `CreatePropertyCondition` | +3 |
| `IUIAutomation` | 25 | `CreateAndCondition` | +3 |
| `IUIAutomationElement` | 5 | `FindFirst` | +3 |
| `IUIAutomationElement` | 6 | `FindAll` | +3 |
| `IUIAutomationElement` | 10 | `GetCurrentPropertyValue` | +3 |
| `IUIAutomationElement` | 16 | `GetCurrentPattern` | +3 |
| `IUIAutomationElement` | 43 | `get_CurrentBoundingRectangle` | +3 |
| `IUIAutomationElementArray` | 3 | `Length` | +1 |
| `IUIAutomationElementArray` | 4 | `GetElement` | +1 |
| `IUIAutomationValuePattern` | 3 | `get_Value` | +1 |
| `IUIAutomationValuePattern` | 4 | `SetValue` | +1 |
| `IUIAutomationInvokePattern` | 3 | `Invoke` | +1 |

### Verified WinRT vtable indices (ocr_com.go, ocr.go, winrt.go)

Source files: `internal/actions/ocr_com.go`, `internal/actions/ocr.go`, `internal/actions/winrt.go` — verified 2026-06-30, Win11 26200 (24H2), SDK 10.0.26200.0

| Interface | Index | Method | IInspectable base |
|-----------|-------|--------|-------------------|
| `ILanguageFactory` | 6 | `CreateLanguage` | +1 |
| `IStorageFileStatics` | 6 | `GetFileFromPathAsync` | +1 |
| `IStorageFile` | 8 | `OpenAsync` | +3 |
| `IBitmapDecoderStatics` | 14 | `CreateAsync` | +9 |
| `IBitmapFrame` | 6 | `GetSoftwareBitmapAsync` | +1 |
| `IOcrEngineStatics` | 7 | `get_AvailableRecognizerLanguages` | +2 |
| `IOcrEngineStatics` | 9 | `TryCreateFromLanguage` | +4 |
| `IOcrEngineStatics` | 10 | `TryCreateFromUserProfileLanguages` | +5 |
| `IOcrEngine` | 6 | `RecognizeAsync` | +1 |
| `IVectorView` (generic) | 6 | `GetAt` | +1 |
| `IVectorView` (generic) | 7 | `get_Size` | +2 |
| `IAsyncInfo` | 7 | `get_Status` | +2 |
| `IAsyncInfo` | 8 | `get_ErrorCode` | +3 |
| `IAsyncOperation<T>` | 8 | `GetResults` | +3 |
| `IUnknown` | 0 | `QueryInterface` | base |
| `IUnknown` | 1 | `AddRef` |
| `IUnknown` | 2 | `Release` |

---

## 3. IUnknown Lifecycle

COM interface pointers are stored as `unsafe.Pointer` (never as `uintptr`, which would be invisible to the GC and could become dangling pointers).

### Release

```go
func comRelease(p unsafe.Pointer) {
    if p != nil {
        syscall.SyscallN(vtblMethod(p, 2), uintptr(p)) // p->Release()
    }
}
```

All COM objects use `defer comRelease(obj)` immediately after acquisition. Nested lifetimes use `defer` in acquisition order (LIFO).

### QueryInterface

```go
func qei(obj unsafe.Pointer, iid *windows.GUID) (unsafe.Pointer, error) {
    var result unsafe.Pointer
    r, _, _ := syscall.SyscallN(vtblMethod(obj, 0), uintptr(obj),
        uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&result)))
    if r != 0 {
        return nil, fmt.Errorf("QI 0x%X", r)
    }
    return result, nil
}
```

Used when an `IAsyncOperation<T>` result needs to be cast to a specific interface, e.g. `IID_IStorageFile` or `IID_IBitmapFrameWithSoftwareBitmap`.

### Ownership rules

1. **Factory/source objects** — released via `defer comRelease()` immediately after use (e.g., `OcrEngineStatics`, `StorageFileStatics`)
2. **Async operations** — released immediately after the async result is extracted
3. **Async results** — QI'd into target interface, then the QI result is released after use
4. **Returned objects** — callers own the reference and must release themselves

---

## 4. HSTRING Management

WinRT strings are `HSTRING` handles (`uintptr` alias). Managed in `winrt.go`.

### Create / Delete

```go
func windowsCreateString(s string) (HSTRING, error) {
    u, _ := syscall.UTF16FromString(s)
    r, _, _ := procWindowsCreateString.Call(
        uintptr(unsafe.Pointer(&u[0])),
        uintptr(len(u)-1),  // exclude null terminator
        uintptr(unsafe.Pointer(&h)),
    )
}

func windowsDeleteString(h HSTRING) error {
    if h == 0 { return nil }
    r, _, _ := procWindowsDeleteString.Call(uintptr(h))
}
```

### Extract

```go
func hstringToString(h HSTRING) (string, error) {
    var length uint32
    bufRaw, _, _ := procWindowsGetStringRawBuffer.Call(uintptr(h), uintptr(unsafe.Pointer(&length)))
    return syscall.UTF16ToString(unsafe.Slice((*uint16)(*(*unsafe.Pointer)(unsafe.Pointer(&bufRaw))), length)), nil
}
```

**Safety**: `WindowsGetStringRawBuffer` returns a pointer into the HSTRING's internal buffer, which stays alive as long as we hold the HSTRING handle. We `defer windowsDeleteString(h)` to release it after extraction.

### Lifecycle pattern

```go
hStr, err := newHString("Windows.Media.Ocr.OcrEngine")
if err != nil { return err }
defer freeHString(hStr)
```

HSTRINGs are always released before the function returns.

---

## 5. BSTR / VARIANT (UIA COM)

Used in `uia_com.go` for UIA property conditions.

### BSTR

```go
func bstrAlloc(s string) uintptr {
    u, _ := syscall.UTF16PtrFromString(s)
    r, _, _ := procSysAllocString.Call(uintptr(unsafe.Pointer(u)))
    return r
}

func bstrFree(p uintptr) {
    if p != 0 { procSysFreeString.Call(p) }
}
```

### VARIANT

```go
type VARIANT struct {
    VT         uint16
    wReserved1 uint16
    wReserved2 uint16
    wReserved3 uint16
    data       [8]byte
}

func varString(s string) *VARIANT {
    b := bstrAlloc(s)
    return &VARIANT{VT: VT_BSTR, data: *(*[8]byte)(unsafe.Pointer(&b))}
}
```

**Cleanup**: BSTR-backed variants require explicit `varFree()`:

```go
func varFree(v *VARIANT) {
    if v != nil && v.VT == VT_BSTR {
        bstrFree(*(*uintptr)(unsafe.Pointer(&v.data)))
    }
}
```

---

## 6. IAsyncOperation Polling (WinRT Async)

WinRT async calls return `IAsyncOperation<T>` — no callbacks, no Go channels. The pattern is **synchronous polling** via `IAsyncInfo::Status`:

```go
func waitForAsync(op unsafe.Pointer, timeout time.Duration) error {
    info, _ := qei(op, IID_IAsyncInfo)
    defer comRelease(info)

    deadline := time.Now().Add(timeout)
    for {
        var status int32
        syscall.SyscallN(vtblMethod(info, 7), uintptr(info), uintptr(unsafe.Pointer(&status)))
        switch status {
        case AsyncStatusCompleted: return nil
        case AsyncStatusError:
            var errCode uint32
            syscall.SyscallN(vtblMethod(info, 8), uintptr(info), uintptr(unsafe.Pointer(&errCode)))
            return fmt.Errorf("async error 0x%X", errCode)
        case AsyncStatusCanceled: return fmt.Errorf("async cancelled")
        }
        if time.Now().After(deadline) { return fmt.Errorf("async timeout") }
        time.Sleep(5 * time.Millisecond)
    }
}

func getAsyncObj(op unsafe.Pointer, timeout time.Duration) (unsafe.Pointer, error) {
    if err := waitForAsync(op, timeout); err != nil { return nil, err }
    var result unsafe.Pointer
    syscall.SyscallN(vtblMethod(op, 8), uintptr(op), uintptr(unsafe.Pointer(&result)))
    return result, nil
}
```

**Timeout**: All OCR async operations use 60-second timeout. Poll interval: 5ms.

### Async flow (OCR example)

```
StorageFile.GetFileFromPathAsync(path)
  → waitForAsync → getAsyncObj → QI(IID_IStorageFile)
  → OpenAsync(Read)
  → waitForAsync → getAsyncObj → stream
  → BitmapDecoder.CreateAsync(stream)
  → waitForAsync → getAsyncObj → QI(IID_IBitmapFrameWithSoftwareBitmap)
  → GetSoftwareBitmapAsync()
  → waitForAsync → getAsyncObj → software bitmap
  → OcrEngine.RecognizeAsync(bitmap)
  → waitForAsync → getAsyncObj → OcrResult
```

---

## 7. WinRT Activation Factory Pattern

All WinRT objects are created via `RoGetActivationFactory`:

```go
func roGetActivationFactory(classID HSTRING, iid *windows.GUID) (unsafe.Pointer, error) {
    var factory unsafe.Pointer
    r, _, _ := procRoGetActivationFactory.Call(
        uintptr(classID),
        uintptr(unsafe.Pointer(iid)),
        uintptr(unsafe.Pointer(&factory)),
    )
    if r != 0 { return nil, fmt.Errorf("RoGetActivationFactory 0x%X", r) }
    return factory, nil
}
```

### Activation table

| WinRT Class | IID Variable | Used By |
|-------------|-------------|---------|
| `Windows.Globalization.Language` | `IID_ILanguageFactory` | `ocr_com.go:createLanguage` |
| `Windows.Storage.StorageFile` | `IID_IStorageFileStatics` | `ocr_com.go:openStorageFile` |
| `Windows.Storage.Streams.RandomAccessStream` | `IID_IRandomAccessStreamStatics` | (reserved) |
| `Windows.Graphics.Imaging.BitmapDecoder` | `IID_IBitmapDecoderStatics` | `ocr_com.go:createDecoder` |
| `Windows.Media.Ocr.OcrEngine` | `IID_IOcrEngineStatics` | `ocr_com.go:createOcrEngine`, `ocr.go:OcrLanguages` |

All IIDs above were discovered and verified on **Win11 26200** using [`scripts\discover-winrt-iids.ps1`](../../scripts/discover-winrt-iids.ps1). To re-verify on a new Windows build: `powershell -File scripts\discover-winrt-iids.ps1 -UpdateDocs` and cross-reference the output against the table below. If any IID differs, update `winrt.go` — do not trust the old value.

### IID table (defined in winrt.go)

<!-- IID_TABLE_START -->
| IID | Value | Purpose | Status |
|-----|-------|---------|--------|
| `IID_IInspectable` | `{AF86E2E0-B12D-4c6a-9C5A-D7AA65101E90}` | Base WinRT interface | unused |
| `IID_IActivationFactory` | `{00000035-0000-0000-C000-000000000046}` | Activation factory | unused |
| `IID_IAsyncInfo` | `{00000036-0000-0000-C000-000000000046}` | Async operation status | internal |
| `IID_IStorageFileStatics` | `5984C710-DAF2-43C8-8BB4-A4D3EACFD03F` | StorageFile factory | used |
| `IID_IStorageFileStatics2` | `5C76A781-212E-4AF9-8F04-740CAE108974` | StorageFile factory v2 | unused |
| `IID_IStorageFile` | `FA3F6186-4214-428C-A64C-14C9AC7315EA` | StorageFile instance | used |
| `IID_IBitmapDecoderStatics` | `438CCB26-BCEF-4E95-BAD6-23A822E58D01` | BitmapDecoder factory | used |
| `IID_IBitmapDecoderStatics2` | `50BA68EA-99A1-40C4-80D9-AEF0DAFA6C3F` | BitmapDecoder factory v2 | unused |
| `IID_IBitmapDecoder` | `ACEF22BA-1D74-4C91-9DFC-9620745233E6` | BitmapDecoder instance | used |
| `IID_IBitmapFrame` | `72A49A1C-8081-438D-91BC-94ECFC8185C6` | Bitmap frame | used |
| `IID_IBitmapFrameWithSoftwareBitmap` | `FE287C9A-420C-4963-87AD-691436E08383` | SoftwareBitmap extraction | used |
| `IID_IBitmapEncoder` | `2BC468E3-E1F8-4B54-95E8-32919551CE62` | BitmapEncoder instance | internal |
| `IID_IBitmapEncoderStatics` | `A74356A7-A4E4-4EB9-8E40-564DE7E1CCB2` | BitmapEncoder factory | unused |
| `IID_ISoftwareBitmap` | `689E0708-7EEF-483F-963F-DA938818E073` | SoftwareBitmap instance | unused |
| `IID_IRandomAccessStream` | `905A0FE1-BC53-11DF-8C49-001E4FC686DA` | RandomAccessStream base | internal |
| `IID_IRandomAccessStreamStatics` | `524CEDCF-6E29-4CE5-9573-6B753DB66C3A` | RandomAccessStream factory | unused |
| `IID_IRandomAccessStreamWithContentType` | `CC254827-4B3D-438F-9232-10C76BC7E038` | Stream with content type | unused |
| `IID_IInputStream` | `905A0FE2-BC53-11DF-8C49-001E4FC686DA` | Input stream | unused |
| `IID_IOutputStream` | `905A0FE6-BC53-11DF-8C49-001E4FC686DA` | Output stream | unused |
| `IID_IDataWriter` | `64B89265-D341-4922-B38A-DD4AF8808C4E` | DataWriter | unused |
| `IID_IDataReader` | `E2B50029-B4C1-4314-A4B8-FB813A2F275E` | DataReader | unused |
| `IID_IFileOpenPicker` | `8CEB6CD2-B446-46F7-B265-90F8E55AD650` | FileOpenPicker instance | internal |
| `IID_IFileOpenPickerStatics` | `6821573B-2F02-4833-96D4-ABBFAD72B67B` | FileOpenPicker factory | unused |
| `IID_IFileSavePicker` | `0EC313A2-D24B-449A-8197-E89104FD42CC` | FileSavePicker instance | internal |
| `IID_IFileSavePickerStatics` | `28E3CF9E-961C-5E2C-AED7-E64737F4CE37` | FileSavePicker factory | unused |
| `IID_IOcrEngineStatics` | `5BFFA85A-3384-3540-9940-699120D428A8` | OcrEngine factory | used |
| `IID_IOcrEngine` | `5A14BC41-5B76-3140-B680-8825562683AC` | OcrEngine instance | used |
| `IID_IOcrResult` | `9BD235B2-175B-3D6A-92E2-388C206E2F63` | OCR result | unused |
| `IID_IOcrLine` | `0043A16F-E31F-3A24-899C-D444BD088124` | OCR line | unused |
| `IID_IOcrWord` | `3C2A477A-5CD9-3525-BA2A-23D1E0A68A1D` | OCR word | unused |
| `IID_ILanguageFactory` | `9B0252AC-0C27-44F8-B792-9793FB66C63E` | Language factory | used |
| `IID_ILanguage` | `EA79A752-F7C2-4265-B1BD-C4DEC4E4F080` | Language instance | used |
| `IID_ILanguageStatics` | `B23CD557-0865-46D4-89B8-D59BE8990F0D` | Language statics | unused |
| `IID_IDeviceInformation` | `ABA0FB95-4398-489D-8E44-E6130927011F` | DeviceInformation instance | internal |
| `IID_IDeviceInformationStatics` | `C17F100E-3A46-4A78-8013-769DC9B97390` | DeviceInformation factory | unused |
| `IID_IMediaDeviceStatics` | `AA2D9A40-909F-4BBA-BF8B-0C0D296F14F0` | MediaDevice factory | unused |
| `IID_IPowerManagerStatics` | `1394825D-62CE-4364-98D5-AA28C7FBD15B` | PowerManager factory | unused |
| `IID_ILauncherStatics` | `277151C3-9E3E-42F6-91A4-5DFDEB232451` | Launcher factory | unused |
| `IID_IUserProfilePersonalizationSettings` | `8CEDDAB4-7998-46D5-8DD3-184F1C5F9AB9` | Personalization settings | internal |
| `IID_IUserProfilePersonalizationSettingsStatics` | `91ACB841-5037-454B-9883-BB772D08DD16` | Personalization settings factory | unused |
| `IID_IUserInformationStatics` | `77F3A910-48FA-489C-934E-2AE85BA8F772` | UserInformation factory | unused |
| `IID_IProcessDiagnosticInfo` | `E830B04B-300E-4EE6-A0AB-5B5F5231B434` | Process info instance | internal |
| `IID_IProcessDiagnosticInfoStatics` | `2F41B260-B49F-428C-AA0E-84744F49CA95` | Process info factory | unused |
| `IID_IDisplayInformation` | `BED112AE-ADC3-4DC9-AE65-851F4D7D4799` | Display info instance | internal |
| `IID_IDisplayInformationStatics` | `C6A02A6C-D452-44DC-BA07-96F3C6ADF9D1` | Display info factory | unused |
| `IID_IToastNotification` | `997E2675-059E-4E60-8B06-1760917C8B80` | Toast notification | internal |
| `IID_IToastNotificationManagerStatics` | `50AC103F-D235-4598-BBEF-98FE4D1A3AD4` | Toast notification factory | unused |
| `IID_IClipboardStatics` | `C627E291-34E2-4963-8EED-93CBB0EA3D70` | Clipboard factory | unused |
| `IID_IDataPackage` | `61EBF5C7-EFEA-4346-9554-981D7E198FFE` | DataPackage instance | unused |
| `IID_IGlobalSystemMediaTransportControlsSessionManager` | `CACE8EAC-E86E-504A-AB31-5FF8FF1BCE49` | Media session manager | internal |
| `IID_IGlobalSystemMediaTransportControlsSessionManagerStatics` | `2050C4EE-11A0-57DE-AED7-C97C70338245` | Media session manager factory | unused |
<!-- IID_TABLE_END -->

All IIDs discovered and verified on **Win11 26200** via [`scripts\discover-winrt-iids.ps1`](../../scripts/discover-winrt-iids.ps1) — run `powershell -File scripts\discover-winrt-iids.ps1 -UpdateDocs` on any new Windows build to re-verify the full set. The script loads WinRT types, enumerates their interfaces via reflection, and calls `Marshal.GenerateGuidForType` to extract COM IIDs. To add more types, extend the `$null = [Type, Namespace, ContentType=WindowsRuntime]` section at the top.

WinRT IIDs are frozen per interface contract — they only change when Microsoft introduces a new interface version suffix (e.g. `IOcrEngineStatics2` would get a new IID). The discovery script is the authoritative source; if it reports a different IID than what's in `winrt.go`, the code is wrong.

---

## 8. UIA COM (UI Automation)

### Initialization

```go
func newUIA() (*uiaAuto, error) {
    var p unsafe.Pointer
    r, _, _ := procCoCreateInstance.Call(
        uintptr(unsafe.Pointer(CLSID_CUIAutomation)),  // {FF48DBA4-60EF-4201-AA87-54103EEF594E}
        0, CLSCTX_INPROC_SERVER,
        uintptr(unsafe.Pointer(IID_IUIAutomation)),     // {30CBE57D-D9D0-452A-AB13-7AC5AC4825EE}
        uintptr(unsafe.Pointer(&p)),
    )
    return &uiaAuto{p: p}, nil
}
```

### Conditions

Conditions are COM objects that filter UIA tree traversal:

| Condition Creator | vtbl Index | Purpose |
|-----------------|-----------|---------|
| `CreateTrueCondition` | 21 | Match all elements |
| `CreatePropertyCondition(propertyId, variant)` | 23 | Match by property value |
| `CreateAndCondition(c1, c2)` | 25 | Combine two conditions |

Conditions are combined in `buildCondition()` (`uia.go:73`):

```go
// For multi-filter (name + control type + automation ID):
cond1 := au.createPropertyCondition(UIA_NamePropertyId, varString("Submit"))
cond2 := au.createPropertyCondition(UIA_ControlTypePropertyId, varInt(50000))
andC, _ := au.createAndCondition(cond1, cond2)
// Individual conds released — AndCondition holds references
comRelease(cond1)
comRelease(cond2)
```

### Property IDs

| Constant | Value | Type |
|----------|-------|------|
| `UIA_NamePropertyId` | 30005 | VT_BSTR |
| `UIA_ControlTypePropertyId` | 30003 | VT_I4 |
| `UIA_ProcessIdPropertyId` | 30020 | VT_I4 |
| `UIA_IsEnabledPropertyId` | 30062 | VT_BOOL |
| `UIA_BoundingRectanglePropertyId` | 30031 | VT_R8 \| VT_ARRAY |
| `UIA_AutomationIdPropertyId` | 30011 | VT_BSTR |

### Control type constants

| Constant | Value | String Name |
|----------|-------|-------------|
| `UIA_ButtonControlType` | 50000 | `Button` |
| `UIA_EditControlType` | 50004 | `Edit` |
| `UIA_WindowControlType` | 50032 | `Window` |
| `UIA_PaneControlType` | 50033 | `Pane` |

Full list: 41 control types mapped in `controlTypeName()` (`uia_com.go:335`).

### Tree traversal strategy

`UIAFindElement()` (`uia.go:161`) uses three strategies:

```
hasExact name/automation_id?
  YES → FindFirst(Descendants)
         Fast (~2ms), returns single element
  NO (control_type only) → FindAll(Children)
         Root children only, ~275ms worst case
```

### Pattern operations

Patterns are accessed via `GetCurrentPattern(patternId)` (vtbl index 16):

```go
elem.getCurrentPattern(InvokePatternId)  // → IUIAutomationInvokePattern (Invoke=3)
elem.getCurrentPattern(ValuePatternId)   // → IUIAutomationValuePattern (get_Value=3, SetValue=4)
```

**Bounding rectangle** uses two paths:
1. Direct `get_CurrentBoundingRectangle` (vtbl 43) — preferred
2. Fallback `GetCurrentPropertyValue(UIA_BoundingRectanglePropertyId)` → `SafeArrayGetElement` × 4

---

## 9. Fallback Chain

OCR uses a COM-first strategy: try native WinRT COM, fall back to PowerShell on failure. UIA has no fallback (native COM only; pre-v0.2.5 used PowerShell).

| Operation | Primary (COM) | Fallback (PowerShell) | File |
|-----------|--------------|----------------------|------|
| **OCR** | `ocrNative()` — WinRT COM via `ocr_com.go` | `ocrExecWithRetry()` — PowerShell script `ocrScript` | `ocr.go:205` |
| **OcrLanguages** | WinRT COM `IOcrEngineStatics.get_AvailableRecognizerLanguages` | None | `ocr.go:151` |
| **UIA find/get text/invoke** | UIA COM via `uia_com.go` | None (pre-v0.2.5 used PowerShell) | `uia.go` |

---

## 10. Memory Safety & unsafe.Pointer Rules

### Do not store COM pointers as uintptr

```go
// CORRECT: stored as unsafe.Pointer (GC-tracked)
type uiaAuto struct { p unsafe.Pointer }

// WRONG: would be invisible to GC, could become dangling
type uiaAuto struct { p uintptr }
```

### Do pass COM pointers as uintptr to SyscallN

```go
// CORRECT: SyscallN accepts uintptr copies for call duration only
r, _, _ := syscall.SyscallN(vtblMethod(a.p, 5), uintptr(a.p), uintptr(unsafe.Pointer(&e)))
```

### HSTRING raw buffer extraction

```go
// WindowsGetStringRawBuffer returns internal pointer — valid only while HSTRING is alive
bufRaw, _, _ := procWindowsGetStringRawBuffer.Call(uintptr(h), uintptr(unsafe.Pointer(&length)))
str := syscall.UTF16ToString(unsafe.Slice((*uint16)(*(*unsafe.Pointer)(unsafe.Pointer(&bufRaw))), length))
```
The HSTRING handle `h` must not be deleted before the string is copied.

### BSTR lifetime

```go
v, _ := elem.getPropValue(UIA_NamePropertyId)  // v contains VT_BSTR + pointer
name := bstrToGo(*(*unsafe.Pointer)(unsafe.Pointer(&v.data)))
// v is stack-allocated — no explicit free needed for the VARIANT itself,
// but UIA owns the BSTR memory. For variants we create (varString),
// explicit varFree() is required.
```

---

## 11. PowerShell-Only Features (Audio & Brightness)

Audio and brightness never touch Go COM code. They shell out to PowerShell, which internally uses WinRT COM or WMI:

| API | PowerShell Uses | Why Not Native COM |
|-----|----------------|-------------------|
| `ListAudioDevices` / `SetDefaultAudioDevice` | `Windows.Media.Devices.MediaDevice`, `Windows.Devices.Enumeration.DeviceInformation` | WinRT async patterns not ported to native Go COM yet |
| `GetBrightness` / `SetBrightness` | WMI `WmiMonitorBrightnessMethods` | WMI-only API, no WinRT equivalent |

---

## 12. Code Map

| File | Lines | Role | COM Technology |
|------|-------|------|---------------|
| `winrt.go` | 242 | WinRT infrastructure: HSTRING, RoInitialize, async polling, helpers | combase.dll |
| `uia_com.go` | 552 | UIA COM bindings: IUIAutomation, conditions, elements, patterns, safe array | ole32.dll + oleaut32.dll |
| `uia.go` | 339 | Public UIA API: UIAFind, UIAGetText, UIAInvoke, WarmupUIA, condition builder | wraps uia_com.go |
| `ocr_com.go` | 425 | WinRT OCR pipeline: StorageFile → async decode → OcrEngine → result | combase.dll |
| `ocr.go` | 233 | OCR orchestration: screen capture → native COM → PowerShell fallback | wraps ocr_com.go |
| `audio.go` | 85 | Audio device enumeration via PowerShell (no Go COM) | PowerShell |
| `brightness.go` | 39 | Brightness via PowerShell WMI (no Go COM) | PowerShell + WMI |

## Cross-Reference

- `docs/reference/windows-dll-ref.md` — DLL proc tables for combase.dll, ole32.dll, oleaut32.dll
- `docs/reference/vtable-verification.md` — stability model, re-verification procedure, CI/CD test plan
- `docs/reference/uipi.md` — UIPI elevation detection (separate from COM)
- `docs/reference/codebase-map.md` — tool→handler→action→file mapping
- `docs/adr/adr-002-windows-automation-strategy.md` — strategic decision: native COM over go-ole/CGO
- `docs/meta/CHANGELOG.md` — history of COM fixes (MTA alignment, async pattern, safety)
- `docs/reference/tools.md` — tool reference for uia_find/uia_get_text/uia_invoke/ocr_languages
