# TODO

## Completed

### Core
- [x] Project spec (plan.md) and ADRs (001, 002)
- [x] Go 1.26.4 installed, module initialized
- [x] modelcontextprotocol/go-sdk v1.6.1 dependency
- [x] Stdio transport entrypoint
- [x] Binary builds clean (~9MB), passes vet

### Screenshot
- [x] `screenshot` — full screen and region via GDI BitBlt → PNG → base64

### Mouse
- [x] `click` — SendInput (left/right, single/double)
- [x] `move_mouse` — cursor position
- [x] `scroll` — wheel up/down
- [x] `drag` — mouse_down + move + mouse_up
- [x] `hover` — move_mouse + wait(300ms)

### Keyboard
- [x] `type` — SendInput KEYBDINPUT with UTF-16
- [x] `key_press` — modifier combos (Ctrl+C, Alt+Tab, etc.)
- [x] `type_and_submit` — type + Enter
- [x] `select_all_and_type` — Ctrl+A + type

### Window Management
- [x] `list_windows` — EnumWindows + GetWindowText
- [x] `focus_window` — SetForegroundWindow
- [x] `move_window` — move/resize
- [x] `minimize_window` / `maximize_window` / `restore_window`
- [x] `close_window` / `get_window_state`
- [x] `find_window` / `wait_for_window`
- [x] `screenshot_element` — screenshot of specific window

### System
- [x] `get_system_info` — hostname, OS, RAM
- [x] `get_volume` / `set_volume` / `set_mute`
- [x] `get_clipboard` / `set_clipboard`
- [x] `get_cursor_position` / `get_screen_size`
- [x] `get_pixel_color` / `get_screen_dpi`
- [x] `open_url` / `open_file_explorer` / `open_file_location`
- [x] `show_notification` — Windows message box
- [x] `lock_workstation`
- [x] `wait` — NtDelayExecution
- [x] `get_uptime`
- [x] `shutdown` / `restart` / `sleep` / `hibernate`

### Display & Battery
- [x] `list_displays` — EnumDisplayMonitors
- [x] `get_battery` — GetSystemPowerStatus
- [x] `get_brightness` / `set_brightness` — WMI
- [x] `get_idle_time` — GetLastInputInfo

### Network
- [x] `get_network_info` — hostname, IPs, DNS, gateway
- [x] `ping` — ping host

### Processes
- [x] `list_processes` — CreateToolhelp32Snapshot
- [x] `launch_app` — CreateProcess
- [x] `kill_process` — TerminateProcess

### OCR
- [x] `ocr` — Windows.Media.Ocr via PowerShell (full screen or region)
- [x] `find_text_and_click` — OCR → find text → click
- [x] `wait_for_text` — poll OCR until text appears
- [x] `click_menu_item` — find window → OCR region → click label

### Input Language
- [x] `get_keyboard_layout` / `set_keyboard_layout`

### Disk
- [x] `get_disk_usage` — all drives (GetDiskFreeSpaceExW)

### Chained / Composite
- [x] `find_text_and_click`
- [x] `type_and_submit`
- [x] `launch_and_wait`
- [x] `screenshot_element`
- [x] `hover`
- [x] `wait_for_text`
- [x] `select_all_and_type`
- [x] `click_menu_item`

### Shared Infrastructure
- [x] `internal/actions/user32.go` — centralized user32.dll proc loading
- [x] `internal/actions/system.go` — kernel32, winmm, shell32, clipboard

### ONNX & ML
- [x] `onnx_status` / `onnx_download` / `onnx_detect` — ONNX backend
- [x] Python/Ultralytics dependency eliminated — pre-exported models only
- [x] Switched from `best.pt` (PyTorch, 7 UI classes) to `yolo11n.onnx` (COCO, 80 classes)
- [x] `savePNG` auto-saves reference PNG on zero-element detection
- [x] Background watcher: `onnx_watch_start/stop/status/cache`

### Memory & Templates
- [x] `memory_set/get/search/list/forget` — SQLite-backed memory store
- [x] `layout_validate` — window drift + OCR keyword verification
- [x] `template_store/find/list/forget` — self-growing template library

### Browser & Explorer
- [x] `browser_focus_url_bar/navigate/new_tab/search`
- [x] `explorer_focus/open_path`

### Bug Fixes
- [x] `memory_set` schema validation (value:true → explicit InputSchema)
- [x] `close_window` Win32 API fix (ShowWindowAsync → PostMessageW)
- [x] `onnx_status` global state bug (modelsDir empty on InitONNX failure)

### Chain Tool
- [x] `chain` — sequential step executor with poll/loop/if/capture

## Next Up

### Slice 4 — Robustness
- [ ] Coordinate bounds validation (screen dimensions)
- [ ] Permission detection with clear error messages
- [ ] Action timeout mechanism
- [ ] JSON config file (~/.config/go-mcp-computer-use/config.json)
- [ ] Structured logging
- [ ] Error wrapping audit for consistency
- [ ] Graceful shutdown (handle stdin EOF)

### Slice 5 — Cross-platform
- [ ] Define platform interface
- [ ] Windows implementation behind interface
- [ ] Linux stub (xdotool/at-spi)
- [ ] macOS stub (Accessibility API)

### Docs
- [ ] README.md with setup and usage
- [ ] Agent config examples (opencode.json, claude_code.json, copilot.json)
- [ ] Install script (Windows PowerShell)
- [ ] Update plan.md with current tool list

### Potential Future Tools
- Template/image matching (find image on screen)
- Screen recording / frame streaming
- Per-monitor DPI awareness
- Clipboard formats beyond text (images, files)
- Mouse gesture recognition
- Audio device management (list, set default)
- WebAuthn/security key API
- Remote desktop / RDP support
