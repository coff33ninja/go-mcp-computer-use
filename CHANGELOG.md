# Changelog

## [0.2.0] - 2026-06-27

### Fixed

- OCR PowerShell script: properly loads WinRT types via `WindowsRuntimeSystemExtensions.GetAwaiter` with `MakeGenericMethod`, fixing OCR on systems where WinRT async extension methods don't resolve in PowerShell 5.1
- Go raw string literal: avoids backtick in `IAsyncOperation`1` by using `-like` wildcard matching

### Changed

- Bumped version to 0.2.0

## [0.1.0] - 2026-06-27

### Added

- Screenshot (full + region) via GDI BitBlt
- Mouse control: click, move, scroll, drag, hover
- Keyboard input: type, key_press, type_and_submit, select_all_and_type
- OCR via Windows.Media.Ocr with language support
- Template matching via normalized cross-correlation
- find_text_and_click, wait_for_text, click_menu_item, launch_and_wait
- Screen recording (duration_ms + interval_ms → base64 frames)
- Window management: list, focus, find, move, resize, minimize, maximize, restore, close, get_state
- Audio devices: list playback/recording, set default
- Clipboard: get/set with retry + timeout
- System: volume, mute, brightness, battery, disk, DPI, display info, uptime, idle
- Network: hostname, IPs, DNS, gateway, ping
- Processes: list, launch, kill
- Power: shutdown, restart, sleep, hibernate
- Per-monitor DPI awareness
- UI Automation via PowerShell: find elements, get text, invoke
- get_display_modes tool (69th tool) — enumerate all display modes
- Config file: `~/.config/go-mcp-computer-use/config.json`
- Install script: `scripts/install.ps1` with Zig cc support

### Changed

- syscall hardening: `ptr()` helper for safe unsafe.Pointer conversion
- performance optimizations across all action modules
- README with comprehensive tool listing and security warning
- MCP client configs documentation for 19 agents

### Security

- Added SECURITY WARNING section to README detailing dangerous capabilities
