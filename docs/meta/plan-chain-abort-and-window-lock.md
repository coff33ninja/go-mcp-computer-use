# Plan: Chain Abort + Window Lock-On

**Target version:** v0.2.42  
**Tool count:** 140 → 143 (3 new tools)  
**Branch:** v0.2.x  
**Created:** 2026-07-19  
**Status:** IN PROGRESS

---

## Problem

1. **No chain interrupt** — `chain` runs server-side with no way to stop it mid-execution. Timeout goroutine leaks (continues running after timeout returns).
2. **No window focus awareness** — AI can interact with wrong monitor/screen silently. No feedback when user switches windows during automation.

## Solution

### Feature 1: Chain Abort via Global Hotkey

Background goroutine polls `GetAsyncKeyState` for a configurable key combo (default: Ctrl+Shift+Escape). When detected, closes shared `chan struct{}`. Chains check between steps, exit with partial results.

### Feature 2: Window Lock-On Context

Package-level mutex-protected struct holds "locked" window handle. Screen-touching tools verify locked window is foreground before operating. Auto-focus back if not (configurable).

---

## Files to Create

### `internal/actions/chainabort.go` (NEW ~120 LOC)

Global state:
- `abortCh chan struct{}` — shared abort signal, closed when hotkey detected
- `abortStop chan struct{}` — stops the polling goroutine
- `abortKeys []uint32` — parsed VK codes from config
- `abortPollMs int` — polling interval

Functions:
- `ParseHotkeyString(hotkey string) []uint32` — splits on `+`, looks up in `vkModMap` (keyboard.go:86-91) and `vkSpecialMap` (keyboard.go:93-134). Returns VK codes. Unknown keys skipped with warning.
- `StartAbortPoller(keys []uint32, pollMs int)` — stops previous poller, starts new goroutine polling `GetAsyncKeyState` every pollMs. When all keys simultaneously down → close `abortCh`.
- `StopAbortPoller()` — closes `abortStop` channel.
- `GetAbortChannel() <-chan struct{}` — returns `abortCh`, creates if nil.
- `ResetAbortChannel()` — creates fresh channel (after previous was closed).
- `IsChainAborted() bool` — non-blocking check on `abortCh`.

Polling pattern (mirrors keylogger.go:195-203):
```go
func abortPollLoop() {
    ticker := time.NewTicker(time.Duration(abortPollMs) * time.Millisecond)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            allDown := true
            for _, vk := range abortKeys {
                if !getAsyncKeyState(vk) {
                    allDown = false
                    break
                }
            }
            if allDown {
                select {
                case <-abortCh:
                default:
                    close(abortCh)
                    slog.Info("chain abort triggered by global hotkey")
                }
                return
            }
        case <-abortStop:
            return
        }
    }
}
```

`getAsyncKeyState` — thin wrapper around Win32 `GetAsyncKeyState`, same as `klGetState` (keylogger.go:100-103). Duplicate the 3-line function.

### `internal/actions/windowlock.go` (NEW ~80 LOC)

Global state:
- `lockMu sync.RWMutex`
- `lockHandle uintptr`
- `lockTitle string`
- `lockPid uint32`

Functions:
- `SetWindowLock(handle uintptr) error` — verify window exists via `IsWindow`/`FindWindow`, store handle/title/pid.
- `ClearWindowLock()` — zero out all fields.
- `GetWindowLock() (handle uintptr, title string, pid uint32, locked bool)`
- `VerifyWindowLock(autoFocus bool) (bool, string)` — check IsWindow exists, check if foreground via `GetForegroundWindow`, auto-focus if configured.

---

## Files to Modify

### `internal/config/config.go`

Add to `Config` struct:
```go
ChainAbortEnabled   bool   `json:"chain_abort_enabled"`
ChainAbortKeys      string `json:"chain_abort_keys"`
ChainAbortPollMs    int    `json:"chain_abort_poll_ms"`
WindowLockEnabled   bool   `json:"window_lock_enabled"`
WindowLockAutoFocus bool   `json:"window_lock_auto_focus"`
```

Add to `Default()`:
```go
ChainAbortEnabled:   true,
ChainAbortKeys:      "Ctrl+Shift+Escape",
ChainAbortPollMs:    50,
WindowLockEnabled:   false,   // opt-in
WindowLockAutoFocus: true,
```

### `internal/actions/chain.go`

**`chainState` struct (line 379)** — add `abort <-chan struct{}` field.

**`ExecuteChain` (line 244):**
- Pass `abort: GetAbortChannel()` to chainState
- Replace goroutine + select with `context.WithCancel` pattern:
  - `ctx, cancel := context.WithCancel(context.Background())`
  - `defer cancel()`
  - Three-way select: `<-done`, `<-state.abort` (return partial + error), `<-time.After` (cancel + error)
- This fixes the goroutine leak: timeout now actually stops the goroutine.

**`execSteps` (line 293):**
- At top of each `for i, step` iteration, check abort:
  ```go
  if state.abort != nil {
      select {
      case <-state.abort:
          results = append(results, StepResult{
              Index: i, Success: false,
              Error: "aborted by user (global hotkey)",
          })
          return results, stepCount + 1
      default:
      }
  }
  ```
- After auto-focus check (line 343), add lock-on verification for screen tools:
  ```go
  if cfg := ActiveConfig; cfg != nil && cfg.WindowLockEnabled {
      if isScreenTool(step.Tool) {
          if valid, reason := VerifyWindowLock(cfg.WindowLockAutoFocus); !valid {
              stepResult = StepResult{
                  Tool: "window_lock_check", Success: false, Error: reason,
              }
              stepResult.Index = i
              results = append(results, stepResult)
              stepCount++
              if state.onError == "stop" { break }
              continue
          }
      }
  }
  ```

**Add `isScreenTool` helper** — static map of ~35 tool names that touch the screen (click, type, scroll, ocr, screenshot, uia_*, browser_*, drag, hover, key_press, etc.). NOT including read-only tools like get_cursor_position, get_pixel_color.

### `internal/server/server.go`

**`SetConfigArgs` (line 1775)** — add:
```go
ChainAbortEnabled   *bool  `json:"chain_abort_enabled,omitempty"`
ChainAbortKeys      string `json:"chain_abort_keys,omitempty"`
ChainAbortPollMs    *int   `json:"chain_abort_poll_ms,omitempty"`
WindowLockEnabled   *bool  `json:"window_lock_enabled,omitempty"`
WindowLockAutoFocus *bool  `json:"window_lock_auto_focus,omitempty"`
```

**`setConfigHandler` (line 2047)** — add handlers for new fields:
- `ChainAbortEnabled` → toggle poller on/off
- `ChainAbortKeys` → restart poller with new keys
- `ChainAbortPollMs` → restart poller with new interval
- `WindowLockEnabled` → toggle, clear lock if disabled
- `WindowLockAutoFocus` → update config

**New tool registrations:**
```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "chain_abort",
    Description: "Send abort signal to running chains. Chains check between steps and exit with partial results.",
}, chainAbortHandler)

mcp.AddTool(server, &mcp.Tool{
    Name:        "set_window_lock",
    Description: "Lock onto a window. Screen-touching tools verify this window is foreground before operating.",
}, setWindowLockHandler)

mcp.AddTool(server, &mcp.Tool{
    Name:        "clear_window_lock",
    Description: "Release the window lock.",
}, clearWindowLockHandler)
```

**New handler functions:**
- `chainAbortHandler` — calls `ResetAbortChannel()`, returns confirmation
- `setWindowLockHandler` — takes `handle`, calls `SetWindowLock`
- `clearWindowLockHandler` — calls `ClearWindowLock`

**`focusWindowHandler`** — after focusing, auto-set lock if `WindowLockEnabled`.

### `cmd/mcp-server/main.go`

After config load, start abort poller if enabled:
```go
if cfg.ChainAbortEnabled {
    keys := actions.ParseHotkeyString(cfg.ChainAbortKeys)
    actions.StartAbortPoller(keys, cfg.ChainAbortPollMs)
}
```

---

## New Config JSON Fields

```json
{
  "chain_abort_enabled": true,
  "chain_abort_keys": "Ctrl+Shift+Escape",
  "chain_abort_poll_ms": 50,
  "window_lock_enabled": false,
  "window_lock_auto_focus": true
}
```

---

## VK Code Reference

From `keyboard.go`:
- `vkModMap`: CTRL/CONTROL=0x11, ALT=0x12, SHIFT=0x10
- `vkSpecialMap`: ESCAPE=0x1B, F1-F12=0x70-0x7B, DELETE=0x2E, etc.
- `klGetState` (keylogger.go:100-103): `GetAsyncKeyState` Win32 API, returns `(ret & 0x8000) != 0`

---

## Testing

| Test | Type | File |
|------|------|------|
| `TestParseHotkeyString` | Unit | `chainabort_test.go` |
| `TestParseHotkeyString_Invalid` | Unit | `chainabort_test.go` |
| `TestAbortChannel` | Unit | `chainabort_test.go` |
| `TestChainAbort` | Unit | `chainabort_test.go` |
| `TestWindowLockSetClear` | Unit | `windowlock_test.go` |
| `TestIsScreenTool` | Unit | `windowlock_test.go` |
| `TestConfigNewFields` | Unit | `config_test.go` |

---

## Tool Count: 143

New tools:
1. `chain_abort` — Chain Automation category
2. `set_window_lock` — Window Management category
3. `clear_window_lock` — Window Management category

---

## Edge Cases

- Abort when no chain running → channel closes, no effect. Next chain resets it.
- Multiple chains running → all share abort channel, all abort.
- Nested loop/if steps → execSteps called recursively, abort check at top of each call.
- Window lock + window closed → VerifyWindowLock detects gone → error stops chain.
- Config changed while running → StartAbortPoller stops old goroutine, starts new one.
- Ctrl+Shift+Esc opens Task Manager → intentional, no conflict.

---

## Implementation Order

1. `chainabort.go` — hotkey parsing + polling
2. `windowlock.go` — lock/clear/verify
3. `config.go` — new fields + defaults
4. `chain.go` — abort wiring + goroutine leak fix + lock-on integration
5. `server.go` — 3 new tools + set_config args + auto-lock on focus
6. `main.go` — startup poller
7. Tests
8. Docs update
