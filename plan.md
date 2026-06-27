# go-mcp-computer-use

## Goal

An MCP server in Go that exposes desktop computer use tools (screenshot, mouse, keyboard, window management, OCR, system control) to AI coding agents via the Model Context Protocol.

## Background

AI agents (opencode, Claude Code, GitHub Copilot, Cursor, etc.) can control the desktop through a screenshot-act-repeat loop:
1. Agent calls `screenshot()` to see what's on screen
2. Agent decides what to do (click, type, drag)
3. Agent calls the corresponding tool
4. Repeat

This project implements 61 MCP tools as an MCP server, using Go's Windows API bindings with zero CGO dependency.

## Architecture

```
cmd/mcp-server/main.go        — entrypoint, stdio transport
internal/server/server.go     — MCP tool registration (61 tools)
internal/actions/             — Win32 API implementations
  ├── user32.go               — shared user32.dll proc loading
  ├── screenshot.go           — GDI BitBlt capture → PNG → base64
  ├── mouse.go                — SendInput click/move/scroll/drag
  ├── keyboard.go             — SendInput key_press/type
  ├── window.go               — EnumWindows list/focus
  ├── window_ext.go           — move/resize/minimize/maximize/close/state
  ├── system.go               — volume, clipboard, system info, processes
  ├── process.go              — list/launch/kill processes
  ├── misc.go                 — battery, displays, pixel color, notification, wait
  ├── ocr.go                  — Windows.Media.Ocr via PowerShell
  ├── chained.go              — composite tools (find_text_and_click, etc.)
  ├── brightness.go           — display brightness via WMI
  ├── idle.go                 — GetLastInputInfo for idle time
  ├── network.go              — network info, ping
  ├── power.go                — uptime, shutdown, restart, sleep, hibernate
  ├── layout.go               — keyboard layout, screen DPI
  ├── disk.go                 — disk usage
```

## Tools (61 total)

### Screenshot & Vision
`screenshot` `get_pixel_color` `get_screen_size` `get_screen_dpi` `ocr`

### Mouse & Keyboard
`click` `move_mouse` `scroll` `drag` `hover`
`type` `key_press` `type_and_submit` `select_all_and_type`

### Window Management
`list_windows` `focus_window` `find_window` `wait_for_window`
`move_window` `minimize_window` `maximize_window` `restore_window`
`close_window` `get_window_state` `screenshot_element`

### Chained / Composite
`find_text_and_click` `wait_for_text` `click_menu_item`
`launch_and_wait` `screenshot_element`

### System
`get_system_info` `get_uptime` `get_idle_time`
`get_volume` `set_volume` `set_mute`
`get_clipboard` `set_clipboard`
`get_brightness` `set_brightness`
`get_battery` `get_disk_usage`
`get_keyboard_layout` `set_keyboard_layout`
`get_network_info` `ping`
`list_displays`
`open_url` `open_file_explorer` `open_file_location`
`show_notification` `lock_workstation`
`shutdown` `restart` `sleep` `hibernate`
`wait`

### Process Management
`launch_app` `kill_process` `list_processes`

## Design Decisions

**ADR-001** — MCP SDK: `modelcontextprotocol/go-sdk` v1.6.1 (official, Google-maintained).
**ADR-002** — Win32 via `syscall.NewLazyDLL` + `golang.org/x/sys/windows`. No CGO. No COM.

## Constraints

- Windows 10/11 only (primary target)
- MCP spec 2025-11-25
- stdio transport only
- 64-bit binary
- No CGO for easy cross-compilation
- No external dependencies beyond MCP SDK (Win32 via syscall, PowerShell for OCR/WMI)

## Future Slices

### Slice 4 — Robustness
- Coordinate bounds validation, config file, structured logging, permission detection

### Slice 5 — Cross-platform
- Platform interface + Linux/macOS stubs
