# Vtable Index Verification

All COM/WinRT calls use raw vtable dispatch with hardcoded method indices (`vtblMethod(iface, 7)`). An incorrect index calls the wrong function pointer — silent corruption, no crash.

## Stability Guarantee

**Published COM interfaces never change their vtable layout.** Microsoft adds new methods via new interface IIDs (`IUIAutomation2`, `IUIAutomation3`, ...). This means:

| Interface | Released | Stability |
|-----------|----------|-----------|
| `IUIAutomation` | Windows 7 (2009) | Frozen — billions of apps depend on it |
| `IUIAutomationElement` | Windows 7 | Same |
| `IAsyncInfo` | Windows 8 | Same — core WinRT contract |
| `IOcrEngineStatics` | Windows 10 (2015) | Same — published Windows SDK API |
| `IStorageFileStatics` | Windows 8 | Same |

**Zero known cases** of a Microsoft-published COM interface changing vtable layout post-release. The real risk is **incorrect initial indexing**, not drift.

## How Indices Are Computed

Every WinRT COM interface inherits from `IInspectable`, which inherits from `IUnknown`:

```
[0]  IUnknown::QueryInterface
[1]  IUnknown::AddRef
[2]  IUnknown::Release
[3]  IInspectable::GetIids
[4]  IInspectable::GetRuntimeClassName
[5]  IInspectable::GetTrustLevel
[6]  <first interface-specific method>
[7]  <second>
...
```

UIA COM interfaces (`IUIAutomation`, `IUIAutomationElement`) inherit from `IDispatch` (which inherits from `IUnknown`), adding more base methods before the application methods start.

The verified indices are published in [`com-patterns.md`](com-patterns.md#2-com-vtable-dispatch).

## Re-Verification Procedure

When adding a new COM call with an unverified vtable index:

### Step 1: Find the Windows SDK header

All WinRT interfaces used here are defined in the Windows SDK, installed at:
```
C:\Program Files (x86)\Windows Kits\10\Include\<version>\winrt\
```

| Interface | File |
|-----------|------|
| `IOcrEngineStatics` | `windows.media.ocr.h` |
| `IOcrEngine` | `windows.media.ocr.h` |
| `ILanguageFactory` | `windows.globalization.h` |
| `IStorageFileStatics` | `windows.storage.h` |
| `IBitmapDecoderStatics` | `windows.graphics.imaging.h` |
| `IAsyncInfo` | `inspectable.h` |
| `IUIAutomation` / `IUIAutomationElement` | `UIAutomation.h` |

### Step 2: Count inherited methods

| Base | Methods | Occupies |
|------|---------|----------|
| `IUnknown` | `QueryInterface`, `AddRef`, `Release` | 0-2 |
| `IInspectable` | `GetIids`, `GetRuntimeClassName`, `GetTrustLevel` | 3-5 |
| `IDispatch` | `GetTypeInfoCount`, `GetTypeInfo`, `GetIDsOfNames`, `Invoke` | 3-6 |

WinRT interfaces use `IInspectable` → first app method at index 6.  
UIA interfaces use `IDispatch` → first app method at index 7.  

### Step 3: Count method declarations in IDL order

```cpp
// From inspectable.h — IOcrEngineStatics inherits IInspectable
struct IOcrEngineStatics : public IInspectable
{
    // [6]  HRESULT get_AvailableRecognizerLanguages(...)
    // [7]  ...
    // [8]  HRESULT TryCreateFromLanguage(...)
    // [9]  HRESULT TryCreateFromUserProfileLanguages(...)
};
```

The declaration order in the `.h` file (after the base vtable entries) determines the index. Read top-to-bottom, starting from the base count.

### Step 4: Verify with a smoke call

Before relying on an index, call it in a one-off test and verify the result makes sense:

```go
func verifyOcrLanguagesIndex() {
    factory := getFactory(...)          // RoGetActivationFactory
    var view unsafe.Pointer
    hr, _, _ := syscall.SyscallN(
        vtblMethod(factory, 7),         // get_AvailableRecognizerLanguages
        uintptr(factory),
        uintptr(unsafe.Pointer(&view)),
    )
    if hr != 0 { panic("index 7 failed") }
    if view == nil { panic("index 7 returned nil") }
    comRelease(view)
}
```

If the call returns `S_OK` and a non-nil result, the index is correct. If it crashes with an illegal instruction or returns `E_POINTER`/`E_NOTIMPL`, the index is wrong.

### Step 5: Mark in source

```go
// verified 2026-06-30 on Windows 11 24H2 (build 26100)
// SDK 10.0.26100.0 - windows.media.ocr.h line 142
```

## Developer Workflow

When adding or modifying a COM vtable call, follow this checklist **before pushing**:

```
┌──────────────────────────────────────────────────────────────┐
│  1. Write the COM call in Go with the guessed vtable index   │
│  2. Verify the index against Windows SDK headers (§Re-Verify)│
│  3. Run a local smoke test to confirm the call works          │
│  4. Run `go vet ./internal/actions/` (unsafe.Pointer rules)   │
│  5. Commit with index and version annotated in source         │
└──────────────────────────────────────────────────────────────┘
```

### Local smoke test (no CI needed)

Add a throwaway test in the same file or a `_test.go` companion:

```go
//go:build windows

func TestNewCOMCallVtable(t *testing.T) {
    if err := ensureRo(); err != nil {
        t.Skip("WinRT not available:", err)
    }
    // ... create interface, call method at target index
    // Assert hr == S_OK, result != nil
}
```

Run locally:
```pwsh
go test -v -run TestNewCOMCallVtable ./internal/actions/
```

| Result | Meaning |
|--------|---------|
| Panic with illegal instruction | Wrong index — calling garbage in the vtable |
| `E_POINTER` | Interface pointer itself is invalid |
| `S_OK` + valid result | Index is correct |

Delete the throwaway test after verification, or convert it into a permanent smoke test (see CI/CD section below).

### Source annotations

Every vtable call site must annotate the index:

```go
// uia_com.go:165
r, _, _ := syscall.SyscallN(vtblMethod(a.p, 5), uintptr(a.p), uintptr(unsafe.Pointer(&e)))
// 5 = GetRootElement (verified 2026-06-30, Win11 24H2, SDK 10.0.26100.0)
```

Format: `// <N> = <MethodName> (verified YYYY-MM-DD, <WindowsVersion>)`

### Local vet check

COM interop uses `unsafe.Pointer` in patterns that `go vet` flags. Run before committing:

```pwsh
go vet ./internal/actions/
```

`go vet` **must** pass. See `.govetallow` for the conventions that satisfy the `unsafeptr` checker.

## CI/CD Integration

CI runs on `windows-latest` (GitHub Actions), making vtable smoke tests possible. Currently CI only builds + vets — the vtable test suite exists (`internal/actions/vtable_test.go`) but is **not yet wired into CI**.

### Existing: `internal/actions/vtable_test.go`

File: `internal/actions/vtable_test.go` — build tag `//go:build vtable && windows`.

Run locally:
```pwsh
go test -v -tags=vtable ./internal/actions/ -run 'TestVtable'
```

| Test (actual) | vtbl Indices | What It Checks | Verified |
|------|------|---------------|----------|
| `TestVtable_IUIAutomation_GetRootElement` | `5` | CoCreateInstance → GetRootElement → non-nil | 2026-06-30 Win11 26200 |
| `TestVtable_IUIAutomationElement_BoundingRect` | `43` | get_CurrentBoundingRectangle → reasonable rect | 2026-06-30 Win11 26200 |
| `TestVtable_IUIAutomationElement_FindFirst` | `5, 21` | CreateTrueCondition → FindFirst Children | 2026-06-30 Win11 26200 |
| `TestVtable_IUIAutomation_Conditions` | `21, 23, 25` | CreateTrueCondition, CreatePropertyCondition, CreateAndCondition → non-nil | 2026-06-30 Win11 26200 |
| `TestVtable_IUIAutomationElement_GetPropValue` | `10` | GetCurrentPropertyValue → non-nil | 2026-06-30 Win11 26200 |
| `TestVtable_IUIAutomationElement_GetCurrentPattern` | `16` | GetCurrentPattern → ValuePattern | 2026-06-30 Win11 26200 |
| `TestVtable_IUIAutomation_ValuePattern_Invoke_Array` | `3, 4, 6, 21` | get_Value, Length, GetElement, Invoke via ValuePattern + `FindAll` on desktop root — **slow** (10-30s, walks all top-level windows) | 2026-06-30 Win11 26200 |
| `TestVtable_OcrEngineStatics_GetLanguages` | `7, 6` | get_AvailableRecognizerLanguages (IVectorView) → count | 2026-06-30 Win11 26200 |
| `TestVtable_OcrEngine_TryCreate` | `10` | TryCreateFromUserProfileLanguages → engine | 2026-06-30 Win11 26200 |
| `TestVtable_OcrEngine_TryCreateFromLanguage` | `9` | TryCreateFromLanguage → engine or nil | 2026-06-30 Win11 26200 |
| `TestVtable_StorageFile_Async` | `6, 8, 14, 0, 7` | GetFileFromPathAsync → waitForAsync → GetResults → QI → IStorageFile | 2026-06-30 Win11 26200 |
| `TestVtable_HSTRING_CreateExtract` | — | WindowsCreateString → WindowsGetStringRawBuffer → round-trip | 2026-06-30 Win11 26200 |
| `TestVtable_vtblMethod_rejectsNil` | — | vtblMethod(nil) panics | 2026-06-30 Win11 26200 |

### Performance note

`TestVtable_IUIAutomation_ValuePattern_Invoke_Array` runs `FindAll(TreeScope_Children)` on the desktop root to exercise vtbl indices `3` (Length), `4` (GetElement), and `6` (FindAll). UIA walks every top-level window — typically 10-30s depending on number of open windows and cross-process COM latency. This is deliberate:

- **Every vtbl index used in production must be exercised** even if slow, so the test suite serves as a complete reference of all live call sites.
- The suite is build-tagged `vtable && windows` — excluded from `go test ./...`, only runs on demand with `-tags=vtable`.
- Testing `FindAll` on the full desktop root exercises the real UIA tree walk path. Narrowing the scope would skip the production code path users actually rely on.

Run the full suite:
```pwsh
go test -v -tags=vtable ./internal/actions/ -run 'TestVtable'
```

### Integration into `ci.yml` (live)

VTable tests run as a separate CI job after lint. See `.github/workflows/ci.yml` for the `vtable-check` step:

```yaml
- name: COM vtable verification
  shell: pwsh
  run: go test -v -tags=vtable ./internal/actions/ -run 'TestVtable'
```

### Trigger table

| When | What to do | Who |
|------|-----------|-----|
| Adding a new COM call | Dev workflow (verify → smoke → annotate → vet) | **Dev** |
| Windows SDK update | Re-verify indices against new headers, update `verified` comments | **Dev** |
| Windows version bump in CI | Run full vtable test suite to confirm no drift | **CI (future)** |
| Unexplained COM crashes | Run vtable tests to rule out index drift | **Dev + CI** |

### Failure response

| Outcome | Action |
|---------|--------|
| **Index mismatch** (extremely rare) | Re-count from SDK header, update index + `verified` comment with new SDK version |
| **Interface not available** | Implement capability check or fall through to PowerShell fallback |
| **CoCreateInstance / RoGetActivationFactory fails** | COM server not registered — test should Skip, not Fail |

## Cross-Reference

- `docs/reference/com-patterns.md` — vtable dispatch pattern, verified index tables
- `docs/reference/windows-dll-ref.md` — DLL proc mapping
- `docs/ci-cd-pipeline.md` — CI workflow definition
- `internal/actions/uia_com.go` — UIA vtable indices (all `// verified`)
- `internal/actions/ocr_com.go` — WinRT OCR vtable indices
- `internal/actions/ocr.go` — `OcrLanguages` vtable indices
- `internal/actions/winrt.go` — async polling, HSTRING helpers
- `.govetallow` — unsafe.Pointer safety conventions for COM interop
