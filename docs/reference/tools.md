# Tools (120)

Auto-generated from `internal/server/server.go`. Total: **120 tools**.

For the full tool→handler→action function→source file mapping, see [`codebase-map.md`](codebase-map.md).

## Screenshot & Vision (10)

- `find_all_images` — Find ALL occurrences of a template image on screen using NCC template matching. Provide template as base64 PNG. Returns array of matches with coordinates and scores.
- `find_image` — Find a template image on screen using NCC template matching. Provide template as base64 PNG. Returns coordinates of best match.
- `get_display_modes` — Get all available display modes (resolution, refresh rate, color depth) for a monitor by device name.
- `get_pixel_color` — Get the hex color at screen coordinates x,y.
- `get_screen_dpi` — Get per-monitor screen DPI and scale percentage.
- `get_screen_size` — Get the screen dimensions.
- `ocr` — Extract text from screen using Windows OCR. Supports full screen or region (x,y,w,h).
- `ocr_languages` — List all available Windows OCR languages. Returns array of language objects with tag, display_name, and native_name.
- `record_screen` — Record screen frames at fixed intervals. Returns base64 images. Duration in ms, interval in ms.
- `screenshot` — Capture the screen or a region. If w/h omitted, captures full screen.

## Mouse (6)

- `click` — Click at screen coordinates x,y. Button: left/right/middle. Clicks: 1 or 2.
- `drag` — Drag mouse from (from_x, from_y) to (to_x, to_y).
- `get_cursor_position` — Get the current mouse cursor position.
- `hover` — Move the mouse to coordinates and wait briefly (for tooltips/hover menus).
- `move_mouse` — Move mouse cursor to x,y.
- `scroll` — Scroll the mouse wheel. Positive clicks = up, negative = down. Set horizontal=true for horizontal scroll.

## Keyboard (9)

- `key_down` — Hold a key down (does not release it). Use key_up to release. Example: "W"
- `key_press` — Press key combination. Example: ["Ctrl", "C"] for copy.
- `key_up` — Release a key that was held down with key_down. Example: "W"
- `keylogger_start` — Start recording keyboard and mouse input for replay
- `keylogger_status` — Check if keylogger is active and event count
- `keylogger_stop` — Stop recording and return recorded sequence as chain steps
- `select_all_and_type` — Select all text (Ctrl+A) and type replacement text.
- `type` — Type text at the currently focused element.
- `type_and_submit` — Type text and press Enter (e.g. for form submission or search).

## Window Management (13)

- `close_window` — Close a window by handle.
- `find_window` — Find a window handle by title.
- `focus_window` — Bring a window to the foreground by handle.
- `focus_window_by_title` — Find a window by title and focus it, clicking its title bar to ensure activation. Useful before keyboard input in chain steps.
- `get_active_window` — Get the current foreground window info.
- `get_window_state` — Get window state (visible, minimized, maximized, position, size).
- `list_windows` — List all visible windows with their handles, titles, and PIDs.
- `maximize_window` — Maximize a window by handle.
- `minimize_window` — Minimize a window by handle.
- `move_window` — Move and resize a window by handle.
- `restore_window` — Restore a minimized or maximized window by handle.
- `screenshot_element` — Take a screenshot of a specific window by handle.
- `wait_for_window` — Wait for a window with the given title to appear. Returns handle or timeout.

## Chained / Composite (4)

- `click_menu_item` — Find a window by title, then click a menu item or button using OCR within that window.
- `find_text_and_click` — Find text on screen using OCR and click at its location. Optional region x,y,w,h to search within.
- `launch_and_wait` — Launch an application and wait for its window to appear.
- `wait_for_text` — Wait for text to appear on screen. Polls OCR until found or timeout.

## Chain Automation (1)

- `chain` — Execute a sequence of steps sequentially server-side. Steps can call any tool, wait, capture output, and use {{variable}} substitution.

## UI Automation (3)

- `uia_find` — Find UI elements by name, automation_id, or control_type using UI Automation. Returns bounding rectangles and properties.
- `uia_get_text` — Get text from a UI element by name or automation_id using UI Automation.
- `uia_invoke` — Click or invoke a UI element by name or automation_id using UI Automation.

## Browser Automation (4)

- `browser_focus_url_bar` — Focus a browser window's URL bar. Supports Firefox (Ctrl+T), Chrome/Edge (Ctrl+L), and other browsers. Provide browser name (firefox, chrome, edge, brave, opera) or window title substring.
- `browser_navigate` — Open a new tab in a browser and navigate to a URL.
- `browser_new_tab` — Open a new tab in a browser window. Uses Ctrl+T for all browsers.
- `browser_search` — Open a new tab in a browser and perform a search query.

## File Explorer (4)

- `explorer_focus` — Focus an existing File Explorer window.
- `explorer_open_path` — Open a File Explorer window at the specified path. Reuses existing window when possible.
- `open_file_explorer` — Open File Explorer to a specified path (default: C:\).
- `open_file_location` — Open File Explorer with a specific file selected.

## Audio (2)

- `list_audio_devices` — List all audio playback and recording devices.
- `set_default_audio_device` — Set the default audio playback device by device ID.

## Memory & Templates (10)

- `layout_validate` — Validate stored UI element layout against the current screen. Checks window existence, position drift, and OCR keyword verification. Returns adjusted coordinates and confidence levels (ok/drifted/stale).
- `memory_forget` — Delete facts by key, scope, or tags. At least one filter is required to prevent accidental mass deletion.
- `memory_get` — Retrieve a fact from the memory store by key and optional scope.
- `memory_list` — List stored facts under a scope with optional tag filter.
- `memory_search` — Full-text search across keys, values, scope, and tags using FTS5. Supports SQLite FTS5 query syntax.
- `memory_set` — Store a fact into the memory store. Fields: key (required), value (required, any JSON value), scope, tags (comma-separated), ttl (optional expiry in seconds).
- `template_find` — Find a stored UI element template on the current screen using NCC template matching. Returns coordinates, score, and drift from stored position.
- `template_forget` — Delete a stored UI element template by element_key and optional scope.
- `template_list` — List stored UI element templates with metadata (element key, scope, window title, hit count, etc.).
- `template_store` — Capture a UI element template from the current screen by cropping around a coordinate. Stores as base64 PNG in the element_templates table for visual re-identification.

## ONNX ML (7)

- `onnx_detect` — Run YOLO-based UI element detection on a screenshot (or full screen if no image provided). Returns detected elements with class labels, confidence scores, and bounding boxes. Requires onnxruntime.dll and YOLO model file.
- `onnx_download` — Check and prepare ONNX model files. Lists which models are present and which need manual download.
- `onnx_status` — Check ONNX runtime and model availability. Returns presence of YOLO model, MobileNet model, and onnxruntime.dll.
- `onnx_watch_cache` — Retrieve cached detections from the background watcher. Returns the most recent detection results with timestamps and saved reference paths.
- `onnx_watch_start` — Start a background watcher that periodically screenshots the screen, runs ONNX detection, and caches results. Takes interval_seconds (default 5).
- `onnx_watch_status` — Get the current ONNX watcher state: running, interval, last run time, cache size.
- `onnx_watch_stop` — Stop the background ONNX watcher.

## Priors & Statistics (1)

- `priors_stats` — Show learned element frequency and position statistics per window. Returns priors with sample count, frequency, and position distributions. Use min_count to filter out low-sample entries.

## Training Pipeline (6)

- `find_ui_element` — Find a UI element on screen by label. Checks memory first (from past ONNX detections), then runs ONNX detection, then falls back to OCR. Stores findings in memory for future reuse. Use this when the AI needs to locate an element it has seen before or needs to find programmatically.
- `training_cleanup_noise` — Delete low-signal (signal_level=0) training samples older than max_age_hours. Use dry_run=true to see what would be deleted without actually removing anything. Returns deleted count and freed bytes.
- `training_list_samples` — List saved training samples, optionally filtered by category or unused-only status.
- `training_mark_used` — Mark a training sample as used (after the model has been trained on it).
- `training_save_sample` — Capture screenshot and save as a training sample with a task prompt (e.g. 'click the submit button'). The ONNX model learns from these during idle retraining.
- `training_stats` — Get training data statistics: total samples, unused samples, breakdown by category, disk usage.

## Data Export (1)

- `export_yolo_dataset` — Export unused training samples as a YOLO-format dataset (images + labels + dataset.yaml) for external training with Ultralytics or other YOLO frameworks. Outputs to a directory of your choice.

## Data Logging (3)

- `datalog_export` — Export OCR+command training pairs as JSON for ML training. Optionally filter by session_id. Returns pairs with before/after OCR text and command JSON.
- `datalog_query` — Query the action/OCR data log. Table: commands, chains, ocr, or pairs. Filter by source, tool, success. Returns recent rows with all columns.
- `datalog_status` — Get data logging statistics: count of commands, chains, OCR snapshots, and training pairs logged to the datalog database.

## Adaptive Agent (3)

- `agent_analyze` — Analyze the adaptive engine state — timing stats, success rates per tool, and learned OCR→command sequences. Returns a full report for AI decision-making.
- `agent_suggest` — Given OCR screen text, predict the best next command based on past successful sequences. Returns ranked predictions with confidence scores and optional coord (x, y, confidence, samples) for click/hover/move_mouse.
- `agent_train` — Train the adaptive engine from datalog training_pairs. Rebuilds the OCR→command word index and sequence cache. Call after the datalog has accumulated new pairs.

## Introspection & Debugging (4)

- `bridge_debug` — Debug the OCR→command bridge state — shows recent OCR buffer, pending command, and timing info.
- `introspection_analyze` — View task history with mined insights from past task_begin/task_end sessions.
- `task_begin` — Mark the start of a task for post-task introspection. Call before the first tool call in a task.
- `task_end` — Mark the end of a task. Returns mined insights: slow/failed tools, OCR stats, repeat patterns, and improvement suggestions.

## Runtime Config (1)

- `set_config` — Update runtime configuration. Accepts any subset of: training_enabled (stop/start background screenshot saving), prior_adjustment (enable/disable ML prior confidence tuning), verify_bounds (toggle coordinate bounds checking), log_level (debug/info/warn/error), watcher_enabled (start/stop the background screenshot watcher), watcher_interval_seconds (change polling frequency while running). Changes persist to disk. Use this to disable data collection or control the tool at runtime.

## System (25)

- `get_battery` — Get battery status (percentage, charging, on battery).
- `get_brightness` — Get the current display brightness level (0-100).
- `get_clipboard` — Read text from the clipboard.
- `get_disk_usage` — Get disk usage information for all drives.
- `get_idle_time` — Get the system idle time (time since last user input) in milliseconds.
- `get_keyboard_layout` — Get the current keyboard layout / input language.
- `get_network_info` — Get network information: hostname, IP addresses, DNS servers, default gateway.
- `get_system_info` — Get system information (hostname, OS, RAM).
- `get_uptime` — Get the system uptime (time since last boot).
- `get_volume` — Get the current system volume level (0-100).
- `hibernate` — Hibernate the computer.
- `list_displays` — List all monitors with resolution and position.
- `lock_workstation` — Lock the workstation.
- `open_url` — Open a URL in the default browser.
- `ping` — Ping a host to check network reachability.
- `restart` — Restart the computer.
- `set_brightness` — Set the display brightness level (0-100).
- `set_clipboard` — Write text to the clipboard.
- `set_keyboard_layout` — Set the keyboard layout / input language (e.g. 'en-US', 'ja-JP').
- `set_mute` — Mute or unmute the system audio.
- `set_volume` — Set the system volume level (0-100).
- `show_notification` — Show a Windows notification message box.
- `shutdown` — Shut down the computer.
- `sleep` — Put the computer to sleep.
- `wait` — Wait for N milliseconds before the next action.

## Process Management (3)

- `kill_process` — Terminate a process by PID.
- `launch_app` — Launch an application by path or shell command.
- `list_processes` — List all running processes with PID, name, and thread count.

<!--
Generated by scripts/gen-tools-doc.go — 120 tools found
-->
