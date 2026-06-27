# go-mcp-computer-use

MCP server for Windows desktop computer use. Exposes mouse, keyboard, screenshot, OCR, template matching, window management, system control, and screen recording to AI agents via [Model Context Protocol](https://modelcontextprotocol.io).

## Features

- **Screenshot** — full screen or region capture (GDI BitBlt → PNG → base64)
- **Mouse** — click, move, scroll, drag, hover
- **Keyboard** — type, key combos (Ctrl+C, Alt+Tab), type+submit, select all+type
- **OCR** — extract text via Windows.Media.Ocr, optional language (en-US, ja-JP, fr-FR...)
- **Template matching** — find an image on screen via NCC (normalized cross-correlation)
- **Find & Click** — OCR + click: find text on screen and click it  
- **Chained tools** — `find_text_and_click`, `launch_and_wait`, `wait_for_text`, `click_menu_item`
- **Screen recording** — capture frames at interval for a duration
- **Window management** — list, focus, move, resize, min/max/restore, close, find
- **Audio devices** — list playback/recording devices, set default
- **Clipboard** — get/set text with retry + timeout
- **System** — volume, mute, brightness, battery, disk, DPI, display info, uptime, idle
- **Network** — hostname, IPs, DNS, gateway, ping
- **Processes** — list, launch, kill
- **Power** — shutdown, restart, sleep, hibernate
- **Per-monitor DPI** — per-monitor DPI awareness, scale reporting
- **UI Automation** — find elements by name/automationID, get text, invoke buttons via UIA
- **68 MCP tools** — full list below

## Requirements

- Windows 10 or 11
- Go 1.26+ (to build from source)
- No CGO, no external dependencies (Zig optional for CGO)

## Quick Start

```powershell
git clone https://github.com/coff33ninja/go-mcp-computer-use.git
cd go-mcp-computer-use
go build -o mcp-server.exe .\cmd\mcp-server\
.\mcp-server.exe
```

Or use the install script:

```powershell
.\scripts\install.ps1
.\scripts\install.ps1 -UseZig              # with Zig cc for CGO
```

## Configuration

`~/.config/go-mcp-computer-use/config.json`:

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
| `verify_bounds` | `true` | Validate coordinates against screen bounds |
| `action_timeout_ms` | `30000` | Max time (ms) for blocking operations |

## Tools (69)

### Screenshot & Vision
`screenshot` `get_screen_size` `get_pixel_color` `get_screen_dpi` `get_display_modes` `ocr` `find_image`

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

### OCR & Language
`ocr` (supports `language` param: en-US, ja-JP, fr-FR...)
`find_text_and_click` `wait_for_text` `click_menu_item` (all pass through language)

### Audio
`list_audio_devices` `set_default_audio_device`

### Screen Recording
`record_screen` (duration_ms, interval_ms → base64 frames)

### Template Matching
`find_image` (template_b64 as base64 PNG, threshold 0-1)

### System
`get_volume` `set_volume` `set_mute`
`get_clipboard` `set_clipboard`
`get_brightness` `set_brightness`
`get_battery` `get_disk_usage`
`get_keyboard_layout` `set_keyboard_layout`
`get_network_info` `ping` `get_system_info`
`get_uptime` `get_idle_time`
`list_displays` `get_screen_dpi` (per-monitor)
`open_url` `open_file_explorer` `open_file_location`
`show_notification` `lock_workstation`
`shutdown` `restart` `sleep` `hibernate` `wait`

### Process Management
`launch_app` `kill_process` `list_processes`

### UI Automation
`uia_find` `uia_get_text` `uia_invoke`

## Agent Configuration

Examples in `examples/`:
- `opencode.json`
- `claude_code.json`
- `copilot.json`

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\tools\\mcp-server.exe"
    }
  }
}
```

## Architecture

```
cmd/mcp-server/main.go        — entrypoint, DPI awareness, signals
internal/server/server.go     — 68 MCP tool registrations
internal/actions/             — Win32 API + PowerShell (no CGO, no COM)
internal/actions/uia.go       — PowerShell UI Automation (find, get_text, invoke)
internal/config/config.go     — JSON config file
```

## Build

```powershell
go build -o mcp-server.exe .\cmd\mcp-server\
```

Cross-compile from Linux/macOS (no CGO):

```bash
GOOS=windows GOARCH=amd64 go build -o mcp-server.exe ./cmd/mcp-server/
```

Cross-compile with CGO via Zig:

```bash
CC="zig cc" CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o mcp-server.exe ./cmd/mcp-server/
```
