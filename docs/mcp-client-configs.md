# MCP Client Configuration Reference

How to load `go-mcp-computer-use` as an MCP server in various AI agents.

All config examples below use `C:\Users\YourUsername\Downloads\mcp-server.exe` as a placeholder — **replace the path with the actual location of your `mcp-server.exe`**.

## Quick Start — Common Config Block

All stdio-based clients use the same JSON shape:

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

Some clients use different top-level keys. The table below shows them.

> **⚠️ CLI/TUI over file edits:** Several clients provide CLI commands (`claude mcp add`, `gemini mcp add`, `opencode mcp add`) to add servers interactively. Prefer these over manual JSON edits when possible — they handle schema validation, correct key names, and env var substitution. Manual file edits may not take effect if:
> - The client caches config at startup (restart required)
> - The client uses a sidecar/sync layer (e.g., OpenCode Desktop has `opencode.global.dat` that can desync from the JSON file)
> - The project config overrides global config in unexpected ways
> - The file extension or schema differs from what's documented (.json vs .jsonc, etc.)
>
> **First attempt with CLI. Fall back to file edits only if the client lacks CLI support.**

---

## Client Details

### Claude Desktop

| Field | Value |
|-------|-------|
| **Config file** | macOS: `~/Library/Application Support/Claude/claude_desktop_config.json` |
| | Windows: `%APPDATA%\Claude\claude_desktop_config.json` |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio (local), SSE via `mcp-remote` |
| **Restart** | Restart Claude Desktop |
| **Docs** | https://support.claude.com/en/articles/10949351-getting-started-with-local-mcp-servers-on-claude-desktop |

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe"
    }
  }
}
```

---

### Claude Code

| Field | Value |
|-------|-------|
| **Config file** | Project: `.mcp.json` (root), User: `~/.claude.json`, Local: `~/.claude.json` per-project |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio, http, sse, websocket |
| **CLI add** | `claude mcp add computer-use -- mcp-server.exe` **(preferred)** |
| **Scopes** | `--scope project` → `.mcp.json` (commit-friendly), `--scope user` → `~/.claude.json`, default → local |
| **Docs** | https://code.claude.com/docs/en/mcp-quickstart |

> **Recommendation:** Use the CLI command over manual file edits. It handles JSON schema, `env` formatting, and scoping automatically. Run `claude mcp list` to verify after adding.

```json
{
  "mcpServers": {
    "computer-use": {
      "type": "stdio",
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

---

### Cursor

| Field | Value |
|-------|-------|
| **Config file** | Project: `.cursor/mcp.json`, Global: `~/.cursor/mcp.json` |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio, http, sse |
| **Variable interpolation** | `${env:NAME}`, `${userHome}`, `${workspaceFolder}` |
| **Restart** | Reload Cursor window |
| **Docs** | https://cursor.com/docs/mcp |

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

---

### Windsurf (Codeium)

| Field | Value |
|-------|-------|
| **Config file** | `~/.codeium/windsurf/mcp_config.json` (global only) |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio, http, sse |
| **Gotchas** | Uses `serverUrl` not `url` for remote servers; path is `.codeium/` not `.windsurf/` |
| **Tool limit** | 100 tools max across all servers |
| **Restart** | Restart Windsurf |
| **Docs** | https://docs.codeium.com/windsurf/mcp |

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

---

### VS Code (GitHub Copilot)

| Field | Value |
|-------|-------|
| **Config file** | Project: `.vscode/mcp.json`, User: `settings.json` |
| **Top-level key** | `servers` (NOT `mcpServers`) |
| **Transport** | stdio, http, sse |
| **User settings path** | `settings.json` under `github.copilot.chat.mcp.servers` |
| **Restart** | Reload VS Code window |
| **Docs** | https://code.visualstudio.com/docs/copilot/chat/mcp-servers |

```json
// .vscode/mcp.json (project-level)
{
  "servers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

```json
// settings.json (user-level)
{
  "github.copilot.chat.mcp.servers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

---

### Continue.dev

| Field | Value |
|-------|-------|
| **Config file** | Global: `~/.continue/config.json` or `~/.continue/config.yaml` |
| **Top-level key** | `mcpServers` (ARRAY, not object) |
| **Transport** | stdio, sse, streamable-http |
| **Per-project** | Create `.continue/config.json` in project root to merge |
| **IDE support** | VS Code + all JetBrains |
| **Docs** | https://docs.continue.dev/customize/mcp |

```json
{
  "models": [],
  "mcpServers": [
    {
      "name": "computer-use",
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "args": [],
      "env": {}
    }
  ]
}
```

---

### Cline

| Field | Value |
|-------|-------|
| **Config file** | VS Code: `%APPDATA%\Code\User\globalStorage\saoudrizwan.claude-dev\settings\cline_mcp_settings.json` |
| | CLI: `~/.cline/data/settings/cline_mcp_settings.json` |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio, streamable-http, sse |
| **Extra fields** | `disabled`, `autoApprove`, `alwaysAllow` |
| **Docs** | https://docs.cline.bot/mcp/adding-and-configuring-servers |

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "disabled": false,
      "autoApprove": []
    }
  }
}
```

---

### LangChain Deep Agents

| Field | Value |
|-------|-------|
| **Config file** | Project: `.mcp.json`, Project: `.deepagents/.mcp.json`, User: `~/.deepagents/.mcp.json` |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio, sse, http |
| **CLI flags** | `--mcp-config PATH`, `--no-mcp` |
| **Security** | Default-deny for project-level stdio servers |
| **Docs** | https://docs.langchain.com/oss/python/deepagents/cli/mcp-tools |

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

---

### mcp-agent

| Field | Value |
|-------|-------|
| **Config file** | `mcp_agent.config.yaml` |
| **Top-level key** | `mcp.servers` |
| **Transport** | stdio, sse, websocket, streamable-http |
| **Docs** | https://github.com/lastmile-ai/mcp-agent |

```yaml
mcp:
  servers:
    computer-use:
      command: "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe"
```

---

### OpenAI Agents SDK

| Field | Value |
|-------|-------|
| **Docs** | https://openai.github.io/openai-agents-python/ |
| **Config** | Programmatic (Python) |

Configured programmatically:

```python
from agents import Agent, MCPServerStdio

server = MCPServerStdio(
    name="computer-use",
    params={
        "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
    }
)

agent = Agent(
    name="Assistant",
    mcp_servers=[server],
    mcp_config={
        "include_server_in_tool_names": True,
    }
)
```

---

### OpenCode

| Field | Value |
|-------|-------|
| **Config file** | Global: `~/.config/opencode/opencode.json`, Project: `opencode.json` (root) |
| **Top-level key** | `mcp` (NOT `mcpServers`) |
| **Transport** | local, remote |
| **Command format** | Array (NOT string) — `"command": ["exe", "arg1", "arg2"]` |
| **Env field** | `environment` (NOT `env`) |
| **Extra fields** | `enabled`, `timeout`, `cwd` |
| **CLI add** | `opencode mcp add` (interactive prompt) |
| **CLI list** | `opencode mcp list` |
| **CLI debug** | `opencode mcp debug [name]` |
| **Docs** | https://opencode.ai/docs/mcp-servers/ |

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "computer-use": {
      "type": "local",
      "command": ["C:\\Users\\YourUsername\\Downloads\\mcp-server.exe"],
      "enabled": true,
      "environment": {}
    }
  }
}
```

> **Note:** OpenCode's `mcp` config follows a different schema than most clients. The `command` is an array, not a string, and env vars live under `environment`.

> **⚠️ Desktop sidecar / .dat file bug:** OpenCode Desktop uses a sidecar process that reads `opencode.json`, but the Desktop UI reads state from `opencode.global.dat` in `%APPDATA%\ai.opencode.desktop\` (Windows). This `.dat` file can desync, causing the Desktop panel to show "0 MCPs" even though the sidecar and CLI have all servers connected. Known in v1.15.13 (race condition in PR #28937). Workarounds:
> 1. **Use CLI** — `opencode mcp list` always reflects real state
> 2. **Restart Desktop fully** (`Cmd+Q`/exit, not just window close)
> 3. **Delete `.dat` cache** — quit Desktop, delete `%APPDATA%\ai.opencode.desktop\opencode.global.dat`, restart
> 4. **Upgrade** — newer versions may include the fix

---

### Gemini CLI

| Field | Value |
|-------|-------|
| **Config file** | User: `~/.gemini/settings.json`, Project: `.gemini/settings.json` |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio, sse, http |
| **CLI add** | `gemini mcp add computer-use -- mcp-server.exe` **(preferred)** |
| **Extra fields** | `args`, `cwd`, `timeout`, `trust`, `includeTools`, `excludeTools` |
| **Env expansion** | `$VAR` / `${VAR}` / `%VAR%` (Windows) in `env` values |
| **Restart** | `gemini quit` then restart |
| **Docs** | https://geminicli.com/docs/tools/mcp-server/ |

> **Recommendation:** Use `gemini mcp add` over manual edits — it validates the schema and sets the correct scope (project vs user). The CLI also supports `--transport` (stdio/sse/http), `--trust`, and `--include-tools`/`--exclude-tools` flags.

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

> For HTTP transport, use `"httpUrl"` instead of `"command"`. Run `/mcp` in the CLI or `gemini mcp list` to verify servers are connected.

---

### Roo Code

| Field | Value |
|-------|-------|
| **Config file** | Global: `mcp_settings.json` (VS Code settings dir), Project: `.roo/mcp.json` |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio, streamable-http, sse |
| **Extra fields** | `disabled`, `alwaysAllow`, `autoApprove` |
| **Precedence** | Project-level overrides global for same server name |
| **Restart** | Roo Code auto-detects file changes |
| **Docs** | https://docs.roocode.com/features/mcp/using-mcp-in-roo |

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "disabled": false,
      "alwaysAllow": []
    }
  }
}
```

---

### Android Studio (MCP Server)

| Field | Value |
|-------|-------|
| **Plugin** | MCP Server (bundled, enabled by default in v2025.2+) |
| **Config** | Settings → Tools → MCP Server |
| **Transport** | SSE, stdio, HTTP Stream |
| **Role** | Android Studio acts as MCP **server** (not client), exposing ~30+ IDE tools |
| **Auto-config** | Auto-Configure for Claude Desktop, Claude Code, Cursor, VS Code, Windsurf |
| **Manual** | Copy SSE/Stdio/HTTP config snippet from Settings UI |
| **Docs** | https://www.jetbrains.com/help/idea/mcp-server.html |

This is a role reversal — Android Studio (and all JetBrains IDEs v2025.2+) can expose IDE tools as MCP endpoints for external AI agents to call (e.g., terminal execution, code analysis, run configurations).

---

### Zed

| Field | Value |
|-------|-------|
| **Config file** | macOS/Linux: `~/.config/zed/settings.json`, Windows: `%APPDATA%\zed\settings.json` |
| **Top-level key** | `context_servers` (NOT `mcpServers`) |
| **Transport** | stdio only (no native HTTP support) |
| **Required field** | `"source": "custom"` for manually added servers |
| **Tool permissions** | `agent.tool_permissions.default` in settings |
| **Restart** | Save file — Zed auto-restarts server processes |
| **Docs** | https://zed.dev/docs/ai/mcp |

```json
{
  "context_servers": {
    "computer-use": {
      "source": "custom",
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "args": [],
      "env": {}
    }
  }
}
```

> **Note:** Without `"source": "custom"` Zed silently skips the entry. Each argument must be a separate string in `args`.

---

### JetBrains IDEs (MCP Client)

| Field | Value |
|-------|-------|
| **Plugin** | AI Assistant (v2+, required for MCP Client support in v2025.1+) |
| **Config UI** | Settings → Tools → AI Assistant → Model Context Protocol (MCP) |
| **Config file** | `%APPDATA%\JetBrains\AIAssistant\mcp.json` (Windows) |
| **Top-level key** | `mcpServers` |
| **Transport** | stdio (configured via UI or JSON file) |
| **Import** | Can import Claude Desktop config directly |
| **Restart** | Restart the IDE |
| **Docs** | https://www.jetbrains.com/help/idea/mcp-server.html |

```json
{
  "mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "env": {}
    }
  }
}
```

> Also supports MCP via Continue.dev plugin (see Continue.dev section above), which works across all JetBrains IDEs.

---

### Obsidian

Requires two components: the **Local REST API** community plugin (inside Obsidian), and an **MCP server** process (outside).

| Field | Value |
|-------|-------|
| **Plugin** | Local REST API (coddingtonbear) — install from Community Plugins |
| **Plugin settings** | Note the API key, default port is `27124` (HTTPS) or `27123` (HTTP) |
| **MCP server** | Various community MCP servers (e.g., `uvx mcp-obsidian`, `npx @obsidian/mcp-server`) |
| **Config** | Add to your MCP client's config (see other sections) |
| **Prerequisite** | Obsidian must be running with the plugin enabled |
| **Plugin docs** | https://github.com/coddingtonbear/obsidian-local-rest-api |
| **Server docs** | https://github.com/cyanheads/obsidian-mcp-server |

```json
{
  "mcpServers": {
    "obsidian": {
      "command": "uvx",
      "args": ["mcp-obsidian"],
      "env": {
        "OBSIDIAN_API_KEY": "your-api-key-here",
        "OBSIDIAN_HOST": "127.0.0.1",
        "OBSIDIAN_PORT": "27124"
      }
    }
  }
}
```

> For OpenCode specifically, use the `mcp` key with array command format:
> ```jsonc
> { "mcp": { "obsidian": { "type": "local", "command": ["uvx", "mcp-obsidian"], "environment": { "OBSIDIAN_API_KEY": "..." } } } }
> ```

---

### Emacs

| Field | Value |
|-------|-------|
| **Package** | `mcp.el` (MELPA, `(use-package mcp :ensure t)`), requires Emacs 30+ |
| **Config variable** | `mcp-hub-servers` |
| **Config format** | Alist of `(NAME . PLIST)` — plist keys: `:command`, `:args`, `:url`, `:env`, `:roots`, `:timeout` |
| **Transport** | stdio (`:command` + `:args`), SSE (`:url`) |
| **Integration** | gptel, llm, agent-shell |
| **Start servers** | `M-x mcp-hub-start-all-server` or via `after-init-hook` |
| **Docs** | https://github.com/lizqwerscott/mcp.el |

```elisp
(setq mcp-hub-servers
      '(("computer-use" . (:command "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe"))))
```

> The `:env` plist key uses keyword-prefixed symbols: `(:API_KEY "value")`, not a JSON object.

---

### Sourcegraph Cody

| Field | Value |
|-------|-------|
| **Feature flag** | `agentic-context-mcp-enabled` (must be enabled on Enterprise instance) |
| **Config location** | VS Code: `settings.json` under `cody.mcpServers`, JetBrains: `cody_settings.json` |
| **Top-level key** | `cody.mcpServers` (in extension settings) |
| **Transport** | stdio only (local servers) |
| **Capabilities** | Tools only (no Resources or Prompts) |
| **Tool filtering** | `disabledTools` array to exclude specific tools |
| **Docs** | https://sourcegraph.com/docs/cody/capabilities/agentic-context-fetching |

```json
{
  "cody.mcpServers": {
    "computer-use": {
      "command": "C:\\Users\\YourUsername\\Downloads\\mcp-server.exe",
      "args": [],
      "env": {},
      "disabledTools": []
    }
  }
}
```

> Cody uses MCP for **agentic context fetching** — it automatically decides which MCP tools to invoke based on your query. The feature is disabled by default and requires an Enterprise Sourcegraph instance.

---

## Quick Reference Table

| Client | Config File | Top-Level Key | Transport | Project Scope? |
|--------|------------|---------------|-----------|----------------|
| Claude Desktop | `%APPDATA%\Claude\claude_desktop_config.json` | `mcpServers` | stdio | No |
| Claude Code | `.mcp.json` / `~/.claude.json` | `mcpServers` | stdio, http | Yes (`.mcp.json`) |
| Cursor | `.cursor/mcp.json` / `~/.cursor/mcp.json` | `mcpServers` | stdio, http | Yes (`.cursor/mcp.json`) |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` | `mcpServers` | stdio, http | No (global only) |
| VS Code Copilot | `.vscode/mcp.json` / `settings.json` | `servers` | stdio, http | Yes (`.vscode/mcp.json`) |
| Continue.dev | `~/.continue/config.json` | `mcpServers` (array) | stdio, http | Yes (`.continue/config.json`) |
| Cline | `%APPDATA%\Code\...\cline_mcp_settings.json` | `mcpServers` | stdio, http | No (global only) |
| Deep Agents | `.mcp.json` / `~/.deepagents/.mcp.json` | `mcpServers` | stdio, http | Yes |
| mcp-agent | `mcp_agent.config.yaml` | `mcp.servers` | stdio, http | Yes |
| OpenAI SDK | Programmatic | N/A | stdio, http | Per-Agent |
| **OpenCode** | `opencode.json` / `~/.config/opencode/opencode.json` | `mcp` | local, remote | Yes |
| **Gemini CLI** | `.gemini/settings.json` / `~/.gemini/settings.json` | `mcpServers` | stdio, sse, http | Yes |
| **Roo Code** | `mcp_settings.json` / `.roo/mcp.json` | `mcpServers` | stdio, http | Yes (`.roo/mcp.json`) |
| **JetBrains IDEs** | `%APPDATA%\JetBrains\AIAssistant\mcp.json` | `mcpServers` | stdio | Yes |
| **Android Studio** | Settings → Tools → MCP Server (acts as server, not client) | N/A | SSE, stdio, HTTP | Per-project |
| **Zed** | `%APPDATA%\zed\settings.json` / `~/.config/zed/settings.json` | `context_servers` | stdio | Yes |
| **Obsidian** | Part of another client's MCP config (requires Local REST API plugin) | per-client | stdio | N/A |
| **Emacs** | Emacs config (`~/.emacs.d/init.el`) via `mcp-hub-servers` variable | Alist plist | stdio, SSE | Per-session |
| **Sourcegraph Cody** | `settings.json` under `cody.mcpServers` | `cody.mcpServers` | stdio | No (global)

---

## Troubleshooting Config Reload Issues

Config file edits don't always take effect immediately (or at all). Here are the common failure modes:

### 1. Sidecar sync desync (OpenCode Desktop)

OpenCode Desktop runs a **sidecar process** that reads `opencode.json`, but the frontend UI reads from `opencode.global.dat` in `%APPDATA%\ai.opencode.desktop\`. If the sidecar and UI get out of sync, MCP servers work in the CLI but show "0 MCPs" in the Desktop panel.

**Fix:** Use `opencode mcp add` via CLI instead of editing JSON files. If already broken, quit Desktop, delete `%APPDATA%\ai.opencode.desktop\opencode.global.dat`, and restart.

### 2. Stale Electron session (Claude Desktop, Cursor, VS Code)

Closing the window may not kill the background process. The old config stays cached in memory.

**Fix:** Fully quit the app (system tray → Quit, or `taskkill /f /im`), not just close the window. On Windows, check Task Manager for lingering processes.

### 3. Project vs global config conflict

If both a project-level config and a global config define the same server name, one silently overrides the other (usually project wins). You may be editing the wrong file.

**Fix:** Check which scope the client resolves. For Claude Code, `--scope project` writes to `.mcp.json`; without it, the default scope may write to a different file.

### 4. Wrong file extension (.json vs .jsonc)

Some clients (OpenCode, VS Code) treat `.jsonc` and `.json` differently. OpenCode Desktop v1.15.13 had a bug where `.jsonc` extension caused MCP config to be ignored entirely.

**Fix:** Try renaming to `.json` (or `.jsonc` to match what the client expects).

### 5. Server startup blocked by UIA warmup (16-37s delay)

The UIA (UI Automation) layer has a one-time cold-start cost of 16-37 seconds. Since v0.2.3, this warmup runs asynchronously at startup so it no longer blocks the MCP initialize handshake. If you're on an older version or still experiencing timeouts:

**Fix:** Add `"uia_warmup": false` to `~/.config/go-mcp-computer-use/config.json` to skip UIA warmup entirely. UIA tools will be slow on first call, but the server will respond instantly.

---

### 6. Env var not expanded in shell-launched process

If you use `$VAR` or `%VAR%` in the `env` block but the client doesn't expand it (Gemini CLI does, but some clients don't), the server starts with literal `$VAR` as the value.

**Fix:** Use absolute values in the config, or check the client docs for env expansion support. Gemini CLI supports `$VAR`, `${VAR}`, and `%VAR%`.

---

## Testing the Server

```powershell
# Direct stdio test (works with any MCP inspector)
npx @modelcontextprotocol/inspector mcp-server.exe
```

Or manually via PowerShell:

```powershell
$env:Path = [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = "C:\Users\YourUsername\Downloads\mcp-server.exe"
$psi.RedirectStandardInput = $true
$psi.RedirectStandardOutput = $true
$psi.UseShellExecute = $false
$p = [System.Diagnostics.Process]::Start($psi)
Start-Sleep -Milliseconds 300
$p.StandardInput.WriteLine('{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0.1.0"}}}')
Start-Sleep -Milliseconds 100
$p.StandardInput.WriteLine('{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}')
Start-Sleep -Milliseconds 800
$p.StandardOutput.ReadLine()  # initialize result
$p.StandardOutput.ReadLine()  # tools/list result
$p.Kill()
```
