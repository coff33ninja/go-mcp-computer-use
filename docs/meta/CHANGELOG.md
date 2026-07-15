# Changelog

## [0.2.39] - 2026-07-15

### Added

- **`get_dpi_for_point(x, y)`** — new MCP tool that returns DPI and scale percentage at a specific screen coordinate. Useful for determining which monitor a coordinate is on and its scaling factor in mixed-DPI multi-monitor setups. Returns `{dpi, scale_percent, x, y}`. Also available in chain dispatch.
- **`FocusHandle` chain step field** — new `focus_handle` field on chain steps accepts a handle directly (via `{{var.handle}}` capture), avoiding title re-resolution. Chain checks `focus_handle` first, falls back to `focus_window` (title-based).
- **`auto_verify_focus` chain option** — new boolean field on chain requests. When `true`, the chain engine tracks the last-focused window handle and re-verifies foreground state before sending input (click, type, key_press, scroll, etc.). If focus was stolen by a popup or notification, it re-focuses before acting.
- **`click_menu_item` accepts handle** — new optional `handle` field. When provided, bypasses title-based window lookup. Falls back to `window_title` if handle is 0.
- **`layout_validate` accepts window handle** — new optional `window_handle` field. When provided, bypasses title-based window lookup. Falls back to `window_title` if handle is 0.

### Fixed

- **`FocusWindow` now verifies and retries** — previously discarded `SetForegroundWindow` return and never confirmed the window actually became foreground. Now uses a 4-attempt fallback chain: (1) `SetForegroundWindow` with `AttachThreadInput`, (2) `BringWindowToTop` + `SW_SHOW`, (3) `SwitchToThisWindow` (bypasses foreground lock), (4) retry `SetForegroundWindow` after delay. Each attempt verified via `GetForegroundWindow`. Returns error if all attempts fail instead of silently returning nil. This fixes the stale-focus failure where clicks land on the wrong window.
- **`click_menu_item` no longer silently fails** — when window title matches but window is obscured, the improved `FocusWindow` (called by browser focus helpers) now actually brings the window to foreground before OCR+click.
- **`ensureWindowFocus` title-bar click guarded by z_order** — the post-focus activation click at `Top+10` was blind to always-on-top windows. Now checks `ZOrder == 0` (truly topmost) before clicking, preventing accidental hits on overlapping windows.

### VERSION

`0.2.38` → `0.2.39`

## [0.2.38] - 2026-07-15

### Added

- **`z_order` field on `get_window_state`** — reports 0=topmost, higher=deeper in Z-order stack. Uses `GetDesktopWindow` + `GW_CHILD` to find the true topmost, then walks `GW_HWNDNEXT` counting visible windows. The AI can compare z_order between handles to determine absolute stacking: window A at z_order=3 is above window B at z_order=12.
- **`uia_get_all_elements(handle, max_results)`** — returns all immediate child UI elements in a window (title bar, menu bar, content panes, toolbars, status bar). Uses `TreeScope_Children` + `TrueCondition` (one level deep, not recursive DOM tree) so a browser window doesn't flood with 10K+ elements. Returns name, control_type, automation_id, bounding rect, is_enabled for each.
- **`uia_get_element_at_point(x, y)`** — identifies which UI element is at screen coordinates using UIA `ElementFromPoint`. Returns name, control_type, automation_id, bounding rect. Use after `get_cursor_position` or click to validate what was under the cursor.
- **`wait_for_ui_element(handle, name, control_type, timeout_ms)`** — polls UIA FindFirst on a window's descendants until the element appears or timeout. Use for content verification: click a button, then wait_for_ui_element for the dialog that should appear. Default timeout 10s.
- **Auto-capture element_at_point in chain** — mouse-based chain steps (`click`, `move_mouse`, `hover`, `drag`) now automatically call `UIAElementFromPoint` at their target coordinates after execution. The UIA element is attached to the step output as `element_at_point` and available as a captured variable for subsequent steps.
- **`verify_ui` step type** — new chain step that verifies a UI element exists or disappears using UIA instead of OCR. Accepts `element_name`, `control_type`, `handle` (window scope), `timeout_ms`, `not_exists` (true = expect absence). Polls UIA `FindFirst` / `WaitForUIElement` until timeout. Complements the existing OCR-based `verify` step for structural post-action validation.
- **`if_uia` step type** — new conditional branch that checks UI element existence via UIA. Branches `then`/`else` based on whether an element with the given name/control_type is found. Like `if` (OCR), but structural rather than pixel/text-based.
- **Chain-callable UIA tools** — `uia_find`, `uia_get_element_at_point`, `uia_get_all_elements`, `uia_set_text`, `wait_for_ui_element` are now registered in the chain tool dispatch and can be called as regular chain steps with `{{variable}}` substitution.
- **Chain-callable window tools** — `get_active_window`, `ocr_window`, `ocr_active_window` added to chain dispatch.
- **`ocr_window` tool** — new MCP tool that extracts text from a specific window by handle using Windows OCR. Calls `OCRWindow(hwnd, language)` which captures the window's bounding rect via `GetWindowRectByHandle`, clamps to screen bounds, then runs WinRT OCR on the region. Accepts `handle` (uintptr) and optional `language`.
- **`ocr_active_window` tool** — new MCP tool that extracts text from the current foreground window. Uses `ForegroundWindowHandle()` to get the active window handle, then delegates to `OCRWindow`. Accepts optional `language` parameter.
- **`ForegroundWindowHandle()`** exported helper (`datalog.go`) — returns the current foreground window's HWND via `GetForegroundWindow`.
- **`clamp32()` helper** (`ocr.go`) — clamps int32 values to [lo, hi] range for safe screen bounds capping.
- **`list_windows` now returns bounding rect** — each window now includes `x`, `y`, `width`, `height` from `GetWindowRectByHandle`. The AI can cross-reference window bounds with `list_displays` monitor positions to determine which screen a window is on.
- **`get_active_window` now returns bounding rect** — same `x`, `y`, `width`, `height` fields added.

### Changed

- **`OCRWindow` now clamps to screen bounds** — window rects with negative coordinates (off-screen title bars) are clamped to the virtual screen dimensions via `ScreenSize()` before capture, preventing "out of bounds" errors from `ValidateRegion`.
- **`LogToolCall` auto-capture unchanged** — remains full-screen `OCRScreen` to preserve the training pair bridge behavior; the AI chooses between `ocr`, `ocr_window`, `ocr_active_window` as needed.
- **Tool descriptions updated** — `get_window_state` documents z_order and the visible-flag vs actual-visibility distinction. `focus_window` tells AI to ALWAYS call before interacting with a non-foreground window. `uia_find` mentions it can find textboxes, address bars, search menus, title bars; advises focusing window first. `ocr_window` warns about minimized/obscured windows. `list_windows` and `get_active_window` descriptions mention bounding rect fields.
- **Chain input schema relaxed** — `args` `AdditionalProperties` now accepts any type (`{}` instead of `{type:object}`), so `{{variable}}` template strings in arg values pass schema validation.

### VERSION

`0.2.37` → `0.2.38`

## [0.2.37] - 2026-07-08

### Added

- **Per-tool enable/disable** — `tool_denylist` config field (array of tool names) removes tools from the MCP server entirely so the AI never sees them. Uses `server.RemoveTools()` after registration — zero changes to existing handler code. Case-insensitive matching. Example: `"tool_denylist": ["shutdown", "restart", "hibernate"]`. Configurable at runtime via `set_config`.
- **Retention policy** — `retention_days` config field (integer, 0=disabled) auto-prunes training samples older than N days. Background pruner runs every 6 hours. Deletes both database rows and image files. Starts automatically on boot when `retention_days > 0` and `training_enabled` is true. Configurable at runtime via `set_config`.
- **Unit tests for v0.2.33 behaviors** — 25 new tests across `internal/actions/adaptive_test.go` and `internal/actions/datalog_test.go`: `uniqueTokens` dedup, `nearbyOCRText` spatial scoping, `capAndDedupeText` fallback, `SaveAdaptiveStat`/`LoadPersistedStats` round-trip, `Analyze()` persisted fallback. 5 new tests in `internal/config/config_test.go` for `ToolEnabled` helper.
- **`ToolEnabled` helper** (`config.go`) — case-insensitive denylist check with empty-string safety.

### Changed

- **`set_config` tool** — accepts new fields: `tool_denylist` (string array), `retention_days` (integer). Description updated to document both.
- **`Config` struct** — added `ToolDenylist []string` and `RetentionDays int` fields.

## [0.2.36] - 2026-07-07

### Fixed

- **All handlers now return structured JSON instead of hardcoded `"ok"`** — `verifiedResult()` helper and ~50+ handler functions set `Content: []mcp.Content{&mcp.TextContent{Text: "ok"}}` which blocked the MCP SDK from auto-populating with the structured second return value. When `Content == nil`, the SDK marshals the second value and sets both `StructuredContent` and `Content` (as JSON text). Fixed `verifiedResult()` to return `nil` Content when extra data is present, and changed every handler that returned explicit `"ok"` Content to return `&mcp.CallToolResult{}, map[string]any{"ok": true}, nil` (no-data tools) or `&mcp.CallToolResult{}, result, nil` (data tools). Verified live post-reboot: `list_windows`, `get_system_info`, `get_active_window`, `ocr` all return their structured JSON payloads instead of `"ok"`.

## [0.2.35] - 2026-07-07

### Fixed

- **`roUninitialize` never called on shutdown** — COM WinRT apartment initialized by `ensureRo()` was never cleaned up. Added `CloseWinRT()` in `ocr_com.go` wired into `main.go`'s signal handler so `RoUninitialize` runs on graceful shutdown.
- **`status` initializer unreachable in `verifiedResult`** — `status := "ok"` was always overwritten by the if/else branches before use. Changed to single default `status := "ok (verified)"` with negation on `!vr.Passed`.
- **`klMouseDown` and `winEventHookProc` dead code** — leftover scaffolding from keylogger refactor. `winEventHookProc` type was unused (callback uses `syscall.NewCallback` inline); `klMouseDown` was shadowed by local vars in `pollLoop`. Removed.
- **Unused `state` params in `execVerify` and `execPoll`** — both functions received `*chainState` but never used it. Renamed to `_ *chainState` to match the pattern already used by `execWait` and `execTool`. Keeps call-site interface compatibility with `execIf`/`execLoop` which do need state.
- **Inefficient string concatenation in `WriteString` calls** — three spots in `readXlsx` and `readPdf` concatenated strings before passing to `buf.WriteString(...)`, allocating intermediate strings. Changed to `fmt.Fprintf(&buf, ...)` which avoids the extra allocation.

### Added

- **`uia_set_text` MCP tool** — writes text into a UI element via UI Automation's `ValuePattern.SetValue`. The COM plumbing (`uiaElement.setValue`) existed but was never wired. Added `UIASetText()` in `uia.go`, handler and registration in `server.go`. Fills a backlog item from `docs/meta/backlog.md`.
- **Prior-based coordinate prediction in `FindUIElement`** — `priors.go` tracked element frequency and position per window but `FindUIElement` never consulted them. Added `FindPriorPrediction()` that returns predicted coordinates for high-confidence priors (frequency >= 70%, sample_count >= 5, StdX/StdY <= 2.0). Inserted as step 1.5 in the cascade: memory → **prior** → ONNX → OCR. A prior hit avoids ONNX (no GPU/CPU inference) and OCR (no PowerShell launch).
- **ONNX/OCR failure logging in `FindUIElement`** — when ONNX detection failed, the function silently fell through to OCR with no trace of the error. Same for OCR failure. Added `log.Printf` calls in both paths so server logs record when these subsystems are unavailable, distinguishing "not found" from "can't detect."
- **Competitive intelligence gathered from landscape survey** — cross-referenced 12 open-source and commercial projects in `docs/comparison-vs-alternatives.md` and extracted features worth adopting. Added a "Competitive Intelligence" section to `docs/meta/plan.md` with prioritized steals (quick wins → medium → high effort). Fleshed out `docs/meta/backlog.md` with 3 new sections (Memory & ML, Transport & Server, Browser Automation, Linux & Container) and enhanced Security & Identity, Vision, Mouse, Screen, and Processes sections — totaling ~41 new backlog items. Shout out to [Cua](https://github.com/trycua/cua), [Agent-S](https://github.com/simular-ai/Agent-S), [Bytebot](https://github.com/bytebot-ai/bytebot), [MS Magentic-UI](https://github.com/microsoft/magentic-ui), [Windows-MCP](https://github.com/CursorTouch/Windows-MCP), [DesktopCtl](https://github.com/yaroshevych/desktopctl), [Windows MCP Server](https://github.com/sbroenne/mcp-windows), [Computer Control MCP](https://github.com/AB498/computer-control-mcp), [Browser Use](https://github.com/browser-use/browser-use), and Microsoft Windows Recall for the reference implementations and design inspiration.

### Fixed

- **Double screenshot in `FindUIElement` OCR fallback** — when ONNX failed, `OCRScreen("")` captured the screen again even though `CaptureScreen()` had already been called at step 2. Changed OCR fallback to call `ocrFromBase64(b64, "")` directly, reusing the existing screenshot. Also performs `pushRecentOCR`/`tryCompletePair`/`LogOCRSnapshot` side effects that `OCRScreen` normally handles.
- **Fragile memory deserialization in `FindUIElement`** — the memory fast path used 4 levels of nested type assertions (`any → map → float64 → int32`) with silent fall-through on any mismatch. Extracted into `memoryToElement()` helper that returns `nil` on any parse failure, keeping it clear and testable.

## [0.2.34] - 2026-07-07

### Fixed

- **Training pair pipeline died permanently after a single missed OCR window** — `LogToolCall` only called `OCRScreen("")` (the auto-capture that completes a pending pair *and* refreshes `recentOCR` for the next action) inside the `if ocrBefore != ""` branch. If one action's `findRecentOCRBefore` call missed the `bridgeWindow` — entirely plausible under normal agent round-trip latency between an explicit `ocr()` call and the action that follows it — the buffer never refreshed again. Every subsequent action would also find nothing, permanently starving `training_pairs` until something called an OCR tool manually to re-seed it. Moved the `OCRScreen("")` auto-capture out of the conditional so it now runs after every action regardless of whether that action found a prior snapshot, making the bridge self-healing instead of a single point of failure.

### Changed

- **`bridgeWindow` widened 30s → 60s** — 30s was tight for realistic agent-driven latency between an OCR call and the action that follows it. Combined with the self-healing refresh above, 60s gives headroom without letting stale context linger indefinitely.

### Verification

Live end-to-end test through the MCP server (`ocr` → `click` → `type` → `key_press` → `click` → `type`): before this fix, 5 real actions produced 0 `training_pairs` rows (the bridge had already gone dark from an earlier missed window). After the fix and a rebuild/restart, the same kind of sequence produced 5/5 pairs, with `agent_train`/`agent_analyze` showing populated `timing_stats`, `success_rates`, and `top_sequences` with counts correctly bounded by `total_commands`.

## [0.2.33] - 2026-07-06

### Fixed

- **`agent_analyze`: timing_stats/success_rates resetting on restart** — Stats were purely in-memory (`map[string]*ToolTiming`, `map[string]*ToolSuccess`), so every server restart wiped them, leaving `agent_analyze` with empty timing_stats/success_rates until new commands arrived. Added `adaptive_stats` SQLite table and `SaveAdaptiveStat()`/`LoadPersistedStats()`/`HydratePersisted()`. On startup, `EnsureAdaptive()` calls `HydratePersisted()` to seed in-memory stats from the durable table, and every `RecordResult()` call persists the aggregate asynchronously via `go SaveAdaptiveStat(...)`.

- **Training token inflation: `uniqueTokens()` dedup** — `tokenize()` on raw OCR text produced duplicate word entries per row (same word appearing multiple times in the OCR dump contributed multiple hits to the (word, command) pair). This caused `Count` in `top_sequences` to exceed `total_commands`/`total_sequences` (which count rows, not word occurrences). Added `uniqueTokens()` to dedupe before insert, so each (word, command) pair gets at most 1 hit per row regardless of how many times the word appears in the OCR context.

- **OCR context scoped to nearby words** — `findRecentOCRBefore()` used to pass the ENTIRE screen's OCR dump as `ocr_before` for every command, so common on-screen words ("the", bullet points, news headlines, unrelated UI labels) all got associated with whatever command followed — producing noise-dominated predictions. Now:
  - For coordinate-based tools (click, move_mouse, drag, hover), it uses word bounding boxes (`words []OCRWord`) to scope context to words within 200px of the target coordinates, nearest-first, capped at 20 words.
  - For non-coordinate tools, it falls back to `capAndDedupeText()` which returns at most 40 unique words from the full dump instead of the raw dump.

- **`LogToolCall` no longer double-counts timing/success** — `LogToolCall` previously called `Adaptive.RecordResult(tool, 0, errVal == nil)` with a phantom 0ms duration. Callers (Click, TypeText, etc.) already call `Adaptive.RecordResult` with the real elapsed duration. Removed the call from `LogToolCall` to avoid every action being double-counted with a fake 0ms sample alongside the real one.

### Changed

- **`pushRecentOCR()` signature** — Now accepts `*OCRResult` (with word bounding boxes) instead of just `string`, enabling spatial scoping in `findRecentOCRBefore`.
- **`findRecentOCRBefore()` signature** — Now takes `tool string, argsJSON string` to enable tool-specific OCR scoping.
- **Datalog query limit relaxed** — From clamped 50–200 range to 1–5000 range (default 50).

## [0.2.32] - 2026-07-05

### Fixed

- **`chain` tool startup panic — shared sub-schema pointers** — `chainInputSchema()` reused the same `*jsonschema.Schema` for `then`, `else`, and `steps` fields. The MCP SDK's `AddTool()` requires schemas to form a tree (not a DAG) and panics on duplicate pointers. Changed to factory functions that return unique instances per call.
- **Module path mismatch** — `go.mod` declared `github.com/user/go-mcp-computer-use` but the repo lives at `github.com/coff33ninja/go-mcp-computer-use`. Updated module path and all 7 import references across the codebase. (User was lazy to update this.)
- **Adaptive engine: `timing_stats` and `success_rates` never populated** — `RecordCommand` (which calls `RecordResult` → `RecordTiming` + `RecordSuccess`) was defined but never called. Added `Adaptive.RecordResult(tool, 0, errVal == nil)` to `LogToolCall` so runtime success/failure is tracked per tool. Previously `agent_analyze` always showed empty `timing_stats: {}` and `success_rates: {}`.
- **Adaptive engine: `rebuildSequences` merge corrupts `Count` and never updates `Freq`** — When merging duplicate (word, command) entries, Count was overwritten instead of accumulated, and Freq (success ratio) was set at creation and never recalculated on merge. Added internal `SuccessCount`/`FailCount` fields to `SequenceExample`, fixed merge to accumulate correctly and recalculate `Freq`.

### Added

- **Chain integration tests** — 7 tests build-tagged `//go:build integration` that start the mcp-server binary and validate chain tool end-to-end via stdio MCP protocol. Covers: simple steps, capture, loop, if/else branching, unknown tool error, timeout, and structured data output. Run with `go test -tags=integration -v -count=1 -timeout 120s ./internal/actions/ -run 'TestChain_'`.
- **CI: `chain-tests` job** — runs chain integration tests after lint in `.github/workflows/ci.yml`.
- **README badges** — Go version, release, CI status, Windows, MCP, last commit, PRs welcome.

## [0.2.31] - 2026-07-05

### Changed

- **All 60+ query/result handlers now return structured data instead of "ok"** — Every handler that returns meaningful data (get_volume, get_battery, list_windows, get_system_info, get_uptime, get_clipboard, get_pixel_color, list_displays, get_disk_usage, get_network_info, ocr, find_image, find_all_images, list_audio_devices, list_processes, uia_find, uia_get_text, memory_get/search/list, template_find/list/store/forget, training_*, onnx_*, datalog_status, chain, launch_and_wait, write_file, delete_file, find_files, list_directory, set/get_working_directory, bridge_debug, set_config, task_begin, and more) now return their structured JSON data instead of the placeholder `"ok"` text. The SDK auto-populates tool results from structured output when no explicit `TextContent` is set, so tools like `get_screen_size` now show `{"width":1920,"height":1080}` instead of `"ok"`.
- **`get_screen_size`, `get_cursor_position`** — same fix applied (noticed during audit).
- **`chain` tool schema — no longer rejected by Gemini** — `IfConfig.Then`/`Else` and `LoopConfig.Steps` changed from `[]any` to `[]ChainStep`, which produced `items: true` and `type: ["null", "array"]` in the auto-generated JSON schema (both rejected by Gemini's MCP schema validator). The chain tool now uses a manually crafted `InputSchema` that avoids the recursive type cycle in `jsonschema-go` and produces clean schema output.

## [0.2.30] - 2026-07-03

### Added

- **Windows icon embedded in mcp-server.exe** — app.ico compiled into a COFF `.syso` resource via `rsrc` (`github.com/akavel/rsrc`), so the binary shows a custom icon in File Explorer, taskbar, and title bar. Icon sizes: 16, 32, 48, 64, 256px with SVG source in `icons/app.svg`.
- **`icons/` directory** — app.svg source, generated PNGs at 5 sizes, app.ico multi-res icon, app.rc resource script.
- **`scripts/gen-icons.ps1`** — PowerShell script that runs `rsrc` to compile `app.ico` into `cmd/mcp-server/rsrc_windows.syso`, auto-installing `rsrc` if missing. Called from `build.ps1`, `lint.ps1`, and CI release workflow.

### Security

- **Bump `golang.org/x/net` v0.54.0 → v0.55.0** — dependency update from Dependabot patching a security vulnerability in the `go_modules` group.
- **`.github/dependabot.yml`** — `package-ecosystem` was empty `""`; set to `"gomod"` so Dependabot properly scans `go.mod` for vulnerabilities.

### Changed

- **`write_file` — `overwrite` is now optional** — changed `Overwrite bool` to `*Overwrite *bool` with `omitempty`. No longer required in tool schema, defaults to `false` when omitted.
- **`get_file_info` — returns metadata instead of "ok"** — handler now returns actual file info JSON (`name`, `size`, `is_dir`, `mod_time`, `mode`).

## [0.2.29] - 2026-07-02

### Added

- **File System tools** — 11 new tools for navigating and manipulating files:
  - `list_directory` — list files/dirs at path (name, size, is_dir, mod_time, mode)
  - `read_file` — read file with automatic type detection + native format parsing
  - `write_file` — write/overwrite files with format-aware creation and editing
  - `find_files` — recursive glob search (e.g. `*.go`, `**/*.md`)
  - `copy_file` — copy file or directory (recursive)
  - `move_file` — move/rename file or directory
  - `delete_file` — delete file/dir to Recycle Bin via SHFileOperationW (not permanent)
  - `create_directory` — mkdir -p (recursive directory creation)
  - `get_file_info` — file/dir metadata (size, mod_time, is_dir, mode)
  - `set_working_directory` — set working directory for relative path resolution
  - `get_working_directory` — get current working directory
- **Format-aware `read_file`** — auto-detects mime type by magic bytes + extension; parses:
  - Plain text (txt, json, csv, yaml, toml, md, source code, configs, etc.) via `io.ReadAll`
  - `.docx` via `nguyenthenguyen/docx` library, extracting text from `<w:t>` XML elements
  - `.xlsx` via `xuri/excelize/v2`, all sheets rendered as TSV
  - `.pdf` via `ledongthuc/pdf`, all pages with separators
  - Images (png, jpg, gif, bmp, tiff, webp) via native WinRT COM OCR (`ocrNative`)
  - Pagination: `page` and `page_size` params (default 8000 chars), returns page/totalPages/truncated
- **Format-aware `write_file`** — detects target extension and creates/edits:
  - Plain text — raw write via `os.WriteFile`
  - `.docx` — new file creates from scratch (ZIP+XML); overwrite preserves existing headers/footers/images by swapping `<w:body>` content, writes to temp + rename
  - `.xlsx` — new file via `excelize.NewFile()`; overwrite opens existing, repopulates cells from TSV content
  - `.pdf` — new file creates from text via `go-pdf/fpdf`; overwrite tries `pdfcpu.FillFormFile` with JSON form field data, falls back to `createPdf`
- **File verification** — `FilePreCheck`/`FilePostVerify` wired into all 5 action file tool handlers (write, copy, move, delete, create_directory) using `ExpConfig` / `VerifyArgs` pattern from v0.2.28
- **Recycle Bin delete** — `delete_file` uses `SHFileOperationW` with `FOF_ALLOWUNDO` to move items to the Recycle Bin instead of permanent `os.RemoveAll`
- **Working directory** — all file tools resolve relative paths against a configurable working directory (defaults to process CWD, changeable via `set_working_directory`)
- **New dependencies** — `nguyenthenguyen/docx`, `ledongthuc/pdf`, `xuri/excelize/v2`, `go-pdf/fpdf`, `pdfcpu/pdfcpu` for native document parsing and creation

### Changed

- `internal/actions/filesystem.go` — Enhanced `ReadFile` with format dispatch + pagination; enhanced `WriteFile` with format-aware creation/editing (docx, xlsx, pdf); file verification helpers; Recycle Bin via SHFileOperationW; working directory support.
- `internal/server/server.go` — Updated `ReadFileArgs{Path, Page, PageSize}`, updated `write_file` tool description; file verification handlers.
- `internal/actions/chain.go` — Updated `chainReadFile` with page/page_size params.
- `VERSION` — bumped to 0.2.29.

### Tool Count

Now at 131 total MCP tools (+11).

## [0.2.28] - 2026-07-02

### Added

- **Auto-verify on 5 remaining high-value tools** — `open_url`, `launch_app`, `find_text_and_click`, `select_all_and_type`, `click_menu_item` now support `auto_verify` and `expected` parameters with OCR-based post-action verification, matching the existing 6 tools.
- **`TrainingCatLaunch` category** — New `"launch"` training category for app launch snapshots.
- **Pre-action validation (`pre_expected`)** — All 11 verification-enabled tools now accept `pre_expected` with the same `ExpConfig` shape (text/not_text/change). Runs OCR before the action and fails fast if precondition not met — action is never executed.
- **`VerifyArgs` embeddable struct** — Replaced duplicate `AutoVerify`/`Expected`/`PreExpected` fields across all 11 arg structs with a single `VerifyArgs` embed, reducing 22 lines of redundancy.
- **`preVerifyCheck` helper** — Common pre-verify logic extracted to server.go.
- **Region-of-typing OCR** — `type`, `type_and_submit`, `select_all_and_type` now capture cursor position (`GetCursorPosition`) before typing and restrict verification OCR to `SmartRegionAround(cursor, 400px)` instead of full-screen scan.
- **Window-aware OCR for `click_menu_item`** — Verification scans only within the target window bounds (found by `FindWindowByTitle` + `GetWindowState`) instead of full screen.
- **Coordinate-reuse for `find_text_and_click`** — `FindTextAndClick` now returns click coordinates `(int32, int32, error)`. The handler uses `SmartRegionAround(click_pos, 400px)` for post-verify instead of full-screen OCR. Pre-verify uses the specified search region.

### Changed

- `internal/actions/chained.go` — `FindTextAndClick` signature changed from `error` → `(int32, int32, error)`. Callers updated: `server.go`, `chain.go`, `cmd/benchmark/main.go`.
- `internal/actions/training.go` — Added `TrainingCatLaunch`.
- `internal/server/server.go` — All 11 verification handlers updated with cursor-aware region (type tools), window-bounds region (click_menu_item), coordinate-reuse region (find_text_and_click), and pre-verify checks. `VerifyArgs` struct replaces repetitive fields.

## [0.2.27] - 2026-06-30

### Added

- **`find_image` / `find_all_images` ONNX + OCR fallback** — When NCC template matching fails (no match, degenerate template), both tools now cascade through ONNX YOLO object detection → Windows OCR. `find_image` returns the highest-confidence ONNX element (or first OCR word if ONNX is empty). `find_all_images` returns all ONNX elements + all OCR words. The fallback captures a fresh screenshot if no screenB64 was provided (`ensureScreenB64`), making it robust against degenerate templates.
- **`findImageONNXFallback` / `findAllONNXFallback` helpers** — Extracted fallback logic with full cascade (ONNX → OCR) and screen capture self-healing.
- **`ensureScreenB64` helper** — Captures screen on demand when the passed-in `screenB64` is empty, fixing the edge case where degenerate templates bypass screen decode.
- **`ocr_languages` tool** — New `OcrLanguages()` function in `ocr.go:145` queries WinRT COM (`IOcrEngineStatics.get_AvailableRecognizerLanguages`) and returns every installed OCR language with tag, display name, and native name.
- **Middle mouse button** — `Click` now supports `button: "middle"` via `mouseEventMiddleDown`/`mouseEventMiddleUp` (0x0020/0x0040).
- **Horizontal scroll** — `Scroll` now takes `horizontal bool` parameter; uses `mouseEventHWheel` (0x1000) flag when horizontal. `chainScroll` passes it through from args.

### Changed

- `internal/actions/template.go` — `FindImage` and `FindAllImages` restructured: degenerate templates (zero-dim, no-variance) skip NCC entirely and go straight to ONNX+OCR fallback. Template-larger-than-screen still errors (unrecoverable). `FindAllImages` added as new NCC implementation with non-maximum suppression (overlap >50% suppressed).
- `internal/actions/window_ext.go` — `GetWindowState` now returns `fullscreen` boolean field. Added `isFullscreen()` helper that checks `MonitorFromWindow` + `GetMonitorInfoW` + `WS_CAPTION` style.
- `internal/actions/window_ext.go` — Added `monitorFromWindow` proc, `MONITORINFO` struct, `WS_CAPTION` constant.
- `docs/tools.md` → `docs/reference/tools.md` — auto-regenerated (2 new tools, 120 total)
- **Doc audit & reorganization** — Audited 20 files across `docs/`, `dev/`, `.github/instructions/`. Deleted `docs/todo.md` (dead) and `dev/models-setup.md` (duplicate of `docs/models-setup.md`). Deduplicated privacy table, accessibility block, inline tool listing (80 lines), and inline release cycle (8 steps) into cross-references. Reorganized flat `docs/` into subdirs: `adr/`, `reference/`, `guides/`, `meta/`. Moved 15 files to new locations. Updated all cross-references in README.md and all 18 docs files.
- **CI/CD path updates** — `ci.yml` references `docs/tools.md` → `docs/reference/tools.md`; `release.yml` references `docs/CHANGELOG.md` → `docs/meta/CHANGELOG.md`; `scripts/gen-tools-doc.go` writes to `docs/reference/tools.md`; `scripts/push-and-release.ps1` references `docs/meta/CHANGELOG.md`.
- **`scripts/gen-tools-doc.go`** — `MkdirAll("docs")` → `MkdirAll("docs/reference")`, output path changed from `docs/tools.md` to `docs/reference/tools.md`.
- **New reference docs** — `docs/reference/windows-dll-ref.md` (every syscall proc, DLL, COM interface used), `docs/reference/uipi.md` (UIPI elevation detection logic + call sites), `docs/reference/com-patterns.md` (COM/WinRT patterns: vtable dispatch, async polling, HSTRING/BSTR lifecycle, threading model, UIA tree traversal)
- **Cross-reference updates** — `README.md`, `codebase-map.md`, `windows-dll-ref.md` all updated with links to the 3 new reference docs
- **`docs/reference/windows-dll-ref.md`** — Fixed inaccurate advapi32.dll proc names (removed `CheckTokenMembership`/`AllocateAndInitializeSid`/`FreeSid` that were never used in `uipi.go`)
- **`docs/reference/vtable-verification.md`** — New doc covering COM vtable stability guarantees, SDK header verification procedure, CI/CD plan, and complete test table (13 tests, 16 unique vtbl indices, all verified)

### Fixed

- **`find_image` / `find_all_images` no longer error on degenerate templates** — Previously a constant-color or 0×0 template returned an unrecoverable error. Now falls through to ONNX + OCR object/text detection.

### VTable Verification System (vtable hardening)

- **36 vtblMethod() call sites annotated** — Every COM/WinRT vtable dispatch in `uia_com.go`, `ocr_com.go`, `ocr.go`, `winrt.go` tagged with inline `// N = MethodName (verified 2026-06-30, Win11 26200)` and block-level `// Verified YYYY-MM-DD — WinBuild SDK` headers above each interface definition.
- **`internal/actions/vtable_test.go`** — 13 smoke tests with build tag `//go:build vtable && windows`. Covers all 16 unique vtbl indices used in production: UIA GetRootElement (vtbl 5), BoundingRect (43), FindFirst (5, 21), Conditions (21, 23, 25), GetPropValue (10), GetCurrentPattern (16), ValuePattern/Invoke/Array (3, 4, 6, 21), OCR GetLanguages (7, 6), OCR TryCreate (10), OCR TryCreateFromLanguage (9), StorageFile async pipeline (6, 8, 14, 0, 7), HSTRING round-trip, vtblMethod nil guard. All pass (~8s).
- **`scripts/verify-vtable-docs.go`** — Parses all 36 vtblMethod() call sites from source, cross-references each unique index against `com-patterns.md` and `vtable-verification.md` doc tables, checks test coverage via `// vtbl:` annotations. Run `go run ./scripts/verify-vtable-docs.go`. Confirms all 16 unique indices are documented and tested.
- **CI: `vtable-check` job** — `.github/workflows/ci.yml` runs `go test -v -tags=vtable ./internal/actions/ -run 'TestVtable'` after build, plus `go run ./scripts/verify-vtable-docs.go` to catch doc drift.
- **`go vet` passes** — zero warnings across all vtable code.
- **`scripts/verify-iid-usage.go`** — New Go script that parses `winrt.go`, scans `internal/actions/` for IID references, categorizes each as `used` (referenced outside winrt.go), `internal` (only within winrt.go), or `unused`. Cross-references against the Status column in `com-patterns.md` docs. With `-update`, rewrites the Status column. Run: `go run ./scripts/verify-iid-usage.go` or `go run ./scripts/verify-iid-usage.go -update`.
- **`scripts/discover-winrt-iids.ps1` -UpdateDocs mode** — Added `-UpdateDocs` switch that auto-generates the 51-entry IID table in `docs/reference/com-patterns.md` from discovered values, replacing the content between `<!-- IID_TABLE_START -->` / `<!-- IID_TABLE_END -->` markers. Includes fallback pass for statics discovered under flat keys. Now also calls `verify-iid-usage.go -update` to set correct usage statuses. CI runs `-UpdateDocs` then `go run verify-iid-usage.go` then `git diff` to catch drift.
- **`ci.yml` vtable-check job** — Added `Verify WinRT IID docs are in sync with source` step: runs `powershell -File scripts\discover-winrt-iids.ps1 -UpdateDocs` + `go run ./scripts/verify-iid-usage.go`, fails if `docs/reference/com-patterns.md` has uncommitted changes.
- **`docs/reference/scripts.md`** — New reference doc cataloging all 8 scripts with purpose, invocation, uniqueness, and cross-references to docs and CI.
- **`docs/reference/com-patterns.md`** — VTable dispatch section (§2) updated with `IUnknown base` column in all index tables. IID table wrapped in `<!-- IID_TABLE_START/END -->` markers with new **Status** column (used/internal/unused). Inline `-UpdateDocs` flag in both "run the script" instructions. 
- **`docs/reference/vtable-verification.md`** — Expanded from "proposed plan" to actual test table with `vtbl Indices` + `Verified` columns, **Performance note** documenting why `FindAll` on desktop root is slow by design, and CI/CD integration reference.
- **`uia.go` `andCondition` dead code fixed** — `CreateAndCondition` (vtbl 25) was called from `buildCondition` but not tested; now exercised by `TestVtable_IUIAutomation_Conditions`.
- **`internal/actions/uia_ctrltype_test.go`** — 3 tests validating the UIA ControlTypeId map covers all 41 constants (50000–50040), no duplicates, no out-of-range values, and unknown names return nil.
- **Cross-references updated** — `README.md`, `docs/reference/codebase-map.md`, `docs/reference/windows-dll-ref.md`, `docs/ci-cd-pipeline.md` all linked to new vtable docs.
- **VTable indices are hard-won** — Every index was researched against SDK headers (UIAutomationClient.h, windows.data.h, windows.storage.h) and verified at runtime on Windows 11 (build 26200). These are the most fragile part of the codebase: an incorrect vtbl index silently calls the wrong method or crashes. The verification script and test suite exist because even a single off-by-one error can take days to diagnose. Microsoft freezes these indices for published interfaces, but when upgrading to a new Windows build, run `go test -tags=vtable` and `go run ./scripts/verify-vtable-docs.go` before shipping.

## [0.2.26] - 2026-06-30

### Fixed

- **Chain tool: `computer_use_*` prefix normalization** — `execTool` now strips the `computer_use_` prefix from tool names before dispatch lookup, so all `computer_use_*` tools (click, type, key_press, ocr, get_screen_size, etc.) work inside chain steps.
- **Chain `success` aggregation** — `result.Success` now initializes to `true` before checking step results, fixing false-negative `success: false` when all steps pass.
- **Keyboard modifier key case sensitivity** — `KeyPress` normalizes modifier key names to uppercase before `vkModMap`/`vkSpecialMap` lookups, so `"Ctrl"`, `"ctrl"`, `"CTRL"` all correctly match instead of being silently skipped.
- **Window focus reliability** — `FocusWindow` uses `AttachThreadInput` to attach to the target window's input thread before `SetForegroundWindow`, working around Windows focus-stealing restrictions for background automation processes.

### Changed files

- `internal/actions/chain.go` — `execTool` adds `strings.TrimPrefix(step.Tool, "computer_use_")` at line 312; `ExecuteChain` initializes `result.Success = true` before failure loop at line 200
- `internal/actions/keyboard.go` — `KeyPress` normalizes keys via `strings.ToUpper` before modifier/special key lookups
- `internal/actions/window.go` — `FocusWindow` uses `AttachThreadInput` with `GetWindowThreadProcessId`/`GetCurrentThreadId`
- `internal/actions/system.go` — added `getCurrentThreadId` (kernel32) and `attachThreadInput` (user32) proc declarations

## [0.2.25] - 2026-06-30

### Fixed

- **Coordinate extraction: case-insensitive key matching** — `getIntArg` now falls back to case-insensitive key lookup when an exact match fails, fixing coordinate extraction for `click` and `move_mouse` tools which store their args with capitalized `X`/`Y` (from Go `json.Marshal` of struct fields) while the code searched for lowercase `x`/`y`. This caused all click coordinate data to be silently ignored by `TrainFromDatalog`, meaning the `__learned__` aggregate and per-token coordinate index never accumulated click coordinates.

### Changed files

- `internal/actions/adaptive.go` — `getIntArg` now does case-insensitive key lookup via `strings.EqualFold` as fallback

## [0.2.24] - 2026-06-30

### Changed

- **Adaptive engine: `__learned__` aggregate built from persisted training data** — `TrainFromDatalog` now aggregates all coordinate samples per tool into the `__learned__` key in `coordIndex`, so coordinate predictions survive server restart. Combined with the v0.2.23 fallback in `predictCoord`, `agent_suggest` now returns coordinate predictions for `click`/`hover`/`move_mouse` using the aggregate average from all training data, even before any runtime samples accumulate.

### Changed files

- `internal/actions/adaptive.go` — `TrainFromDatalog` stores aggregated coords under `__learned__` key per tool

## [0.2.23] - 2026-06-30

### Changed

- **Adaptive engine: coord prediction fallback to `__learned__` aggregate** — `predictCoord` now falls back to the runtime-learned `__learned__` aggregate coordinate when per-token samples in `coordIndex` are below the threshold of 3. This ensures `agent_suggest` returns coordinate predictions for `click`/`hover`/`move_mouse` even when the specific OCR tokens haven't accumulated 3+ samples yet — the aggregate `__learned__` accumulates across all invocations of the same tool.

### Changed files

- `internal/actions/adaptive.go` — `predictCoord` now checks `__learned__` in `toolMap` as fallback when `tCount < 3`, returning the aggregate coord instead of `nil`

## [0.2.22] - 2026-06-30

### Changed

- **Adaptive engine: real timing_stats and success_rates** — `RecordResult` is now called from every action tool's defer with a captured start time, so `timing_stats` (mean, stddev, count, min, max) and `success_rates` per tool populate correctly. Previously `RecordCommand` was defined but never called, leaving both maps permanently empty.

### Changed files

- `internal/actions/datalog.go` — removed `Adaptive.RecordResult(tool, 0, ...)` from `LogToolCall` (moved to per-action defer with real timing)
- `internal/actions/chained.go` — added `start` capture + `RecordResult` to `LaunchAndWait` and `Hover`
- `internal/actions/keyboard.go` — added `start` capture + `RecordResult` to `KeyDown`, `KeyUp`, `KeyPress`, `TypeText`
- `internal/actions/mouse.go` — added `start` capture + `RecordResult` to `Click`, `MoveMouse`, `Scroll`, `Drag`
- `internal/actions/window.go` — added `time` import + `start` capture + `RecordResult` to `FocusWindow`

## [0.2.21] - 2026-06-30

### Changed

- **LogToolCall coverage: all 11 MCP action tools now instrumented** — Added `LogToolCall` to `key_down`, `key_up`, `focus_window`, and `launch_and_wait`, completing adaptive engine training pair coverage for every non-query action tool. Previously 4 tools produced commands without OCR context pairs, leaving gaps in the training index.

### Changed files

- `internal/actions/keyboard.go` — added `LogToolCall("key_down", ...)` to `KeyDown`, `LogToolCall("key_up", ...)` to `KeyUp`
- `internal/actions/window.go` — added `LogToolCall("focus_window", ...)` to `FocusWindow`
- `internal/actions/chained.go` — added `LogToolCall("launch_and_wait", ...)` to `LaunchAndWait`

## [0.2.20] - 2026-06-30

### Changed

- **Adaptive engine: OCR bridge auto-complete in `LogToolCall`** — `LogToolCall` now synchronously captures OCR after setting a pending training pair, ensuring every action produces a complete `(ocr_before, tool, ocr_after)` pair. Previously pairs only completed when the next explicit `OCRScreen()` call happened, causing all training sequences to cluster under "click". Also added `LogToolCall` to `Hover` and `MoveMouse` which were missing it entirely.

### Changed files

- `internal/actions/datalog.go` — `LogToolCall` auto-captures OCR after pending pair set
- `internal/actions/chained.go` — added `LogToolCall("hover", ...)` to `Hover`
- `internal/actions/mouse.go` — added `LogToolCall("move_mouse", ...)` to `MoveMouse`

## [0.2.19] - 2026-06-30

### Changed

- **Keylogger rewrite: hooks → polling** — Replaced `WH_MOUSE_LL` + `WH_KEYBOARD_LL` low-level hooks with `GetAsyncKeyState` polling loop (50ms ticker). Eliminates the system-wide input lag caused by the Go hook callback trampoline on every mouse event. The polling loop runs in a goroutine with no locked OS thread and no Windows message loop. Trade-off: scroll wheel events no longer detectable (acceptable cost for eliminating system-wide input lag).

### Fixed

- **CI lint failure — stale tools.md & uncategorized tools** — `scripts/gen-tools-doc.go` was missing category entries for 4 tools (`bridge_debug`, `introspection_analyze`, `task_begin`, `task_end`), causing them to fall under "Uncategorized" and `docs/tools.md` to show 114 instead of 118 tools. The lint check (regenerate + diff) then failed, skipping the build job. Added `"Introspection & Debugging"` category, removed stale `docs2/` staging output from the script, and regenerated `docs/tools.md`.

- **`yolo_dataset` location inconsistency** — removed stale `yolo_dataset/` from repo root (empty train/val dirs). `export_yolo_dataset` now defaults to `%APPDATA%\go-mcp-computer-use\yolo_dataset\` when `output_dir` is omitted. Added `yolo_dataset/` to `.gitignore` to prevent future repo root drift.

## [0.2.18] - 2026-06-29

### Added

- **Post-Task Introspection Engine** (`internal/actions/introspection.go`) — three new MCP tools for task-aware self-improvement:
  - `task_begin` — marks task start with description, timestamps
  - `task_end` — closes task, mines insights from command_log between start/end: slowest tools, most failed tools, OCR stats, repeated command patterns, and improvement suggestions
  - `introspection_analyze` — browse completed task history with full insight data
  - Uses existing `command_log` + `ocr_log` tables — no new logging infra needed
  - `task_log` table added to datalog DB

### Changed

- `datalog_status` now reports `task_count` in stats

## [0.2.17] - 2026-06-29

### Fixed

- **OCR→Training bridge window** — `bridgeWindow` increased from 3s to 30s. The OCR→AI→MCP→Click round trip regularly exceeded the original 3-second window, preventing training pair creation. Debugged via new `bridgeBufferSize()` and `BridgeDebugInfo()` diagnostic functions exposed through the `bridge_debug` MCP tool.

### Added

- **`bridge_debug` MCP tool** — debug the OCR→command bridge state, showing recent OCR buffer contents, pending command, and timing info.

## [0.2.16] - 2026-06-29

### Added

- **Adaptive Engine** (`internal/actions/adaptive.go`) — pure Go statistical ML system with three components:
  - **TimingTracker** — rolling-window (N=100) per-tool statistics: mean, stddev, min, max. Auto-suggests adaptive delays based on historical execution time plus success-rate multiplier (1.5× by default, 3× when success rate < 50%).
  - **SuccessTracker** — per-tool success/failure ratios. Queried on every `SuggestDelay()` call to adjust timeouts.
  - **SequencePredictor** — TF-IDF-style word index from `training_pairs`. Given OCR text, tokenizes and scores each word→command mapping by historical success frequency. Returns ranked predictions with confidence (0.0–1.0) and sample size.

- **MCP Resources (5)** — auto-exposed to the AI client, read on every session context:
  - `datalog://stats` — current row counts for all four datalog tables
  - `datalog://commands` — 20 most recent command log entries
  - `datalog://ocr` — 10 most recent OCR snapshots
  - `datalog://pairs` — 20 most recent training pairs
  - `adaptive://analysis` — full adaptive engine analysis (timing stats, success rates, learned sequences)

- **Agent MCP Tools (3)** — AI-queryable loop for context-aware decisions:
  - `agent_analyze` — returns full timing stats, success rates, and top learned sequences for AI decision-making
  - `agent_suggest` — given OCR screen text, predicts the best next command ranked by confidence
  - `agent_train` — rebuilds the word→command index from current `training_pairs` table

- **Auto training pair generation** — passive OCR bridge creates triple (ocr_before, command, ocr_after) without slowing commands:
  - Ring buffer of last 5 OCR snapshots with timestamps
  - Every command auto-pairs with most recent OCR (within 3s window) as `ocr_before`
  - Next OCR snapshot completes as `ocr_after`
  - Command stored as `{"tool":"name","args":"..."}` JSON for robust parsing

### Fixed

- **`datalog_query` table name mismatch** — switch-case expected short names (`"commands"`, `"ocr"`, `"chains"`, `"pairs"`) but the handler passed raw table names. Now accepts both forms as aliases.
- **`TrainFromDatalog` JSON parsing** — robust `extractToolFromJSON` helper handles both JSON `{"tool":"..."}` and plain string command values.

### Changed

- **Tool count** — 111 → 114
- **VERSION** — bumped 0.2.15 → 0.2.16
- **gen-tools-doc.go** — added "Adaptive Agent" category
- **LogCommand** — now releases SQLite lock before OCR bridge to avoid deadlock with LogOCRSnapshot (no cross-lock ordering)
- **LogTrainingPair** — `Command` field stores structured `{"tool":"name","args":"..."}` JSON instead of raw args string

### Documentation

- **docs/tools.md** — regenerated with 114 tools across categories including "Adaptive Agent"

## [0.2.15] - 2026-06-29

### Added

- **Data logging database** (`internal/actions/datalog.go`) — new SQLite DB at `%APPDATA%/go-mcp-computer-use/datalog/datalog.db` with four tables:
  - `command_log` — every chain/tool execution with args, success, duration, error text
  - `chain_log` — full chain executions with step counts, success/fail breakdown, chain JSON
  - `ocr_log` — OCR snapshots with full OCR text, word count, linked screenshot image path
  - `training_pairs` — OCR-before + command + OCR-after triples for ML sequence learning

- **Automatic logging hooks** — chains, individual commands, and OCR calls are logged automatically via goroutines with no performance impact on the main execution path.

- **Three new MCP tools:**
  - `datalog_query` — query any table (commands, chains, ocr, pairs) with filters (source, tool, success), returns rows as JSON
  - `datalog_export` — export training pairs as JSON array for downstream ML training pipelines
  - `datalog_status` — get row counts for all four tables

### Changed

- **VERSION** — bumped 0.2.14 → 0.2.15
- **Tool count** — 108 → 111

## [0.2.14] - 2026-06-29

### Added

- **`NormalizedElement` coordinate system** — element positions stored as window-relative 0.0–1.0 fractions via `WindowNormalizer` in `internal/actions/dpi.go`. Layout-independent across screen resolutions and multi-monitor. Includes `GetDPIScaleForWindow`, `Normalize`/`Denormalize` helpers, and `ProportionalRegion` for computing screen-absolute OCR crops as a percentage of the active window.

- **`OCRProportionalWindowRegion`** — new OCR function in `ocr.go` that takes a window handle + proportional fractions, eliminating hardcoded pixel crops.

- **Auto-expand tiny OCR regions** — `FindTextAndClick` now detects crops <300px in any dimension and falls back to a generous 5%–95% of the active window. Prevents "Desktop not found" failures on small fixed-pixel regions.

- **Window context in ONNX detection** — `DetectionOutput` carries `WindowTitle` and `Normalized []NormalizedElement` alongside absolute coordinates. Computed per-active-window during inference.

- **Training schema migration** — `training_samples` table gains `window_rect TEXT` and `normalized_coords TEXT` columns. `saveTrainingSampleDirect` accepts and persists both normalized coords and window rect JSON.

### Fixed

- **`NormalizeElement` missing Class/Confidence copy** — `WindowNormalizer.NormalizeElement` returned a `NormalizedElement` with zeroed `Class` and `Confidence` fields. Exposed by round-trip test (`TestNormalizeElementRoundTrip`). Now copies both fields before returning.

### Changed

- **Watcher cache** — `CachedDetection` includes `Normalized` elements alongside absolute `Elements`. Training samples from watcher snaps now carry window rect context.
- **VERSION** — bumped 0.2.13 → 0.2.14

### Tests

- **Coordinate system tests** — `dpi_test.go` with 6 tests covering: normalize/denormalize round-trip, coordinate bounds (corners, center, size), proportional region math, `NormalizeElement` class/confidence round-trip, and zero-size window edge case.

## [0.2.13] - 2026-06-29

### Fixed

- **ONNX detection timeout (65s → 599ms)** — root cause was not DLL incompatibility but performance:
  - `parseYOLOOutput` passed all 8400 raw detections through NMS at O(n²) = ~15M iterations
  - `MemoryStoreDetectionElements` called `MemorySet` 5507 times — each a separate SQLite INSERT with global mutex lock
  - Fixed: `parseYOLOOutput` now applies confidence threshold early (0.25), pre-filtering to ~50 boxes before NMS
  - Fixed: `MemoryStoreDetectionElements` rewritten with batched SQLite inserts in a single transaction, capped at 200 elements

### Changed

- **ONNX Runtime DLL updated** — v1.20.1 → v1.26.0 to support opset 22 (required by yolo11n.onnx). Limited opset support warning is non-fatal.

## [0.2.12] - 2026-06-29

### Fixed

- **Release binaries crash with STATUS_ILLEGAL_INSTRUCTION** — Zig cc on GHA runners defaults to `-march=native`, generating CPU-specific instructions incompatible with older machines (Pentium Gold G5400). Pinned `-mcpu=x86_64_v2` in `CGO_CFLAGS` so binaries run on any x86-64 CPU.
- **CGO_LDFLAGS also needs `-mcpu=x86_64_v2`** — `actions/setup-go@v5` overrides `CGO_LDFLAGS` with `-O2 -g`, dropping the CPU baseline. Both `CGO_CFLAGS` (compile) and `CGO_LDFLAGS` (link) now pin `-mcpu=x86_64_v2`.

### Changed

- **`scripts/build.ps1`** — added `CGO_CFLAGS` with `-mcpu=x86_64_v2` baseline for portable builds
- **`.github/workflows/release.yml`** — same CPU baseline pin in both `CGO_CFLAGS` and `CGO_LDFLAGS`, plus `-fno-sanitize=all` and `-Wno-error`

## [0.2.11] - 2026-06-29

### Added

- **`scripts/gen-tools-doc.go`** — parses `internal/server/server.go` for `mcp.AddTool` calls, generates `docs/tools.md` with categorized 108-tool listing. CI validates freshness on every push/PR.
- **`scripts/push-and-release.ps1`** — one-shot auto-release: reads VERSION, commits with changelog body, tags, pushes, waits for release workflow, downloads binary, replaces `mcp-server.exe`, restarts OpenCode Desktop as admin.
- **`docs/tools.md`** — auto-generated tool reference doc (never stale).
- **`docs/security.md`**, **`docs/configuration.md`**, **`docs/build.md`**, **`docs/architecture.md`**, **`docs/accessibility.md`** — split from monolithic README.
- **Weekly module maintenance** — `.github/workflows/mod-maintenance.yml` runs `go get -u ./...` + auto-PR every Monday.
- **CI: `go mod tidy` validation** — fails if `go.mod`/`go.sum` drifts from tidy state.

### Changed

- **README.md** — collapsed 383→92 lines, links to focused docs/ split.
- **Root docs moved** — `plan.md`, `todo.md`, `backlog.md`, `known-issues.md`, `CHANGELOG.md` relocated to `docs/`.
- **CGO mandatory** — removed all `-NoCGO` flags, pure-Go fallback paths, and optional-CGO language across 9 files. `release.yml` now produces a single `mcp-server.exe` (CGO+Zig).
- **Release workflow** — single binary output, no `-cgo` suffix variant.
- **`scripts/build.ps1`** — removed `-NoCGO` switch, always requires Zig cc.

### Documentation

- **README split** — large sections moved into focused docs for maintainability.
- **All NoCGO references removed** — across `plan.md`, `adr-002`, `comparison-vs-alternatives.md (formerly comparison-vs-windows-recall.md)`, `ci-cd-pipeline.md`, `build.md`, `README.md`.

## [0.2.10] - 2026-06-29

### Documentation

- **Systematic doc audit** — fixed 90 stale statements across 12 docs: tool counts (103→108 restored from actual registrations), version refs, CGO/dependency claims, category counts, missing tool listings, completed Slice 4 checkboxes, stale future-tool lists
- **Architecture guide** — added Part 6 to computer-use-guide: layered agent stack (LLM→MCP→Controller→Perception→Memory→World), ML vision + spatial memory, division of responsibilities, convergence of LLM+MCP+ML
- **Source fix** — server.go tool count hardcode corrected 103→108 to match actual registrations
- **Config auto-start** — watcher_auto_start config created on dev machine

### Changed

- **VERSION** — bumped 0.2.9 → 0.2.10

## [0.2.9] - 2026-06-29

### Added

- **`scripts/build.ps1`** — unified build script with `-UseZig` flag for CGO-enabled builds
- **CI/CD: CGO + Zig cc build pipeline** — CI now runs two jobs: no-CGO lint+build and CGO+Zig build. Release workflow produces both `mcp-server.exe` (no CGO) and `mcp-server-cgo.exe` (with ONNX support).
- **Zig 0.16.0 support** — `scripts/install.ps1` updated to download Zig 0.16.0

### Documentation

- **README.md** — documented CGO requirements for ONNX tools with Zig cc build instructions
- **known-issues.md** — B13: ONNX tools require CGO (documented workaround)
- **Tool count docs updated** — all docs updated to 108 tools, stale CGO claims corrected

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
- **Tool count**: 103 → 108 (added `key_down`, `key_up`, `keylogger_start`, `keylogger_stop`, `keylogger_status`).

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
