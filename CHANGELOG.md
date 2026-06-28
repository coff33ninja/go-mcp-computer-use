# Changelog

## [0.2.0] - 2026-06-27

### Changed

- **v0.2.x branch baseline** — cut from v0.1.11 as starting point for v0.2 development. All subsequent changes on this branch increment as `+0.0.1` (v0.2.1, v0.2.2, etc.).

## [0.2.1] - 2026-06-27

### Added

- **`chain` tool** — sequential step executor that runs multiple tools server-side without round trips. Supports `tool` (call any registered tool), `wait` (sleep N ms), and `capture` (save step output as `{{variable}}` for use in subsequent steps). Error modes: `stop` (halt on first error, default) or `skip`. Global timeout. 40+ tools dispatched.
- **Variable substitution** — `{{variable_name}}` in string args is replaced with captured output from earlier steps.
- **ChainFromJSON** — convenience entry point for programmatic chain execution from JSON string.

## [0.2.2] — *planned*

### Major

- **SQLite Memory Store** — `memory_set/get/search/list/forget` tools backed by `modernc.org/sqlite` (pure Go, zero CGO). Persists learned facts, sequences, and element templates across sessions.
- **Layout Validation** — before replaying a stored sequence, validates window position, element coordinates, and OCR signatures haven't drifted. Auto-adjusts or marks stale on mismatch.
- **Self-growing Template Library** — each discovered UI element stores a 32×32px template crop. Over time, `find_image` locates elements visually instead of relying on stale coordinates.
- **ONNX ML Backend** (optional v2+) — two-tier Windows UI element detection: YOLO11s (18 MB, 7 classes) + MobileNetV3-small classifier (6 MB, 15 classes). Hierarchical validation chain from O(1) coords → OCR → template → YOLO → classifier.
- **Prompt Engineering Guide** — documented Learn-Once-Reuse-Forever pattern for AI agents: store sequences + layout facts after every successful interaction, recall and replay next session with zero rediscovery.

## [0.1.11] - 2026-06-27

### Added

- **VERSION file + ldflags** — single source of truth at project root, injected via `-X main.Version`, replaces hardcoded string
- **CI/CD pipeline** — `.github/workflows/ci.yml` (build + vet on push/PR), `.github/workflows/release.yml` (tag-triggered GitHub Release with binary + SHA256 + changelog)
- **`.govetallow`** — documents COM/WinRT unsafe.Pointer conventions for vet policy
- **`scripts/lint.ps1`** — local CI runner: vet + build + tests

### Changed

- **COM types** — all interface pointers stored as `unsafe.Pointer` instead of `uintptr`:
  `uiaAuto.p`, `uiaCondition.p`, `uiaElement.p`, `uiaElementArray.p`,
  `bstrToGo` parameter, `getCurrentPattern` return type
- **`vtblMethod`** — rewritten with `unsafe.Pointer` parameter + `unsafe.Add`, satisfies vet's unsafeptr checker
- **Syscall output params** — all local variables receiving COM pointers via SyscallN declared as `unsafe.Pointer` instead of `uintptr`
- **GUID literals** — all 14 `windows.GUID` values in `winrt.go` use keyed fields
- **CI workflows** — use `scripts/lint.ps1` instead of raw `go vet`

## [0.1.10] - 2026-06-27

### Fixed

- Keyboard VK-coded keys (Enter, Backspace, Tab, Ctrl+letter) sent via `sendKey`/`KeyPress` were silently dropped by the system — only `KEYEVENTF_UNICODE` path worked. Rewrote keyboard handling to send **all** keys through `KEYEVENTF_UNICODE` where possible: special keys map to Unicode control characters (Enter=0x0D, Backspace=0x08, Ctrl+A-Z=0x01-0x1A). VK fallback only for non-printable keys (arrows, F-keys, Insert, etc.)
- `TypeAndSubmit` and `SelectAllAndType` now use Unicode path instead of VK-coded `KeyPress` for Enter and Ctrl+A

## [0.1.9] - 2026-06-27

### Added

- B9: UIPI elevation detection for keyboard input (`TypeText`, `KeyPress`) — returns clear warning when foreground window is elevated (admin), instead of silently dropping input

## [0.1.8] - 2026-06-27

### Fixed

- B3: `list_displays` only returned primary monitor — `monitorEnumProc` gated on `MONITORINFOF_PRIMARY` flag, skipping all non-primary displays

## [0.1.7] - 2026-06-27

### Fixed

- B4: `uia_get_text` / `uia_invoke` no longer crash MCP transport — `GetCurrentPattern` nil check added before pattern operations

## [0.1.6] - 2026-06-27

### Fixed

- B2: `list_audio_devices` returns `[]` instead of `null` — empty PowerShell output produced nil slice which serialized as JSON `null`

## [0.1.5] - 2026-06-27

### Fixed

- B6: `Wait()` calculation was **1 million times too long** — `NtDelayExecution` argument was `-(ns * 10000)` instead of `-(ms * 10000)`, causing `hover` (and any tool calling `Wait`) to block for hours instead of milliseconds

## [0.1.4] - 2026-06-27

### Fixed

- B1: `get_brightness` returns clear "brightness not supported on this display" instead of parse error when display doesn't support WMI brightness control (desktop monitors)

## [0.1.3] - 2026-06-27

### Fixed

- B5: `screenshot_element` now clamps off-screen window coordinates to screen bounds instead of rejecting them (e.g., windows with `x=-8` from Aero Snap)
- Multi-monitor: `ScreenSize()` now returns virtual desktop dimensions (`SM_CXVIRTUALSCREEN`/`SM_CYVIRTUALSCREEN`) instead of primary monitor only, fixing coordinate validation across multiple displays

## [0.1.2] - 2026-06-27

### Fixed

- UIA COM and OCR WinRT apartment model conflict: changed `RoInitialize` from `RO_INIT_SINGLETHREADED` to `RO_INIT_MULTITHREADED` so both UIA and OCR use MTA on the same thread, preventing `RPC_E_CHANGED_MODE` error

## [0.1.1] - 2026-06-27

### Added

- **Native WinRT COM OCR** — replaces PowerShell OCR with direct COM calls: `StorageFile.GetFileFromPathAsync` → `OpenAsync` → `BitmapDecoder.CreateAsync` → `GetSoftwareBitmapAsync` → `OcrEngine.RecognizeAsync`. Zero CGO, no Windows SDK needed.
- **Native COM UI Automation** — replaced PowerShell UIA with direct COM calls to `UIAutomationCore.dll` (IUIAutomation, IUIAutomationElement, conditions, patterns). All operations via native COM.
- **WinRT COM infrastructure** (`winrt.go`) — HSTRING management, `RoInitialize`, `RoGetActivationFactory`, `IAsyncInfo` polling, COM helpers
- OCR falls back to PowerShell if native COM fails

### Changed

- All OCR and UIA operations now use native COM instead of PowerShell — **2-8x faster**
  - OCR full screen: 653→292ms (2.2x)
  - OCR region 400×400: 542→68ms (8x)
  - find_text_and_click: 809→275ms (2.9x)
- `comRelease` signature changed from `uintptr` to `unsafe.Pointer` for unified COM cleanup
- ADR-002 updated: project now uses native COM/WinRT, not just Win32 API

### Fixed

- WindowsGetStringRawBuffer signature: actual DLL export returns buffer pointer in RAX (2 params), not as out parameter (3 params) — MSDN docs differ from Win10 10.0.26100 behavior
- All vtable reads: corrected `*(*[N]uintptr)(obj)` pattern (reads object data) to `vtblMethod()` (reads actual vtable entries)
- OCR PowerShell script: properly loads WinRT types via `WindowsRuntimeSystemExtensions.GetAwaiter` with `MakeGenericMethod`, fixing OCR on systems where WinRT async extension methods don't resolve in PowerShell 5.1
- Go raw string literal: avoids backtick in `IAsyncOperation`1` by using `-like` wildcard matching

## [0.1.0] - 2026-06-27

### Added

- Screenshot (full + region) via GDI BitBlt
- Mouse control: click, move, scroll, drag, hover
- Keyboard input: type, key_press, type_and_submit, select_all_and_type
- OCR via Windows.Media.Ocr with language support
- Template matching via normalized cross-correlation
- find_text_and_click, wait_for_text, click_menu_item, launch_and_wait
- Screen recording (duration_ms + interval_ms → base64 frames)
- Window management: list, focus, find, move, resize, minimize, maximize, restore, close, get_state
- Audio devices: list playback/recording, set default
- Clipboard: get/set with retry + timeout
- System: volume, mute, brightness, battery, disk, DPI, display info, uptime, idle
- Network: hostname, IPs, DNS, gateway, ping
- Processes: list, launch, kill
- Power: shutdown, restart, sleep, hibernate
- Per-monitor DPI awareness
- UI Automation via PowerShell: find elements, get text, invoke
- get_display_modes tool (69th tool) — enumerate all display modes
- Config file: `~/.config/go-mcp-computer-use/config.json`
- Install script: `scripts/install.ps1` with Zig cc support

### Changed

- syscall hardening: `ptr()` helper for safe unsafe.Pointer conversion
- performance optimizations across all action modules
- README with comprehensive tool listing and security warning
- MCP client configs documentation for 19 agents

### Security

- Added SECURITY WARNING section to README detailing dangerous capabilities
