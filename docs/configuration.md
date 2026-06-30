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

## Privacy Controls

See [`security.md`](security.md) for the full data collection and privacy controls reference.

---

<sub><sup>
here are 12 configuration fields, and you will still need to tweak all of them because the defaults were chosen at 3am. `watcher_auto_start` defaults to `false` because we didn't trust ourselves not to fill your disk with screenshots. `training_enabled` defaults to `true` because we DO trust ourselves to fill your disk with screenshots. the duality of the developer. `verify_bounds` exists because someone clicked at coordinates that didn't exist and blamed us. fair.
</sup></sub>
