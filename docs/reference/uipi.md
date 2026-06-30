# UIPI (User Interface Privilege Isolation) Detection

File: `internal/actions/uipi.go` (110 lines)

Windows UIPI blocks low-integrity (non-admin) processes from sending input (`SendInput`) to elevated (admin) windows. The MCP server detects this mismatch and returns a clear error so the user knows to relaunch as admin.

## Detection Functions

### isForegroundElevated() → (bool, error)
1. `GetForegroundWindow` → hwnd
2. `GetWindowThreadProcessId` → pid
3. `OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION)` → hProcess
   - If `ERROR_ACCESS_DENIED` → process is elevated (admin processes deny `PROCESS_QUERY_LIMITED_INFORMATION` to non-elevated callers)
4. `OpenProcessToken(hProcess, TOKEN_QUERY)` → hToken
5. `GetTokenInformation(hToken, TokenElevation=20)` → `TOKEN_ELEVATION` struct
6. Returns `elevated != 0`

### isProcessElevated(pid uint32) → (bool, error)
Same logic as above but for an arbitrary PID.

### isSelfElevated() → (bool, error)
Calls `isProcessElevated(GetCurrentProcessId())`.

### warnElevated() → error
Three-tier check:
1. **Target check** — is foreground window elevated? If not → return nil
2. **Self check** — is the MCP server itself elevated? If yes → UIPI allows same-integrity input → return nil
3. **Mismatch** — foreground is elevated, server is not → return descriptive error with remediation

## Error Message

```
foreground window is elevated (admin). Input from non-elevated MCP server
is blocked by Windows UIPI. Run mcp-server.exe as Administrator or target
a non-elevated window
```

## C DLL Calls (advapi32.dll)

| Proc | Signature |
|------|-----------|
| `OpenProcessToken` | `OpenProcessToken(hProcess uintptr, DesiredAccess uint32, TokenHandle *uintptr) bool` |
| `GetTokenInformation` | `GetTokenInformation(TokenHandle uintptr, TokenInformationClass uint32, TokenInformation *uint32, TokenInformationLength uint32, ReturnLength *uint32) bool` |

Also calls via `user32.dll`: `GetForegroundWindow`, `GetWindowThreadProcessId`.<br>
Also calls via `kernel32.dll`: `OpenProcess`, `CloseHandle`.

## Call Sites

| Caller File | Function | Trigger |
|-------------|----------|---------|
| `chained.go:72` | `chainStep` | Before each chain step |
| `chained.go:185` | `chainStep` | Before chain step (alt path) |
| `keyboard.go:203` | `keyDown` (type/send) | Before key down |
| `keyboard.go:222` | `keyUp` | Before key up |
| `keyboard.go:241` | `keyPress` | Before key press |
| `keyboard.go:297` | `typeString` | Before typing text |

## Cross-Reference

- `docs/reference/windows-dll-ref.md` — advapi32.dll and user32.dll proc listings
- `docs/reference/codebase-map.md` — tool→handler→action→file mapping
