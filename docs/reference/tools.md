# Tools (146)

Auto-generated from `internal/server/server.go`. Total: **146 tools**.

## Screenshot & Vision (11)

- `find_all_images` — Find ALL occurrences of a template image on screen using NCC template matching. Provide template as base64 PNG. Returns array of matches with coordinates and scores.
- `find_image` — Find a template image on screen using NCC template matching. Provide template as base64 PNG. Returns coordinates of best match.
- `get_display_modes` — Get all available display modes (resolution, refresh rate, color depth) for a monitor by device name.
- `get_pixel_color` — Get the hex color at screen coordinates x,y.
- `get_screen_dpi` — Get per-monitor screen DPI and scale percentage.
- `get_screen_size` — Get the screen dimensions.
- `image_diff` — Compare two base64-encoded PNG screenshots pixel by pixel. Returns statistics: changed_pixels, total_pixels, change_ratio (0-1), mean_diff (0-255), max_diff (0-255), same (bool). Optionally generates a diff image with changed pixels highlighted in red. Use threshold (0-255, default 30) to control sensitivity.
- `ocr` — Extract text from screen using Windows OCR. Supports full screen, specific monitor (screen=N where N is the display index from list_displays, 0-based), or region (x,y,w,h).
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

## Window Management (15)

- `clear_window_lock` — Release the window lock. Screen-touching tools will no longer be restricted to a specific window.
- `close_window` — Close a window by handle.
- `find_window` — Find a window handle by title.
- `focus_window` — Bring a window to the foreground by handle. Also restores the window if it is minimized. ALWAYS call this before typing, clicking, or OCR-ing a window that is not the current foreground window. Use get_window_state to check if a window is already foreground before calling.
- `focus_window_by_title` — Find a window by title and focus it, clicking its title bar to ensure activation. Useful before keyboard input in chain steps.
- `get_active_window` — Get the current foreground window info (handle, title, PID, and bounding rect).
- `get_window_state` — Get window state: visible (WS_VISIBLE flag — NOT obscured-by-other-windows), minimized, maximized, fullscreen, foreground (is this the active/focused window?), z_order (0=topmost, higher=deeper behind other windows), and bounding rect. A window can be visible with high z_order but completely hidden behind other windows. Use this before interacting: if NOT foreground, call focus_window first (z_order will become 0); if visible vs other windows check, compare z_order values between windows.
- `list_windows` — List all visible windows with their handles, titles, PIDs, and bounding rect (x, y, width, height). This includes background windows (minimized, behind other windows). Cross-reference with list_displays monitor positions to determine which screen each window occupies. Returns every top-level window — use get_window_state on a handle to check if it is actually foreground, minimized, or behind other windows.
- `maximize_window` — Maximize a window by handle.
- `minimize_window` — Minimize a window by handle.
- `move_window` — Move and resize a window by handle.
- `restore_window` — Restore a minimized or maximized window by handle.
- `screenshot_element` — Take a screenshot of a specific window by handle.
- `set_window_lock` — Lock the active chain to a specific window by handle. Screen-touching tools (click, type, OCR, etc.) will verify the locked window is foreground before executing. If window_lock_auto_focus is enabled, automatically re-focuses the locked window when it loses foreground. Use GetWindowState or list_windows to find the handle.
- `wait_for_window` — Wait for a window with the given title to appear. Returns handle or timeout.

## Chained / Composite (4)

- `click_menu_item` — Find a window by title, then click a menu item or button using OCR within that window.
- `find_text_and_click` — Find text on screen using OCR and click at its location. Uses a smart cascade: checks spatial memory (where text was seen before), then system find-text (Ctrl+F in browsers/apps), then OCR with optional scrolling. Use max_scrolls=5 for scrollable pages. Returns error with visible text if not found.
- `launch_and_wait` — Launch an application and wait for its window to appear.
- `wait_for_text` — Wait for text to appear on screen. Polls OCR until found or timeout. Supports scrolling with max_scrolls to find text on scrollable pages.

## Chain Automation (2)

- `chain` — Execute a sequence of steps sequentially server-side. Steps can call any tool, wait, capture output, and use {{variable}} substitution. Mouse-based tools (click, move_mouse, hover, drag) auto-capture the UIA element at their target coordinates and include it in step output as 'element_at_point'. New step types: verify_ui (UIA element presence/absence check), if_uia (branch on element existence). New chain-callable tools: uia_find, uia_get_element_at_point, uia_get_all_elements, uia_set_text, wait_for_ui_element.
- `chain_abort` — Check if the global chain abort hotkey has been pressed since last check. Returns {aborted: true} when the configured hotkey combo is detected. The abort is consumed on read (auto-resets). Call before starting long chains or poll periodically.

## UI Automation (3)

- `uia_find` — Find UI elements within windows by name, automation_id, or control_type using UI Automation. Returns bounding rectangles and properties (type, enabled state, etc.). Use this to locate text boxes, address bars, search menus, title bars, buttons, and other controls by their automation identity. The target window should be foreground (use focus_window first) for reliable results — some UIA providers only respond when the window is active.
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

## Introspection & Debugging (6)

- `bridge_debug` — Debug the OCR→command bridge state — shows recent OCR buffer, pending command, and timing info.
- `get_logs` — Read server log entries from the file-based log. Returns recent log lines with timestamps, levels, and messages. Useful for diagnosing tool failures, crashes, and errors after they occur.
- `introspection_analyze` — View task history with mined insights from past task_begin/task_end sessions.
- `report_issue` — Generate a GitHub issue report with system info, recent error logs, and context. If gh CLI is available, creates the issue automatically. Otherwise returns the markdown body for manual submission.
- `task_begin` — Mark the start of a task for post-task introspection. Call before the first tool call in a task.
- `task_end` — Mark the end of a task. Returns mined insights: slow/failed tools, OCR stats, repeat patterns, and improvement suggestions.

## Runtime Config (1)

- `set_config` — Update runtime configuration. Accepts any subset of: training_enabled (stop/start background screenshot saving), prior_adjustment (enable/disable ML prior confidence tuning), verify_bounds (toggle coordinate bounds checking), log_level (debug/info/warn/error), watcher_enabled (start/stop the background screenshot watcher), watcher_interval_seconds (change polling frequency while running), tool_denylist (list of tool names to disable, e.g. ["shutdown","restart"]), retention_days (auto-prune training samples older than N days, 0=disabled), chain_abort_enabled (enable/disable global hotkey abort), chain_abort_keys (hotkey combo like "Ctrl+Shift+Escape"), chain_abort_poll_ms (polling interval), window_lock_enabled (enable/disable screen tool locking), window_lock_auto_focus (auto re-focus locked window), log_file_enabled (enable/disable file-based logging), log_file_max_size_mb (max MB per log file before rotation), log_file_retention (number of rotated log files to keep). Changes persist to disk.

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

## Uncategorized (20)

- `copy_file` — Copy a file or directory (recursively) from source to destination.
- `create_directory` — Create a directory (recursive, like mkdir -p).
- `delete_file` — Delete a file or directory to the Recycle Bin (uses SHFileOperationW with FOF_ALLOWUNDO).
- `dismiss_all_menus` — Press Escape to dismiss open context menus/dialogs. OCRs before and after to detect which menus were open and whether they closed.
- `find_files` — Recursively search for files matching a glob pattern (e.g. '*.go', '**/*.md').
- `get_dpi_for_point` — Get DPI and scale percentage at a specific screen coordinate. Useful for determining which monitor a coordinate is on and its scaling factor, especially in mixed-DPI multi-monitor setups.
- `get_file_info` — Get file or directory metadata: size, mod_time, is_dir, mode.
- `get_working_directory` — Get the current working directory used for relative path resolution.
- `list_directory` — List directory contents. Returns entries with name, size, is_dir, mod_time, and mode.
- `move_file` — Move or rename a file or directory.
- `ocr_active_window` — Extract text from the currently active/foreground window using Windows OCR.
- `ocr_window` — Extract text from a specific window by handle using Windows OCR. Captures what is currently visible in the window's region. If the window is minimized, behind other windows, or off-screen, the captured region will show whatever is on top at those screen coordinates. Use get_window_state to check state, then focus_window or restore_window first if needed.
- `read_file` — Read a file with automatic type detection. Supports plaintext (txt, json, csv, yaml, etc.), docx, xlsx, pdf, and images (via OCR). Use page and page_size to paginate long content. Default page_size=8000 chars.
- `reset_state` — Clear accumulated server state (adaptive engine stats, bridge buffer). Use between heavy batch operations to prevent state accumulation and timeouts.
- `set_working_directory` — Set the working directory for relative path resolution in file tools.
- `uia_get_all_elements` — Get all immediate child UI elements in a window by handle (title bar, menu bar, content panes, toolbars, status bar — one level deep, not recursive DOM tree). Returns name, control_type, automation_id, bounding rect, and enabled state for each. Use this to understand a window's full control surface — text boxes, buttons, search fields, address bars, menus, etc. The window should be foreground (use focus_window first) for reliable results. Use max_results to cap output.
- `uia_get_element_at_point` — Identify a UI element at screen coordinates (x, y) using UI Automation. Returns the element's name, control_type, automation_id, bounding rect, and whether it is enabled. Use this after clicking or hovering to validate what was under the cursor, or to determine what element exists at a given point before interacting.
- `uia_set_text` — Set text in a UI element by name or automation_id using UI Automation.
- `wait_for_ui_element` — Wait for a UI element to appear in a window, identified by name or control_type. Polls UIA FindFirst on the window's descendants until found or timeout. Use this for content verification after an action (e.g., wait for a dialog to appear after clicking a button). Default timeout is 10 seconds.
- `write_file` — Write content to a file. Supports plaintext, docx (creates from text, preserves structure on overwrite), xlsx (TSV content becomes cells), and PDF (text creates PDF, JSON fills existing form fields). Requires overwrite=true to replace existing files.

<!--
Generated by scripts/gen-tools-doc.go — 146 tools found
-->
