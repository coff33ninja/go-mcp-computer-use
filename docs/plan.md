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

## Current State: v0.2.17 — 115 tools

All tools registered in `internal/server/server.go`, auto-documented in `docs/tools.md`.

### Screenshot & Vision (8)
- `screenshot` / `screenshot_element` — GDI BitBlt, full screen or window
- `ocr` — Windows.Media.Ocr via PowerShell (full screen or region)
- `find_image` — NCC template matching (base64 template)
- `get_pixel_color` — hex color at coordinates
- `record_screen` — frame polling at interval

### Mouse (6)
- `click` / `move_mouse` / `scroll` / `drag` / `hover` — SendInput
- `get_cursor_position`

### Keyboard (9)
- `type` — SendInput KEYBDINPUT UTF-16
- `key_press` / `key_down` / `key_up` — VK code hold/release
- `type_and_submit` / `select_all_and_type`
- `keylogger_start/stop/status` — LL hook input recording
- `get_keyboard_layout` / `set_keyboard_layout`

### Window Management (7)
- `list_windows` / `find_window` / `get_active_window`
- `focus_window` / `focus_window_by_title`
- `move_window` / `minimize` / `maximize` / `restore` / `close` / `get_window_state`
- `wait_for_window`

### Chained / Composite (9)
- `find_text_and_click` / `type_and_submit` / `launch_and_wait`
- `screenshot_element` / `hover` / `wait_for_text`
- `select_all_and_type` / `click_menu_item`
- `chain` — sequential multi-step executor (poll/loop/if/capture)

### Browser Automation (4)
- `browser_focus_url_bar` / `browser_navigate` / `browser_new_tab` / `browser_search`

### File Explorer (2)
- `explorer_focus` / `explorer_open_path`

### UI Automation (3)
- `uia_find` / `uia_get_text` / `uia_invoke` — COM UI Automation

### System (23)
- `get_system_info` / `get_volume` / `set_volume` / `set_mute`
- `get_clipboard` / `set_clipboard` / `get_screen_size` / `get_screen_dpi`
- `get_cursor_position` / `get_pixel_color` / `get_brightness` / `set_brightness`
- `get_idle_time` / `get_uptime` / `get_disk_usage` / `get_display_modes`
- `get_network_info` / `ping` / `get_battery`
- `open_url` / `open_file_explorer` / `open_file_location` / `show_notification`
- `lock_workstation` / `shutdown` / `restart` / `sleep` / `hibernate`

### Process Management (3)
- `list_processes` / `launch_app` / `kill_process` / `launch_and_wait`

### Audio (2)
- `list_audio_devices` / `set_default_audio_device`

### Memory & Templates (9)
- `memory_set/get/search/list/forget` — SQLite-backed facts store
- `layout_validate` — window drift + OCR keyword verification
- `template_store/find/list/forget` — self-growing PNG template library

### ONNX ML (7)
- `onnx_status` / `onnx_download` / `onnx_detect` — YOLO11n + MobileNetV3
- `onnx_watch_start/stop/status/cache` — background detection watcher
- `export_yolo_dataset` — unused samples as YOLO-format dataset

### Data Logging & Adaptive (8)
- `datalog_query` / `datalog_status` / `datalog_export`
- `agent_analyze` / `agent_suggest` / `agent_train`
- `bridge_debug` — bridge state inspection
- `set_config` — runtime training/logging/privacy toggles

### Training Pipeline (5)
- `training_list_samples` / `training_mark_used` / `training_save_sample`
- `training_cleanup_noise` / `training_stats` / `export_yolo_dataset`
- `priors_stats` — element frequency + position priors

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

### 2. Keylogger Rewrite (MEDIUM)
Replace `WH_MOUSE_LL` hook with polling (`GetAsyncKeyState` every 50ms) to eliminate system-wide input lag.

### 3. Chain Interruption (MEDIUM)
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

```
v<major>.<minor>.<patch>
```

| Bump | When | Examples |
|------|------|----------|
| `+0.0.1` (patch) | Bug fixes, tool tweaks, doc updates | Bridge window fix, keylogger rewrite |
| `+0.1.0` (minor) | New capabilities, architecture changes | Introspection engine, skill library |
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
- Keylogger uses `WH_KEYBOARD_LL` + `WH_MOUSE_LL` hooks — planned rewrite to polling
- CGO mandatory for ONNX — Zig cc with x86_64_v2 CPU baseline
- Adaptive engine: pure Go (rolling averages, TF-IDF) — no Python/external ML
- Bridge window: 30s — OCR→AI→MCP→Command round trip ceiling
- Data logging SQLite: same pattern as memory/training stores, WAL journal mode
