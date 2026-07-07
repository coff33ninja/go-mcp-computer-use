<!-- Auto-generated from project docs. Run `go run ./scripts/gen-wiki.go` to regenerate. -->

# Changelog

[Home](Home.md) ← # Changelog

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


_... older versions omitted._


<!-- Generated by scripts/gen-wiki.go -->
