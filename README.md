# go-mcp-computer-use

MCP server for Windows desktop computer use. Exposes mouse, keyboard, screenshot, OCR, window management, and system control to AI agents via [Model Context Protocol](https://modelcontextprotocol.io).

## Features

- **Screenshot** — full screen or region capture (GDI BitBlt → PNG → base64)
- **Mouse** — click, move, scroll, drag, hover, double-click, right-click
- **Keyboard** — type text, key combos (Ctrl+C, Alt+Tab), type+submit, select all+type
- **OCR** — extract text from screen via Windows.Media.Ocr (Windows 11)
- **Find & Click** — OCR + click: find text on screen and click at its location
- **Window management** — list, focus, move, resize, minimize, maximize, close, find by title
- **Clipboard** — get/set text with retry
- **System** — volume, mute, brightness, display info, battery, uptime, idle time, DPI
- **Network** — hostname, IPs, DNS, ping
- **Processes** — list, launch, kill
- **Power** — shutdown, restart, sleep, hibernate
- **File Explorer** — open explorer to a path or with a file selected
- **60+ MCP tools** — see full list below

## Requirements

- Windows 10 or 11
- Go 1.26+ (to build from source)
- No CGO, no external dependencies

## Quick Start

```powershell
# Clone and build
git clone https://github.com/coff33ninja/go-mcp-computer-use.git
cd go-mcp-computer-use
go build -o mcp-server.exe .\cmd\mcp-server\

# Run (stdio mode)
.\mcp-server.exe
```

## Configuration

Config file at `~/.config/go-mcp-computer-use/config.json`:

```json
{
  "log_level": "info",
  "mouse_speed": 500,
  "click_delay_ms": 100,
  "verify_bounds": true,
  "action_timeout_ms": 30000
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `log_level` | `info` | One of: `debug`, `info`, `warn`, `error` |
| `mouse_speed` | `500` | Mouse move speed (unused, reserved) |
| `click_delay_ms` | `100` | Delay between clicks (unused, reserved) |
| `verify_bounds` | `true` | Validate coordinates against screen bounds |
| `action_timeout_ms` | `30000` | Max time (ms) for blocking operations |

## Agent Configuration

### opencode

`~/.config/opencode/opencode.json` or `.opencode.json` in project root:

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\path\\to\\mcp-server.exe"
    }
  }
}
```

### Claude Code

`~/.claude/claude_code.json`:

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\path\\to\\mcp-server.exe"
    }
  }
}
```

### GitHub Copilot

`.github/copilot.json` in project root:

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\path\\to\\mcp-server.exe"
    }
  }
}
```

See `examples/` for more variants.

## Tools

### Screenshot & Vision
`screenshot` `get_pixel_color` `get_screen_size` `get_screen_dpi` `ocr`

### Mouse
`click` `move_mouse` `scroll` `drag` `hover`

### Keyboard
`type` `key_press` `type_and_submit` `select_all_and_type`

### Window Management
`list_windows` `focus_window` `find_window` `wait_for_window`
`move_window` `minimize_window` `maximize_window` `restore_window`
`close_window` `get_window_state` `screenshot_element`

### Chained (composite)
`find_text_and_click` `wait_for_text` `click_menu_item` `launch_and_wait`

### System
`get_volume` `set_volume` `set_mute`
`get_clipboard` `set_clipboard`
`get_brightness` `set_brightness`
`get_battery` `get_disk_usage`
`get_keyboard_layout` `set_keyboard_layout`
`get_network_info` `ping`
`get_system_info` `get_uptime` `get_idle_time`
`list_displays`
`open_url` `open_file_explorer` `open_file_location`
`show_notification` `lock_workstation`
`shutdown` `restart` `sleep` `hibernate`
`wait`

### Process Management
`launch_app` `kill_process` `list_processes`

## Build

```powershell
go build -o mcp-server.exe .\cmd\mcp-server\
```

Cross-compile from Linux/macOS (no CGO):

```bash
GOOS=windows GOARCH=amd64 go build -o mcp-server.exe ./cmd/mcp-server/
```

## Architecture

```
cmd/mcp-server/main.go        — entrypoint, signal handling
internal/server/server.go     — 61 MCP tool registrations
internal/actions/             — Win32 API implementations (no CGO)
internal/config/config.go     — JSON config file
```
