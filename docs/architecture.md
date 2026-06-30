# Architecture

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
