# Changelog

## [0.2.9] - 2026-06-29

### Added

- **`scripts/build.ps1`** — unified build script with `-UseZig` flag for CGO-enabled builds
- **CI/CD: CGO + Zig cc build pipeline** — CI now runs two jobs: no-CGO lint+build and CGO+Zig build. Release workflow produces both `mcp-server.exe` (no CGO) and `mcp-server-cgo.exe` (with ONNX support).
- **Zig 0.16.0 support** — `scripts/install.ps1` updated to download Zig 0.16.0

### Documentation

- **README.md** — documented CGO requirements for ONNX tools with Zig cc build instructions
- **known-issues.md** — B13: ONNX tools require CGO (documented workaround)

## [0.2.8] - 2026-06-29

### Added

- **`key_down` / `key_up` MCP tools** — separate key hold/release for game-play sequences. Chains can now hold movement keys while dragging camera and pressing abilities, all server-side with no round-trip latency. `KeyDown("W")` holds the key, `KeyUp("W")` releases. Full VK support including modifiers, letters, digits, and special keys.

- **`keylogger_start` / `keylogger_stop` / `keylogger_status` MCP tools** — record real keyboard + mouse input (keys, clicks, drags, moves, scroll) via low-level Windows hooks (`WH_KEYBOARD_LL` + `WH_MOUSE_LL`). Output is a chain-compatible JSON sequence for AI replay. Includes timing-accurate delays between events. Mouse clicks auto-detect drag vs click by distance/time thresholds. Mouse moves throttled to meaningful position changes.

- **`sendVKPress` helper with 50ms inter-key delay** — `KeyPress`, `TypeText`, `sendCharWithVK` now use `sendVKPress(vk)` which inserts a 50ms `time.Sleep` between key down and key up. Fixes game engines and DirectInput applications that miss instant down/up sequences (character switch hotkeys 1-4, ability keys).

### Fixed

- **`warnElevated` false positive when both server and target are elevated** — `warnElevated()` only checked if the foreground window was elevated, not the MCP server itself. If both are elevated (server running as Admin targeting an admin game), `SendInput` keyboard works fine, but the check falsely blocked it. Added `isSelfElevated()` — only blocks keyboard when server is non-elevated AND target is elevated.

- **`KeyPress` modifier ordering** — `["CTRL", "C"]` sent `C` via Unicode first, then pressed Ctrl down, then released Ctrl. The key arrived before the modifier was held. Rewrote to process keys in order: modifiers are pressed immediately, target keys are sent while held, all modifiers released in reverse at end.

- **Keyboard input uses VK codes instead of `KEYEVENTF_UNICODE`** — `KEYEVENTF_UNICODE` synthesizes `WM_CHAR` messages, which many applications ignore (game engines, terminals, code editors, browser input fields). Rewrote all keyboard functions to use VK codes:
  - `TypeText` and `TypeAndSubmit` use `sendCharWithVK()` — maps each character to its VK code + Shift state using `charToVK` table (US keyboard layout). Letters, digits, punctuation all handled.
  - `KeyPress` sends all keys (letters, digits, special keys) as VK codes. Modifier combos like Ctrl+C now work correctly: Ctrl down → VK_C → Ctrl up.

- **`Drag` incremental movement** — was sending a single jump from start to end (mouseDown → teleport → mouseUp). Games and map UIs ignored this as a teleport. Now sends 5–50 incremental steps with 5ms delays, proportional to distance. Map panning now works correctly.

### Changed

- **`sendUnicode` removed** — no longer used. All keyboard input via VK codes.

### Documentation

- **Elevation & UIPI** section in README — explains admin vs non-admin behavior
- **Known issues B11, B12** — documented keyboard issues and fixes

## [0.2.7] - 2026-06-29

### Added

- **Statistical prior model** (`priors_stats` tool) — Go-native "training" without Python. Element frequency + position distributions are learned per window from collected training samples. Priors boost confidence for expected elements (e.g., "laptop" in browser windows) and suppress unlikely ones (e.g., "tv" in code editor). Position outliers beyond 3σ are penalized.
- **Prior-based confidence adjustment** — `ONNXDetect` now calls `AdjustConfidenceWithPriors()` after NMS, adjusting every detection's confidence based on learned per-window statistics. Gated by `prior_adjustment` config field (default: `true`).
- **`export_yolo_dataset` tool** — exports unused training samples (signal_level >= 1) as a YOLO-format dataset (images + normalized label files + train/val split + `dataset.yaml`). Users with Python can train externally via Ultralytics.
- **`training_cleanup_noise` tool** — deletes low-signal (signal_level=0) samples older than a threshold. Supports `dry_run=true` to preview deletions. Frees disk space from watcher noise frames.
- **`training_enabled` config field** — when set to `false`, disables all auto-save training snapshots (both from actions and the background watcher). Default: `true`.
- **`prior_adjustment` config field** — when set to `false`, disables prior-based confidence adjustment in ONNXDetect. Default: `true`.
- **Priors auto-populated on save** — every training sample save (raw or watcher) also updates element priors via `UpdatePriorsFromDetections`. Negative samples (zero elements) update frequency denominators.

### Changed

- **`set_config` tool** — runtime config changes without restart. Accepts: `training_enabled` (stop/start background data collection), `prior_adjustment`, `verify_bounds`, `log_level`, `watcher_enabled` (start/stop watcher), `watcher_interval_seconds` (change polling frequency live). Changes persist to disk immediately. Enables users to disable data collection and control the watcher mid-session for privacy or debugging.
- **`watcher_auto_start` / `watcher_interval_seconds` config** — `watcher_auto_start: true` starts the background watcher on server boot with the configured interval. Default: `false`.
- **Tool count**: 99 → 103 (added `priors_stats`, `export_yolo_dataset`, `training_cleanup_noise`, `set_config`).

### Fixed

- **`SendInput` silently dropping mouse clicks** — the `input` struct in `mouse.go` had an orphan `_ [8]byte` padding field, making `unsafe.Sizeof` = 48 bytes. Windows `sizeof(INPUT)` on x64 is 40 bytes. `SendInput` returns 0 when `cbSize` doesn't match, so `SetCursorPos` moved the cursor but the click event never fired. Removed the extra padding — struct is now exactly 40 bytes.
- **Network struct layout mismatches** — `IP_ADDR_STRING` was missing `_ [4]byte` trailing padding (44→48 bytes). `IP_ADAPTER_INFO` and `FIXED_INFO` used `[260/132]uint16` for `char` arrays (2x Windows size, shifting every subsequent field). Changed to `[260/132]byte` and added alignment padding after `DhcpEnabled`.
- **All Windows API structs verified** — audited every struct passed to Win32 via `unsafe.Pointer` in `internal/actions/`: `mouseInput` (32B ✓), `input` (40B ✓), `point` (8B ✓), `keyboardInput` (24B ✓), `inputKbd` (40B ✓), `BITMAPINFOHEADER` (40B ✓), `RECT`, `MONITORINFOEXW`, `DEVMODEW`, `MEMORYSTATUSEX`, `SYSTEM_POWER_STATUS`, `PROCESSENTRY32W`, `LASTINPUTINFO`, `VARIANT`, `UiaRect`, `WinRect` — all match Windows x64 sizes.

### Changed

- **`Drag` rewritten for raw input games** — replaced `SetCursorPos` (invisible to DirectInput/raw input) with `SendInput` + `MOUSEEVENTF_MOVE | MOUSEEVENTF_ABSOLUTE`. Coordinates normalized to 0–65535 range. Game engines using raw input now see the movement between mouse-down and mouse-up.

### Documentation

- **Elevation & UIPI section** added to README — explains admin vs non-admin behavior (keyboard warns, mouse silently fails), how to run elevated, and reassurance that normal apps work fine without elevation.

## [0.2.6] - 2026-06-28

### Added

- **Training data pipeline** (`training_save_sample`, `training_list_samples`, `training_stats`, `training_mark_used`) — persistent screenshot + ONNX detection storage for model fine-tuning. Images saved to categorized folders (`raw/click/`, `raw/type/`, `raw/navigate/`, `raw/ocr/`, `raw/general/`, `watcher/elements_found/`, `watcher/no_elements/`) with metadata in `samples.db`. Each sample carries a `task_prompt` string that the ML learns to predict during training.
- **Auto-save on every UI action** — `click`, `type`, `scroll`, `drag`, `hover`, `key_press`, `type_and_submit`, `select_all_and_type`, `browser_navigate`, `browser_search`, `open_url`, and `find_text_and_click` handlers (both direct MCP and chain steps) automatically capture a screenshot + ONNX detection + save to `raw/{category}/` with the action description as `task_prompt`.
- **`find_ui_element` tool** — three-layer cascading element locator: checks memory first (cached ONNX detections by window+label), runs ONNX detection with label matching, falls back to OCR for text elements. Stores findings in memory for reuse. Saves training samples (positive + negative).
- **Memory-backed element caching** — every `ONNXDetect` call auto-stores detected elements as memory facts (`memory_set`, scope `ui`, keyed `ui:{window_title}:{class}`) with 1-hour TTL. AI can query memory for known element locations without re-running ML.
- **Quality/signal filtering** — every training sample gets a `signal_level` (0=noise, 1=elements found, 2=elements+task context). `training_list_samples` accepts `min_signal` filter. Noise samples (watcher frames with zero elements) are flagged for discard.

### Changed

- **Restructured training directories** — from flat `samples/{cat}_{ts}.png` to `raw/{cat}/{ts}.png` + `watcher/{cat}/{ts}.png` layout. Database renamed from `training.db` to `samples.db`.
- **Watcher save path** — frames now save to `watcher/elements_found/` or `watcher/no_elements/` instead of flat `references/` dir.
- **ONNXDetect no longer auto-saves** — removal of inline `saveTrainingSampleDirect` in ONNXDetect to avoid caller confusion. Watcher handles persistence; explicit calls handle the rest.

## [0.2.5] - 2026-06-28

### Fixed

- **`memory_set` schema validation** — `MemorySetArgs.Value any` generated `"value": true` in JSON Schema, which OpenCode's MCP validator rejected. Fixed with explicit `InputSchema` using `json.RawMessage` + description-only schema.
- **`close_window` Win32 API** — was calling `ShowWindowAsync(hwnd, 0x10)` but `0x10 = WM_CLOSE` is not a `ShowWindow` command. Changed to `PostMessageW(hwnd, WM_CLOSE, 0, 0)`.
- **`onnx_status` global state bug** — used global `modelsDir` which was empty when `InitONNX` failed. Now calls `getModelsDir()` directly.

### Added

- **Background watcher** (`onnx_watch_start/stop/status/cache`) — goroutine that periodically captures screen, runs ONNX detection, caches last 20 results, and auto-saves reference PNGs when detection returns zero elements.
- **`savePNG` auto-save in detection** — `onnx_detect` now saves a `ref_<ts>.png` to `%APPDATA%/go-mcp-computer-use/models/references/` when detection returns zero elements (AI confusion signal).
- **`focus_window_by_title`** — finds window by title, focuses, and clicks title bar to ensure activation.
- **Browser automation** — `browser_focus_url_bar`, `browser_new_tab`, `browser_navigate`, `browser_search`.
- **File Explorer automation** — `explorer_focus`, `explorer_open_path`.
- **`uia_warmup` config field** and async UIA warmup on startup.

### Changed

- **Eliminated Python dependency entirely** — removed `convertYoloToONNX()`, `detectWithPython()`, `pythonDetectResult` struct, `os/exec`, `bytes`, `strings` imports.
- **Switched YOLO model** — from HuggingFace `best.pt` (PyTorch, 57 MB, 7 UI classes) to Ultralytics pre-exported `yolo11n.onnx` (10.9 MB, 80 COCO classes).

## [0.2.0] - 2026-06-27

### Changed

- **v0.2.x branch baseline** — cut from v0.1.11 as starting point for v0.2 development. All subsequent changes on this branch increment as `+0.0.1` (v0.2.1, v0.2.2, etc.).

## [0.2.1] - 2026-06-27

### Added

- **`chain` tool** — sequential step executor that runs multiple tools server-side without round trips. Supports `tool` (call any registered tool), `wait` (sleep N ms), and `capture` (save step output as `{{variable}}` for use in subsequent steps). Error modes: `stop` (halt on first error, default) or `skip`. Global timeout. 40+ tools dispatched.
- **Variable substitution** — `{{variable_name}}` in string args is replaced with captured output from earlier steps.
- **ChainFromJSON** — convenience entry point for programmatic chain execution from JSON string.

## [0.2.4] - 2026-06-28

### Added

- **`memory_set` / `memory_get` / `memory_search` / `memory_list` / `memory_forget` tools** — SQLite-backed memory store using `modernc.org/sqlite` (pure Go, zero CGO). Database at `%APPDATA%/go-mcp-computer-use/memory.db` with WAL mode, FTS5 full-text search, auto-syncing triggers, TTL support, scope isolation, and tag filtering.
- **`layout_validate` tool** — validates stored UI element layouts against the current screen. Checks window existence, position drift (with tolerance), and OCR keyword verification around element coordinates. Returns per-element confidence (`ok`/`drifted`/`stale`) with adjusted coordinates.
- **`template_store` / `template_find` / `template_list` / `template_forget` tools** — self-growing template library. `template_store` auto-crops a 48×48 PNG template around a coordinate from the current screen and stores it in the `element_templates` table. `template_find` uses NCC template matching (`find_image`) to relocate the element visually on the current screen, returning coordinates and drift. Hit count auto-increments on each successful find, enabling the system to self-train over time.
- **`onnx_status` / `onnx_detect` / `onnx_download` tools** — ONNX ML backend for UI element detection. `onnx_status` checks runtime and model availability. `onnx_detect` runs YOLO11s inference on a screenshot or full screen to detect UI elements (button, textbox, checkbox, dropdown, icon, tab, menu_item) with bounding boxes and confidence scores. Uses `github.com/yalue/onnxruntime_go` for native ONNX Runtime support. Requires manual download of `onnxruntime.dll` and model files. Falls back gracefully when runtime/models are missing.
- **`focus_window_by_title` tool** — focus management for reliable keyboard input. Finds a window by title, focuses it, and clicks its title bar to ensure activation.
- **`ChainStep.FocusWindow` field** — chain steps can specify `focus_window: "window title"` to auto-focus and activate the window before executing the step. The chain executor handles window lookup, focus, title bar click, then runs the step.
- **`browser_focus_url_bar` / `browser_new_tab` / `browser_navigate` / `browser_search` tools** — generic browser automation (Firefox, Chrome, Edge, Brave, Opera). `browser_focus_url_bar` focuses the URL bar (Ctrl+T for Firefox, Ctrl+L for others). `browser_new_tab` opens a new tab (Ctrl+T). `browser_navigate` opens a new tab and navigates to a URL. `browser_search` opens a new tab and performs a search query. Backed by `BrowserFocusURLBar`, `BrowserNewTab`, `BrowserNavigate`, `BrowserSearch` in `internal/actions/browseruse.go` — reusable composite functions that import existing modules instead of duplicating logic.
- **`explorer_focus` / `explorer_open_path` tools** — File Explorer automation. `explorer_focus` finds and activates an existing File Explorer window by title. `explorer_open_path` opens explorer at a given path, reusing existing windows when possible (Ctrl+L + path) or launching a new one. Backed by `ExplorerFocus`, `ExplorerOpenPath`, `ExplorerNavigateTo` in `internal/actions/windowexploreruse.go`.

### Changed

- **Replaced `firefox_focus_url_bar`** — removed Firefox-specific function from `chained.go`. Replaced with generic `browseruse.go` that detects browser type from window title and uses browser-specific keyboard shortcuts (Ctrl+T for Firefox URL bar, Ctrl+L for Chrome/Edge).
- **Refactored `FocusWindowByTitle`** — now delegates to shared `focusAndActivateWindow` helper, reducing duplication across browser, explorer, and generic focus code paths.

### Removed

- **`FirefoxFocusURLBar`** — removed from `internal/actions/chained.go`. Superseded by `BrowserFocusURLBar`. Tool name changed from `firefox_focus_url_bar` to `browser_focus_url_bar`.

## [0.2.3] - 2026-06-28

### Fixed

- **`TypeAndSubmit` Enter via `KeyPress`** — appended `\r` used `sendUnicode(0x0D)` which sends the CR character via `KEYEVENTF_UNICODE`, unreliable in Firefox/browser address bars. Replaced with `KeyPress([]string{"ENTER"})` with a 50ms pause, matching the same code path used by the `key_press` handler.

## [0.2.2] - 2026-06-28

### Added

- **`poll` step type** — polls OCR at `every_ms` interval until `ocr_contains` text is found or `timeout_ms` elapses. Syntax: `{"poll": {"every_ms": 1000, "timeout_ms": 30000, "ocr_contains": "Submit"}}`.
- **`if` step type** — OCR checks for `ocr_contains` text, executes `then` or `else` branch. Syntax: `{"if": {"ocr_contains": "Error", "then": [...], "else": [...]}}`.
- **`loop` step type** — repeats sub-steps `times` iterations. Syntax: `{"loop": {"times": 5, "steps": [...]}}`.
- **`StepResult.Steps`** — nested step results for if/loop sub-steps, visible in chain output.
- **UIA warmup at server startup** — pre-initializes COM and creates/releases a UIA instance, absorbing the one-time 15-42s cold-start cost so handlers respond instantly.
- **`WarmupUIA()`** — exported function to pre-warm COM/UIA at server startup.

### Fixed

- **StepResult.Index always `0`** — `execWait`/`execTool` created fresh `StepResult` structs discarding the loop index. Index is now set after the switch.
- **`SelectAllAndType` uses VK codes** — `sendUnicode(0x01)` used `KEYEVENTF_UNICODE` (VK_PACKET) which doesn't trigger select-all in most apps. Replaced with `sendVK(VK_CONTROL)` + `sendVK(VK_A)` for reliable Ctrl+A.
- **Variable substitution supports dotted paths** — regex `[a-zA-Z0-9_]+` didn't match `{{size.width}}`. Updated to `[a-zA-Z0-9_.]+` with `resolveVarPath()` for nested map lookups.
- **`SelectAllAndType` elevated warning** — now calls `warnElevated()` before sending input, preventing silent drops on admin windows.

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
