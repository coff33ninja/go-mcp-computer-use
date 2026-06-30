# Agent Prompt Engineering Guide

> How to configure AI agents with the right tool subset for different tasks.
> The full 118-tool set (v0.2.19) is powerful but bloated for simple tasks. Load only what you need.

---

## Tool Subsets by Task Type

### 1. Web Browsing & Research

When the agent needs to navigate a browser, fill forms, extract text.

**Load these tools:**
```
screenshot, get_screen_size, get_screen_dpi
click, move_mouse, scroll, hover
type, key_press, type_and_submit, select_all_and_type
get_clipboard, set_clipboard
ocr (full screen)
find_text_and_click (click links/buttons by visible text)
wait (page load timing)
get_pixel_color (verify color changes)
```

**Omit:**
```
all window management (unless switching tabs)
process management
power/shutdown
audio, recording
template matching (slow, OCR is better for web)
```

**System prompt hint:**
> You control a Windows desktop. Take a screenshot to see the browser, then use OCR to find
> links or buttons by their visible text and click them. Use `type_and_submit` for search bars
> and `type` for form fields. Scroll when content is below the fold.

---

### 2. File & Folder Management

When the agent needs to navigate Explorer, select files, copy/move.

**Load these tools:**
```
screenshot, get_screen_size
click, move_mouse, type, key_press
get_keyboard_layout (for path entry)
open_file_explorer, open_file_location
get_disk_usage
list_windows, focus_window (switch between Explorer windows)
wait (for Explorer to open)
```

**Omit:**
```
ocr (overkill for Explorer)
audio, recording
power
template matching
```

**System prompt hint:**
> Use `open_file_explorer` to launch Explorer to a path. Focus the window with `focus_window`.
> Navigate by typing paths into the address bar with `type_and_submit` for fast navigation.
> Use `open_file_location` to reveal a specific file. For drag-drop, use mouse coordinates.

---

### 3. Text Editing & Document Work

When the agent needs to write or edit documents (Word, Notepad, VS Code, etc.).

**Load these tools:**
```
screenshot, get_pixel_color
click, move_mouse, type, key_press
type_and_submit, select_all_and_type
get_clipboard, set_clipboard
get_screen_size (for window positioning)
list_windows, focus_window, move_window (arrange editor windows)
wait
```

**Omit:**
```
ocr (already typing)
audio
power
template matching
recording
```

**System prompt hint:**
> Position the editor window first with `move_window`, then `focus_window`.
> Use `select_all_and_type` to replace all content. Use `set_clipboard` + paste (Ctrl+V)
> for large text. Use `type_and_submit` for search/replace dialogs.

---

### 4. System Administration

When the agent needs to check system state, configure settings, manage processes.

**Load these tools:**
```
get_system_info, get_disk_usage, get_uptime, get_idle_time
list_processes, kill_process, launch_app
get_volume, set_volume, set_mute
get_battery, get_brightness, set_brightness
get_network_info, ping
get_keyboard_layout, set_keyboard_layout
get_clipboard, set_clipboard
shutdown, restart, sleep, hibernate
open_url, show_notification, lock_workstation
list_displays, get_screen_dpi
```

**Omit:**
```
mouse/keyboard (mostly CLI operations)
OCR, template matching
recording
```

**System prompt hint:**
> Use `list_processes` to find running apps, `launch_app` to start tools like Task Manager
> or Settings. Use `get_system_info` and `get_disk_usage` for health checks.
> For admin tasks, use `launch_app` with paths like `control`, `ms-settings:` URIs,
> or direct exe paths.

---

### 5. Debugging & Troubleshooting

When the agent needs to diagnose a problem, check error messages, inspect logs.

**Load these tools:**
```
screenshot, get_pixel_color
click, move_mouse, scroll
type, key_press, type_and_submit
ocr (read error dialogs)
list_windows, focus_window, get_window_state
list_processes, kill_process
get_clipboard, set_clipboard
wait, wait_for_text
get_system_info, get_disk_usage
find_text_and_click (dismiss error dialogs)
screenshot_element (capture specific window)
datalog_query, datalog_export, datalog_status (audit past actions)
bridge_debug (inspect OCR→command bridge state)
task_begin, task_end, introspection_analyze (post-mortem analysis)
agent_analyze (check per-tool success rates and timing stats)
```

**Omit:**
```
power commands
audio
recording
```

**System prompt hint:**
> Take a screenshot to see the error. Use OCR to extract error text into clipboard for analysis.
> Use `find_text_and_click` to click "OK", "Cancel", or "Close" buttons on dialogs.
> For console apps, use `launch_and_wait` to run and capture output.
> Use `task_begin` + `task_end` to wrap each debugging session — introspection mines slow/failed
> tools and suggests improvements. Query `datalog_query(commands)` to audit what happened.

---

### 6. Automation Scripting

When the agent needs to record and replay actions.

**Load these tools:**
```
screenshot
click, move_mouse, scroll, drag, hover
type, key_press, type_and_submit, select_all_and_type
wait, wait_for_text, wait_for_window
find_text_and_click
record_screen (capture result)
get_cursor_position (for coordinate recording)
chain (multi-step sequences with capture, poll, and branching)
```

**Omit:**
```
system info (unless needed)
power
audio
template matching (OCR is more robust)
```

**System prompt hint:**
> Break the automation into a sequence: screenshot → analyze → act → wait for result → repeat.
> Use `wait_for_text` and `wait_for_window` to make the script robust (not hardcoded delays).
> Use `find_text_and_click` instead of hardcoded coordinates.
> For complex sequences, use `chain` with poll steps and variable capture for conditional logic.

---

### 7. Self-Improving & Adaptive

When the agent needs to learn from past actions, adapt timing, or audit its own performance.

**Load these tools:**
```
agent_analyze (full timing stats, success rates, learned sequences)
agent_suggest (predict best next command from OCR context)
agent_train (rebuild word→command index from training_pairs)
task_begin, task_end, introspection_analyze (wrap tasks with post-mortem)
datalog_query, datalog_export, datalog_status (query execution history)
memory_list, memory_get, memory_search (recall stored facts)
priors_stats (element frequency/position stats per window)
```

**Omit:**
```
power commands
recording (unless capturing demo)
```

**System prompt hint:**
> Call `task_begin` at the start of every major task. On completion, `task_end(summary)` mines
> what went wrong — slowest tools, most failed, repeat patterns. Read `agent_analyze` periodically
> to adapt timing: if `type` has low success rate, try `uia_invoke` or add a focus step first.
> Use `agent_suggest(ocr_text)` when unsure what to do next — it predicts the best command
> and can return `coord` (x, y, samples) for click/hover/move_mouse, based on past
> successful sequences in similar screen contexts.

---

### 7. Audio & Media

When the agent needs to control audio devices or record.

**Load these tools:**
```
list_audio_devices, set_default_audio_device
get_volume, set_volume, set_mute
record_screen (capture media playback)
screenshot
```

**Omit:**
```
keyboard (minimal)
power
processes (unless launching media app)
```

**System prompt hint:**
> Use `list_audio_devices` to see available endpoints. Use `set_default_audio_device`
> to switch output. Combine with `launch_app` to open media apps.

---

## Agent-Specific Configurations

### opencode

In `opencode.json`, use the `env` field to set `MCP_TOOL_SUBSET` and write a system prompt:

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\tools\\mcp-server.exe",
      "env": {
        "MCP_TOOL_SUBSET": "web"
      }
    }
  }
}
```

### Claude Code

Use Claude's project knowledge to store the tool guide. Reference it in CLAUDE.md:

```
See .claude/computer-use-guide.md for which tools to use when browsing, reading docs, etc.
```

### GitHub Copilot

Copilot picks tools based on your prompt. Prefix your instruction with the task type:

```
[Task: Web Browsing] Navigate to example.com and search for...
[Task: System Admin] Check disk space and running processes...
```

---

## Prompt Patterns That Work

### Pattern 1: Screenshot-Think-Act loop
```
1. SCREENSHOT to see current state
2. Analyze what needs to happen
3. Call one tool (click, type, etc.)
4. SHORT WAIT (300-500ms)
5. Repeat from step 1
```
*Best for: most desktop automation tasks*

### Pattern 2: OCR-Then-Click
```
1. SCREENSHOT (or region)
2. OCR to find text and its coordinates
3. CLICK at those coordinates
```
*Best for: clicking buttons/links with unknown positions*
*(Also available as single `find_text_and_click` call)*

### Pattern 3: Wait-For-Change
```
1. SCREENSHOT
2. Perform action (click, type)
3. WAIT_FOR_TEXT or WAIT_FOR_WINDOW
4. Continue
```
*Best for: operations with unpredictable timing*

### Pattern 4: Window-Scoped
```
1. FIND_WINDOW to get handle
2. FOCUS_WINDOW to bring to front
3. OCR within window bounds
4. CLICK relative to window position
```
*Best for: working inside a specific application window*

### Pattern 5: Task-Wrap (Self-Audit)
```
1. TASK_BEGIN("description")
2. ... perform actions ...
3. TASK_END(summary="what happened", success=true|false)
4. INTROSPECTION_ANALYZE to review mined insights
```
*Best for: long-running tasks where you want post-mortem on what failed*

### Pattern 6: Adaptive Timing
```
1. AGENT_ANALYZE to check timing stats
2. If "wait" has high stddev, use WAIT_FOR_TEXT instead
3. If "click" has low success rate, verify focus first
```
*Best for: recovering from repeated failures — let the data guide you*

### Pattern 7: Context-Aware Prediction
```
1. OCR to get current screen state
2. AGENT_SUGGEST(ocr_text) to get ranked next-command predictions
3. Pick the highest-confidence command and execute
4. AGENT_TRAIN to rebuild index after new successful pairs
```
*Best for: when you're stuck or the screen state maps to a known workflow*

---

## General Tips

1. **Screenshot first** — always start with a screenshot to know where you are
2. **Prefer text over coordinates** — use `find_text_and_click` over hardcoded x,y
3. **Use `wait_for_text` over `wait`** — waiting for visual confirmation is more robust
4. **Keep delays moderate** — 200-500ms between actions is usually enough
5. **Prefer `type_and_submit`** — saves a separate Enter key press
6. **Use `screenshot_element`** — capture only the relevant window to reduce processing
7. **DPI awareness** — coordinates on a 150% scaled display need DPI scaling; use `get_screen_dpi`
8. **OCR language** — set the `language` parameter if the UI isn't in English
9. **Region OCR is faster** — OCR a region instead of full screen when possible
10. **Template matching threshold** — start at 0.8 and lower if no match found
11. **Wrap tasks** — `task_begin`/`task_end` around major work; introspection mines what went wrong automatically
12. **Query the datalog** — `datalog_query(table="commands", success=false)` shows which tools keep failing
13. **Let the adaptive engine help** — `agent_suggest(ocr_text)` predicts the best command based on past successes
14. **Retrain after learning** — call `agent_train` after a successful new workflow to update the prediction index
