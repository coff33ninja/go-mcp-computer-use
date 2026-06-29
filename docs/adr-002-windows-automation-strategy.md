# ADR-002: Windows Desktop Automation Strategy

## Status

Accepted (updated 2026-06-27: now also uses native COM/WinRT)

## Context

We need to control the Windows desktop (screenshot, mouse, keyboard) from Go. Options:

1. **Go + syscall/windows** — Direct Win32 API calls via `golang.org/x/sys/windows`
2. **Go + COM via go-ole** — UI Automation COM interface for richer introspection
3. **PowerScripts invoked from Go** — Shell out to PowerShell for automation commands
4. **Go + CGO with MS UI Automation** — C bindings to Windows UIAutomationCore.dll

Key requirements:
- Screenshot capture (GDI + D3D)
- Mouse click/move (SendInput Win32 API)
- Keyboard input (SendInput)
- Window enumeration (EnumWindows)
- Reliability and speed (sub-second actions)
- No CGO dependency for core tools (ONNX ML inference is the exception — uses CGO via Zig cc; optional `-NoCGO` build flag excludes ONNX tools)

## Decision

Use **Go + syscall/windows** with direct Win32 API calls, plus **native COM** where beneficial:

- Screenshot: `CreateDC` + `BitBlt` via GDI (`golang.org/x/sys/windows`)
- Mouse/keyboard: `SendInput` Win32 API
- Window management: `EnumWindows`, `SetForegroundWindow`, `GetWindowText`
- Cursor: `GetCursorPos`
- **UI Automation**: Native COM calls to `UIAutomationCore.dll` (IUIAutomation, IUIAutomationElement) via raw vtable dispatch — no CGO, no go-ole
- **OCR**: Native WinRT COM via `combase.dll` (RoGetActivationFactory, WindowsCreateString, IAsyncOperation polling) — HSTRING management, activation factories, async result extraction, all via raw syscall

Avoid CGO for core tools to keep builds simple and cross-compilable. ONNX ML inference requires CGO (Zig cc) — excluded via `-NoCGO` build flag for pure-Go builds.

## Consequences

- Easier: pure Go for core tools, no CGO toolchain needed (ONNX is optional CGO via Zig cc)
- Easier: cross-compilation works out of the box
- Easier: direct COM vtable dispatch via syscall — no CGO, no go-ole dependency
- Faster: OCR 2-8x faster, UIA no longer shells out to PowerShell
- Harder: more manual struct definitions and FFI boilerplate
- Harder: COM threading model must be managed per-thread
- Involved: WinRT async operations require polling IAsyncInfo::Status rather than callback pattern
- Unofficial: brightness control still shells out to PowerShell for WMI APIs
