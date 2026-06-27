# go-mcp-computer-use

## Goal

An MCP server in Go that exposes desktop computer use tools (screenshot, mouse, keyboard, window management, OCR, system control) to AI coding agents via the Model Context Protocol.

## Background

AI agents (opencode, Claude Code, GitHub Copilot, Cursor, etc.) can control the desktop through a screenshot-act-repeat loop:
1. Agent calls `screenshot()` to see what's on screen
2. Agent decides what to do (click, type, drag)
3. Agent calls the corresponding tool
4. Repeat

This project implements **70 MCP tools** as an MCP server, using Go's Windows API bindings with zero CGO dependency — including native COM WinRT (OCR, UIA) and Win32 via syscall.

## Architecture

```
cmd/mcp-server/main.go        — entrypoint, stdio transport
internal/server/server.go     — MCP tool registration (69 tools + 1 chain)
internal/actions/
  ├── user32.go               — shared user32.dll proc loading
  ├── screenshot.go           — GDI BitBlt capture → PNG → base64
  ├── mouse.go                — SendInput click/move/scroll/drag
  ├── keyboard.go             — SendInput KEYEVENTF_UNICODE (key_press/type)
  ├── window.go               — EnumWindows list/focus
  ├── window_ext.go           — move/resize/minimize/maximize/close/state
  ├── system.go               — volume, clipboard, system info, processes
  ├── process.go              — list/launch/kill processes
  ├── misc.go                 — battery, displays, pixel color, notification, wait
  ├── ocr.go                  — OCR orchestration (native COM → PowerShell fallback)
  ├── ocr_com.go              — Native COM WinRT OCR pipeline (WinRT)
  ├── winrt.go                — Core WinRT infrastructure (HSTRING, RoInitialize, async)
  ├── winrt_test.go           — WinRT tests
  ├── uia.go                  — Native COM UIA wrappers (find, get text, invoke)
  ├── uia_com.go              — COM vtable dispatch, patterns, vtblMethod()
  ├── uia_com_test.go         — UIA tests
  ├── uipi.go                 — UIPI elevation detection (admin vs non-admin)
  ├── chained.go              — composite tools (find_text_and_click, etc.)
  ├── audio.go                — Audio devices via PowerShell
  ├── brightness.go           — display brightness via WMI
  ├── idle.go                 — GetLastInputInfo for idle time
  ├── network.go              — network info, ping
  ├── power.go                — uptime, shutdown, restart, sleep, hibernate
  ├── layout.go               — keyboard layout, screen DPI
  ├── disk.go                 — disk usage
  └── validate.go             — coordinate bounds validation
```

## Tools (70 total)

### Screenshot & Vision (7)
`screenshot` `get_pixel_color` `get_screen_size` `get_screen_dpi`
`ocr` `find_image` `record_screen`

### Mouse (6)
`click` `move_mouse` `scroll` `drag` `hover` `get_cursor_position`

### Keyboard (5)
`type` `key_press` `type_and_submit` `select_all_and_type`

### Window Management (11)
`list_windows` `focus_window` `find_window` `wait_for_window`
`move_window` `minimize_window` `maximize_window` `restore_window`
`close_window` `get_window_state` `screenshot_element`

### UI Automation (3)
`uia_find` `uia_get_text` `uia_invoke`

### Chained / Composite (8)
`find_text_and_click` `wait_for_text` `click_menu_item`
`launch_and_wait` `hover` `type_and_submit` `select_all_and_type`

### System (22)
`get_system_info` `get_uptime` `get_idle_time`
`get_volume` `set_volume` `set_mute` `list_audio_devices` `set_default_audio_device`
`get_clipboard` `set_clipboard`
`get_brightness` `set_brightness`
`get_battery` `get_disk_usage`
`get_keyboard_layout` `set_keyboard_layout`
`get_network_info` `ping`
`list_displays` `get_display_modes`
`open_url` `open_file_explorer` `open_file_location`
`show_notification` `lock_workstation` `wait`
`shutdown` `restart` `sleep` `hibernate`

### Process Management (3)
`launch_app` `launch_and_wait` `kill_process` `list_processes`

### Automation Pipeline (1)
`chain` — executes a sequence of steps with polling, waiting, and branching

## Design Decisions

**ADR-001** — MCP SDK: `modelcontextprotocol/go-sdk` v1.6.1 (official, Google-maintained).
**ADR-002** — Win32 via `syscall.NewLazyDLL` + `golang.org/x/sys/windows`. No CGO. COM/WinRT via raw uintptr vtbl dispatch (`vtblMethod()`). WinRT uses `RO_INIT_MULTITHREADED` to match UIA's COM apartment model.

## Versioning

```
v<major>.<minor>.<patch>
```

| Bump | When | Examples |
|------|------|----------|
| `+0.0.1` (patch) | Bug fixes, tool tweaks, doc updates, minor refactors | Fixing UIPI detection, adjusting OCR timing, renaming a tool parameter |
| `+0.1.0` (minor) | New tools, new capabilities, architecture changes, dependency adds | Adding native COM OCR, adding UIA layer, adding `chain` tool, introducing SQLite memory store |
| `+1.0.0` (major) | Stable release with proven architecture, all planned slices complete, field-tested | Full automation pipeline working, memory store battle-tested, ONNX integration verified |

**Current trajectory:** v0.1.x (bug-fix cycle on initial tools) → v0.2.0 (automation pipeline + memory + ML) → v0.3.x (iterative improvements) → v1.0.0 (stable release)

## Constraints

- Windows 10/11 only
- MCP spec 2025-11-25
- stdio transport only
- 64-bit binary
- No CGO
- No external dependencies beyond MCP SDK + `golang.org/x/sys`

## Chained Automation Pipeline

### Problem
The screenshot-act-repeat loop requires the AI agent to make 5-50+ round trips to accomplish a task like "open Chrome, search Google, scroll results, read pricing". Each round trip adds latency, token cost, and fragility.

### Solution: `chain` tool
A single MCP tool `chain` that accepts a JSON array of steps and executes them **sequentially on the server side** — no round trips between steps.

```json
{
  "steps": [
    { "tool": "type", "args": { "text": "hello" } },
    { "tool": "key_press", "args": { "keys": ["Enter"] } },
    { "tool": "wait", "args": { "ms": 2000 } },
    { "tool": "ocr", "args": {} }
  ]
}
```

### Step Types

| Type | Description | Example |
|------|-------------|---------|
| `tool` | Call any registered MCP tool | `{tool: "click", args: {x,y}}` |
| `wait` | Sleep N ms | `{wait_ms: 2000}` |
| `poll` | Poll until condition met | `{poll: {every_ms: 1000, timeout_ms: 30000, ocr_contains: "Submit"}}` |
| `if` | Conditional branch | `{if: {ocr_contains: "Error"}, then: [...], else: [...]}` |
| `loop` | Repeat N times | `{loop: {times: 5, steps: [...]}}` |
| `capture` | Save step output as variable | `{capture: "screenshot1", tool: "screenshot"}` |

### Variable substitution
Step outputs feed into subsequent steps via `{{variable}}` syntax. E.g., OCR text → click coordinates.

### State machine
Chain execution maintains state:
- `state.variables` — captured outputs
- `state.last_screenshot` — for fast diffing
- `state.step_index` — current step
- `state.errors` — per-step error log

### Example: Open Chrome → Google → Search → Scroll → Read
```json
{
  "steps": [
    { "tool": "launch_app", "args": {"path": "chrome.exe"} },
    { "tool": "wait_for_window", "args": {"title": "Google Chrome", "timeout_ms": 10000} },
    { "tool": "wait", "args": {"ms": 2000} },
    { "tool": "ocr", "args": {"x": 0, "y": 0, "w": 3200, "h": 80}, "capture": "top_bar" },
    { "tool": "click", "args": {"x": "{{top_bar.address_bar_x}}", "y": "{{top_bar.address_bar_y}}" }},
    { "tool": "type_and_submit", "args": {"text": "google.com"} },
    { "tool": "wait", "args": {"ms": 5000} },
    { "tool": "type_and_submit", "args": {"text": "call of duty pricing"} },
    { "tool": "wait", "args": {"ms": 4000} },
    { "loop": { "times": 3, "steps": [
        { "tool": "scroll", "args": {"clicks": -6} },
        { "tool": "wait", "args": {"ms": 500} }
    ]}},
    { "tool": "ocr", "args": {}, "capture": "results" }
  ],
  "timeout_ms": 60000,
  "on_error": "stop"
}
```

### Key behaviors
- **Survey-first approach**: Always OCR the target window region before clicking — tools return `ok` even when aimed at the wrong element
- **Verify after action**: Each step can optionally verify with a post-action OCR check
- **Graceful degradation**: If `find_text_and_click` fails, try `uia_find` fallback, then log and continue
- **Timeout per step + global timeout**: Prevents runaway chains
- **Error modes**: `stop` (halt on first error), `skip` (log and continue), `retry(N)` (retry step N times)

## Future Slices

### Slice 4 — Robustness (done)
- Coordinate bounds validation (`validate.go`) ✅
- Permission detection / UIPI warnings (`uipi.go`) ✅
- Action timeout mechanism 🔲
- JSON config file 🔲
- Structured logging 🔲
- Error wrapping audit 🔲
- Graceful shutdown 🔲

### Slice 5 — Automation Pipeline
- `chain` tool — sequential step executor 🔲
- Poll/loop/if primitives 🔲
- Variable capture and substitution 🔲
- OCR-guided click positioning (find element → click center) 🔲
- Chain result reporting (which steps succeeded/failed) 🔲
- Chain templates (reusable automation recipes) 🔲

### Slice 6 — Built-in Memory Store (SQLite) with Layout Validation
A persistence layer inside the MCP server so learned facts survive across AI agent sessions — no reliance on external memory tools.

| Tool | Description |
|------|-------------|
| `memory_set` | Store a fact/sequence. Fields: `key`, `value` (JSON), `scope` (app:firefox), `tags`, `ttl` (optional expiry) |
| `memory_get` | Retrieve by key + scope |
| `memory_search` | SQL FTS5 full-text search across keys, values, tags |
| `memory_list` | List entries under a scope with optional tag filter |
| `memory_forget` | Delete by key, scope, or tag pattern |

**Storage:** SQLite via `modernc.org/sqlite` (pure Go, zero CGO). Database at `%APPDATA%/go-mcp-computer-use/memory.db`. Tables:
- `facts` — key, value (JSON text), scope, tags (comma-separated), created_at, updated_at, ttl
- `sequences` — app_name, steps (JSON), window_layout (JSON), screen_size, element_signatures (JSON), verified_at, success_count, fail_count
- `element_templates` — base64 PNG templates of UI elements for visual re-identification
- FTS5 virtual table for full-text search across all text fields

**Layout Validation & Staleness Detection**
Users resize windows, update apps, change themes — stored coordinates rot. Before replaying any stored sequence, the server validates the layout hasn't changed:

1. **Window existence check** — does the target window still exist with the expected title?
2. **Size/position drift detection** — is the window still at the stored rect? If within tolerance (e.g. ±20px), adjust coordinates. If not, flag stale.
3. **Element signature verification** — for each click target, OCR a 40×40px region around the stored coordinate and verify at least one expected keyword matches. E.g., URL bar → expect "http" or "google.com" or "about:" in that region.
4. **Visual template matching** — optional: store a 32×32px template of the element (URL bar padlock + first letter) using `find_image` to verify visual match before clicking.

**Staleness resolution:**
- **Drift**: auto-adjust all coordinates by the delta
- **Mild mismatch**: fall back to `find_text_and_click` / `uia_find` to re-locate the element, update stored coordinates
- **Complete mismatch**: mark sequence stale, return `memory_get` with `confidence: "stale"` so the AI knows to re-discover

**Tiny ML / UI Pattern Recognition**
Hardcoded coordinates + OCR keywords work for known layouts but break when users have custom themes, different Firefox versions, or non-standard window arrangements. A lightweight pattern recognition layer helps the server generalize:

1. **OCR element classifier** — given a screenshot region, classify it as: `url_bar`, `tab_bar`, `search_field`, `button`, `text_block`, `list_item`, `dropdown`. Based on geometry heuristics (aspect ratio, position relative to window top, background color uniformity).
2. **Layout fingerprint hashing** — perceptual hash of the window's structural layout (tab bar position, URL bar position, content area). If the hash changes, the layout has drifted — trigger re-discovery.
3. **Click target inference** — instead of "click at x=350,y=105", store "click on the URL bar" and at replay time: OCR the top 150px of the window → find the text input region → click its center. Works regardless of window size or position.
4. **Self-growing template library** — every time the server discovers a new UI element, it crops a small template image (32×32 or 48×48 PNG) around the element center and stores it in the database via `element_templates`. Over time, the template library grows organically with real screen captures from actual usage. At replay time, `find_image` locates the element visually instead of relying on stale coordinates. The system effectively trains itself — no ML model needed.
5. **Template matching fallback chain**: for any stored element → try coordinate (fastest) → verify via OCR keywords → if mismatch, try `find_image` with stored template → if still fail, mark stale and re-discover.
6. **ONNX runtime (v2+)** — two-tier model architecture using existing pre-trained models:

   **Tier 1: UI Element Detector — `IndextDataLab/windows-ui-locator` (YOLO11s)**
   - Trained specifically on Windows-style UI screenshots (3,000 synthetic, but covers standard Win10/11 controls)
   - 7 classes: `button`, `textbox`, `checkbox`, `dropdown`, `icon`, `tab`, `menu_item`
   - mAP50: **0.9886** — near-perfect on its domain
   - 9.4M params, ~18 MB ONNX export, ~44-80ms on CPU (M2 Pro)
   - MIT licensed — no copyleft concerns
   - Purpose: given a full screenshot region, returns bounding boxes + class labels for all visible UI elements
   - **Usage in pipeline**: before clicking a stored coordinate, run detector → verify expected element type exists at that location → if not, template-match fallback → if still fail, mark stale

   **Tier 2: Element Type Classifier — `diogoneno/gui-element-classifier` (MobileNetV3-small)**
   - 15 element types: `button`, `checkbox`, `container`, `dropdown`, `icon_button`, `image`, `label`, `link`, `menu_item`, `scrollbar`, `slider`, `tab`, `text_input`, `toggle`, `unknown`
   - Only 6 MB ONNX model, ~5ms per crop on CPU
   - Acts as a refinement layer: crop each detected bounding box from Tier 1 → classify precisely
   - Catches what YOLO misses: `scrollbar`, `slider`, `toggle`, `text_input` vs plain `textbox`

   **Why two-tier:** The YOLO detector answers "what is this region?" → gives approximate element type + location. The classifier answers "exactly what control is this?" → refines the label + confidence. Together they provide a fallback when OCR keywords fail or coordinates drift.

   **Hierarchical validation chain (3-layer → 100x layer):**
   ```
   Layer 1: Coordinate check (O(1)) — is window at expected position?
   Layer 2: OCR keyword match (O(n)) — does expected text exist at stored coord?
   Layer 3: Template matching (find_image) (O(m)) — visual match with stored crop
   Layer 4: YOLO detector ~9M params (O(screenshot)) — full UI element detection
   Layer 5: MobileNetV3 classifier ~3M params (O(crops)) — per-element type refinement
   ```
   Each layer catches what the previous layer misses. The AI decides how deep to go based on confidence score — if Layer 2 passes (OCR keyword matches), skip Layers 3-5 and move fast. Only on mismatch/failure does the pipeline descend deeper.

   **Go ONNX runtime:** `github.com/yalue/onnxruntime_go` — pure Go bindings to ONNX Runtime shared library (~10-15MB). Both models above can be downloaded on first run or bundled. The architecture supports plugging any ONNX model without code changes.

**Element signature storage schema:**
```json
{
  "window_title": "Qwen — Mozilla Firefox",
  "window_rect": {"left": 295, "top": 39, "right": 1418, "bottom": 830},
  "layout_hash": "a1b2c3d4",
  "elements": [
    {
      "id": "url_bar",
      "type": "text_input",
      "stored_coord": {"x": 350, "y": 105},
      "signature_keywords": ["http", "google", "about:", "qwen", "firefox"],
      "expected_ocr_region": {"x": 280, "y": 90, "w": 600, "h": 30},
      "template_b64": "...32×32 PNG...",
      "rel_position": {"from_top": 66, "from_left": 55}
    },
    {
      "id": "new_tab_button",
      "type": "button",
      "stored_coord": {"x": 1370, "y": 55},
      "note": "Firefox containers intercepts + click — use Ctrl+T instead",
      "workaround": "key_press(Ctrl+T)"
    }
  ]
}
```

**Auto-save pattern:** After any successful chain execution, the `chain` tool automatically inserts into `sequences` with the steps, element signatures, layout hash, and screen context. Next session, the AI runs `memory_search("firefox open google")`, gets the sequence back, the server validates the layout hasn't drifted, and if clean — replays with zero rediscovery.

**Dependency cost:** `modernc.org/sqlite` adds ~2MB to the binary, zero CGO. Worth it for ACID, indexing, and FTS5.

### Slice 7 — Cross-platform
- Platform interface + Linux/macOS stubs 🔲
