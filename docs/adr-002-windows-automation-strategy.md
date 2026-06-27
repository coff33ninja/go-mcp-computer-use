# ADR-002: Windows Desktop Automation Strategy

## Status

Accepted

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
- No CGO dependency (easier cross-compilation, no GCC requirement)

## Decision

Use **Go + syscall/windows** with direct Win32 API calls:

- Screenshot: `CreateDC` + `BitBlt` via GDI (`golang.org/x/sys/windows`)
- Mouse/keyboard: `SendInput` Win32 API
- Window management: `EnumWindows`, `SetForegroundWindow`, `GetWindowText`
- Cursor: `GetCursorPos`

Avoid CGO and COM to keep builds simple and cross-compilable.

## Consequences

- Easier: pure Go, no CGO toolchain needed
- Easier: cross-compilation works out of the box
- Easier: no COM initialization/threading complexity
- Harder: more manual struct definitions and FFI boilerplate
- Harder: no UI Automation tree walking (cannot "find button labeled X" — but OCR via PowerShell + Windows.Media.Ocr fills this gap)
- Unofficial: OCR and brightness control shell out to PowerShell for WinRT/WMI APIs that lack Go bindings
