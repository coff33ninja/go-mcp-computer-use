# Configuration

`~/.config/go-mcp-computer-use/config.json`:

```json
{
  "log_level": "info",
  "mouse_speed": 500,
  "click_delay_ms": 100,
  "verify_bounds": true,
  "action_timeout_ms": 30000,
  "uia_warmup": true,
  "training_enabled": true,
  "prior_adjustment": true,
  "watcher_auto_start": false,
  "watcher_interval_seconds": 5
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `log_level` | `info` | One of: `debug`, `info`, `warn`, `error` |
| `mouse_speed` | `500` | Mouse movement speed |
| `click_delay_ms` | `100` | Delay between mouse down/up (ms) |
| `verify_bounds` | `true` | Validate coordinates against screen bounds |
| `action_timeout_ms` | `30000` | Max time (ms) for blocking operations |
| `uia_warmup` | `true` | Warm up UIA at startup (async) to avoid cold-start delay. Set `false` if clients timeout during init. |
| `training_enabled` | `true` | Enable auto-save training snapshots on every UI action. Set `false` to stop all background data collection (also controllable at runtime via `set_config`). |
| `prior_adjustment` | `true` | Apply learned element frequency/position priors to ONNX detection scores. Set `false` for raw YOLO output only. |
| `watcher_auto_start` | `false` | Auto-start the background watcher on server boot. Watcher polls screen every N seconds and saves frames for training. |
| `watcher_interval_seconds` | `5` | How often the watcher captures and analyzes the screen (if running). Also used as default when starting via `set_config`. |
| `tool_denylist` | `[]` | Array of tool names to remove from the MCP server entirely (case-insensitive). Denied tools are invisible to the AI agent. Configurable at runtime via `set_config`. |
| `retention_days` | `0` | Auto-prune training samples older than N days. Deletes both database rows and image files. Background pruner runs every 6 hours. Set `0` to disable (default). Requires `training_enabled: true`. Configurable at runtime via `set_config`. |

## Transformer Model (Auto-Discovery)

The Go-native transformer engine requires no configuration. On server startup:

1. If `model.gob` exists in the data directory → loads it automatically
2. If no model exists → trains from the `training_pairs` table in SQLite
3. After each session → new training pairs are available for the next training cycle

No config fields needed — the transformer is self-managing.

## Privacy Controls

See [`../security.md`](../security.md) for the full data collection and privacy controls reference.
