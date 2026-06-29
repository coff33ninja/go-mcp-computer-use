# Security

## ⚠️ DANGEROUS CAPABILITIES

This executable can **fully control the Windows machine it runs on**. It exposes these capabilities to any connected AI agent:

- **Read anything on screen** — screenshot, OCR, screen recording
- **Control input** — mouse clicks/moves, keyboard typing, key combos
- **Read and write clipboard** — steal or replace clipboard contents
- **Kill processes, launch executables, shutdown/restart** the machine
- **Change system audio, volume, mute, default devices**
- **Enumerate and interact with windows** — move, resize, close, find
- **Read network config, ping hosts, enumerate adapters**
- **Read disk usage, battery state, display modes**
- **Automate UI elements** via UI Automation (find/invoke buttons, read text)

**Treat this binary with the same caution as a remote-admin tool.** Only connect it to MCP clients you trust. The AI agent receiving these tools has equivalent access to a logged-in user at the keyboard. Do not expose it over a network without authentication, and never run it on a machine where you wouldn't let a remote user operate the mouse and keyboard.

## Elevation & UIPI (Admin vs Non-Admin)

Windows **UIPI** (User Interface Privilege Isolation) silently blocks input from non-elevated processes targeting elevated (Administrator) windows.

**If you run an app as Administrator** (game installers, system tools, some games like `HTGame.exe`):
→ You must also run `mcp-server.exe` **as Administrator** for mouse clicks and keyboard input to reach it.

**Without elevation:**
- **Keyboard** (`type`, `key_press`, `type_and_submit`): returns a clear warning — UIPI blocks `SendInput` with `KEYEVENTF_UNICODE`
- **Mouse** (`click`, `scroll`, `drag`): **silently fails** — no error, no feedback. The cursor moves (via `SetCursorPos`) but the click never fires

**To run elevated:** right-click your terminal/launcher → "Run as Administrator" → start `mcp-server.exe`. Or set your MCP client config to launch it through an admin shell.

**The good news:** this is a Windows security feature, not a bug. Normal (non-admin) applications work fine without elevation — browsers, terminals, editors, chat apps, file explorers, most games. You only need admin mode when targeting admin windows.

## Data Collection & Privacy Controls

The server has **no telemetry, no network calls, no data exfiltration**. All collected data stays in `%APPDATA%/go-mcp-computer-use/training/`. But users have full runtime control:

| Goal | How |
|------|-----|
| **Stop all screenshot saving** | `set_config` with `training_enabled: false` — disables auto-saves from actions AND the background watcher instantly |
| **Re-enable data collection** | `set_config` with `training_enabled: true` |
| **Stop the background watcher** | `set_config` with `watcher_enabled: false` — or `onnx_watch_stop` |
| **Start the background watcher** | `set_config` with `watcher_enabled: true` — uses interval from config or `watcher_interval_seconds` |
| **Change watcher frequency** | `set_config` with `watcher_interval_seconds: 10` — restarts watcher with new interval if running |
| **Disable ML prior learning** | `set_config` with `prior_adjustment: false` |
| **Delete noise samples** | `training_cleanup_noise` with `max_age_hours: 0` — purges low-quality frames |
| **Clear cached element data** | `memory_forget` with `scope: ui` — removes cached ONNX detection positions |
| **Inspect collected data** | `training_stats` — see counts, sources, disk usage |
| **Export collected data** | `export_yolo_dataset` — dump all images + labels to a directory |
| **Persistent disable** | Set `"training_enabled": false` in `~/.config/go-mcp-computer-use/config.json` |

The `set_config` tool can be called by the AI agent or directly by the user via their MCP client. All changes persist to disk and survive server restarts.

**For maximum privacy:** set `training_enabled: false` in config before starting the server.

## Agent Configuration

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\tools\\mcp-server.exe"
    }
  }
}
```

See [`mcp-client-configs.md`](mcp-client-configs.md) for per-agent config examples.
