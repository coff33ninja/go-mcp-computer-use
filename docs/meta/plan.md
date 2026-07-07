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

## Design Principles

- **No overlap in responsibility** — each layer does one job well
- **Stateless → stateful** — element memory replaces blind re-discovery
- **ML informs MCP, doesn't replace it** — perception feeds structure, not commands
- **Feedback loop** — every action is verified by perception before continuing
- **Stable planner/executor interface** — high-level skills decoupled from tool layer so vision models, LLMs, or backends can be swapped

## Current State: v0.2.27 — 120+ tools

All tools registered in `internal/server/server.go`, auto-documented in [`docs/reference/tools.md`](../reference/tools.md). Adaptive engine now includes timing stats, success rates, coordinate prediction, and full OCR-bridge training pair coverage across all 11 action tools.

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
| **Windows-MCP** | Per-tool enable/disable | Config map allow/deny list per MCP tool | `server.go` — filter registration |
| **Recall** | Retention policy | Auto-prune training samples older than N days | `training/cleanup.go` — add TTL config + background goroutine |
| **Windows MCP Server** | DPI normalization | Scale coordinates per-monitor DPI | `actions/verify.go` — `SmartRegionAround` / coordinate transforms |

### Medium Effort

| Steal From | Feature | What It Does | Where to Implement |
|-----------|---------|-------------|-------------------|
| **Windows-MCP** | Bearer token + TLS | Auth for TCP transport (needed before SSE) | New `transport/` package |
| **Windows-MCP** | SSE transport | Switchable from stdio so server can run standalone | New `transport/` + MCP SSE support |
| **Windows-MCP** | Browser DOM mode | CDP/Playwright integration for Chrome/Edge DOM snapshots | New `internal/browser/` package |
| **Agent-S** | In-context RL | Track last-N action outcomes per session, surface as reflection context | `actions/introspection.go` — session-scoped outcome buffer |
| **Recall** | Sensitive content filtering | Regex PII/password detection on OCR text before saving training samples | `training/save.go` — filter hook |

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
