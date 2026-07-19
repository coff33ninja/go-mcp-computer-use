# Plan: Self-Update System with Client-Aware Restart

**Platform:** Windows-only. All paths, process handling, and scripts assume Windows.

## Problem

The MCP server has no way to:
1. Tell the AI if a newer version exists
2. Let the AI trigger an update
3. Handle restart across different AI clients (desktop apps, CLI agents, direct invocation)
4. Communicate to the user when an update requires their manual intervention

Each AI client launches `mcp-server.exe` differently. The update script must understand
which client is running it, where the binary lives, and how to restart that specific client.

## Client Taxonomy

There are three distinct client types with fundamentally different restart semantics:

### Type 1: Desktop Apps (GUI)
Process tree: `OpenCode.exe → mcp-server.exe`
- The desktop app is the direct parent
- Kill the parent, relaunch it — MCP restarts automatically
- Examples: OpenCode Desktop, Claude Desktop, Cursor, Windsurf

### Type 2: CLI Agents (Terminal)
Process tree: `pwsh.exe → opencode.exe → mcp-server.exe`
- The CLI agent is the parent, but it runs inside a terminal
- The terminal is the grandparent — you can't relaunch a terminal session
- The CLI agent manages the MCP server lifecycle
- Examples: opencode CLI, claude CLI, gemini CLI, codex CLI, copilot CLI

### Type 3: Direct Invocation
Process tree: `pwsh.exe → mcp-server.exe`
- User ran the binary directly from a terminal
- No AI client involved — the user is the "client"
- Restart means: kill, replace, user re-runs the command

**The update script must detect which type it's dealing with and use the right strategy.**

## Key Insight

**The MCP server should report what program is using it.** This makes the update system
client-agnostic. The server already knows its parent process via `os/ppid`. The update
script receives this info and adjusts its behavior accordingly.

## Architecture

### 1. Client Detection

On startup, the MCP server walks the process tree upward:

```
mcp-server.exe (self, PID 9999)
  └── parent: opencode.exe (PID 5555)     ← CLI agent detected
        └── grandparent: pwsh.exe (PID 1111)  ← terminal
```

```
mcp-server.exe (self, PID 8888)
  └── parent: OpenCode.exe (PID 4444)     ← desktop app detected
```

```
mcp-server.exe (self, PID 7777)
  └── parent: pwsh.exe (PID 3333)         ← direct invocation
```

Detection uses `os/ppid` + `os.FindProcess` + process name resolution.
The server stores:
- `parent_process_name` (e.g., "opencode", "OpenCode", "pwsh")
- `parent_process_path` (e.g., `C:\...\opencode.exe`)
- `parent_pid`
- `grandparent_process_name` (for CLI detection)
- `client_type` — "desktop" | "cli" | "direct" | "unknown"

This is exposed via `get_client_info` tool or as fields in `update_server` response.

### 2. Client Registry

```go
type ClientInfo struct {
    Name            string   // Display name: "OpenCode CLI"
    ProcessName     string   // Process name without .exe: "opencode"
    ClientType      string   // "desktop" | "cli" | "direct"
    RestartMethod   string   // "relaunch_parent" | "replace_and_notify" | "replace_and_exit" | "needs_permission" | "manual"
    ElevateOnRestart bool    // Whether to use RunAs
    KillParent      bool    // Whether to kill the client process
}
```

Known clients:

| Client | Process | Type | Restart Method | Kill Parent | Elevate |
|--------|---------|------|---------------|-------------|---------|
| OpenCode Desktop | `OpenCode` | desktop | relaunch_parent | Yes | Yes |
| Claude Desktop | `Claude` | desktop | relaunch_parent | Yes | No |
| Cursor | `cursor` | desktop | relaunch_parent | Yes | No |
| Windsurf | `windsurf` | desktop | relaunch_parent | Yes | No |
| opencode CLI | `opencode` | cli | replace_and_notify | No | No |
| claude CLI | `claude` | cli | replace_and_notify | No | No |
| gemini CLI | `gemini` | cli | replace_and_notify | No | No |
| codex CLI | `codex` | cli | replace_and_notify | No | No |
| copilot CLI | `copilot` | cli | replace_and_notify | No | No |
| pwsh/cmd | `*` | direct | replace_and_exit | No | No |
| Unknown | `*` | unknown | needs_permission | No | No |

### 3. Restart Strategies by Client Type

#### Desktop Apps → `relaunch_parent`
```
1. Sleep 3s (let MCP response flush)
2. Download new binary
3. Kill mcp-server.exe (self)
4. Kill parent (OpenCode.exe, Claude.exe, etc.)
5. Sleep 2s (process cleanup)
6. Copy new binary over old
7. Relaunch parent (RunAs if needed)
```
The desktop app restarts and spawns a new mcp-server.exe automatically.

#### CLI Agents → `replace_and_notify`
```
1. Sleep 3s (let MCP response flush)
2. Download new binary
3. Kill mcp-server.exe (self) — CLI agent detects child death
4. Copy new binary over old
5. Write marker file: %APPDATA%/go-mcp-computer-use/update-pending.json
   { "version": "0.2.44", "replaced_at": "..." }
6. Exit
```
The CLI agent's MCP transport detects the connection drop. On next invocation,
the CLI agent spawns the new binary. The marker file lets the AI check if an
update was applied: `update_server(check_only: true)` reads the marker and
reports "updated to 0.2.44, restart your CLI agent to use the new version."

**Why not relaunch the CLI?** The CLI runs in the user's terminal. Killing it
would destroy the terminal session, lose scrollback, break other commands. The
user must restart the CLI agent themselves — but they get a clear message telling
them to.

#### Direct Invocation → `replace_and_exit`
```
1. Sleep 3s (let MCP response flush)
2. Download new binary
3. Kill mcp-server.exe (self)
4. Copy new binary over old
5. Exit
```
The terminal returns to the prompt. The user re-runs `mcp-server.exe` or their
AI client restarts it. Simple, predictable.

#### Unknown Client → `needs_permission`
```
1. Download new binary to %APPDATA%/go-mcp-computer-use/mcp-server-new.exe
2. Do NOT kill anything
3. Return to AI: {
     "status": "needs_permission",
     "downloaded_to": "C:\\Users\\...\\mcp-server-new.exe",
     "binary_path": "E:\\...\\mcp-server.exe",
     "message": "Cannot determine how to restart your AI agent automatically.
                 To update, please:
                 1. Close your AI agent
                 2. Copy mcp-server-new.exe to mcp-server.exe
                 3. Restart your AI agent"
   }
4. The AI relays this message to the user
5. If user says "yes, do it" → AI tells user the exact steps
```

This is the **permission-required flow**. The system never guesses. If it can't
determine the restart strategy, it asks the user to intervene.

**When does `needs_permission` trigger?**
- Parent process is not in the known client registry
- Parent is a service host (`svchost.exe`), task scheduler, or other system process
- Binary is on a network drive or read-only location
- Permission denied when attempting to stop the parent process
- Parent process is elevated (admin) but MCP server is not

### 4. Update Flow (Full)

```
AI calls update_server(check_only: false)
  │
  ├─→ Server detects:
  │     current = 0.2.43
  │     latest  = 0.2.44 (from GitHub API)
  │     client  = { name: "opencode", type: "cli", restart: "replace_and_notify" }
  │
  ├─→ Writes update script to %APPDATA%/go-mcp-computer-use/update.ps1
  │   Script receives args:
  │     -binary_path     "E:\SCRIPTS\Servers\go-mcp-computer-use\mcp-server.exe"
  │     -download_url    "https://github.com/.../v0.2.44/mcp-server.exe"
  │     -parent_name     "opencode"
  │     -parent_type     "cli"
  │     -restart_method  "replace_and_notify"
  │     -elevate         "false"
  │     -appdata_dir     "C:\Users\...\AppData\Roaming\go-mcp-computer-use"
  │     -mcp_pid         "9999"
  │
  ├─→ Spawns script DETACHED (survives process kill)
  │
  ├─→ Returns response to AI immediately
  │
  └─→ Script executes based on -restart_method:
        "replace_and_notify"  → download, kill self, replace, write marker, exit
        "relaunch_parent"     → download, kill self, kill parent, replace, relaunch
        "replace_and_exit"    → download, kill self, replace, exit
        "needs_permission"    → download to safe location, don't kill, return path
        "manual"              → download to safe location, don't kill anything
```

### 5. Binary Path Discovery

The MCP server knows its own executable path via:
```go
execPath, _ := os.Executable()
execPath, _ = filepath.EvalSymlinks(execPath)
```

This is the canonical binary location. The update script replaces this file.

**Edge cases:**
- **Network drive / read-only location** — Detect via `os.Stat` permissions. Return
  `needs_permission` with message: "Binary is in a read-only location. Download
  saved to <path>. Please replace manually."
- **Symlinked binary** — `EvalSymlinks` resolves to the real file. Replace the real
  file, not the symlink.
- **Architecture mismatch** — If `runtime.GOARCH` doesn't match download, reject.
  Don't replace an amd64 binary with arm64 or vice versa.

### 6. Config Additions

```json
{
  "update_check_on_startup": false,
  "update_auto_apply": false
}
```

- `update_check_on_startup`: Server checks GitHub on start, logs if update available
- `update_auto_apply`: Not recommended — always require AI/user confirmation

### 7. CLI Agent Considerations

CLI agents have unique constraints that desktop apps don't:

**Terminal session preservation:**
- Killing a CLI agent kills the user's terminal scrollback
- The user may have other work in that terminal
- Strategy: never kill the CLI agent. Replace binary, let it die naturally, user restarts.

**Transport detection:**
- CLI agents use `stdio` transport (stdin/stdout pipes)
- When mcp-server.exe dies, the pipe breaks
- CLI agents handle this differently:
  - opencode: detects pipe close, reports to user, ready for restart
  - claude: shows "MCP server disconnected" warning
  - codex: reconnects on next prompt
- The update script doesn't need to know transport details — just replace and exit

**Marker file for CLI agents:**
```json
// %APPDATA%/go-mcp-computer-use/update-pending.json
{
  "version": "0.2.44",
  "replaced_at": "2026-07-19T19:30:00Z",
  "previous_version": "0.2.43",
  "binary_path": "E:\\SCRIPTS\\Servers\\go-mcp-computer-use\\mcp-server.exe"
}
```
- Written by update script after successful replacement
- Read by `update_server(check_only: true)` on next startup
- Cleaned up after first successful startup post-update
- Allows the AI to tell the user: "Updated to v0.2.44. Please restart your CLI."

**What if the CLI agent auto-restarts the MCP server?**
Some CLI agents may attempt to restart the MCP server after detecting the pipe close.
If the binary was already replaced, the new version starts automatically. The marker
file gets cleaned up. No user intervention needed. This is the ideal case.

### 8. Tool Schema

#### `get_client_info`

Reports what program is using the MCP server. Useful for debugging, logging, and the
update system.

```json
{
  "name": "get_client_info",
  "description": "Report which program is using this MCP server. Returns parent process info, client type, and restart semantics."
}
```

**Response:**
```json
{
  "client": {
    "name": "opencode",
    "display_name": "opencode CLI",
    "type": "cli",
    "pid": 5555,
    "path": "C:\\Users\\DRAGOHN\\go\\bin\\opencode.exe",
    "restart_method": "replace_and_notify",
    "grandparent": {
      "name": "pwsh",
      "pid": 1111
    }
  },
  "binary": {
    "path": "E:\\SCRIPTS\\Servers\\go-mcp-computer-use\\mcp-server.exe",
    "pid": 9999,
    "version": "0.2.43"
  }
}
```

#### `update_server`

```json
{
  "name": "update_server",
  "description": "Check for MCP server updates and optionally apply them. Returns version info, client detection, and update status. If a pending update exists (CLI agent marker file), reports it.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "check_only": {
        "type": "boolean",
        "default": false,
        "description": "If true, only check for updates without downloading. If false, download and schedule replacement."
      }
    }
  }
}
```

**Response (check_only: true, update available):**
```json
{
  "update_available": true,
  "current_version": "0.2.43",
  "latest_version": "0.2.44",
  "release_notes": "...",
  "download_url": "https://github.com/.../v0.2.44/mcp-server.exe",
  "client": {
    "name": "opencode",
    "type": "cli",
    "pid": 5555,
    "restart_method": "replace_and_notify",
    "binary_path": "E:\\SCRIPTS\\Servers\\go-mcp-computer-use\\mcp-server.exe"
  }
}
```

**Response (check_only: true, no update):**
```json
{
  "update_available": false,
  "current_version": "0.2.43",
  "latest_version": "0.2.43",
  "client": {
    "name": "opencode",
    "type": "cli"
  }
}
```

**Response (check_only: true, CLI agent with pending update from previous session):**
```json
{
  "update_available": false,
  "current_version": "0.2.44",
  "latest_version": "0.2.44",
  "pending_update": {
    "version": "0.2.44",
    "replaced_at": "2026-07-19T19:30:00Z",
    "previous_version": "0.2.43"
  },
  "message": "Binary was updated to 0.2.44. Restart your CLI agent to use the new version.",
  "client": {
    "name": "opencode",
    "type": "cli"
  }
}
```

**Response (check_only: false, update applied):**
```json
{
  "update_available": true,
  "current_version": "0.2.43",
  "latest_version": "0.2.44",
  "status": "updating",
  "client": {
    "name": "opencode",
    "type": "cli",
    "restart_method": "replace_and_notify"
  },
  "message": "Update scheduled. Server will restart in ~10 seconds. If the AI becomes unavailable, the update is in progress."
}
```

**Response (check_only: false, needs_permission):**
```json
{
  "update_available": true,
  "current_version": "0.2.43",
  "latest_version": "0.2.44",
  "status": "needs_permission",
  "downloaded_to": "C:\\Users\\DRAGOHN\\AppData\\Roaming\\go-mcp-computer-use\\mcp-server-new.exe",
  "client": {
    "name": "pwsh",
    "type": "direct"
  },
  "message": "Cannot automatically restart your AI agent. To complete the update:\n1. Close your AI agent or terminal\n2. Replace mcp-server.exe with mcp-server-new.exe in the binary directory\n3. Restart your AI agent"
}
```

## Safety Considerations

1. **Never auto-apply** — The AI must explicitly call `update_server` with `check_only: false`.
   Even then, the response warns that a restart will happen.

2. **Download validation** — Check file exists and size > 1MB before replacing. Checksum
   support later if GitHub releases include SHA256 files.

3. **Rollback** — If the new binary fails to start (crashes immediately), the user must
   manually reinstall. No automatic rollback — too complex for v1. Consider:
   - Keep old binary as `mcp-server.exe.bak` for 24 hours
   - Add `rollback_update` tool that restores the backup

4. **Concurrent updates** — Lock file at `%APPDATA%/go-mcp-computer-use/update.lock`.
   If lock exists, another update is in progress. Return error.

5. **Network failures** — If GitHub is unreachable, return clear error. Don't retry
   indefinitely. Max 2 retries with 5s timeout.

6. **Permission errors** — If binary is in a protected location (Program Files), detect
   and return `needs_permission`. Don't silently fail. Don't assume admin rights.

7. **CLI agent safety** — Never kill a CLI agent process. The user's terminal session
   is sacred. Only replace the binary and exit. The marker file communicates the
   update status on next invocation.

8. **Binary in use** — Windows locks running executables. When replacing a running
   binary, the copy may fail. Strategy:
   - Rename old binary to `mcp-server.exe.old` (succeeds even while running)
   - Copy new binary to `mcp-server.exe`
   - On next start, old binary is gone (or cleaned up by the update script)
   - This is the same technique used by Chrome, VS Code, and other auto-updaters

9. **User consent** — The AI must always tell the user what will happen before calling
   `update_server(check_only: false)`. The user should confirm. If the user says "no"
   or "not now", the AI respects that. The tool is a capability, not an automation.

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/update/update.go` | CheckForUpdate, BuildUpdateScript, ApplyUpdate, DetectClient |
| `internal/server/server.go` | Register update_server and get_client_info tools |
| `internal/config/config.go` | Add update config fields |
| `docs/meta/backlog.md` | Update relevant sections |
| `docs/meta/CHANGELOG.md` | v0.2.44 entry |

## What This Does NOT Cover (Future Work)

- **Checksum verification** — Requires CI to generate SHA256 files alongside releases
- **Auto-update** — Config option to silently apply (dangerous, low priority)
- **Docker/Container** — Different update mechanism entirely (re-pull image)
- **Update channel** — "stable" vs "beta" vs "nightly" (future config option)
- **Changelog preview** — Show release notes before applying (currently just version diff)
- **CLI agent auto-restart detection** — Some CLI agents may restart the MCP server
  automatically after pipe close. Detect this and skip the marker file flow.
