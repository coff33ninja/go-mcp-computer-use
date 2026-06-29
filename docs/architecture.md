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

```
cmd/mcp-server/main.go        — entrypoint, DPI awareness, signals
internal/server/server.go     — MCP tool registrations (108 tools)
internal/actions/
  ├── user32.go               — shared user32.dll proc loading
  ├── screenshot.go           — GDI BitBlt capture → PNG → base64
  ├── mouse.go                — SendInput click/move/scroll/drag
  ├── keyboard.go             — SendInput KEYEVENTF_UNICODE
  ├── window.go               — EnumWindows list/focus
  ├── window_ext.go           — move/resize/minimize/maximize/close/state
  ├── process.go              — list/launch/kill processes
  ├── system.go               — volume, clipboard, system info
  ├── misc.go                 — battery, displays, pixel color, notification, wait
  ├── chained.go              — composite tools (find_text_and_click, etc.)
  ├── chain.go                — chain step executor (poll, if/else, loop, variables)
  ├── validate.go             — coordinate bounds validation
  ├── uia_com.go              — COM UIAutomation (IUIAutomation via vtblMethod)
  ├── uia.go                  — UIA wrappers (find, get text, invoke)
  ├── ocr_com.go              — WinRT COM OCR pipeline
  ├── winrt.go                — WinRT infrastructure (HSTRING, RoInitialize, async)
  ├── ocr.go                  — OCR orchestration (native + PowerShell fallback)
  ├── uipi.go                 — UIPI elevation detection
  ├── audio.go                — audio devices via PowerShell
  ├── idle.go                 — GetLastInputInfo
  ├── network.go              — network info, ping
  ├── power.go                — shutdown, restart, sleep, hibernate
  ├── layout.go               — keyboard layout, screen DPI
  ├── disk.go                 — disk usage
  ├── brightness.go           — display brightness via WMI
  ├── browseruse.go           — browser automation (navigate, search, tab, url bar)
  ├── windowexploreruse.go    — File Explorer automation (focus, open path)
  ├── onnx.go                 — ONNX ML inference (YOLO detection)
  ├── watcher.go              — background ONNX detection loop with caching
  ├── memory.go               — SQLite-backed facts + element templates
  ├── training.go             — training data storage (categorized PNGs + samples.db)
  ├── ui_finder.go            — cascading element locator (memory → ONNX → OCR)
internal/config/config.go     — JSON config file
```

## Agent Architecture

See [`computer-use-guide-for-ai-agents.md`](computer-use-guide-for-ai-agents.md) for the full layered agent stack: LLM → MCP → Controller/Perception/Memory/Training → World.
