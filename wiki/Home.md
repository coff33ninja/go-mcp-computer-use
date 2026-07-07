<!-- Auto-generated from project docs. Run `go run ./scripts/gen-wiki.go` to regenerate. -->

# go-mcp-computer-use Wiki


# go-mcp-computer-use

[![Go version](https://img.shields.io/github/go-mod/go-version/coff33ninja/go-mcp-computer-use?logo=go&labelColor=2d333b)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/coff33ninja/go-mcp-computer-use?logo=github&labelColor=2d333b&color=orange)](https://github.com/coff33ninja/go-mcp-computer-use/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/coff33ninja/go-mcp-computer-use/ci.yml?branch=v0.2.x&logo=github&labelColor=2d333b)](https://github.com/coff33ninja/go-mcp-computer-use/actions)
[![Windows](https://img.shields.io/badge/Windows-0078D6?logo=windows&logoColor=white&labelColor=2d333b)](https://github.com/coff33ninja/go-mcp-computer-use)
[![MCP](https://img.shields.io/badge/MCP-Server-5B5BD6?logoColor=white&labelColor=2d333b)](https://modelcontextprotocol.io)
[![Last commit](https://img.shields.io/github/last-commit/coff33ninja/go-mcp-computer-use?labelColor=2d333b&color=yellowgreen)](https://github.com/coff33ninja/go-mcp-computer-use/commits/v0.2.x)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-brightgreen?labelColor=2d333b)](https://github.com/coff33ninja/go-mcp-computer-use/pulls)

</div>

> **Built iteratively** across AI-assisted development sessions, with [`v0.1.x`](https://github.com/coff33ninja/go-mcp-computer-use/tree/v0.1.x) covering 70+ bug-fixed Win32/COM tools and [`v0.2.x`](https://github.com/coff33ninja/go-mcp-computer-use/tree/v0.2.x) (default branch) adding the chained automation pipeline, SQLite memory store, ONNX ML detection, introspection engine, adaptive ML, and the training data pipeline for user-specific model fine-tuning.
> The AI agent was guided by a curated set of quality-enforcement skills from [coff33ninja/ai-skills](https://github.com/coff33ninja/ai-skills) — anti-hallucination, anti-slop, safe-code-modifications, anti-sycophancy, code-simplification, context-engineering, don't-kill-tokens, os-awareness, anti-tool-sprawl, follow-existing-patterns, no-dead-code-removal, universal-format-lint, self-validate, verify-and-cite, and others.
>
> **Status:** v0.2.32 — 120 tools including statistical prior model, training pipeline, memory-backed UI element cache, ONNX detection, runtime privacy controls, key hold/release, input recording, set_config, YOLO dataset export, introspection engine, adaptive ML engine, OCR→command training bridge, and ONNX cascade fallback for template matching. See [`docs/reference/tools.md`](docs/reference/tools.md) for the full listing.


---

## Features

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


## Quick Links

- [Tools Reference](Tools-Reference) — all 132 MCP tools by category
- [Architecture](Architecture) — agent stack and code map
- [Project Plan](Project-Plan) — progress and prioritized work
- [Backlog](Backlog) — 385-item roadmap
- [Changelog](Changelog) — release history
- [Known Issues](Known-Issues) — bugs and workarounds
- [Security](Security) — security model and data collection
- [Build & Usage](Build-Usage) — requirements and build commands
- [Configuration](Configuration) — config file reference
- [Reference Docs](Reference-Docs) — full reference documentation
- [Guides](Guides) — usage guides
- [CI/CD Pipeline](CICD) — CI/CD workflow documentation

## Documentation


- [`WIKI.md`](WIKI.md) — auto-generated comprehensive wiki from all project docs (run `go run ./scripts/gen-wiki.go` to regenerate)
- [`docs/reference/codebase-map.md`](docs/reference/codebase-map.md) — complete tool→handler→action→file mapping for all 96 tools
- [`docs/reference/windows-dll-ref.md`](docs/reference/windows-dll-ref.md) — Windows DLL, COM, and WinRT API reference — every syscall proc, DLL, and COM interface used
- [`docs/reference/uipi.md`](docs/reference/uipi.md) — UIPI elevation detection logic, call sites, and error semantics
- [`docs/reference/com-patterns.md`](docs/reference/com-patterns.md) — COM/WinRT patterns: vtable dispatch, async polling, HSTRING/BSTR lifecycle, threading model, UIA tree traversal, IID table with usage status
- [`docs/reference/scripts.md`](docs/reference/scripts.md) — all 8 scripts: purpose, invocation, uniqueness, cross-references
- [`docs/reference/vtable-verification.md`](docs/reference/vtable-verification.md) — COM vtable stability model, 13-test suite, drift detection workflow — **read this before upgrading Windows**
- [`docs/reference/mcp-client-configs.md`](docs/reference/mcp-client-configs.md) — MCP client configuration for 19 agents (Claude, Cursor, Windsurf, Cline, Continue, OpenCode, Gemini CLI, Roo Code, Android Studio, Zed, JetBrains, Obsidian, Emacs, Sourcegraph Cody, and more) with CLI setup commands and troubleshooting
- [`docs/guides/agent-guides.md`](docs/guides/agent-guides.md) — tool subsets per task type, prompt patterns, and agent-specific workflows
- [`docs/adr/adr-001-mcp-sdk-selection.md`](docs/adr/adr-001-mcp-sdk-selection.md) — why `modelcontextprotocol/go-sdk` was chosen
- [`docs/adr/adr-002-windows-automation-strategy.md`](docs/adr/adr-002-windows-automation-strategy.md) — Windows automation approach (Win32 API + native COM/WinRT, CGO only for ONNX)
- [`docs/guides/computer-use-guide-for-ai-agents.md`](docs/guides/computer-use-guide-for-ai-agents.md) — full layered agent architecture guide
- [`docs/meta/plan.md`](docs/meta/plan.md) — project plan, progress, and prioritized work items
- [`docs/meta/backlog.md`](docs/meta/backlog.md) — 326-tool roadmap covering every desktop ability a human has on Windows
- [`docs/meta/known-issues.md`](docs/meta/known-issues.md) — known issues and workarounds
- [`docs/meta/CHANGELOG.md`](docs/meta/CHANGELOG.md) — release history



<!-- Generated by scripts/gen-wiki.go -->
