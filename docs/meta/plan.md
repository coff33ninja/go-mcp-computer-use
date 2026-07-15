# go-mcp-computer-use — Plan & Progress

## Goal

A closed-loop embodied agent for Windows — an MCP server in Go that exposes desktop computer use tools (screenshot, mouse, keyboard, window management, OCR, ONNX ML detection, memory store, data logging, adaptive engine) to AI agents via the Model Context Protocol. The system implements perception, action, memory, and self-improvement layers, trending toward a locally-hosted autonomous desktop agent.

## Architecture Layers

```
User Goal
     │
     ▼
Strategic Planner (LLM in AI client)
     │
     ▼
Skill Library (Macros) — NEXT SLICE
     │
     ├── Install Software
     ├── Configure Settings
     ├── Edit Document
     ├── Browse Website
     └── ...
     │
     ▼
Action Executor (MCP Server)
     │
     ├── Mouse • Keyboard • Vision (OCR/ONNX/UIA)
     ├── Window Management • Browser • Explorer
     ├── Chain Tool — sequential step execution
     └── Keylogger — input recording/replay
     │
     ▼
Verification & Feedback (OCR, ONNX, UIA)
     │
     ▼
Memory & Learning
     ├── Data Logging (commands, OCR, chains, training pairs)
     ├── Adaptive Engine (timing stats, success rates, sequence predictions)
     ├── SQLite Memory Store (facts, sequences, templates)
     └── Training Data Pipeline (screenshot + label export)
     │
     ▼
Post-Task Introspection — NEXT SLICE
     ├── What worked? What slowed me down?
     ├── Which macro was reused? Which element took too long?
     └── Generalize into reusable skills
```

## Personal Security System

Five-layer defense for protecting user data during AI-driven automation:

### Layer 1: Identity Awareness (who am I, what can I do)
- `is_admin` — check elevation status (affects what actions are safe)
- `get_username` / `get_user_sid` — identity for audit trails
- `get_user_groups` — permission groups (admin, user, guests)
- `list_tools` — available tools (respects `tool_denylist`)

### Layer 2: Input Sanitization (what gets captured)
- **OCR text redaction** — scan for PII before training DB write:
  Credit cards, SSNs, emails, passwords/auth tokens, phone numbers
- **Screenshot region masking** — blank detected sensitive areas
- **Clipboard sanitization** — clear clipboard after AI copies sensitive data

### Layer 3: Source Filtering (what gets captured from where)
- **App exclusion list** — `set_config excluded_apps: ["password-manager"]`
- **URL exclusion list** — `set_config excluded_urls: ["*.bank.com"]`
- **Private browsing detection** — skip capture in incognito mode
- **Window title patterns** — skip "Login", "Enter Password", "Two-Factor"

### Layer 4: Storage Security (what happens after capture)
- **Encryption at rest** — DPAPI or AES for stored screenshots
- **Access control** — auth token to access training data directory
- **Audit log** — who accessed what, when (SQLite table)

### Layer 5: Retention & Cleanup (data lifecycle)
- `retention_days` — auto-prune old samples ✅ done v0.2.37
- `training_cleanup_noise` — manual noise cleanup ✅
- **Auto-redact on retention** — re-scan old samples with updated filter patterns

### Data flow
```
Screenshot taken
  → [L3] App/URL exclusion check → skip if excluded
  → [L2] OCR text extracted → PII regex scan → redact matches
  → [L2] Screenshot region masking → blank detected areas
  → [L4] Encrypt before write to disk
  → Training DB + image file saved
```

### What's done vs. what's next
| Layer | Feature | Status |
|-------|---------|--------|
| L1 | `tool_denylist` | ✅ v0.2.37 |
| L5 | `retention_days` | ✅ v0.2.37 |
| L5 | `training_cleanup_noise` | ✅ manual |
| L1 | `is_admin` / `get_username` / `list_tools` | NEXT — P0 |
| L2 | `set_sensitive_content_filter` (PII regex) | NEXT — P1 |
| L3 | App/URL exclusion list | NEXT — P2 |
| L3 | Private browsing detection | NEXT — P3 |
| L4 | Encryption at rest | FAR — P4 |
| L4 | Audit log | FAR — P5 |

## Design Principles

- **No overlap in responsibility** — each layer does one job well
- **Stateless → stateful** — element memory replaces blind re-discovery
- **ML informs MCP, doesn't replace it** — perception feeds structure, not commands
- **Feedback loop** — every action is verified by perception before continuing
- **Stable planner/executor interface** — high-level skills decoupled from tool layer so vision models, LLMs, or backends can be swapped

## Current State: v0.2.38 — Desktop Awareness + Chain Integration

All tools registered in `internal/server/server.go`, auto-documented in [`docs/reference/tools.md`](../reference/tools.md). Adaptive engine now includes timing stats, success rates, coordinate prediction, and full OCR-bridge training pair coverage across all 11 action tools.

Chain system integrated with UIA: mouse steps auto-capture `element_at_point`, `verify_ui` and `if_uia` step types use UIA instead of OCR for structural verification and branching.

Desktop awareness layer is now complete:
- **Window state** — `get_window_state` returns full state: visible, minimized, maximized, fullscreen, foreground, z_order (0=topmost, higher=deeper behind), and bounding rect. AI can detect obscured windows and focus/restore before interacting.
- **Window stacking** — `z_order` field lets AI compare which windows are on top. If window A has z_order=3 and window B has z_order=12, A is above B.
- **Element discovery** — `uia_get_all_elements(handle)` dumps all child UI elements of a window (textboxes, address bars, title bars, buttons). `uia_find` searches by name/type. `find_ui_element` cascades memory→ONNX→OCR.
- **Element at point** — `uia_get_element_at_point(x, y)` reverse-looks up screen coordinates to identify the element under the cursor.
- **Content verification** — `wait_for_ui_element(handle, name, control_type)` polls UIA until element appears. Complements `wait_for_text` (OCR polling) for post-action validation.

See [`docs/reference/tools.md`](../reference/tools.md) for the full categorized tool listing and [`backlog.md`](backlog.md) for the roadmap.

## Completed Work

### v0.2.16 — Adaptive Engine + Data Logging
- **Adaptive Engine** (`internal/actions/adaptive.go`) — pure Go statistical ML: TimingTracker (rolling window), SuccessTracker (per-tool ratios), SequencePredictor (TF-IDF word→command index)
- **Data Logging** (`internal/actions/datalog.go`) — SQLite action/OCR/chain/pair logging with `datalog_query/status/export` MCP tools
- **MCP Resources (5)** — `datalog://stats/commands/ocr/pairs`, `adaptive://analysis`
- **Agent MCP Tools (3)** — `agent_analyze/suggest/train`
- **Auto training pair generation** — OCR bridge creates `(ocr_before, command, ocr_after)` triples
- **Bridge race fix** — bridge logic moved from goroutine to synchronous in `LogToolCall`

### v0.2.17 — Bridge Window Fix
- **`bridgeWindow`** increased 3s → 30s (OCR→AI→Click round trip exceeded 3s)
- **`bridge_debug` MCP tool** — inspect bridge state for debugging
- **First training pair created and indexed** via `agent_train`

### v0.2.20 — OCR Bridge Auto-Complete
- **`LogToolCall`** now synchronously captures OCR after setting a pending training pair.
- Every action produces a complete `(ocr_before, tool, ocr_after)` triple. Previously all training pairs clustered under "click".
- Added `LogToolCall` to `Hover` and `MoveMouse`.

### v0.2.21 — Full Action Coverage
- Added `LogToolCall` to `key_down`, `key_up`, `focus_window`, `launch_and_wait`.
- All 11 MCP action tools now produce training pairs.

### v0.2.22 — Real Timing Stats
- `RecordResult` called from every action tool's defer with real captured start time.
- `timing_stats` (mean, stddev, min, max, count) and `success_rates` now populate correctly.

### v0.2.23–24 — Coordinate Prediction
- **Coordinate learning** — `LearnFromCommand` stores per-tool coordinate aggregates. `TrainFromDatalog` persists and rebuilds the `__learned__` aggregate on restart.
- **`predictCoord`** — `agent_suggest` returns `coord: {x, y, confidence, samples}` for `click`/`hover`/`move_mouse` from aggregate training data.

### v0.2.25 — Case-Insensitive Coordinate Match
- `getIntArg` uses `strings.EqualFold` fallback when exact key match fails, fixing click coordinate extraction (Go struct marshaling produces capitalized `X`/`Y`).

### v0.2.28 — Action Verification System
- **Auto-verify on 5 more tools** — `open_url`, `launch_app`, `find_text_and_click`, `select_all_and_type`, `click_menu_item` support `auto_verify`/`expected`.
- **Pre-action validation** — `pre_expected` field on all 11 verification tools, fails fast before action execution.
- **Region-aware OCR** — type tools capture cursor position (`SmartRegionAround`), `click_menu_item` uses window bounds, `find_text_and_click` reuses click coordinates.
- **`VerifyArgs` embeddable struct** — eliminated 22 lines of field duplication across arg structs.

### v0.2.38 — Desktop Awareness + Chain Integration

- **`z_order` on `get_window_state`** — 0=topmost, higher=deeper in stack. Uses `GetDesktopWindow` + `GW_CHILD` to find topmost, walks `GW_HWNDNEXT` counting visible windows. AI compares z_order between handles to find true stacking.
- **`uia_get_all_elements(handle, max_results)`** — dumps all immediate child UI elements in a window via UIA `FindAll(TreeScope_Children, TrueCondition)`. One level deep — returns title bar, menu bar, content panes, toolbars, status bar without flooding thousands of nested DOM elements.
- **`uia_get_element_at_point(x, y)`** — reverse-looks up element at screen coordinates via UIA `ElementFromPoint`. Use after `get_cursor_position` or click.
- **`wait_for_ui_element(handle, name, control_type, timeout)`** — polls UIA `FindFirst` on window descendants until element appears. Structural alternative to `wait_for_text`.
- **Auto-capture `element_at_point` in chain** — mouse steps (`click`, `move_mouse`, `hover`, `drag`) automatically call `UIAElementFromPoint` at target coordinates after execution. Element is attached to step output.
- **`verify_ui` / `if_uia` step types** — UIA-based element presence verification and conditional branching. Structural alternatives to OCR-based `verify`/`if`.
- **Chain-callable UIA tools** — `uia_find`, `uia_get_element_at_point`, `uia_get_all_elements`, `uia_set_text`, `wait_for_ui_element`, `get_active_window`, `ocr_window`, `ocr_active_window`.
- **Schema fix** — `args.AdditionalProperties` relaxed so `{{variable}}` template strings pass validation.
- **`ocr_window` / `ocr_active_window` tools** — window-targeted OCR. `ocr_window` by handle, `ocr_active_window` for foreground.
- **`list_windows` / `get_active_window` now return bounding rects** — `x`, `y`, `width`, `height` for screen cross-referencing.
- **Tool descriptions teach workflow** — `get_window_state`, `focus_window`, `uia_find`, `ocr_window` describe state→focus→interact→validate sequence.

### v0.2.27 — ONNX + OCR Fallback for Template Matching
- **find_image / find_all_images** — NCC failure cascades to ONNX YOLO → OCR. Degenerate templates (zero-dim, no variance) skip NCC entirely.
- **ocr_languages** — new tool, native COM (no PowerShell)
- **fullscreen detection** — `get_window_state` returns `fullscreen: bool`
- **middle_click / horizontal_scroll** — button=middle on click, horizontal=true on scroll

---

## Next Up — Prioritized

### 1. Post-Task Introspection Engine (COMPLETED — v0.2.18)
Extend the adaptive engine from passive stats → active self-improvement. After every task, log:
- **What worked** — which tools/macros succeeded, completion time
- **What slowed down** — retries, OCR failures, window drift, element re-discovery
- **Macro reusability** — which command sequences repeat across tasks
- **Element discovery time** — how long did finding each UI element take?
- **Skill generalization** — auto-suggest reusable macros from successful sequences

Implementation sketch:
```
internal/actions/introspection.go
├── TaskLogger    — record task start/end, steps, outcomes
├── SkillMiner    — analyze logs for reusable sequences
├── Suggestions   — surface improvement opportunities
└── MCP tools     — introspection_analyze, introspection_suggest
```

### 2. Keylogger Rewrite (COMPLETED — v0.2.19)
Replaced `WH_MOUSE_LL` + `WH_KEYBOARD_LL` hooks with `GetAsyncKeyState` polling loop (50ms ticker). Eliminates system-wide input lag. Polling runs in a goroutine — no locked OS thread, no Windows message loop.

### 3. Test Validation of Recent Fixes (HIGH)

The v0.2.33 changes introduced several correctness-sensitive behaviors that need unit test coverage:

- **`uniqueTokens` dedup** — verify token deduplication works (duplicates, empty strings, case handling) and that `TrainFromDatalog` counts never exceed `total_commands`.
- **`nearbyOCRText` spatial scoping** — verify words within radius are returned, words outside are excluded, ordering by distance, dedup, empty/edge cases.
- **`capAndDedupeText` fallback** — verify max word cap, dedup, empty input, strings with mixed whitespace.
- **`SaveAdaptiveStat`/`LoadPersistedStats` round-trip** — verify aggregates persist to `adaptive_stats` table and reload correctly, including concurrent writes and MIN/MAX accumulation.
- **`Analyze()` persisted fallback** — verify timing_stats/success_rates fall through to `e.persisted` when no live samples exist.

Test files: `internal/actions/adaptive_test.go`, `internal/actions/datalog_test.go` (unit tests, no build tag; integration tests via `//go:build integration`).

### 4. Chain Interruption (MEDIUM)
Ability to stop mid-chain on error/state change — `on_error: "stop"` already exists, needs `interrupt` signaling.

### 4. Cross-platform Interface (LOW)
- Define platform interface
- Linux/macOS stubs

### 5. Skill Library (v0.3.0 — ML Setup Endgame)
- High-level macro abstraction layer
- Reusable recipes (install, browse, configure, etc.)
- Stable planner/executor interface
- End of ML setup phase — AI + ML work hand-in-hand
- Self-growing knowledge: save tokens, build over time, share models equally

---

## Versioning

See [`docs/reference/versioning-strategy.md`](../reference/versioning-strategy.md) for the full versioning scheme, bump rules, tagging policy, and release process.

---

## Competitive Intelligence — Features Worth Stealing

High-impact features from competing projects that fill gaps in go-mcp-computer-use, ordered by effort:

### Quick Wins (low effort, high value)

| Steal From | Feature | What It Does | Where to Implement |
|-----------|---------|-------------|-------------------|
| **Windows-MCP** | Per-tool enable/disable | Config map allow/deny list per MCP tool | `server.go` — filter registration — **done v0.2.37** (`tool_denylist` config + `server.RemoveTools`) |
| **Recall** | Retention policy | Auto-prune training samples older than N days | `training/cleanup.go` — **done v0.2.37** (`retention_days` config + `StartRetentionPruner`) |
| **Builtin** | Security tools | `is_admin`, `get_username`, `list_tools` — identity & permission awareness | `internal/actions/system.go` — Win32 API calls (GetTokenInformation, GetUserNameW) |
| **Builtin** | Sensitive content filter | Regex PII/password detection on OCR text before saving training samples | `internal/actions/training.go` — filter hook in save pipeline |

### Medium Effort

| Steal From | Feature | What It Does | Where to Implement |
|-----------|---------|-------------|-------------------|
| **Windows-MCP** | Bearer token + TLS | Auth for TCP transport (needed before SSE) | New `transport/` package |
| **Windows-MCP** | SSE transport | Switchable from stdio so server can run standalone | New `transport/` + MCP SSE support |
| **Windows-MCP** | Browser DOM mode | CDP/Playwright integration for Chrome/Edge/Firefox DOM snapshots | Start in `actions/browseruse.go`; extract to `internal/browser/` if it outgrows a single file. Chrome/Edge use native CDP; Firefox via Playwright driver. |
| **Agent-S** | In-context RL | Track last-N action outcomes per session, surface as reflection context | `actions/introspection.go` — session-scoped outcome buffer |

### High Effort (architectural)

| Steal From | Feature | What It Does | Where to Implement |
|-----------|---------|-------------|-------------------|
| **Cua** | Background control | `SendInput`/UIA instead of cursor-steal for clicks | `actions/click.go` — new input backend |
| **Cua** | VM sandboxing | QEMU/Docker target instead of host OS | New `sandbox/` package |
| **Recall** | Semantic indexing | Vector embeddings for screenshot search | New `memory/vector/` package + embedding model |
| **DesktopCtl** | GPU-accelerated OCR | Swap WinRT OCR for GPU-backed | `ocr/` — new backend |

## Constraints

- Windows 10/11 only
- MCP spec 2025-11-25
- stdio transport only
- 64-bit binary
- CGO required for ONNX runtime (Zig cc as C cross-compiler)
- External deps: `modernc.org/sqlite` (pure Go), `github.com/yalue/onnxruntime_go`, `golang.org/x/sys`

## Key Decisions

- `sendVKPress` with 50ms delay — UE5 games require minimum key hold duration
- Keylogger uses `GetAsyncKeyState` polling loop (50ms ticker) — avoids system-wide input lag from low-level hooks
- CGO mandatory for ONNX — Zig cc with x86_64_v2 CPU baseline
- Adaptive engine: pure Go (rolling averages, TF-IDF) — no Python/external ML
- Bridge window: 30s — OCR→AI→MCP→Command round trip ceiling
- Data logging SQLite: same pattern as memory/training stores, WAL journal mode
