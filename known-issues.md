# Known Issues

## Test Session: v0.1.2 — 2026-06-27

### Tools Tested & Working (22)

| Tool | Status | Notes |
|------|--------|-------|
| `get_screen_size` | ✅ | 1600×900 (primary) |
| `get_system_info` | ✅ | Hostname, OS, RAM |
| `get_cursor_position` | ✅ | Confirms second screen (x=2142) |
| `get_battery` | ✅ | "no_battery" (desktop) |
| `get_disk_usage` | ✅ | 6 drives enumerated |
| `get_idle_time` | ✅ | |
| `get_keyboard_layout` | ✅ | 00000409 (en-US) |
| `get_network_info` | ✅ | 8 IPs, hostname |
| `get_screen_dpi` | ✅ | 2 monitors, both 96dpi 100% |
| `get_uptime` | ✅ | |
| `get_volume` | ✅ | 49% |
| `list_displays` | ✅ | Only DISPLAY1 returned (**bug** - see below) |
| `list_windows` | ✅ | 24 windows |
| `list_processes` | ✅ | 195 processes |
| `get_active_window` | ✅ | OpenCode |
| `get_pixel_color` | ✅ | #2b2a33 |
| `get_clipboard` | ✅ | "ready" |
| `find_window` | ✅ | OpenCode handle 463694 |
| `get_window_state` | ✅ | Visible, maximized, rect (1592,-8,3208,860) |
| `get_display_modes` | ✅ | 37 modes for DISPLAY1 |
| `screenshot` | ✅ | Returns base64 PNG |
| `find_text_and_click` | ✅ | Found "OpenCode" and clicked |

### Tools Tested This Session (additional)

| Tool | Status | Notes |
|------|--------|-------|
| `set_clipboard` | ✅ | Write "test from go-mcp-computer-use" |
| `open_url` | ✅ | https://example.com — opens in default browser |
| `scroll` | ✅ | -3 clicks |
| `click` (left) | ✅ | 100,100 |
| `click` (double) | ✅ | 200,200 clicks=2 |
| `click` (right) | ✅ | 300,300 button=right |
| `click` (middle) | ✅ | 400,400 button=middle |
| `drag` | ✅ | (500,500) → (600,600) |
| `uia_find` | ✅ | GitHub Desktop window found |
| `wait` | ✅ | v0.1.5 — B6: Wait() calc was 1Mx too long |
| `hover` | ✅ | v0.1.5 — B6: same root cause as wait |
| `screenshot_element` | ✅ | v0.1.3 — B5: clamps to screen bounds |
| `uia_get_text` | ✅ | v0.1.7 — B4: nil pattern check added |

### Tools Not Yet Tested (7)

| Tool | Reason |
|------|--------|
| `type_text` | Interactive — needs terminal |
| `key_press` | Interactive |
| `key_sequence` | Interactive |
| `find_image` | Needs template image |
| `uia_invoke` | ✅ | B4 fixed — nil pattern check added |
| `type_text` | ✅ | Returns ok — input blocked by UIPI on elevated targets (B9) |
| `type_and_submit` | ✅ | Returns ok — same UIPI restriction (B9) |
| `key_press` | ✅ | Ctrl+C, Enter, VolumeMute all return ok |
| `select_all_and_type` | ⚠️ | "Tool execution aborted" — possible client timeout |
| `find_text_and_click` | ✅ | Works (session 1) |

---

## Bug Reports

### ~~B1. `get_brightness` — parse failure~~ *(fixed v0.1.4)*
**Error:** `parse brightness: strconv.Atoi: parsing "": invalid syntax`
**Root cause:** PowerShell/WMI brightness query returns empty string instead of a numeric value. Likely because the display (DVI/HDMI desktop monitor) doesn't support WMI brightness control (laptops only).
**Fix:** Return a meaningful error or handle gracefully (e.g., `"not supported"` instead of crash).

### ~~B2. `list_audio_devices` — returns null~~ *(fixed v0.1.6)*
**Result:** `{"devices":null}`
**Issue:** No audio devices enumerated. PowerShell `Get-AudioDevice -List` may not be installed on this system (requires `AudioDeviceCmdlets` module).
**Fix:** Return empty slice `[]` instead of nil slice `null`.

### ~~B3. `list_displays` — second monitor not enumerated~~ *(fixed v0.1.8)*
**Evidence:**
- Cursor position: x=2142 (primary is 1600 wide, so x≥1600 = second screen)
- OpenCode window rect: `left=1592, right=3208, width=1616` — spans across to a second screen at x~1600

But `list_displays` only returns DISPLAY1.

**Root cause:** `monitorEnumProc` callback gated processing on `mi.Flags&1 != 0` (`MONITORINFOF_PRIMARY` = 0x1). Non-primary monitors were silently skipped.
**Fix:** Removed the primary-only gate — all enumerated monitors are now included, with `Primary: mi.Flags&1 != 0` set correctly per-monitor.

### ~~B4. `uia_get_text` / `uia_invoke` — server disconnect~~ *(fixed v0.1.7)*
**Action:** 
- `uia_get_text(name="Taskbar")` — connection lost
- `uia_get_text(name="GitHub Desktop")` — connection lost  
- `uia_invoke(name="Taskbar")` — connection lost
**Root cause:** `GetCurrentPattern` returns `S_OK` with `nil` pointer when element doesn't support pattern. Code then calls `comRelease(0)` and vtbl methods on `0` — nil pointer dereference crashes MCP transport.
**Fix:** Added `p == 0` check in `getCurrentPattern()` — returns clear error instead of nil dereference.

### ~~B5. `screenshot_element` — negative coordinates rejected~~ *(fixed v0.1.3)*
**Error:** `x=-8 out of bounds (screen width=1600)`
**Context:** Firefox window handle 132490 had rect `left=-8` (window decorations positioned off-screen by Windows snap behaviour).
**Fix:** Screenshot element should clamp/clip the region to screen bounds rather than rejecting negative coordinates. A window with off-screen decorations is a normal state (Aero Snap, multi-monitor).

### ~~B6. `hover` — consistently hangs/"Tool execution aborted"~~ *(fixed v0.1.5)*
**Root cause:** `Wait()` used `int64(duration) * 10000` (where `duration` is nanoseconds), producing a timeout **1 million times too long**. `Wait(100ms)` blocked for ~27.7 hours.
**Fix:** Changed to `int64(ms) * 10000` (1ms = 10,000 × 100ns intervals). Same fix applies to B7.

### ~~B7. `wait` — "Tool execution aborted"~~ *(same as B6, fixed v0.1.5)*

### B8. `find_text_and_click` — steals focus
**Observation:** Calling `find_text_and_click` brings the target window to foreground. This is expected behavior for a computer-use tool, but worth documenting as a caveat.
**Workaround:** None — by design.

### ~~B9. Keyboard input blocked by UIPI on elevated windows~~ *(fixed v0.1.9)*
**Observation:** All `type`, `type_and_submit`, `key_press`, `select_all_and_type` return `ok` but input does not reach elevated (Administrator) PowerShell.
**Root cause:** Windows UIPI — `SendInput` with `KEYEVENTF_UNICODE` from non-elevated MCP server is silently blocked from reaching admin-elevated windows.
**Fix:** Added `isForegroundElevated()` check using `OpenProcess` + `GetTokenInformation(TokenElevation)`. Returns clear warning message instead of silent failure.

### B10. `click` may silently fail on elevated windows
**Note:** `Click` uses `SetCursorPos` + `SendInput` mouse events. Same UIPI restriction applies — no error feedback when targeting admin windows.

---

## Notes

- First `UIA FindAll` call costs **16-37s** (one-time per process lifetime). Subsequent calls are fast (~280ms children, ~2ms FindFirst).
- `RoInitialize` now uses `RO_INIT_MULTITHREADED` (v0.1.2 fix) to match UIA's `COINIT_MULTITHREADED` — both paths use MTA.
- OCR uses native COM WinRT path, falls back to PowerShell on failure.
- Server was built with `-ldflags="-s -w"` to reduce binary size.
