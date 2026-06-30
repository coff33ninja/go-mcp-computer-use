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

## Current State: v0.3.x — Skill Library Phase

All tools registered in `internal/server/server.go`, auto-documented in `docs/tools.md`.
**118 tools** across 20 categories. Stable release line: `v0.2.x` (default branch). Active development: `v0.3.x`.

### Tool Categories

| Category | Count | Key Tools |
|----------|-------|-----------|
| Screenshot & Vision | 8 | screenshot, ocr, find_image, record_screen, get_pixel_color, get_screen_size, get_screen_dpi, get_display_modes |
| Mouse | 6 | click, move_mouse, scroll, drag, hover, get_cursor_position |
| Keyboard | 9 | type, key_press, key_down, key_up, type_and_submit, select_all_and_type, keylogger (polling-based), get/set_keyboard_layout |
| Window Management | 13 | list/find/focus/move/minimize/maximize/restore/close/get_state, screenshot_element, wait_for_window, get_active_window |
| Chained / Composite | 5 | find_text_and_click, wait_for_text, click_menu_item, launch_and_wait, chain (poll/loop/if/capture) |
| Chain Automation | 1 | chain — sequential multi-step executor |
| UI Automation | 3 | uia_find, uia_get_text, uia_invoke |
| Browser Automation | 4 | navigate, search, new_tab, focus_url_bar |
| File Explorer | 4 | explorer_focus/open_path, open_file_explorer/location |
| Audio | 2 | list_audio_devices, set_default_audio_device |
| Memory & Templates | 10 | memory CRUD, template store/find/list/forget, layout_validate |
| ONNX ML | 7 | onnx_detect/status/download, onnx_watch_start/stop/status/cache |
| Priors & Statistics | 1 | priors_stats |
| Training Pipeline | 6 | training_save_sample/list_samples/stats/mark_used/cleanup_noise, find_ui_element |
| Data Export | 1 | export_yolo_dataset |
| Data Logging | 3 | datalog_query/status/export |
| Adaptive Agent | 3 | agent_analyze/suggest/train |
| Introspection & Debugging | 4 | task_begin/end, introspection_analyze, bridge_debug |
| Runtime Config | 1 | set_config |
| System | 26 | volume, mute, brightness, battery, clipboard, keyboard layout, network, ping, system info, uptime, idle, displays, open_url, notification, power, wait, displays, DPI, disk usage |
| Process Management | 3 | list_processes, launch_app, kill_process |

---

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

### v0.2.18 — Introspection Engine
- **Post-Task Introspection** (`internal/actions/introspection.go`) — three new MCP tools:
  - `task_begin` / `task_end` — mark task boundaries and mine insights (slow/failed tools, OCR stats, repeat patterns, improvement suggestions)
  - `introspection_analyze` — browse completed task history
- Uses existing `command_log` + `ocr_log` — `task_log` table added to datalog DB
- `datalog_status` now reports `task_count`

### v0.2.19 — Keylogger Rewrite + CI Fix
- **Keylogger rewritten: hooks → polling** — `WH_MOUSE_LL` + `WH_KEYBOARD_LL` replaced with `GetAsyncKeyState` polling loop (50ms ticker). Eliminates system-wide input lag. No locked OS thread, no Windows message loop. Scroll wheel sacrificed (acceptable trade-off).
- **CI lint failure fixed** — `gen-tools-doc.go` missing categories for 4 tools, stale `docs2/` output removed, docs regenerated.

---

## Next Up — Prioritized (v0.3.x)

### 1. Skill Library (v0.3.0 — Core)
- High-level macro abstraction layer
- Reusable recipes (install, browse, configure, edit document, etc.)
- Stable planner/executor interface
- Self-growing knowledge: save tokens, build over time

### 2. Chain Interruption
- Stop mid-chain on error/state change
- `interrupt` signaling for running chains

### 3. Cross-platform Interface
- Define platform interface
- Linux/macOS stubs

---

## Versioning

```
v<major>.<minor>.<patch>
```

| Bump | When | Examples |
|------|------|----------|
| `+0.0.x` (patch) | Bug fixes, tool tweaks, doc updates | Bridge window fix, keylogger rewrite, CI fix |
| `+0.x.0` (minor) | New capabilities, architecture changes | Introspection engine, skill library |
| `+1.0.0` (major) | Stable, field-tested | Full pipeline working |

## Constraints

- Windows 10/11 only
- MCP spec 2025-11-25
- stdio transport only
- 64-bit binary
- CGO required for ONNX runtime (Zig cc as C cross-compiler)
- External deps: `modernc.org/sqlite` (pure Go), `github.com/yalue/onnxruntime_go`, `golang.org/x/sys`

## Key Decisions

- `sendVKPress` with 50ms delay — UE5 games require minimum key hold duration
- Keylogger uses `GetAsyncKeyState` polling at 50ms — replaced `WH_*_LL` hooks which caused system-wide input lag
- CGO mandatory for ONNX — Zig cc with x86_64_v2 CPU baseline
- Adaptive engine: pure Go (rolling averages, TF-IDF) — no Python/external ML
- Bridge window: 30s — OCR→AI→MCP→Command round trip ceiling
- Data logging SQLite: same pattern as memory/training stores, WAL journal mode

---

<sub><sup>
look at this beautiful architecture diagram. layers. responsibilities. no overlap. it's a work of art. now look at the code — it's a single file called `misc.go` that somehow handles battery, brightness, AND clipboard. "no overlap in responsibility" sounds great in a markdown file. in practice, we have a file called `chained.go` that's the junk drawer of tools nobody knew where to put. the plan is aspirational. the code is chaotic. and somehow both are equally valid descriptions of this project.
</sup></sub>
