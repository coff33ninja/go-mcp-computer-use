# Windows DLL & COM Reference

All native Windows API calls are made via `syscall.SyscallN` and `windows.NewLazySystemDLL` — no CGo, no `.c`/`.h` files. CGo is used only indirectly via the Zig cc build toolchain for ONNX runtime linking.

## System DLLs

### user32.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `user32.go` | `GetDC`, `ReleaseDC`, `GetSystemMetrics`, `GetWindowDC`, `GetDesktopWindow`, `WindowFromPoint`, `GetUpdateRect`, `GetWindowRect`, `SetWindowPos`, `ShowWindow`, `EnumWindows`, `GetWindowTextW`, `GetWindowTextLengthW`, `IsWindowVisible`, `GetForegroundWindow`, `SetForegroundWindow`, `AttachThreadInput`, `GetWindowThreadProcessId`, `GetCurrentThreadId`, `GetMenuItemCount`, `GetMenuItemRect`, `ClientToScreen` | Core window management, focus, enumeration |
| `keyboard.go` | `SendInput` | Input simulation via `INPUT` struct |
| `mouse.go` | `SendInput`, `GetCursorPos` | Mouse simulation |
| `keylogger.go` | `SetWinEventHook`, `UnhookWinEvent`, `GetMessageW`, `GetAsyncKeyState`, `GetKeyState`, `VkKeyScanW`, `MapVirtualKeyW`, `ToUnicode`, `GetKeyboardState`, `GetKeyboardLayout`, `GetForegroundWindow` | WinEvent hook for input recording |
| `uia_com.go` | via IUIAutomation COM interface | Accessibility tree navigation |
| `layout.go` | `GetKeyboardLayout`, `GetKeyboardLayoutNameW`, `ActivateKeyboardLayout`, `LoadKeyboardLayoutW` | Keyboard layout enumeration/switching |
| `window.go` | `EnumWindows`, `FindWindowW`, `SetForegroundWindow`, `MoveWindow`, `ShowWindowAsync`, `GetWindowRect`, `IsIconic`, `IsZoomed`, `GetWindowTextW`, `GetWindowThreadProcessId`, `AttachThreadInput` | Window enumeration, focus, state |
| `window_ext.go` | `MoveWindow`, `GetWindowRect`, `ShowWindowAsync`, `PostMessageW`, `IsIconic`, `IsZoomed`, `FindWindowW`, `MonitorFromWindow` | Extended window ops, fullscreen detection |

### gdi32.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `screenshot.go` | `CreateCompatibleDC`, `CreateCompatibleBitmap`, `SelectObject`, `BitBlt`, `GetDIBits`, `DeleteDC`, `DeleteObject`, `CreateDCW` | GDI screen capture, bitmap conversion |
| `dpi.go` | `GetDeviceCaps` | LOGPIXELSX/Y for DPI |

### kernel32.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `system.go` | `GetComputerNameW`, `GetNativeSystemInfo`, `GlobalMemoryStatusEx` | System info |
| `process.go` | `CreateToolhelp32Snapshot`, `Process32First`, `Process32Next`, `OpenProcess`, `TerminateProcess`, `CloseHandle` | Process listing/management |
| `window.go` | `OpenProcess`, `GetModuleBaseNameW`, `GetWindowThreadProcessId` | Process-per-window |
| `idle.go` | `GetLastInputInfo` | Idle time |
| `power.go` | `GetDiskFreeSpaceW` | Disk usage |
| `audio.go` | `GetVolumeInformationW` | Volume info |

### shell32.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `system.go` | `ShellExecuteW` | Open URLs, launch files |
| `windowexploreruse.go` | `ShellExecuteW` | File Explorer automation |

### shcore.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `dpi.go` | `SetProcessDpiAwareness`, `GetDpiForMonitor`, `GetScaleFactorForMonitor` | DPI awareness, per-monitor scaling |

### ntdll.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `misc.go` | `NtQuerySystemInformation` | System power status |
| `system.go` | `RtlGetVersion` | OS version details |

### powrprof.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `power.go` | `SetSuspendState`, `GetPwrCapabilities` | Sleep, hibernate |

### advapi32.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `uipi.go` | `OpenProcessToken`, `GetTokenInformation` | UIPI elevation detection via `TOKEN_ELEVATION` (see [`uipi.md`](uipi.md) for full logic) |

### iphlpapi.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `network.go` | `GetNetworkParams`, `GetAdaptersInfo` | Network info |

### winmm.dll
| File | Procs Used | Purpose |
|------|-----------|---------|
| `system.go` | `waveOutGetVolume`, `waveOutSetVolume` | System audio volume |

## COM & WinRT Infrastructure

### ole32.dll / oleaut32.dll
| File | Functions Used | Purpose |
|------|---------------|---------|
| `uia_com.go` | `CoInitializeEx`, `CoCreateInstance`, `CoTaskMemFree`, `SysFreeString`, `VariantClear` | UIA COM initialization, BSTR/VARIANT management |

### combase.dll
| File | Functions Used | Purpose |
|------|---------------|---------|
| `winrt.go` | `RoInitialize`, `RoGetActivationFactory`, `WindowsCreateStringReference`, `WindowsDeleteString`, `WindowsGetStringRawBuffer` | WinRT COM infrastructure |
| `ocr_com.go` | `RoInitialize`, `RoGetActivationFactory`, `WindowsCreateString`, `WindowsDeleteString`, `WindowsGetStringRawBuffer` | WinRT OCR via `Windows.Media.Ocr.OcrEngine` |
| `ocr.go` | `RoGetActivationFactory`, `WindowsCreateString`, `WindowsDeleteString`, vtbl method calls on `IOcrEngineStatics` | `OcrLanguages()` enumeration |

### onnxruntime.dll
| File | Purpose |
|------|---------|
| `onnx.go` | Loaded at runtime via `onnxruntime_go` Go library. YOLO detection + MobileNet classifier inference. Downloaded by `onnx_download` tool from https://github.com/microsoft/onnxruntime/releases |

## COM Interface Architecture

```
WinRT (combase.dll)
  ├── Windows.Media.Ocr.OcrEngine — OCR (ocr_com.go)
  │     └── IOcrEngineStatics → OcrEngine → OcrResult → OcrLine[] → OcrWord[]
  ├── Windows.Storage.StorageFile — file access for OCR images
  └── Windows.Graphics.Imaging — bitmap decode for OCR

UIA (ole32.dll → IUIAutomation)
  └── UIA COM vtbl interface (uia_com.go)
        ├── IUIAutomation → CreateTrueCondition, CreatePropertyCondition
        ├── IUIAutomationElement → GetCurrentPropertyValue, GetCurrentPattern
        └── IUIAutomationCondition → tree traversal

SendInput (user32.dll — kernel-mode input injection)
  ├── keyboard.go: KEYEVENTF_UNICODE + virtual key codes
  └── mouse.go: MOUSEEVENTF_LEFTDOWN/UP, RIGHT, MIDDLE, WHEEL, HWHEEL
```

## Calling Convention

All DLL calls use the same pattern:

```go
var user32 = windows.NewLazySystemDLL("user32.dll")
var findWindowW = user32.NewProc("FindWindowW")

func FindWindow(title string) uintptr {
    ptr, _ := syscall.UTF16PtrFromString(title)
    hwnd, _, _ := findWindowW.Call(0, uintptr(unsafe.Pointer(ptr)))
    return hwnd
}
```

COM vtbl calls use `syscall.SyscallN` with method index on the vtable:

```go
func vtblMethod(unk unsafe.Pointer, idx int) uintptr {
    return *(*uintptr)(unsafe.Pointer(
        *(*uintptr)(unk) + uintptr(idx)*unsafe.Sizeof(uintptr(0)),
    ))
}
```

## Cross-Reference

- `docs/reference/codebase-map.md` — tool→handler→action→file mapping
- `docs/reference/com-patterns.md` — COM threading, vtable dispatch, async polling, HSTRING/BSTR lifecycle
- `docs/guides/build.md` — Zig cc build, CGO linking
- `docs/reference/models-setup.md` — ONNX model download
