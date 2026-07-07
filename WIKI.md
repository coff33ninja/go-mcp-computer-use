# go-mcp-computer-use Wiki

Auto-generated from project docs. Run `go run ./scripts/gen-wiki.go` to regenerate.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Features](#features)
3. [Architecture](#architecture)
4. [Tools Reference](#tools-reference)
5. [Build & Usage](#build--usage)
6. [Configuration](#configuration)
7. [Project Plan](#project-plan)
8. [Backlog](#backlog)
9. [Changelog](#changelog)
10. [Known Issues](#known-issues)
11. [Security](#security)
12. [Reference Docs](#reference-docs)
13. [Guides](#guides)
14. [CI/CD Pipeline](#cicd-pipeline)

---

## Project Overview


## Features

- **Screenshot** — full screen or region capture (GDI BitBlt → PNG → base64)
- **Mouse** — click, move, scroll, drag, hover
- **Keyboard** — type, key combos (Ctrl+C, Alt+Tab), type+submit, select all+type
- **OCR** — extract text via Windows.Media.Ocr, optional language (en-US, ja-JP, fr-FR...)
- **Template matching** — find an image on screen via NCC (normalized cross-correlation)
- **Find & Click** — OCR + click: find text on screen and click it  
- **Chained tools** — `find_text_and_click`, `launch_and_wait`, `wait_for_text`, `click_menu_item`, `select_all_and_type`
- **Screen recording** — capture frames at interval for a duration
- **Window management** — list, focus, move, resize, min/max/restore, close, find, state
- **Audio devices** — list playback/recording devices, set default
- **Clipboard** — get/set text with retry + timeout
- **System** — volume, mute, brightness, battery, disk, DPI, display info, uptime, idle
- **Network** — hostname, IPs, DNS, gateway, ping
- **Processes** — list, launch, kill
- **Power** — shutdown, restart, sleep, hibernate, lock
- **Per-monitor DPI** — per-monitor DPI awareness, scale reporting
- **UI Automation** — find elements by name/automationID, get text, invoke buttons via native COM UIAutomation (no PowerShell)
- **OCR via native WinRT COM** — StorageFile → BitmapDecoder → OcrEngine pipeline, 2-8x faster than PowerShell (falls back to PowerShell on error)
- **UIPI detection** — warns when keyboard input targets elevated/admin windows
- **Training data pipeline** — persistent screenshot collection with categorized folders (`raw/click/`, `raw/type/`, `raw/navigate/`, `watcher/elements_found/`, etc.) and SQLite metadata. Auto-saves on every UI action for model fine-tuning.
- **Memory-backed UI element cache** — ONNX detections auto-stored as memory facts (`ui:{window}:{class}`) with TTL. AI reuses cached coordinates across sessions.
- **`find_ui_element` tool** — cascading lookup: memory → ONNX → OCR. Self-learning: saves findings to memory + training store.
- **120 MCP tools** — see [`docs/reference/tools.md`](docs/reference/tools.md) for the full listing

## Tools

Auto-generated reference at [`docs/reference/tools.md`](docs/reference/tools.md) — always in sync with `internal/server/server.go`. Run `go run ./scripts/gen-tools-doc.go` to regenerate.

Categories: Screenshot & Vision, Mouse, Keyboard, Window Management, Chained / Composite, Chain Automation, UI Automation, Browser Automation, File Explorer, Audio, Memory & Templates, ONNX ML, Priors & Statistics, Training Pipeline, Data Export, Data Logging, Adaptive Agent, Introspection & Debugging, Runtime Config, System, Process Management.

## Security

**⚠️ This server can fully control your Windows machine.** See [`docs/security.md`](docs/security.md) for:
- Security warning and dangerous capabilities
- Elevation & UIPI (Admin vs Non-Admin)
- Data collection & privacy controls
- Agent configuration

## Accessibility

See [`docs/guides/accessibility.md`](docs/guides/accessibility.md) for assistive technology use cases, hands-free computer operation, and the dual-use nature of these tools.

## Build & Usage

See [`docs/guides/build.md`](docs/guides/build.md) for:
- Requirements (Windows 10+, Go 1.26+, Zig 0.16+)
- Quick start & installation
- Build commands (CGO via Zig cc — always ONNX-enabled)
- Performance benchmarks

## Configuration

See [`docs/reference/configuration.md`](docs/reference/configuration.md) for the full config file reference.

## Architecture

See [`docs/architecture.md`](docs/architecture.md) for the agent stack diagram and code map.

---

## Features


- **Screenshot** — full screen or region capture (GDI BitBlt → PNG → base64)
- **Mouse** — click, move, scroll, drag, hover
- **Keyboard** — type, key combos (Ctrl+C, Alt+Tab), type+submit, select all+type
- **OCR** — extract text via Windows.Media.Ocr, optional language (en-US, ja-JP, fr-FR...)
- **Template matching** — find an image on screen via NCC (normalized cross-correlation)
- **Find & Click** — OCR + click: find text on screen and click it  
- **Chained tools** — `find_text_and_click`, `launch_and_wait`, `wait_for_text`, `click_menu_item`, `select_all_and_type`
- **Screen recording** — capture frames at interval for a duration
- **Window management** — list, focus, move, resize, min/max/restore, close, find, state
- **Audio devices** — list playback/recording devices, set default
- **Clipboard** — get/set text with retry + timeout
- **System** — volume, mute, brightness, battery, disk, DPI, display info, uptime, idle
- **Network** — hostname, IPs, DNS, gateway, ping
- **Processes** — list, launch, kill
- **Power** — shutdown, restart, sleep, hibernate, lock
- **Per-monitor DPI** — per-monitor DPI awareness, scale reporting
- **UI Automation** — find elements by name/automationID, get text, invoke buttons via native COM UIAutomation (no PowerShell)
- **OCR via native WinRT COM** — StorageFile → BitmapDecoder → OcrEngine pipeline, 2-8x faster than PowerShell (falls back to PowerShell on error)
- **UIPI detection** — warns when keyboard input targets elevated/admin windows
- **Training data pipeline** — persistent screenshot collection with categorized folders (`raw/click/`, `raw/type/`, `raw/navigate/`, `watcher/elements_found/`, etc.) and SQLite metadata. Auto-saves on every UI action for model fine-tuning.
- **Memory-backed UI element cache** — ONNX detections auto-stored as memory facts (`ui:{window}:{class}`) with TTL. AI reuses cached coordinates across sessions.
- **`find_ui_element` tool** — cascading lookup: memory → ONNX → OCR. Self-learning: saves findings to memory + training store.
- **120 MCP tools** — see [`docs/reference/tools.md`](docs/reference/tools.md) for the full listing


---

## Architecture

The server implements the execution and perception layers of a closed-loop embodied agent:

```
┌────────────────────────────────────────────────────┐
│  AGENT STACK (runs in the AI client, not the MCP)   │
│                                                    │
│  LLM ────── Cognitive Layer (reasoning + planning)  │
│   ↓                                                │
│  MCP ────── Orchestration Layer (skill dispatch)    │
│   ↓                                                │
│  ─── MCP protocol boundary ───                     │
│   ↓                                                │
│  go-mcp-computer-use (this server)                  │
│                                                    │
│  Controller ── Physical Layer (mouse, keyboard,     │
│  │              window system, clipboard)           │
│  Perception ── Vision Layer (screenshot, OCR,       │
│  │              ONNX ML detection, screen capture)  │
│  Memory ────── State Layer (SQLite facts, element   │
│                 templates, UI position cache)       │
│  Training ──── Data pipeline (screenshot store,     │
│                 YOLO export, sample management)     │
│                                                    │
│  World ─────── Desktop / Browser / Applications     │
└────────────────────────────────────────────────────┘
```

Each layer has a distinct responsibility — no overlap. The server handles execution, perception, memory, and training. The AI client handles reasoning and orchestration via MCP.

## Code Map

Complete tool→handler→action→file mapping in [`reference/codebase-map.md`](reference/codebase-map.md).

```
cmd/
  ├── mcp-server/main.go       — entrypoint, DPI awareness, signals
  ├── benchmark/               — performance benchmark tool
  └── ocrhelper/               — WinRT OCR helper binary

internal/server/server.go      — MCP tool registrations (96 tools, 120+ registrations)
internal/config/config.go      — JSON config file (~/.config/go-mcp-computer-use/config.json)

internal/actions/              — 46 files, organized by capability:
  ├── Input:
  │   ├── mouse.go             — SendInput click/move/scroll/drag
  │   ├── keyboard.go          — SendInput KEYEVENTF_UNICODE
  │   └── chained.go           — composite tools (find_text_and_click, hover, etc.)
  │
  ├── Perception:
  │   ├── screenshot.go        — GDI BitBlt capture → PNG → base64
  │   ├── ocr.go               — OCR orchestration (native COM + PowerShell fallback)
  │   ├── ocr_com.go           — WinRT COM OCR pipeline
  │   ├── template.go          — NCC template matching (find_image, find_all_images)
  │   ├── onnx.go              — YOLO/MobileNet inference via onnxruntime
  │   ├── watcher.go           — background ONNX detection loop with caching
  │   ├── priors.go            — element frequency/position priors per window
  │   ├── ui_finder.go         — cascading locator (memory → ONNX → OCR)
  │   └── recording.go         — screen recording (frames at intervals)
  │
  ├── Window & Desktop:
  │   ├── window.go            — EnumWindows, focus, find, move/resize
  │   ├── window_ext.go        — get window state
  │   ├── dpi.go               — DPI awareness, coordinate scaling, WindowNormalizer
  │   ├── uia.go               — UI Automation (find, get text, invoke)
  │   ├── uia_com.go           — IUIAutomation COM interface
  │   ├── uipi.go              — UIPI elevation detection
  │   ├── validate_layout.go   — stored UI element position validation
  │   ├── browseruse.go        — browser automation (navigate, search, tab, url bar)
  │   └── windowexploreruse.go — File Explorer automation (focus, open path)
  │
  ├── System:
  │   ├── process.go           — list/launch/kill processes
  │   ├── system.go            — system info, active window
  │   ├── power.go             — shutdown, restart, sleep, hibernate, disk usage, explorer
  │   ├── misc.go              — battery, displays, display modes, pixel color, notification
  │   ├── audio.go             — audio devices via PowerShell
  │   ├── brightness.go        — display brightness via WinRT
  │   ├── idle.go              — GetLastInputInfo
  │   ├── network.go           — network info, ping
  │   ├── layout.go            — keyboard layout, screen DPI
  │   └── winrt.go             — WinRT infrastructure (HSTRING, RoInitialize, async)
  │
  ├── Automation:
  │   ├── chain.go             — chain step executor (poll, if/else, loop, variables)
  │   ├── keylogger.go         — WinEvent hook input recording
  │   └── adaptive.go          — adaptive engine (timing, success rates, coord prediction)
  │
  ├── Persistence:
  │   ├── memory.go            — SQLite facts (set/get/search/list/forget) + templates
  │   ├── datalog.go           — tool call/OCR/chain logging, training pair export
  │   └── training.go          — training data storage (categorized PNGs + samples.db)
  │
  └── Introspection:
      ├── introspection.go     — task lifecycle + performance mining
      ├── timeout.go           — WithTimeout helper
      ├── validate.go          — coordinate bounds validation
      └── user32.go            — shared user32.dll proc declarations
```

## Agent Architecture

See [`guides/computer-use-guide-for-ai-agents.md`](guides/computer-use-guide-for-ai-agents.md) for the full layered agent stack.

For the complete tool→handler→action function→file mapping, see [`reference/codebase-map.md`](reference/codebase-map.md).


See [`docs/architecture.md`](docs/architecture.md) for the full architecture document.

---

## Tools Reference

# Tools (132)

Auto-generated from `internal/server/server.go`. Total: **132 tools**.

## Screenshot & Vision (10)

- `find_all_images` — Find ALL occurrences of a template image on screen using NCC template matching. Provide template as base64 PNG. Returns array of matches with coordinates and scores.
- `find_image` — Find a template image on screen using NCC template matching. Provide template as base64 PNG. Returns coordinates of best match.
- `get_display_modes` — Get all available display modes (resolution, refresh rate, color depth) for a monitor by device name.
- `get_pixel_color` — Get the hex color at screen coordinates x,y.
- `get_screen_dpi` — Get per-monitor screen DPI and scale percentage.
- `get_screen_size` — Get the screen dimensions.
- `ocr` — Extract text from screen using Windows OCR. Supports full screen or region (x,y,w,h).
- `ocr_languages` — List all available Windows OCR languages. Returns array of language objects with tag, display_name, and native_name.
- `record_screen` — Record screen frames at fixed intervals. Returns base64 images. Duration in ms, interval in ms.
- `screenshot` — Capture the screen or a region. If w/h omitted, captures full screen.

## Mouse (6)

- `click` — Click at screen coordinates x,y. Button: left/right/middle. Clicks: 1 or 2.
- `drag` — Drag mouse from (from_x, from_y) to (to_x, to_y).
- `get_cursor_position` — Get the current mouse cursor position.
- `hover` — Move the mouse to coordinates and wait briefly (for tooltips/hover menus).
- `move_mouse` — Move mouse cursor to x,y.
- `scroll` — Scroll the mouse wheel. Positive clicks = up, negative = down. Set horizontal=true for horizontal scroll.

## Keyboard (9)

- `key_down` — Hold a key down (does not release it). Use key_up to release. Example: "W"
- `key_press` — Press key combination. Example: ["Ctrl", "C"] for copy.
- `key_up` — Release a key that was held down with key_down. Example: "W"
- `keylogger_start` — Start recording keyboard and mouse input for replay
- `keylogger_status` — Check if keylogger is active and event count
- `keylogger_stop` — Stop recording and return recorded sequence as chain steps
- `select_all_and_type` — Select all text (Ctrl+A) and type replacement text.
- `type` — Type text at the currently focused element.
- `type_and_submit` — Type text and press Enter (e.g. for form submission or search).

## Window Management (13)

- `close_window` — Close a window by handle.
- `find_window` — Find a window handle by title.
- `focus_window` — Bring a window to the foreground by handle.
- `focus_window_by_title` — Find a window by title and focus it, clicking its title bar to ensure activation. Useful before keyboard input in chain steps.
- `get_active_window` — Get the current foreground window info.
- `get_window_state` — Get window state (visible, minimized, maximized, position, size).
- `list_windows` — List all visible windows with their handles, titles, and PIDs.
- `maximize_window` — Maximize a window by handle.
- `minimize_window` — Minimize a window by handle.
- `move_window` — Move and resize a window by handle.
- `restore_window` — Restore a minimized or maximized window by handle.
- `screenshot_element` — Take a screenshot of a specific window by handle.
- `wait_for_window` — Wait for a window with the given title to appear. Returns handle or timeout.

## Chained / Composite (4)

- `click_menu_item` — Find a window by title, then click a menu item or button using OCR within that window.
- `find_text_and_click` — Find text on screen using OCR and click at its location. Optional region x,y,w,h to search within.
- `launch_and_wait` — Launch an application and wait for its window to appear.
- `wait_for_text` — Wait for text to appear on screen. Polls OCR until found or timeout.

## Chain Automation (1)

- `chain` — Execute a sequence of steps sequentially server-side. Steps can call any tool, wait, capture output, and use {{variable}} substitution.

## UI Automation (3)

- `uia_find` — Find UI elements by name, automation_id, or control_type using UI Automation. Returns bounding rectangles and properties.
- `uia_get_text` — Get text from a UI element by name or automation_id using UI Automation.
- `uia_invoke` — Click or invoke a UI element by name or automation_id using UI Automation.

## Browser Automation (4)

- `browser_focus_url_bar` — Focus a browser window's URL bar. Supports Firefox (Ctrl+T), Chrome/Edge (Ctrl+L), and other browsers. Provide browser name (firefox, chrome, edge, brave, opera) or window title substring.
- `browser_navigate` — Open a new tab in a browser and navigate to a URL.
- `browser_new_tab` — Open a new tab in a browser window. Uses Ctrl+T for all browsers.
- `browser_search` — Open a new tab in a browser and perform a search query.

## File Explorer (4)

- `explorer_focus` — Focus an existing File Explorer window.
- `explorer_open_path` — Open a File Explorer window at the specified path. Reuses existing window when possible.
- `open_file_explorer` — Open File Explorer to a specified path (default: C:\).
- `open_file_location` — Open File Explorer with a specific file selected.

## Audio (2)

- `list_audio_devices` — List all audio playback and recording devices.
- `set_default_audio_device` — Set the default audio playback device by device ID.

## Memory & Templates (10)

- `layout_validate` — Validate stored UI element layout against the current screen. Checks window existence, position drift, and OCR keyword verification. Returns adjusted coordinates and confidence levels (ok/drifted/stale).
- `memory_forget` — Delete facts by key, scope, or tags. At least one filter is required to prevent accidental mass deletion.
- `memory_get` — Retrieve a fact from the memory store by key and optional scope.
- `memory_list` — List stored facts under a scope with optional tag filter.
- `memory_search` — Full-text search across keys, values, scope, and tags using FTS5. Supports SQLite FTS5 query syntax.
- `memory_set` — Store a fact into the memory store. Fields: key (required), value (required, any JSON value), scope, tags (comma-separated), ttl (optional expiry in seconds).
- `template_find` — Find a stored UI element template on the current screen using NCC template matching. Returns coordinates, score, and drift from stored position.
- `template_forget` — Delete a stored UI element template by element_key and optional scope.
- `template_list` — List stored UI element templates with metadata (element key, scope, window title, hit count, etc.).
- `template_store` — Capture a UI element template from the current screen by cropping around a coordinate. Stores as base64 PNG in the element_templates table for visual re-identification.

## ONNX ML (7)

- `onnx_detect` — Run YOLO-based UI element detection on a screenshot (or full screen if no image provided). Returns detected elements with class labels, confidence scores, and bounding boxes. Requires onnxruntime.dll and YOLO model file.
- `onnx_download` — Check and prepare ONNX model files. Lists which models are present and which need manual download.
- `onnx_status` — Check ONNX runtime and model availability. Returns presence of YOLO model, MobileNet model, and onnxruntime.dll.
- `onnx_watch_cache` — Retrieve cached detections from the background watcher. Returns the most recent detection results with timestamps and saved reference paths.
- `onnx_watch_start` — Start a background watcher that periodically screenshots the screen, runs ONNX detection, and caches results. Takes interval_seconds (default 5).
- `onnx_watch_status` — Get the current ONNX watcher state: running, interval, last run time, cache size.
- `onnx_watch_stop` — Stop the background ONNX watcher.

## Priors & Statistics (1)

- `priors_stats` — Show learned element frequency and position statistics per window. Returns priors with sample count, frequency, and position distributions. Use min_count to filter out low-sample entries.

## Training Pipeline (6)

- `find_ui_element` — Find a UI element on screen by label. Checks memory first (from past ONNX detections), then runs ONNX detection, then falls back to OCR. Stores findings in memory for future reuse. Use this when the AI needs to locate an element it has seen before or needs to find programmatically.
- `training_cleanup_noise` — Delete low-signal (signal_level=0) training samples older than max_age_hours. Use dry_run=true to see what would be deleted without actually removing anything. Returns deleted count and freed bytes.
- `training_list_samples` — List saved training samples, optionally filtered by category or unused-only status.
- `training_mark_used` — Mark a training sample as used (after the model has been trained on it).
- `training_save_sample` — Capture screenshot and save as a training sample with a task prompt (e.g. 'click the submit button'). The ONNX model learns from these during idle retraining.
- `training_stats` — Get training data statistics: total samples, unused samples, breakdown by category, disk usage.

## Data Export (1)

- `export_yolo_dataset` — Export unused training samples as a YOLO-format dataset (images + labels + dataset.yaml) for external training with Ultralytics or other YOLO frameworks. Outputs to a directory of your choice.

## Data Logging (3)

- `datalog_export` — Export OCR+command training pairs as JSON for ML training. Optionally filter by session_id. Returns pairs with before/after OCR text and command JSON.
- `datalog_query` — Query the action/OCR data log. Table: commands, chains, ocr, or pairs. Filter by source, tool, success. Returns recent rows with all columns.
- `datalog_status` — Get data logging statistics: count of commands, chains, OCR snapshots, and training pairs logged to the datalog database.

## Adaptive Agent (3)

- `agent_analyze` — Analyze the adaptive engine state — timing stats, success rates per tool, and learned OCR→command sequences. Returns a full report for AI decision-making.
- `agent_suggest` — Given OCR screen text, predict the best next command based on past successful sequences. Returns ranked predictions with confidence scores and optional coord (x, y, confidence, samples) for click/hover/move_mouse.
- `agent_train` — Train the adaptive engine from datalog training_pairs. Rebuilds the OCR→command word index and sequence cache. Call after the datalog has accumulated new pairs.

## Introspection & Debugging (4)

- `bridge_debug` — Debug the OCR→command bridge state — shows recent OCR buffer, pending command, and timing info.
- `introspection_analyze` — View task history with mined insights from past task_begin/task_end sessions.
- `task_begin` — Mark the start of a task for post-task introspection. Call before the first tool call in a task.
- `task_end` — Mark the end of a task. Returns mined insights: slow/failed tools, OCR stats, repeat patterns, and improvement suggestions.

## Runtime Config (1)

- `set_config` — Update runtime configuration. Accepts any subset of: training_enabled (stop/start background screenshot saving), prior_adjustment (enable/disable ML prior confidence tuning), verify_bounds (toggle coordinate bounds checking), log_level (debug/info/warn/error), watcher_enabled (start/stop the background screenshot watcher), watcher_interval_seconds (change polling frequency while running). Changes persist to disk. Use this to disable data collection or control the tool at runtime.

## System (25)

- `get_battery` — Get battery status (percentage, charging, on battery).
- `get_brightness` — Get the current display brightness level (0-100).
- `get_clipboard` — Read text from the clipboard.
- `get_disk_usage` — Get disk usage information for all drives.
- `get_idle_time` — Get the system idle time (time since last user input) in milliseconds.
- `get_keyboard_layout` — Get the current keyboard layout / input language.
- `get_network_info` — Get network information: hostname, IP addresses, DNS servers, default gateway.
- `get_system_info` — Get system information (hostname, OS, RAM).
- `get_uptime` — Get the system uptime (time since last boot).
- `get_volume` — Get the current system volume level (0-100).
- `hibernate` — Hibernate the computer.
- `list_displays` — List all monitors with resolution and position.
- `lock_workstation` — Lock the workstation.
- `open_url` — Open a URL in the default browser.
- `ping` — Ping a host to check network reachability.
- `restart` — Restart the computer.
- `set_brightness` — Set the display brightness level (0-100).
- `set_clipboard` — Write text to the clipboard.
- `set_keyboard_layout` — Set the keyboard layout / input language (e.g. 'en-US', 'ja-JP').
- `set_mute` — Mute or unmute the system audio.
- `set_volume` — Set the system volume level (0-100).
- `show_notification` — Show a Windows notification message box.
- `shutdown` — Shut down the computer.
- `sleep` — Put the computer to sleep.
- `wait` — Wait for N milliseconds before the next action.

## Process Management (3)

- `kill_process` — Terminate a process by PID.
- `launch_app` — Launch an application by path or shell command.
- `list_processes` — List all running processes with PID, name, and thread count.

## Uncategorized (12)

- `copy_file` — Copy a file or directory (recursively) from source to destination.
- `create_directory` — Create a directory (recursive, like mkdir -p).
- `delete_file` — Delete a file or directory to the Recycle Bin (uses SHFileOperationW with FOF_ALLOWUNDO).
- `find_files` — Recursively search for files matching a glob pattern (e.g. '*.go', '**/*.md').
- `get_file_info` — Get file or directory metadata: size, mod_time, is_dir, mode.
- `get_working_directory` — Get the current working directory used for relative path resolution.
- `list_directory` — List directory contents. Returns entries with name, size, is_dir, mod_time, and mode.
- `move_file` — Move or rename a file or directory.
- `read_file` — Read a file with automatic type detection. Supports plaintext (txt, json, csv, yaml, etc.), docx, xlsx, pdf, and images (via OCR). Use page and page_size to paginate long content. Default page_size=8000 chars.
- `set_working_directory` — Set the working directory for relative path resolution in file tools.
- `uia_set_text` — Set text in a UI element by name or automation_id using UI Automation.
- `write_file` — Write content to a file. Supports plaintext, docx (creates from text, preserves structure on overwrite), xlsx (TSV content becomes cells), and PDF (text creates PDF, JSON fills existing form fields). Requires overwrite=true to replace existing files.

<!--
Generated by scripts/gen-tools-doc.go — 132 tools found
-->

---

## Build & Usage


See [`docs/guides/build.md`](docs/guides/build.md) for:
- Requirements (Windows 10+, Go 1.26+, Zig 0.16+)
- Quick start & installation
- Build commands (CGO via Zig cc — always ONNX-enabled)
- Performance benchmarks

## Configuration

See [`docs/reference/configuration.md`](docs/reference/configuration.md) for the full config file reference.


---

## Configuration

`~/.config/go-mcp-computer-use/config.json`:

```json
{
  "log_level": "info",
  "mouse_speed": 500,
  "click_delay_ms": 100,
  "verify_bounds": true,
  "action_timeout_ms": 30000,
  "uia_warmup": true,
  "training_enabled": true,
  "prior_adjustment": true,
  "watcher_auto_start": false,
  "watcher_interval_seconds": 5
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `log_level` | `info` | One of: `debug`, `info`, `warn`, `error` |
| `mouse_speed` | `500` | Mouse movement speed |
| `click_delay_ms` | `100` | Delay between mouse down/up (ms) |
| `verify_bounds` | `true` | Validate coordinates against screen bounds |
| `action_timeout_ms` | `30000` | Max time (ms) for blocking operations |
| `uia_warmup` | `true` | Warm up UIA at startup (async) to avoid cold-start delay. Set `false` if clients timeout during init. |
| `training_enabled` | `true` | Enable auto-save training snapshots on every UI action. Set `false` to stop all background data collection (also controllable at runtime via `set_config`). |
| `prior_adjustment` | `true` | Apply learned element frequency/position priors to ONNX detection scores. Set `false` for raw YOLO output only. |
| `watcher_auto_start` | `false` | Auto-start the background watcher on server boot. Watcher polls screen every N seconds and saves frames for training. |
| `watcher_interval_seconds` | `5` | How often the watcher captures and analyzes the screen (if running). Also used as default when starting via `set_config`. |

## Privacy Controls

See [`../security.md`](../security.md) for the full data collection and privacy controls reference.


See [`docs/reference/configuration.md`](docs/reference/configuration.md) for the full config reference.

---

## Project Plan


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



See [`docs/meta/plan.md`](docs/meta/plan.md) for the full project plan.

---

## Backlog


## How to Read

- **HAVE** = implemented (134 tools)
- **NEXT** = high-impact, feasible additions
- **FAR** = possible but lower priority or complex
- Items within a section ordered roughly by priority

---

## 1. VISION — See What's on Screen

### HAVE
- `screenshot` — full screen or region → base64 PNG
- `get_pixel_color` — hex color at x,y
- `get_screen_size` — virtual screen dimensions
- `get_screen_dpi` — per-monitor DPI + scale %
- `ocr` — extract text via Windows.Media.Ocr (any language)
- `find_image` — NCC template matching with ONNX + OCR fallback
- `find_all_images` — NCC template matching with ONNX + OCR fallback, returns all matches
- `ocr_languages` — list installed OCR languages (so agent knows what's available)
- `record_screen` — frame polling at interval → base64 frame array

### NEXT
| Tool | Why |
|------|-----|
| `image_diff` | pixel-level diff between two screenshots (detect changes) |
| `image_histogram` | color histogram analysis (detect dark/bright screens) |
| `match_template_multi` | multi-pass NCC at different scales (handle DPI variation) |

### FAR
| Tool | Why |
|------|-----|
| `detect_ui_elements` | use OCR bounding boxes to classify buttons/text fields/lists |
| `barcode_reader` | decode QR/DataMatrix/Code128 from screen region |
| `face_detection` | WinRT face detection via Camera |
| `screen_recording_video` | encode frames to actual video file (mp4) |
| `real_time_stream` | WebSocket stream of frames for live agent view |
| `color_detection` | find all pixels of a given color on screen |
| `snap_diff` | compare screenshot to a reference "golden" image |
| `ocr_gpu_accelerated` — GPU-backed OCR backend | 10x faster text extraction (from DesktopCtl) |
| `screen_tokenize` — structured UI token output | deterministic CLI-style UI description (from DesktopCtl) |

---

## 2. MOUSE — Point and Click

### HAVE
- `click` — left/right/middle, single/double at x,y
- `move_mouse` — move cursor to x,y
- `scroll` — wheel up/down/left/right (clicks + horizontal)
- `drag` — click-hold from→to
- `hover` — move + wait 300ms (for tooltips)
- `get_cursor_position` — current x,y

### NEXT
| Tool | Why |
|------|-----|
| `click_hold` / `release` — separate hold/release | complex drag-and-drop, slider manipulation |
| `scroll_smooth` — pixel-based scrolling | precise scroll in lists/canvases |
| `scroll_horizontal` — horizontal wheel/tilt | horizontal scrolling in wide content |
| `drag_relative` — drag by (dx, dy) from current pos | relative drag gestures |

### FAR
| Tool | Why |
|------|-----|
| `mouse_gesture` — recognize shape (L, V, Z) | gesture-based commands |
| `click_all_matches` — click every occurrence on screen | dismiss all notifications, close all tabs |
| `right_click_menu` — right-click + get menu items | context menu interaction |
| `multi_touch` — simulate touch gestures (pinch, swipe) | tablet/touch scenarios |
| `pen_stylus` — WinRT pen simulation | drawing, handwriting |
| `background_click` — click without stealing cursor | non-disruptive automation (from Cua) |
| `background_type` — type without focusing target window | invisible input (from Cua) |

---

## 3. KEYBOARD — Type and Command

### HAVE
- `type` — send text string
- `key_press` — key combos (Ctrl+C, Alt+Tab, etc.)
- `type_and_submit` — type + Enter
- `select_all_and_type` — Ctrl+A + type

### NEXT
| Tool | Why |
|------|-----|
| `type_character_by_character` — type with per-char delay | simulate human typing for apps that buffer input |
| `type_with_modifiers` — e.g. type "hello" while holding Shift | advanced text entry |
| `key_hold` / `key_release` — separate hold/release | games, modifier management |
| `get_keyboard_state` — which modifier keys are pressed | check CapsLock/NumLock/ScrollLock state |
| `set_keyboard_state` — toggle CapsLock/NumLock | fix common input issues |

### FAR
| Tool | Why |
|------|-----|
| `text_from_clipboard_paste` — paste instead of type | faster, avoids IME/input issues |
| `text_file_input` — type contents of a text file | paste large text without clipboard |
| `ime_text_input` — send text through IME composition | Japanese/Chinese input methods |
| `macro_record` / `macro_playback` — record key sequence | repeatable automation |
| `send_keys_advanced` — SendKeys-style with pauses | legacy app compatibility |

---

## 4. WINDOWS — Manage Screen Real Estate

### HAVE
- `list_windows` — all visible windows (handle, title, PID)
- `focus_window` — bring to foreground
- `find_window` — by title substring
- `wait_for_window` — poll until window appears
- `move_window` — set x,y,w,h
- `minimize_window` / `maximize_window` / `restore_window`
- `close_window`
- `get_window_state` — visible, minimized, maximized, fullscreen, position
- `screenshot_element` — screenshot a specific window

### NEXT
| Tool | Why |
|------|-----|
| `snap_window_left` / `snap_window_right` — snap to half screen | common multi-window workflow |
| `snap_window_top` / `snap_window_bottom` / `snap_window_corner` | quarter-screen layouts |
| `get_window_z_order` — window stacking order | understand overlap |
| `set_window_z_order` — bring to top/bottom/above/below | reorder windows |
| `set_window_transparency` — alpha blend (SetLayeredWindowAttributes) | peek through windows |
| `set_window_title` — change window title | identification |
| `cascade_windows` / `tile_windows` — arrange all windows | classic Windows arrange |
| `find_window_by_pid` — get window owned by a process | process→window mapping |

### FAR
| Tool | Why |
|------|-----|
| `get_window_children` — enumerate child windows | UI Automation for controls |
| `window_screenshot_multi` — screenshot all windows at once | full desktops |
| `scroll_window` — scroll content inside a window | scroll without focus |
| `click_window_at` — click relative to window origin | element positioning |
| `set_window_always_on_top` | floating windows |
| `get_window_class_name` — Win32 class name | identify control types |

---

## 5. VIRTUAL DESKTOPS — Multiple Desktops

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `list_virtual_desktops` | enumerate all desktops |
| `switch_virtual_desktop` (by index or GUID) | move between workspaces |
| `create_virtual_desktop` | new workspace |
| `remove_virtual_desktop` | delete workspace |
| `move_window_to_desktop` | move window to different desktop |
| `get_current_desktop` | which desktop is active |

### FAR
| Tool | Why |
|------|-----|
| `pin_window_to_all_desktops` | show on every desktop |
| `get_desktop_wallpaper` | background image |

---

## 6. PROCESSES — Run and Manage Apps

### HAVE
- `list_processes` — name, PID, threads, parent PID
- `launch_app` — ShellExecute (open verb)
- `kill_process` — TerminateProcess by PID

### NEXT
| Tool | Why |
|------|-----|
| `launch_app_with_args` — specify arguments | run commands with params |
| `launch_app_hidden` — no window | run background processes |
| `launch_app_as_admin` — UAC elevation | admin tasks |
| `launch_app_with_env` — set env vars for child proc | custom environment |
| `wait_for_process_exit` — wait until PID exits | sequential task chaining |
| `launch_and_wait_exit` — launch + wait for exit | run-and-collect |
| `get_process_info` — memory, CPU, command line | process details |

### FAR
| Tool | Why |
|------|-----|
| `set_process_priority` — idle/below_normal/normal/above_normal/high/realtime | CPU management |
| `suspend_process` / `resume_process` | freeze/thaw |
| `get_process_threads` — list threads | diagnostics |
| `run_as_user` — impersonation | different user context |
| `create_process_group` / `kill_process_tree` | manage process families |
| `vm_sandbox_create` — launch QEMU/Docker sandbox | isolated execution (from Cua) |
| `vm_sandbox_destroy` — tear down sandbox | cleanup (from Cua) |
| `sandbox_exec` — run command inside sandbox | safe testing (from Cua) |

---

## 7. FILE SYSTEM — Navigate and Manipulate Files

### HAVE
- `get_disk_usage` — all drives (total, free, used %)
- `open_file_explorer` — open Explorer to path
- `open_file_location` — open Explorer with file selected
- `list_directory` — list files/subdirs (name, size, is_dir, mod_time, mode)
- `read_file` — format-aware (txt, docx, xlsx, pdf, images via OCR), pagination
- `write_file` — format-aware write (txt, docx, xlsx, pdf), requires `overwrite=true` for existing
- `find_files` — recursive glob search (e.g. `*.go`, `**/*.md`)
- `copy_file` — copy file or directory recursively
- `move_file` — move/rename file or directory
- `delete_file` — move to Recycle Bin (SHFileOperationW)
- `create_directory` — mkdir -p
- `get_file_info` — size, mod_time, is_dir, mode

### NEXT
| Tool | Why |
|------|-----|
| `append_to_file` — add content to end of file | logging |
| `read_file_lines` — read specific line range | large file navigation |

### FAR
| Tool | Why |
|------|-----|
| `download_file` — HTTP download to local path | fetch remote files |
| `archive_create` / `archive_extract` — zip/tar,7z | compression |
| `watch_directory` — file system watcher | monitor for changes |
| `read_file_binary` — read as base64 | binary file handling |
| `check_disk_health` — SMART, error counts | diagnostics |
| `get_recycle_bin` — list recycle bin contents | recovery |

---

## 8. CLIPBOARD — Copy and Paste

### HAVE
- `get_clipboard` — text content
- `set_clipboard` — write text

### NEXT
| Tool | Why |
|------|-----|
| `get_clipboard_image` — read image from clipboard | image copy operations |
| `get_clipboard_files` — read copied file paths | file cut/copy |
| `get_clipboard_formats` — list available formats | clipboard inspection |
| `clear_clipboard` — empty clipboard | security cleanup |

### FAR
| Tool | Why |
|------|-----|
| `clipboard_history` — access clipboard history | multi-paste |
| `set_clipboard_image` — copy image to clipboard | image editing context |
| `set_clipboard_files` — copy files to clipboard | file operations |

---

## 9. AUDIO — Hear and Speak

### HAVE
- `get_volume` / `set_volume` — system volume 0-100
- `set_mute` — toggle mute (sets to 0 or 50)
- `list_audio_devices` — render + capture devices
- `set_default_audio_device` — set default render device

### NEXT
| Tool | Why |
|------|-----|
| `get_microphone_mute` / `set_microphone_mute` | mic control for voice |
| `play_sound` — play a .wav file or system sound | audio feedback |
| `play_beep` — beep at frequency/duration | alerts |
| `get_audio_levels` — current VU meter level | volume visualization |
| `record_audio` — record from microphone to file | voice capture |

### FAR
| Tool | Why |
|------|-----|
| `text_to_speech` — SAPI/WinRT speech synthesis | speak to user |
| `speech_to_text` — WinRT speech recognition | transcribe microphone |
| `get_system_sounds` — list available system sounds | customization |
| `set_audio_device_volume` — per-device volume | granular control |
| `audio_mixer` — per-app volume control | app-specific audio |

---

## 10. TEXT-TO-SPEECH & SPEECH-TO-TEXT — Voice Interaction

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `speak` — TTS via WinRT SpeechSynthesis | voice feedback to user |
| `speak_to_file` — generate speech .wav | save speech output |
| `list_tts_voices` — available voices | selection |
| `set_tts_voice` — choose voice | personalization |

### FAR
| Tool | Why |
|------|-----|
| `transcribe_microphone` — live speech→text | voice commands |
| `transcribe_file` — transcribe audio file | offline transcription |
| `list_speech_languages` — supported recognition languages | multi-lingual |
| `continuous_speech_recognition` — real-time streaming | always-listening mode |

---

## 11. POWER & SYSTEM — Control the Machine

### HAVE
- `shutdown` / `restart` / `sleep` / `hibernate`
- `get_uptime` — time since boot
- `get_idle_time` — time since last user input
- `get_battery` — percentage, charging, AC status
- `get_system_info` — hostname, OS, total/free RAM
- `get_disk_usage` — all drives
- `lock_workstation`

### NEXT
| Tool | Why |
|------|-----|
| `set_screen_power` — turn monitor on/off | power saving |
| `get_power_scheme` — current power plan GUID | power management |
| `set_power_scheme` — high performance / balanced / power saver | performance control |
| `get_computer_name` / `set_computer_name` | system identity |
| `get_timezone` / `set_timezone` | time management |
| `get_system_locale` — current locale | localization awareness |

### FAR
| Tool | Why |
|------|-----|
| `get_cpu_usage` / `get_memory_usage` | performance monitoring |
| `get_installed_updates` — Windows Update status | patch management |
| `get_event_log` — tail system/application logs | diagnostics |
| `get_os_version` — detailed version/build | compatibility |
| `screen_off` / `screen_on` — display power state | energy saving |
| `get_system_uptime_seconds` (already have via duration) | — |

---

## 12. NETWORK — Connect and Communicate

### HAVE
- `get_network_info` — hostname, IPs, DNS, gateway
- `ping` — ICMP reachability

### NEXT
| Tool | Why |
|------|-----|
| `get_wifi_ssid` — connected WiFi network name | connectivity awareness |
| `get_wifi_signal_strength` — RSSI | signal quality |
| `list_network_adapters` — adapters + status | diagnostics |
| `get_active_network_connections` — active TCP/UDP | app network awareness |
| `get_firewall_status` — Windows Defender Firewall state | security check |
| `get_proxy_settings` — current proxy config | internet access |
| `speed_test` — network speed (download/upload) | bandwidth check |

### FAR
| Tool | Why |
|------|-----|
| `set_static_ip` / `set_dhcp` | network configuration |
| `enable_network_adapter` / `disable_network_adapter` | connectivity control |
| `get_public_ip` — external IP | internet presence |
| `connect_to_wifi` / `disconnect_wifi` | wireless management |
| `traceroute` / `nslookup` — network diagnostics | troubleshooting |

---

## 13. REGISTRY — System Configuration

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `registry_read` — read a registry value | system settings |
| `registry_list_keys` — enumerate subkeys | registry browsing |

### FAR
| Tool | Why |
|------|-----|
| `registry_write` / `registry_delete` | system configuration |

---

## 14. ENVIRONMENT — User and System Environment

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `get_env` — get a single env var | config awareness |
| `list_env` — all env vars | full environment |
| `set_env` — set env var for current process | runtime config |

### FAR
| Tool | Why |
|------|-----|
| `set_env_permanent` — persistent env var | system configuration |
| `get_path` / `set_path` — PATH management | executable lookup |

---

## 15. UI AUTOMATION — Interact with Controls

### HAVE
- `uia_find` — find elements by name, automation_id, control_type → bounding rect + properties
- `uia_get_text` — read text from a UI element
- `uia_invoke` — click/invoke a button or control via Invoke/Toggle/Click pattern

### NEXT
| Tool | Why |
|------|-----|
| `uia_get_focused_control` — AutomationElement for focused element | know what's focused |
| `uia_get_text` — get text from a text control | read text fields |
| `uia_set_text` — set text in a text control | input without keyboard simulation |
| `uia_invoke` — click a button by AutomationId/Name | reliable click on controls |
| `uia_get_children` — list child elements of a window | explore UI structure |

### FAR
| Tool | Why |
|------|-----|
| `uia_select` — select from combobox/listbox | dropdown selection |
| `uia_get_table` — read table/grid content | data extraction |
| `uia_scroll` / `uia_select_tab` | control interaction |
| `uia_wait_for_element` — wait for control by AutomationId | robust automation |
| `uia_get_bounding_rect` — get element screen rect | positioning |

---

## 16. TASKBAR & START MENU — Shell Interaction

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `taskbar_search` — type in taskbar search box | quick access |
| `start_menu_open` / `start_menu_close` | start menu access |
| `get_pinned_taskbar_items` — list pinned apps | quick launch awareness |

### FAR
| Tool | Why |
|------|-----|
| `pin_to_taskbar` / `unpin_from_taskbar` | taskbar customization |
| `get_jump_list` — right-click menu for taskbar item | recent files, tasks |
| `action_center_open` / `action_center_close` | notification center |
| `clock_flyout` — open calendar/clock | date/time access |
| `system_tray_open` — show hidden icons | tray access |

---

## 17. NOTIFICATIONS — Alert the User

### HAVE
- `show_notification` — blocking MessageBox

### NEXT
| Tool | Why |
|------|-----|
| `toast_notify` — Windows Toast notification (non-blocking) | modern notifications |
| `toast_dismiss` — dismiss a notification | clean up |

### FAR
| Tool | Why |
|------|-----|
| `get_notification_history` — read notification center | missed alerts |

---

## 18. USB & DEVICES — Peripheral Management

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `list_usb_devices` — connected USB devices | peripheral awareness |
| `eject_usb` — safely remove USB drive | safe removal |

### FAR
| Tool | Why |
|------|-----|
| `list_printers` / `get_default_printer` | printing awareness |
| `get_connected_displays` — external monitors | display config |
| `list_bluetooth_devices` — paired BT devices | wireless peripherals |

---

## 19. TIME & DATE — System Clock

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `get_system_time` — current date/time | time awareness |
| `get_timezone` | timezone awareness |

### FAR
| Tool | Why |
|------|-----|
| `set_system_time` | time synchronization |
| `get_uptime_display` — human-readable uptime | user communication |

---

## 20. ACCESSIBILITY — Ease of Access

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `get_high_contrast` — is high contrast mode on | accessibility awareness |
| `get_magnifier_state` — Magnifier on/off + zoom level | support magnification |
| `get_narrator_state` — is Narrator running | screen reader awareness |

### FAR
| Tool | Why |
|------|-----|
| `set_high_contrast` | toggle accessibility
| `get_cursor_size` / `get_text_size` | scaling awareness |

---

## 21. REMOTE SESSION — Terminal Services

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `get_session_type` — console vs RDP | know if remote |
| `get_active_sessions` — list terminal sessions | multi-user awareness |

### FAR
| Tool | Why |
|------|-----|
| `disconnect_session` / `logoff_session` | session management |

---

## 22. SECURITY & IDENTITY

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `is_admin` — check if running as admin | permission awareness |
| `get_username` — current user | identity |
| `get_user_sid` — user security identifier | identity |
| `get_user_groups` — group membership | permission awareness |
| `list_tools` — list all MCP tools with metadata | tool discovery |
| `enable_tool` / `disable_tool` — per-tool allow/deny | granular security (from Windows-MCP) |
| `set_auth_token` — configure bearer token for TCP transport | transport security (from Windows-MCP) |
| `set_tls` — configure TLS cert for TCP transport | transport encryption (from Windows-MCP) |

### FAR
| Tool | Why |
|------|-----|
| `uac_prompt` — trigger elevation dialog | admin escalation |
| `get_logged_in_users` — active users | multi-user |
| `get_bitlocker_status` — drive encryption | security status |
| `get_defender_status` — antivirus status | security awareness |
| `set_cors_origin` — restrict MCP clients by origin | remote access control |
| `set_ip_allowlist` / `set_ip_denylist` — IP-based access control | network security |

---

## 23. SCREEN — Display Hardware

### HAVE
- `list_displays` — name, res, position, primary
- `get_screen_size` — virtual width/height
- `get_screen_dpi` — per-monitor DPI + scale %
- `get_display_modes` — all available resolutions, refresh rates, color depths per monitor

### NEXT
| Tool | Why |
|------|-----|
| `get_display_orientation` — landscape/portrait | rotation |
| `get_virtual_screen_bounds` — total spanning rect | multi-monitor layout |
| `dpi_normalize_coordinates` — transform coords between monitors with different DPI | mixed-DPI setups (from Windows MCP Server) |

### FAR
| Tool | Why |
|------|-----|
| `set_display_resolution` — change resolution | display config |
| `set_display_orientation` | rotation |
| `identify_displays` — flash display number | user communication |

---

## 24. WINDOWS SHELL — OS Integration

### HAVE
- `open_file_explorer`
- `open_file_location`
- `open_url`

### NEXT
| Tool | Why |
|------|-----|
| `open_run_dialog` — Win+R | quick command |
| `open_command_prompt` / `open_powershell` | terminal access |
| `open_control_panel` — specific applet | system settings |
| `open_settings_app` — Windows Settings page | modern settings |
| `get_default_browser` | URL handling awareness |

### FAR
| Tool | Why |
|------|-----|
| `open_recycle_bin` | file recovery |
| `empty_recycle_bin` | cleanup |
| `open_task_manager` | process management |
| `get_association` — file extension→app | default program |

---

## 25. CHAINED / COMPOSITE — Higher-Level Abstractions

### HAVE
- `find_text_and_click` — OCR + click at text
- `wait_for_text` — poll OCR until text appears
- `click_menu_item` — within a window by title
- `launch_and_wait` — launch app + wait for window
- `screenshot_element` — screenshot a window
- `hover` — move + wait
- `type_and_submit` — type + Enter
- `select_all_and_type` — Ctrl+A + type

### NEXT
| Tool | Why |
|------|-----|
| `find_text_and_right_click` | right-click on OCR text |
| `find_text_and_double_click` | double-click on OCR text |
| `find_all_text` — return all matches | multi-target |
| `fill_form` — map field labels→values | form filling |
| `drag_and_drop_file` — drag file from Explorer | file operation |
| `download_and_open` — download URL then open | remote file workflow |
| `scroll_until_text_visible` — scroll + OCR poll | infinite scroll |
| `click_all_matching_text` — click every occurrence | bulk dismissals |
| `dismiss_all_notifications` — find+close | cleanup |
| `type_password` — type from env var | secure input |

### FAR
| Tool | Why |
|------|-----|
| `login_to_website` — URL + credentials → logged in page | web automation |
| `navigate_file_dialog` — Open/Save dialog → select file | file dialog handling |
| `complete_wizard` — button-by-button wizard completion | installer automation |
| `install_app` — download + run installer + accept EULA | app installation |
| `ocr_and_chain` — general OCR → tool call pipeline | agent-in-a-box |

---

## 26. VERIFICATION & FEEDBACK

### HAVE
- `auto_verify` — all 11 action tools support post-action OCR verification
- `pre_expected` — all 11 action tools support pre-action precondition check
- `expected` — configurable text/change/not_text criteria
- chain `verify` step type — retry loop with OCR diff

### NEXT
| Tool | Why |
|------|-----|
| `verify_image_diff` — screenshot comparison verification | visual change detection |
| `verify_with_timeout` — poll verification until pass/timeout | async UI transitions |

---

## 27. DEBUGGING & DIAGNOSTICS

### HAVE
| Tool | Why |
|------|-----|
| `structured_logging` — per-module structured logging to `%APPDATA%\go-mcp-computer-use\logs\` via `slog` with rotation | debug & audit trail |

### NEXT
| Tool | Why |
|------|-----|
| `log_memory` — current heap usage | memory monitoring |
| `benchmark_screenshot` — capture timing | performance baseline |
| `debug_ocr` — raw OCR output with confidence | OCR quality assessment |

### FAR
| Tool | Why |
|------|-----|
| `save_screenshot_to_file` — save debug screenshot | examination |
| `save_all_state` — windows + processes + disks | snapshot |

---

## 28. MEMORY & ML — Learn and Adapt

### HAVE
- `memory_set` / `memory_get` / `memory_search` / `memory_list` / `memory_forget` — SQLite FTS5 persistent fact store with TTL
- `agent_analyze` / `agent_suggest` / `agent_train` — adaptive engine (timing stats, success rates, coordinate prediction)
- `training_save_sample` / `training_list_samples` / `training_stats` / `training_mark_used` / `training_cleanup_noise` / `export_yolo_dataset` — auto-collect + export pipeline
- `datalog_query` / `datalog_status` / `datalog_export` — action/OCR/chain/pair datalog
- Statistical priors — element frequency + position per window
- `introspection_analyze` / `task_begin` / `task_end` — post-task mining

### NEXT
| Tool | Why |
|------|-----|
| `set_retention_policy` — auto-prune training samples older than N days | bound disk usage (from Recall) |
| `set_sensitive_content_filter` — regex patterns to redact before saving | privacy guard (from Recall) |
| `training_set_category_prompt` — per-category task prompt | richer training context |

### FAR
| Tool | Why |
|------|-----|
| `in_context_rl` — session-scoped outcome buffer for within-session learning | adaptive on-the-fly (from Agent-S) |
| `semantic_search_screenshots` — vector embedding index over past screenshots | find-by-content (from Recall) |
| `rl_training_environment` — simulated desktop for reinforcement learning | RL pipeline (from Cua) |
| `model_fine_tune` — trigger local fine-tuning of ONNX model | self-improving vision |

---

## 29. TRANSPORT & SERVER — Connectivity

### HAVE
- stdio transport only

### NEXT
| Feature | Why |
|---------|-----|
| SSE transport | run as standalone server, support remote MCP clients (from Windows-MCP) |
| Streamable HTTP transport | modern MCP transport, bidirectional streaming |
| Bearer token auth | authenticate MCP clients (from Windows-MCP) |
| TLS support | encrypt transport layer (from Windows-MCP) |
| CORS support | restrict browser-based MCP clients (from Windows-MCP) |
| `set_transport` / `get_transport_config` — runtime transport switching | flexible deployment |

### FAR
| Feature | Why |
|---------|-----|
| OAuth 2.0 + PKCE | enterprise auth (from Windows-MCP) |
| WebSocket transport | real-time bidirectional streaming |
| Unix domain socket transport | local high-performance IPC |

---

## 30. BROWSER AUTOMATION — Web Tasks

### HAVE
- `open_url` — open URL in default browser
- `find_text_and_click` — basic OCR-based browser interaction
- `wait_for_text` — poll OCR until text appears

### NEXT
| Tool | Why |
|------|-----|
| `browser_snapshot` — capture DOM snapshot of current page | structured page data (from Windows-MCP) |
| `browser_dom_get` — query DOM elements by CSS selector | fast element access (from Windows-MCP) |
| `browser_dom_get_text` — get text content of DOM element | read page content (from Windows-MCP) |
| `browser_dom_invoke` — click/submit DOM element | reliable browser interaction (from Windows-MCP) |
| `browser_dom_get_attributes` — get element attributes | page structure analysis |
| `browser_execute_js` — run JavaScript in page context | advanced page scripting |

### FAR
| Tool | Why |
|------|-----|
| `browser_new_tab` / `browser_close_tab` — tab management | multi-page workflows |
| `browser_get_cookies` / `browser_set_cookies` — session management | authenticated browsing |
| `browser_wait_for_page_load` — wait until page fully loaded | reliable navigation |
| `browser_get_console_logs` — capture console output | debugging |
| `browser_fill_form` — auto-fill form fields by label | form automation (from Browser Use) |
| `browser_download_file` — capture download from browser | file acquisition |
| CAPTCHA solving | farm out to third-party service (from Browser Use) |

---

## 31. LINUX & CONTAINER — Cross-Platform Desktop

### HAVE
— *none*

### NEXT
| Tool | Why |
|------|-----|
| `container_list` — list running containers | container awareness |
| `container_exec` — run command inside container | cross-platform execution (from Bytebot) |

### FAR
| Tool | Why |
|------|-----|
| `linux_x_desktop_control` — control Linux X11/Wayland desktops | Linux support (from Bytebot) |
| `macos_a11y_control` — control macOS via Accessibility API | macOS support |
| `container_desktop_launch` — launch desktop app inside container | isolated Linux desktop (from Bytebot) |
| `cross_platform_interface` — abstract Windows/Linux/macOS behind common API | portable agent tasks |

---

## Summary

| Domain | HAVE | NEXT | FAR | Total Possible |
|--------|------|------|-----|---------------|
| Vision | 10 | 5 | 9 | 24 |
| Mouse | 6 | 6 | 7 | 19 |
| Keyboard | 10 | 5 | 6 | 21 |
| Windows | 13 | 9 | 6 | 28 |
| Virtual Desktops | 0 | 6 | 2 | 8 |
| Processes | 4 | 7 | 8 | 19 |
| File System | 12 | 3 | 6 | 21 |
| Clipboard | 2 | 4 | 3 | 9 |
| Audio | 4 | 5 | 6 | 15 |
| TTS / STT | 0 | 4 | 4 | 8 |
| Power & System | 11 | 8 | 6 | 25 |
| Network | 2 | 8 | 5 | 15 |
| Registry | 0 | 2 | 2 | 4 |
| Environment | 0 | 3 | 2 | 5 |
| UI Automation | 3 | 4 | 5 | 12 |
| Taskbar / Start | 0 | 3 | 5 | 8 |
| Notifications | 1 | 2 | 1 | 4 |
| USB & Devices | 0 | 2 | 3 | 5 |
| Time & Date | 0 | 2 | 2 | 4 |
| Accessibility | 0 | 3 | 2 | 5 |
| Remote Session | 0 | 2 | 2 | 4 |
| Security & Identity | 0 | 8 | 6 | 14 |
| Screen (HW) | 6 | 3 | 3 | 12 |
| Windows Shell | 3 | 5 | 3 | 11 |
| Chained | 10 | 11 | 5 | 26 |
| Debugging | 1 | 3 | 2 | 6 |
| Memory & ML | 10 | 3 | 4 | 17 |
| Transport & Server | 0 | 6 | 3 | 9 |
| Browser Automation | 3 | 6 | 6 | 15 |
| Linux & Container | 0 | 2 | 4 | 6 |
| **TOTAL** | **135** | **128** | **122** | **385** |

## Strategy

1. **Build out NEXT items** — these are straightforward and high value (another ~128 tools)
2. **Error wrapping audit** — remaining Slice 4 item for consistent error feedback across all tools
3. **Security + Transport** — per-tool enable/disable + TLS + SSE are prerequisites for remote deployment
4. **Browser DOM mode** — CDP/Playwright integration for 10x faster web tasks (steal from Windows-MCP)
5. **Background control** — SendInput/UIA for non-disruptive clicks (steal from Cua)
6. **Cross-platform interface** — Linux/macOS stubs + container support (steal from Bytebot)
7. **ML model improvement** — fix YOLO11n opset 22 incompatibility, explore UI-specific fine-tuning



See [`docs/meta/backlog.md`](docs/meta/backlog.md) for the full 385-item backlog.

---

## Changelog

# Changelog

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

_... older versions truncated. See [`docs/meta/CHANGELOG.md`](docs/meta/CHANGELOG.md) for full history._


---

## Known Issues


See [`../security.md`](../security.md) for data collection controls, privacy settings, and the full reference including `set_config` options, watcher management, and noise cleanup.

## v0.2.7 — Statistical priors, noise cleanup, config gating

New in v0.2.7:

| Feature | Status | Notes |
|---------|--------|-------|
| **Statistical prior model** (`priors_stats`) | Code complete | Element frequency + position per-window, updated on every training save. Go-native, no Python. |
| **Prior confidence adjustment** | Code complete | Applied in `ONNXDetect` after NMS. Boosts expected elements, suppresses outliers. Gated by `prior_adjustment` config. |
| **`export_yolo_dataset`** | Code complete | Exports unused samples as YOLO-format dataset for external training. |
| **`training_cleanup_noise`** | Code complete | Deletes signal_level=0 samples older than N hours. Supports dry_run. |
| **`training_enabled` config** | Code complete | When `false`, disables all auto-save snapshots (actions + watcher). Default: `true`. |
| Element priors not yet verified with real accumulated data | Untested | Priors are updated asynchronously; first detections in a session have no priors loaded until the first `loadPriorsFromDB` call. |
| No periodic auto-cleanup of noise | Not implemented | `training_cleanup_noise` is manual. Could add auto-prune background goroutine later. |

## Test Session: v0.1.2 — 2026-06-27

### Tools Tested & Working (22)

| Tool | Status | Notes |
|------|--------|-------|
| `get_screen_size` | ✅ | 3200×900 (virtual — 2×1600) |
| `get_system_info` | ✅ | Hostname, OS, RAM |
| `get_cursor_position` | ✅ | Confirms second screen (x=2142) |
| `get_battery` | ✅ | "no_battery" (desktop) |
| `get_disk_usage` | ✅ | 6 drives enumerated |
| `get_idle_time` | ✅ | |
| `get_keyboard_layout` | ✅ | 00000409 (en-US) |
| `get_network_info` | ✅ | 8 IPs, hostname |
| `get_screen_dpi` | ✅ | 2 monitors, both 96dpi 100% |
| `get_uptime` | ✅ | |
| `get_volume` | ✅ | 49% |
| `list_displays` | ✅ | DISPLAY1 + DISPLAY2 (v0.1.8) |
| `list_windows` | ✅ | 24 windows |
| `list_processes` | ✅ | 195 processes |
| `get_active_window` | ✅ | OpenCode |
| `get_pixel_color` | ✅ | #2b2a33 |
| `get_clipboard` | ✅ | "ready" |
| `find_window` | ✅ | OpenCode handle 463694 |
| `get_window_state` | ✅ | Visible, maximized, rect (1592,-8,3208,860) |
| `get_display_modes` | ✅ | 37 modes for DISPLAY1 |
| `screenshot` | ✅ | Returns base64 PNG |
| `find_text_and_click` | ✅ | Found "OpenCode" and clicked |

### Tools Tested This Session (additional)

| Tool | Status | Notes |
|------|--------|-------|
| `set_clipboard` | ✅ | Write "test from go-mcp-computer-use" |
| `open_url` | ✅ | https://example.com — opens in default browser |
| `scroll` | ✅ | -3 clicks |
| `click` (left) | ✅ | 100,100 |
| `click` (double) | ✅ | 200,200 clicks=2 |
| `click` (right) | ✅ | 300,300 button=right |
| `click` (middle) | ✅ | 400,400 button=middle |
| `drag` | ✅ | (500,500) → (600,600) |
| `uia_find` | ✅ | GitHub Desktop window found |
| `wait` | ✅ | v0.1.5 — B6: Wait() calc was 1Mx too long |
| `hover` | ✅ | v0.1.5 — B6: same root cause as wait |
| `screenshot_element` | ✅ | v0.1.3 — B5: clamps to screen bounds |
| `uia_get_text` | ✅ | v0.1.7 — B4: nil pattern check added |

### Tools Not Yet Tested (3)

| Tool | Reason |
|------|--------|
| `type_text` | Interactive — needs terminal |
| `key_press` | Interactive |
| `key_sequence` | Interactive |

## Test Session: v0.2.27 — 2026-06-30

### Tools Tested (v0.2.27 additions)

| Tool | Status | Notes |
|------|--------|-------|
| `find_all_images` | ✅ | Degenerate template (1×1 constant color) → NCC skipped → ONNX detected 68+ COCO objects → OCR 200+ words. Valid template at 0.99 threshold → same cascade. |
| `find_image` | ✅ | Degenerate template → ONNX best element returned `{x:-19, y:4, w:168, h:47, score:1}`. |
| `click` (button=middle) | ✅ | No error on middle click at (100,100). |
| `scroll` (horizontal=true) | ✅ | No error with 3 clicks horizontal scroll. |
| `get_window_state` (fullscreen) | ✅ | Returns `fullscreen: false` for normal windows, `true` for borderless fullscreen games. |
| `ocr_languages` | ✅ | Native COM — 2 languages (en-GB, en-US). PowerShell fallback eliminated. |

---

## Bug Reports

### ~~B1. `get_brightness` — parse failure~~ *(fixed v0.1.4)*
**Error:** `parse brightness: strconv.Atoi: parsing "": invalid syntax`
**Root cause:** PowerShell/WMI brightness query returns empty string instead of a numeric value. Likely because the display (DVI/HDMI desktop monitor) doesn't support WMI brightness control (laptops only).
**Fix:** Return a meaningful error or handle gracefully (e.g., `"not supported"` instead of crash).

### ~~B2. `list_audio_devices` — returns null~~ *(fixed v0.1.6)*
**Result:** `{"devices":null}`
**Issue:** No audio devices enumerated. PowerShell `Get-AudioDevice -List` may not be installed on this system (requires `AudioDeviceCmdlets` module).
**Fix:** Return empty slice `[]` instead of nil slice `null`.

### ~~B3. `list_displays` — second monitor not enumerated~~ *(fixed v0.1.8)*
**Evidence:**
- Cursor position: x=2142 (primary is 1600 wide, so x≥1600 = second screen)
- OpenCode window rect: `left=1592, right=3208, width=1616` — spans across to a second screen at x~1600

But `list_displays` only returns DISPLAY1.

**Root cause:** `monitorEnumProc` callback gated processing on `mi.Flags&1 != 0` (`MONITORINFOF_PRIMARY` = 0x1). Non-primary monitors were silently skipped.
**Fix:** Removed the primary-only gate — all enumerated monitors are now included, with `Primary: mi.Flags&1 != 0` set correctly per-monitor.

### ~~B4. `uia_get_text` / `uia_invoke` — server disconnect~~ *(fixed v0.1.7)*
**Action:** 
- `uia_get_text(name="Taskbar")` — connection lost
- `uia_get_text(name="GitHub Desktop")` — connection lost  
- `uia_invoke(name="Taskbar")` — connection lost
**Root cause:** `GetCurrentPattern` returns `S_OK` with `nil` pointer when element doesn't support pattern. Code then calls `comRelease(0)` and vtbl methods on `0` — nil pointer dereference crashes MCP transport.
**Fix:** Added `p == 0` check in `getCurrentPattern()` — returns clear error instead of nil dereference.

### ~~B5. `screenshot_element` — negative coordinates rejected~~ *(fixed v0.1.3)*
**Error:** `x=-8 out of bounds (screen width=1600)`
**Context:** Firefox window handle 132490 had rect `left=-8` (window decorations positioned off-screen by Windows snap behaviour).
**Fix:** Screenshot element should clamp/clip the region to screen bounds rather than rejecting negative coordinates. A window with off-screen decorations is a normal state (Aero Snap, multi-monitor).

### ~~B6. `hover` — consistently hangs/"Tool execution aborted"~~ *(fixed v0.1.5)*
**Root cause:** `Wait()` used `int64(duration) * 10000` (where `duration` is nanoseconds), producing a timeout **1 million times too long**. `Wait(100ms)` blocked for ~27.7 hours.
**Fix:** Changed to `int64(ms) * 10000` (1ms = 10,000 × 100ns intervals). Same fix applies to B7.

### ~~B7. `wait` — "Tool execution aborted"~~ *(same as B6, fixed v0.1.5)*

### B8. `find_text_and_click` — steals focus
**Observation:** Calling `find_text_and_click` brings the target window to foreground. This is expected behavior for a computer-use tool, but worth documenting as a caveat.
**Workaround:** None — by design.

### ~~B9. Keyboard input blocked by UIPI on elevated windows~~ *(fixed v0.1.9)*
**Observation:** All `type`, `type_and_submit`, `key_press`, `select_all_and_type` return `ok` but input does not reach elevated (Administrator) PowerShell.
**Root cause:** Windows UIPI — `SendInput` with `KEYEVENTF_UNICODE` from non-elevated MCP server is silently blocked from reaching admin-elevated windows.
**Fix:** Added `isForegroundElevated()` check using `OpenProcess` + `GetTokenInformation(TokenElevation)`. Returns clear warning message instead of silent failure.

### ~~B10. `click` may silently fail on elevated windows~~ *(documented)*
**Note:** `Click` uses `SetCursorPos` + `SendInput` mouse events. Same UIPI restriction applies — no error feedback when targeting admin windows. Unlike keyboard (which always targets foreground window), mouse targets coordinates, making elevation check impractical. Run MCP server elevated to avoid this.

### B11. `KeyPress` modifier key ordering — Ctrl+C sends C before pressing Ctrl
**Observation:** `KeyPress(["CTRL", "C"])` sends `sendUnicode('C')` first, then presses Ctrl down, then releases Ctrl. The `C` arrives **before** Ctrl is held, so Ctrl+C works as just `c` in most apps/games.
**Root cause:** `KeyPress` was splitting keys into three phases: Unicode chars → VK downs → VK ups. Modifiers and their target keys were sent in separate batches, not interleaved.
**Fix applied:** Replaced batch processing with in-order processing. Modifiers are pressed immediately when encountered, then their paired keys are sent while the modifier is held. All pressed modifiers are released in reverse order after the key sequence.

### ~~B12. `KEYEVENTF_UNICODE` may not work in game engines~~ *(fixed v0.2.8)*
**Observation:** `sendUnicode` injects characters via `KEYEVENTF_UNICODE`, which synthesizes `WM_CHAR` messages. Game engines using raw input or `GetAsyncKeyState` for keyboard polling typically don't see Unicode-injected characters — they check VK codes and scan codes instead. Same issue affects terminals, code editors, and browser input fields in some configurations.
**Root cause:** All keyboard input used `KEYEVENTF_UNICODE` — letters, digits, TypeText, TypeAndSubmit. Only modifier keys and special keys (arrows, F-keys) used VK codes.
**Fix:** Removed `KEYEVENTF_UNICODE` entirely. Rewrote keyboard input to use VK codes for everything:
- Letters A-Z/a-z → `VK_A`–`VK_Z` (0x41–0x5A) with Shift state for case
- Digits and punctuation → VK codes with Shift mapping for symbols
- TypeText/TypedAndSubmit → `sendCharWithVK()` handles shift state per character
- `KeyPress` modifier order fixed: modifiers are pressed before their target keys

---

## Prompt Engineering: Learn-Once-Reuse-Forever Pattern

The MCP server exposes 120 tools, but an AI agent using them starts **cold** every session — no knowledge of:
- What windows exist on this user's desktop and where they're positioned
- How specific applications render (Firefox tab bar vs URL bar, Outlook email list vs reading pane)
- What sequences of tool calls successfully completed a task last time
- What edge cases exist (Firefox containers, UIPI elevation blocks, OCR timing)

### The Pattern

**After any successful GUI interaction sequence, the AI should:**
1. **Store the sequence** as a named macro/recipe (e.g., "open_chrome_search_google")
2. **Annotate it** with application name, window layout details, and screen dimensions
3. **Scope it** to the application so it's reusable across sessions
4. **Next time the same task is asked**, recall the stored sequence and execute it directly — no need to rediscover coordinates and timings

### Example memory entry

```
Application: Firefox (v134+ with Multi-Account Containers)
Window size: 1123x791 (positioned at x=295, y=39)
Tab bar: y≈50-70
URL bar: y≈90-110 (click at x=350, y=105 to focus)
Container new-tab: Ctrl+T bypasses popup, or click "No Container" at x=830,y=105
Bookmarks bar: y≈120-140 (when enabled)

Sequence "open_google_and_search":
1. focus_window(handle=132490)
2. click(x=350, y=105) — focus URL bar
3. type_and_submit("google.com")
4. wait(4000)
5. type_and_submit("search query")
6. wait(3000)
7. scroll(clicks=-6)
8. ocr(x=295, y=140, w=1123, h=700) — read results below URL bar
```

### Why this matters
Without this pattern, the AI wastes time and tokens rediscovering basic facts each session — where the URL bar is, that Firefox uses containers, that scroll takes negative values for down. With it, the AI builds a **living knowledge base** that compounds with every session.

## Lessons Learned (from live testing)

### L1. Screen layout awareness is critical — always survey before acting
**Problem:** Commands like `click(x,y)` / `type_text` fail silently when the AI doesn't know the screen layout — what windows exist, their positions, what UI elements are where.

**Example from session:** Firefox had:
- Window rect: `{left:295, top:39, right:1418, bottom:830}` (1123×791)
- Multi-Account Containers extension modified the new-tab `+` button behavior — clicking it showed a container picker menu instead of opening a blank tab
- The URL bar was at y≈96 (below the tab bar at y≈56), not at the very top of the window
- Tab bar labels were partially visible but non-obvious ("< Intern PocketStac", "discwc")

**Procedure for any GUI interaction:**
1. `get_window_state(handle)` — get target window position
2. `ocr(x,y,w,h)` over the window region — see what's actually displayed
3. Locate the target element (button, text field, link) from OCR coordinates
4. Click at the element's center position
5. Verify with another OCR call after action

**Firefox-specific layout (tested v134+):**
- Tab bar: y≈50-70 (depends on title bar visibility). Compact tab mode changes spacing.
- URL bar: y≈90-110. Contains: padlock icon + "about:" or URL text.
- Container extensions add a popup menu on `+` click — must click "No Container" to open a regular tab.
- Bookmarks toolbar: y≈120-140 (if enabled). Can shift content down.
- Window top (y=39 for this session) includes the OS window title bar (if not maximized).

### L2. Tools return "ok" even when the action had no visible effect
`type`, `key_press`, `type_and_submit`, `click` all return `ok` — but the input may hit the wrong element or be dropped by UIPI. Always verify with OCR/screenshot after each action.

### L3. Firefox containers intercept the `+` new tab button
Firefox Multi-Account Containers changes the new-tab `+` behavior — instead of opening a blank tab immediately, it shows a popup asking which container to use. Click "No Container" (≈x=830, y=105 in this layout) or use `Ctrl+T` which bypasses the popup.

### B13. ONNX tools require CGO (disabled in default CGO_ENABLED=0 build)

**Observation:** `go build ./...` fails with `build constraints exclude all Go files` for `github.com/yalue/onnxruntime_go` when `CGO_ENABLED=0`. The onnxruntime_go library uses cgo for native shared library bindings.

**Impact:** ONNX ML tools (`onnx_detect`, `onnx_status`, `onnx_download`) are excluded from CGO-free builds. All other tools work.

**Workaround:** Build with `CGO_ENABLED=1` and a C compiler:
- **Zig cc:** `CC="zig cc" CGO_ENABLED=1 go build -o mcp-server.exe .\cmd\mcp-server\`
- **GCC (Mingw-w64):** `CGO_ENABLED=1 go build -o mcp-server.exe .\cmd\mcp-server\`

**Status:** Documented in v0.2.x. Not a bug — by design.

## B14. ONNX YOLO11n model uses unsupported opset 22

**Observation:** `onnx_download` pulls YOLO11n from Ultralytics v8.3.0, which exports to opset 22. `onnxruntime_go` linked against ORT 1.20.x supports only opset 21 max. Detection fails silently when running `onnx_detect`.

**Root cause:** Upstream model format drift — Ultralytics incrementally bumps ONNX opset with releases. ORT 1.20.x predates opset 22 support. The `yalue/onnxruntime_go` v1.13.0 is pinned to ORT 1.20.x API.

**Workaround:** None — MobileNetV3-small still works for UI element classification, but YOLO object detection is offline.

**Planned fix:** Either download an older YOLO11n export (opset 21) from an earlier Ultralytics release, or update ORT to 1.21+ when `onnxruntime_go` releases a compatible version.

## B15. No action verification — tools return "ok" for API call success, not for real-world effect

**Severity:** High — this is the #1 reason the server feels "hit or miss" to an AI agent.

Every action tool (`click`, `type`, `key_press`, `scroll`, `drag`, etc.) uses fire-and-forget Win32 APIs (`SendInput`, `SetCursorPos`, `keybd_event`). They return success if the API call didn't crash — not if the action had any visible effect on screen. There is no built-in verification loop between executing an action and confirming it achieved its goal.

**What tools actually return:**
- `click(x=100, y=200)` → `"ok"` means `SetCursorPos` + `SendInput` returned non-zero. It does **not** mean the click hit an interactive element, changed UI state, or reached the intended window.
- `type("hello")` → `"ok"` means key events were queued. It does **not** mean text appeared in an input field.
- `key_press(["Enter"])` → `"ok"` means the key was sent. It does **not** mean anything happened on screen.
- `screenshot` / `ocr` → Returns actual pixel/text data, but the AI has no way to confirm it matches what *should* be on screen.

**The AI's blind spots:**
1. **No "did the click land?"** — Was the intended window in focus? Was the target element at those coordinates? Did the button respond?
2. **No "did the text appear?"** — Was the right input field focused? Did the text render? Did it go to the right app?
3. **No "did the shortcut work?"** — Was the target window elevated (UIPI block)? Did the key combo reach the intended app?
4. **No "did I actually see what I think I saw?"** — OCR returns text, but if a window overlaps between capture and processing, the result is stale by the time it's returned.

**Why it compounds in chain automation:**
- A chain step `click` returns `success: true` even on empty desktop
- The next step assumes the prior action worked and compounds the error
- Chain has no built-in "verify after action" step type — poll steps must be authored manually

**What partially helps (but doesn't solve):**
- `find_text_and_click` — OCRs first, then clicks. But doesn't verify the click had an effect.
- `layout_validate` — checks stored element positions for drift. Only works on pre-registered layouts.
- Chain `poll` step — can wait for text to appear. Requires explicit authoring and knowing *what* to wait for.
- `uia_invoke` — UI Automation has better success reporting. But only works with UIA-compliant apps.
- `wait_for_text` — single-tool version of poll.

**The core problem:**
Verification is left entirely to the AI (call `ocr()` after `click()` to check). Most AI agents skip this because they trust the `"ok"` response, don't know what to verify against, and can't afford doubling token cost per action. Chain has no auto-verify primitive.

**Planned fix path:**
1. Add `auto_verify` parameter to action tools (`click`, `type`, `key_press`, `type_and_submit`, `scroll`, `drag`) — when enabled, the tool captures OCR/screenshot context before and after the action and returns a diff/verdict
2. Return `before` and `after` OCR text + screenshot region in verified action results
3. Add `verify` step type to chain — wraps any tool call with automatic before/after capture
4. Add `expected` parameter to action tools — AI specifies what it expects to see after the action (e.g., `click(x=100, y=200, expected_text="Submit")`)

**Related:** L2 — "Tools return 'ok' even when the action had no visible effect"

---

## Roadmap / Future Possibilities

### R1. Chain interruption — abort mid-sequence

The `chain` tool runs to completion or global timeout with no stop mechanism. For gameplay chains tied to game state (e.g., react to hit-stun, dodge indicator), the AI needs the ability to abort a running chain and switch to a different sequence.

**Approach (not started):**
- Add `chain_stop` tool that sets an atomic abort flag
- `ChainExecutor` checks abort flag between every step
- Poll/loop steps check abort flag before each iteration
- Returns partial result: `{"status": "aborted", "completed_steps": N, "partial_output": {...}}`

### R2. Database-backed training dataset

AI currently has no long-term memory of what VK sequences worked in previous game sessions. Each session is a cold start.

**Approach (not started):**
- SQLite schema for gameplay recordings: `recording_id`, `window_title`, `game_state_snapshot` (OCR text), `vk_sequence` (JSON), `timestamp`
- Combined OCR + VK logging during keylogger recording
- Queries for replaying sequences that succeeded in similar game states

### R3. Custom ML model for adaptive gameplay timing

Gemini suggested a Seq2Seq/LSTM model that takes "desired abilities" as input and outputs optimal VK code sequences + wait timings based on recorded human play.

**Approach (not started):**
- Export recorded gameplay sequences as labeled training data
- Train lightweight model (not LLM-scale) in isolated Docker container
- Load ONNX-exported model in Go server for real-time combo generation
- Adaptive timing: model learns per-ability cast times from human latency patterns

### R4. Smart cropping for OCR performance

Current OCR screenshots capture entire window or full screen. For game UI reading (cooldown numbers, health bars, ability tooltips), this wastes tokens and latency.

**Approach (not started):**
- Define per-application "UI regions" (e.g., NTE: bottom-center for hotbar, top-left for health)
- Crop screenshot to known UI regions before OCR
- Store region definitions in memory store for reuse across sessions

### R5. Video frame analysis

Currently all screen analysis is single-frame (OCR or ONNX on still images). Temporal awareness would enable:
- Detecting game state transitions (loading screen → gameplay, combat → menu)
- Recognizing animation tells (enemy wind-up → dodge window)
- Reading dynamic UI elements (damage numbers, cast bars)

**Approach (not started):**
- Frame buffer: keep last N screenshots in memory for temporal queries
- Simple state machine: DIFF between consecutive frames to detect large-scale changes
- Video model API integration for frame series → action prediction (long-term)

---

## Notes

- First `UIA FindAll` call costs **16-37s** (one-time per process lifetime). Subsequent calls are fast (~280ms children, ~2ms FindFirst).
- `RoInitialize` now uses `RO_INIT_MULTITHREADED` (v0.1.2 fix) to match UIA's `COINIT_MULTITHREADED` — both paths use MTA.
- OCR uses native COM WinRT path, falls back to PowerShell on failure.
- Server was built with `-ldflags="-s -w"` to reduce binary size.


See [`docs/meta/known-issues.md`](docs/meta/known-issues.md) for the full list.

---

## Security


**⚠️ This server can fully control your Windows machine.** See [`docs/security.md`](docs/security.md) for:
- Security warning and dangerous capabilities
- Elevation & UIPI (Admin vs Non-Admin)
- Data collection & privacy controls
- Agent configuration



This executable can **fully control the Windows machine it runs on**. It exposes these capabilities to any connected AI agent:

- **Read anything on screen** — screenshot, OCR, screen recording
- **Control input** — mouse clicks/moves, keyboard typing, key combos
- **Read and write clipboard** — steal or replace clipboard contents
- **Kill processes, launch executables, shutdown/restart** the machine
- **Change system audio, volume, mute, default devices**
- **Enumerate and interact with windows** — move, resize, close, find
- **Read network config, ping hosts, enumerate adapters**
- **Read disk usage, battery state, display modes**
- **Automate UI elements** via UI Automation (find/invoke buttons, read text)

**Treat this binary with the same caution as a remote-admin tool.** Only connect it to MCP clients you trust. The AI agent receiving these tools has equivalent access to a logged-in user at the keyboard. Do not expose it over a network without authentication, and never run it on a machine where you wouldn't let a remote user operate the mouse and keyboard.

## Elevation & UIPI (Admin vs Non-Admin)

Windows **UIPI** (User Interface Privilege Isolation) silently blocks input from non-elevated processes targeting elevated (Administrator) windows.

**If you run an app as Administrator** (game installers, system tools, some games like `HTGame.exe`):
→ You must also run `mcp-server.exe` **as Administrator** for mouse clicks and keyboard input to reach it.

**Without elevation:**
- **Keyboard** (`type`, `key_press`, `type_and_submit`): returns a clear warning — UIPI blocks `SendInput` with `KEYEVENTF_UNICODE`
- **Mouse** (`click`, `scroll`, `drag`): **silently fails** — no error, no feedback. The cursor moves (via `SetCursorPos`) but the click never fires

**To run elevated:** right-click your terminal/launcher → "Run as Administrator" → start `mcp-server.exe`. Or set your MCP client config to launch it through an admin shell.

**The good news:** this is a Windows security feature, not a bug. Normal (non-admin) applications work fine without elevation — browsers, terminals, editors, chat apps, file explorers, most games. You only need admin mode when targeting admin windows.

## Data Collection & Privacy Controls

The server has **no telemetry, no network calls, no data exfiltration**. All collected data stays in `%APPDATA%/go-mcp-computer-use/training/`. But users have full runtime control:

| Goal | How |
|------|-----|
| **Stop all screenshot saving** | `set_config` with `training_enabled: false` — disables auto-saves from actions AND the background watcher instantly |
| **Re-enable data collection** | `set_config` with `training_enabled: true` |
| **Stop the background watcher** | `set_config` with `watcher_enabled: false` — or `onnx_watch_stop` |
| **Start the background watcher** | `set_config` with `watcher_enabled: true` — uses interval from config or `watcher_interval_seconds` |
| **Change watcher frequency** | `set_config` with `watcher_interval_seconds: 10` — restarts watcher with new interval if running |
| **Disable ML prior learning** | `set_config` with `prior_adjustment: false` |
| **Delete noise samples** | `training_cleanup_noise` with `max_age_hours: 0` — purges low-quality frames |
| **Clear cached element data** | `memory_forget` with `scope: ui` — removes cached ONNX detection positions |
| **Inspect collected data** | `training_stats` — see counts, sources, disk usage |
| **Export collected data** | `export_yolo_dataset` — dump all images + labels to a directory |
| **Persistent disable** | Set `"training_enabled": false` in `~/.config/go-mcp-computer-use/config.json` |

The `set_config` tool can be called by the AI agent or directly by the user via their MCP client. All changes persist to disk and survive server restarts.

**For maximum privacy:** set `training_enabled: false` in config before starting the server.

## Agent Configuration

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\tools\\mcp-server.exe"
    }
  }
}
```

See [`reference/mcp-client-configs.md`](reference/mcp-client-configs.md) for per-agent config examples.


See [`docs/security.md`](docs/security.md) for the full security document.

---

## Reference Docs

- [Codebase Map](docs/reference/codebase-map.md) — 
- [Com Patterns](docs/reference/com-patterns.md) — 
- [Configuration](docs/reference/configuration.md) — 
- [Mcp Client Configs](docs/reference/mcp-client-configs.md) — **⚠️ CLI/TUI over file edits:** Several clients provide CLI commands (`claude mcp add`, `gemini mcp add`, `opencode mcp add`) to add servers interactively. Prefer these over manual JSON edits when possible — they handle schema validation, correct key names, and env var substitution. Manual file edits may not take effect if:
- [Models Setup](docs/reference/models-setup.md) — **Known incompatibility:** YOLO11n from Ultralytics v8.3.0 uses ONNX opset 22, which requires ONNX Runtime 1.21+. The bundled ORT 1.20.x only supports opsets up to 21. If YOLO detection fails, manually export YOLO11n with `opset=21` or upgrade ORT. MobileNetV3-small works with 1.20.x.
- [Scripts](docs/reference/scripts.md) — 
- [Tools](docs/reference/tools.md) — 
- [Uipi](docs/reference/uipi.md) — 
- [Versioning Strategy](docs/reference/versioning-strategy.md) — 
- [Vtable Verification](docs/reference/vtable-verification.md) — 
- [Windows Dll Ref](docs/reference/windows-dll-ref.md) — 

---

## Guides

- [Accessibility](docs/guides/accessibility.md)
- [Agent Guides](docs/guides/agent-guides.md)
- [Build](docs/guides/build.md)
- [Computer Use Guide For Ai Agents](docs/guides/computer-use-guide-for-ai-agents.md)

---

## CI/CD Pipeline


Windows-only Go project. CI builds + vets on every push/PR. Release workflow cuts a GitHub Release with the binary + changelog when tagged.

## Version File

`VERSION` at repository root — single plain-text file containing the semver string (e.g. `0.1.10`). This is the canonical source:

- `go build -ldflags="-X main.Version=$(cat VERSION)"` injects it into the binary
- CI reads it for artifact naming
- Release workflow validates the git tag matches `VERSION` before building
- `meta/CHANGELOG.md` headings must match

## Workflows

### CI (`.github/workflows/ci.yml`)

| Trigger | Action |
|---------|--------|
| Push to `main`, `v0.2.x` | Build + vet + upload artifact |
| PR to `main`, `v0.2.x` | Build + vet |

### Vtable verification

Vtable smoke tests are planned for CI integration. See [`docs/reference/vtable-verification.md`](reference/vtable-verification.md) for the proposed test suite and integration steps.

Artifact name: `mcp-server-windows-<sha>` (uses `${{ github.sha }}` in CI workflow)

### Release (`.github/workflows/release.yml`)

| Trigger | Action |
|---------|--------|
| Push tag `v*` | Build + SHA256 + GitHub Release |

Validates tag matches VERSION file. Builds with Zig cc + CGO. Extracts the corresponding section from `meta/CHANGELOG.md` as release body. Uploads `mcp-server.exe` + `mcp-server.exe.sha256`.

## Branching Strategy

```
main  ───────────────────────────────────●─── (stable releases)
                                         │
v0.2.0-alpha ──●──●──●──●──●──●──●──────┘
               (feature work, chain/memory/ML)
```

| Branch | Purpose |
|--------|---------|
| `main` | Stable — release-ready. CI runs. Tags cut here. |
| `v0.2.0-alpha` | Feature branch for v0.2.0 work (chain tool, SQLite memory store, layout validation, template library, ONNX ML). CI runs. PRs merge into `main`. |
| Feature branches | Short-lived forks from `v0.2.0-alpha`. Squash-merged. |

### Release Cycle

See [`reference/versioning-strategy.md`](reference/versioning-strategy.md) for the full release process including version bump, changelog update, pre-release gates, tagging, and pushing. The CI workflow triggers on tag push and handles the automated build + publish.

## Running CI Locally

```powershell
# Full lint (vet + build)
.\scripts\lint.ps1

# Just vet
go vet ./...

# Build with version injection
$ver = (Get-Content VERSION -Raw).Trim()
go build -ldflags="-X main.Version=$ver" -o mcp-server.exe ./cmd/mcp-server/

# Benchmark
go run ./cmd/benchmark/
```

## Cross-References

- `VERSION` — canonical version source
- `meta/CHANGELOG.md` — release notes per version
- `.govetallow` — vet allowance conventions for COM/WinRT interop
- `scripts/lint.ps1` — local CI runner (vet + build)
- `.github/workflows/ci.yml` — CI workflow
- `.github/workflows/release.yml` — release workflow
- `reference/versioning-strategy.md` — version bump rules


See [`docs/ci-cd-pipeline.md`](docs/ci-cd-pipeline.md) for the full CI/CD documentation.

<!--
Generated by scripts/gen-wiki.go
-->
