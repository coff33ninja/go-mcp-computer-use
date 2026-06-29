# Known Issues

## Safety & Security: Data Collection Controls

The server collects training data (screenshots + ONNX detections) for ML improvement. Users have full runtime control:

| Capability | Mechanism |
|------------|-----------|
| **Disable all auto-save** | `set_config` with `training_enabled: false` — stops action snapshots + watcher saves instantly. No restart needed. Persists to disk. |
| **Re-enable auto-save** | `set_config` with `training_enabled: true` |
| **Disable prior adjustment** | `set_config` with `prior_adjustment: false` — ONNXDetect returns raw YOLO scores without learned bias |
| **Stop watcher** | `onnx_watch_stop` — halts background screenshot polling |
| **Delete collected data** | `training_cleanup_noise` to purge noise; `memory_forget` scope=ui to clear element caches |
| **Export/download data** | `export_yolo_dataset` to inspect what was collected |
| **Persistent config** | Changes to `~/.config/go-mcp-computer-use/config.json` survive server restarts |

All training data is stored locally in `%APPDATA%/go-mcp-computer-use/training/`. No data is sent anywhere — the server has no telemetry, no network calls beyond model downloads.

## v0.2.7 — Statistical priors, noise cleanup, config gating

New in v0.2.7:

| Feature | Status | Notes |
|---------|--------|-------|
| **Statistical prior model** (`priors_stats`) | Code complete | Element frequency + position per-window, updated on every training save. Go-native, no Python. |
| **Prior confidence adjustment** | Code complete | Applied in `ONNXDetect` after NMS. Boosts expected elements, suppresses outliers. Gated by `prior_adjustment` config. |
| **`export_yolo_dataset`** | Code complete | Exports unused samples as YOLO-format dataset for external training. |
| **`training_cleanup_noise`** | Code complete | Deletes signal_level=0 samples older than N hours. Supports dry_run. |
| **`training_enabled` config** | Code complete | When `false`, disables all auto-save snapshots (actions + watcher). Default: `true`. |
| Element priors not yet verified with real accumulated data | Untested | Priors are updated asynchronously; first detections in a session have no priors loaded until the first `loadPriorsFromDB` call. |
| No periodic auto-cleanup of noise | Not implemented | `training_cleanup_noise` is manual. Could add auto-prune background goroutine later. |

## Test Session: v0.1.2 — 2026-06-27

### Tools Tested & Working (22)

| Tool | Status | Notes |
|------|--------|-------|
| `get_screen_size` | ✅ | 3200×900 (virtual — 2×1600) |
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
| `list_displays` | ✅ | DISPLAY1 + DISPLAY2 (v0.1.8) |
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

### ~~B10. `click` may silently fail on elevated windows~~ *(documented)*
**Note:** `Click` uses `SetCursorPos` + `SendInput` mouse events. Same UIPI restriction applies — no error feedback when targeting admin windows. Unlike keyboard (which always targets foreground window), mouse targets coordinates, making elevation check impractical. Run MCP server elevated to avoid this.

### B11. `KeyPress` modifier key ordering — Ctrl+C sends C before pressing Ctrl
**Observation:** `KeyPress(["CTRL", "C"])` sends `sendUnicode('C')` first, then presses Ctrl down, then releases Ctrl. The `C` arrives **before** Ctrl is held, so Ctrl+C works as just `c` in most apps/games.
**Root cause:** `KeyPress` was splitting keys into three phases: Unicode chars → VK downs → VK ups. Modifiers and their target keys were sent in separate batches, not interleaved.
**Fix applied:** Replaced batch processing with in-order processing. Modifiers are pressed immediately when encountered, then their paired keys are sent while the modifier is held. All pressed modifiers are released in reverse order after the key sequence.

### ~~B12. `KEYEVENTF_UNICODE` may not work in game engines~~ *(fixed v0.2.8)*
**Observation:** `sendUnicode` injects characters via `KEYEVENTF_UNICODE`, which synthesizes `WM_CHAR` messages. Game engines using raw input or `GetAsyncKeyState` for keyboard polling typically don't see Unicode-injected characters — they check VK codes and scan codes instead. Same issue affects terminals, code editors, and browser input fields in some configurations.
**Root cause:** All keyboard input used `KEYEVENTF_UNICODE` — letters, digits, TypeText, TypeAndSubmit. Only modifier keys and special keys (arrows, F-keys) used VK codes.
**Fix:** Removed `KEYEVENTF_UNICODE` entirely. Rewrote keyboard input to use VK codes for everything:
- Letters A-Z/a-z → `VK_A`–`VK_Z` (0x41–0x5A) with Shift state for case
- Digits and punctuation → VK codes with Shift mapping for symbols
- TypeText/TypedAndSubmit → `sendCharWithVK()` handles shift state per character
- `KeyPress` modifier order fixed: modifiers are pressed before their target keys

---

## Prompt Engineering: Learn-Once-Reuse-Forever Pattern

The MCP server exposes 103 tools, but an AI agent using them starts **cold** every session — no knowledge of:
- What windows exist on this user's desktop and where they're positioned
- How specific applications render (Firefox tab bar vs URL bar, Outlook email list vs reading pane)
- What sequences of tool calls successfully completed a task last time
- What edge cases exist (Firefox containers, UIPI elevation blocks, OCR timing)

### The Pattern

**After any successful GUI interaction sequence, the AI should:**
1. **Store the sequence** as a named macro/recipe (e.g., "open_chrome_search_google")
2. **Annotate it** with application name, window layout details, and screen dimensions
3. **Scope it** to the application so it's reusable across sessions
4. **Next time the same task is asked**, recall the stored sequence and execute it directly — no need to rediscover coordinates and timings

### Example memory entry

```
Application: Firefox (v134+ with Multi-Account Containers)
Window size: 1123x791 (positioned at x=295, y=39)
Tab bar: y≈50-70
URL bar: y≈90-110 (click at x=350, y=105 to focus)
Container new-tab: Ctrl+T bypasses popup, or click "No Container" at x=830,y=105
Bookmarks bar: y≈120-140 (when enabled)

Sequence "open_google_and_search":
1. focus_window(handle=132490)
2. click(x=350, y=105) — focus URL bar
3. type_and_submit("google.com")
4. wait(4000)
5. type_and_submit("search query")
6. wait(3000)
7. scroll(clicks=-6)
8. ocr(x=295, y=140, w=1123, h=700) — read results below URL bar
```

### Why this matters
Without this pattern, the AI wastes time and tokens rediscovering basic facts each session — where the URL bar is, that Firefox uses containers, that scroll takes negative values for down. With it, the AI builds a **living knowledge base** that compounds with every session.

## Lessons Learned (from live testing)

### L1. Screen layout awareness is critical — always survey before acting
**Problem:** Commands like `click(x,y)` / `type_text` fail silently when the AI doesn't know the screen layout — what windows exist, their positions, what UI elements are where.

**Example from session:** Firefox had:
- Window rect: `{left:295, top:39, right:1418, bottom:830}` (1123×791)
- Multi-Account Containers extension modified the new-tab `+` button behavior — clicking it showed a container picker menu instead of opening a blank tab
- The URL bar was at y≈96 (below the tab bar at y≈56), not at the very top of the window
- Tab bar labels were partially visible but non-obvious ("< Intern PocketStac", "discwc")

**Procedure for any GUI interaction:**
1. `get_window_state(handle)` — get target window position
2. `ocr(x,y,w,h)` over the window region — see what's actually displayed
3. Locate the target element (button, text field, link) from OCR coordinates
4. Click at the element's center position
5. Verify with another OCR call after action

**Firefox-specific layout (tested v134+):**
- Tab bar: y≈50-70 (depends on title bar visibility). Compact tab mode changes spacing.
- URL bar: y≈90-110. Contains: padlock icon + "about:" or URL text.
- Container extensions add a popup menu on `+` click — must click "No Container" to open a regular tab.
- Bookmarks toolbar: y≈120-140 (if enabled). Can shift content down.
- Window top (y=39 for this session) includes the OS window title bar (if not maximized).

### L2. Tools return "ok" even when the action had no visible effect
`type`, `key_press`, `type_and_submit`, `click` all return `ok` — but the input may hit the wrong element or be dropped by UIPI. Always verify with OCR/screenshot after each action.

### L3. Firefox containers intercept the `+` new tab button
Firefox Multi-Account Containers changes the new-tab `+` behavior — instead of opening a blank tab immediately, it shows a popup asking which container to use. Click "No Container" (≈x=830, y=105 in this layout) or use `Ctrl+T` which bypasses the popup.

### B14. ONNX YOLO11n model uses unsupported opset 22

**Observation:** `onnx_download` pulls YOLO11n from Ultralytics v8.3.0, which exports to opset 22. `onnxruntime_go` linked against ORT 1.20.x supports only opset 21 max. Detection fails silently when running `onnx_detect`.

**Root cause:** Upstream model format drift — Ultralytics incrementally bumps ONNX opset with releases. ORT 1.20.x predates opset 22 support. The `yalue/onnxruntime_go` v1.13.0 is pinned to ORT 1.20.x API.

**Workaround:** None — MobileNetV3-small still works for UI element classification, but YOLO object detection is offline.

**Planned fix:** Either download an older YOLO11n export (opset 21) from an earlier Ultralytics release, or update ORT to 1.21+ when `onnxruntime_go` releases a compatible version.

---

## Roadmap / Future Possibilities

### R1. Chain interruption — abort mid-sequence

The `chain` tool runs to completion or global timeout with no stop mechanism. For gameplay chains tied to game state (e.g., react to hit-stun, dodge indicator), the AI needs the ability to abort a running chain and switch to a different sequence.

**Approach (not started):**
- Add `chain_stop` tool that sets an atomic abort flag
- `ChainExecutor` checks abort flag between every step
- Poll/loop steps check abort flag before each iteration
- Returns partial result: `{"status": "aborted", "completed_steps": N, "partial_output": {...}}`

### R2. Database-backed training dataset

AI currently has no long-term memory of what VK sequences worked in previous game sessions. Each session is a cold start.

**Approach (not started):**
- SQLite schema for gameplay recordings: `recording_id`, `window_title`, `game_state_snapshot` (OCR text), `vk_sequence` (JSON), `timestamp`
- Combined OCR + VK logging during keylogger recording
- Queries for replaying sequences that succeeded in similar game states

### R3. Custom ML model for adaptive gameplay timing

Gemini suggested a Seq2Seq/LSTM model that takes "desired abilities" as input and outputs optimal VK code sequences + wait timings based on recorded human play.

**Approach (not started):**
- Export recorded gameplay sequences as labeled training data
- Train lightweight model (not LLM-scale) in isolated Docker container
- Load ONNX-exported model in Go server for real-time combo generation
- Adaptive timing: model learns per-ability cast times from human latency patterns

### R4. Smart cropping for OCR performance

Current OCR screenshots capture entire window or full screen. For game UI reading (cooldown numbers, health bars, ability tooltips), this wastes tokens and latency.

**Approach (not started):**
- Define per-application "UI regions" (e.g., NTE: bottom-center for hotbar, top-left for health)
- Crop screenshot to known UI regions before OCR
- Store region definitions in memory store for reuse across sessions

### R5. Video frame analysis

Currently all screen analysis is single-frame (OCR or ONNX on still images). Temporal awareness would enable:
- Detecting game state transitions (loading screen → gameplay, combat → menu)
- Recognizing animation tells (enemy wind-up → dodge window)
- Reading dynamic UI elements (damage numbers, cast bars)

**Approach (not started):**
- Frame buffer: keep last N screenshots in memory for temporal queries
- Simple state machine: DIFF between consecutive frames to detect large-scale changes
- Video model API integration for frame series → action prediction (long-term)

---

## Notes

- First `UIA FindAll` call costs **16-37s** (one-time per process lifetime). Subsequent calls are fast (~280ms children, ~2ms FindFirst).
- `RoInitialize` now uses `RO_INIT_MULTITHREADED` (v0.1.2 fix) to match UIA's `COINIT_MULTITHREADED` — both paths use MTA.
- OCR uses native COM WinRT path, falls back to PowerShell on failure.
- Server was built with `-ldflags="-s -w"` to reduce binary size.
