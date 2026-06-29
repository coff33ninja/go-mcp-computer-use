# TODO

## Completed

### Core
- [x] Project spec (docs/plan.md) and ADRs (001, 002)
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

### Key Logger
- [x] `keylogger_start/stop/status` — input recording and replay
- [x] `key_down/key_up` — separate hold/release

### Training & Priors
- [x] `priors_stats` — statistical prior model (element frequency + position)
- [x] `training_cleanup_noise` — purge low-quality training samples
- [x] `export_yolo_dataset` — dump unused samples as YOLO-format dataset
- [x] `set_config` — runtime privacy/training/behavior toggles

### Audio
- [x] `list_audio_devices` / `set_default_audio_device`
- [x] `get_display_modes` — available resolutions and refresh rates
- [x] `get_keyboard_layout` / `set_keyboard_layout`

### UI Automation (COM)
- [x] `uia_find` — find elements by name/automation_id/control_type
- [x] `uia_get_text` — read text from a UI element
- [x] `uia_invoke` — click/invoke buttons via UIA

### Misc
- [x] `find_image` — template matching (NCC)
- [x] `record_screen` — frame polling at interval
- [x] `get_active_window` / `focus_window_by_title`
- [x] `per-monitor DPI` awareness (`get_screen_dpi`)

## Next Up

### Slice 4 — Robustness (completed)
- [x] Coordinate bounds validation (screen dimensions)
- [x] Permission detection with clear error messages
- [x] Action timeout mechanism
- [x] JSON config file (~/.config/go-mcp-computer-use/config.json)
- [x] Structured logging
- [ ] Error wrapping audit for consistency
- [x] Graceful shutdown (handle stdin EOF)

### Slice 5 — Cross-platform
- [ ] Define platform interface
- [ ] Windows implementation behind interface
- [ ] Linux stub (xdotool/at-spi)
- [ ] macOS stub (Accessibility API)

### Docs
- [x] README.md with setup and usage
- [x] Agent config examples (opencode.json, claude_code.json, copilot.json)
- [x] Install script (Windows PowerShell)
- [x] Computer-use guide for AI agents
- [x] Update plan.md with current tool list

### Potential Future Tools
- Clipboard formats beyond text (images, files)
- Mouse gesture recognition
- WebAuthn/security key API
- Remote desktop / RDP support
