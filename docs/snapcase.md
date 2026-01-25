# snapcase.json

`snapcase.json` defines a test case. The case type is determined by the directory (`regression/` or `golden/`).

## Minimal example

```json
{
  "version": 1,
  "title": "Basic Regression",
  "tags": ["ui", "regression"]
}
```

## Fields

- `version` (integer, required)
  - Schema version. Currently `1`.
- `title` (string, optional)
  - Display name. Defaults to the case directory name.
- `tags` (string[], optional)
  - Tags used by `list --tag` and filters.
- `width` / `height` (integer, optional)
  - UI size used when capturing.
- `wait` (integer, optional)
  - Wait for redraw flush (ms).
- `post_wait` (integer, optional)
  - Wait after scenario execution (ms).
- `wait_done` (boolean, optional)
  - Wait for scenario completion notification.
- `done_timeout` (integer, optional)
  - Timeout for `wait_done` (ms).
- `rpc_timeout` (integer, optional)
  - RPC timeout (ms). If `wait_done` is enabled, this is bumped above `done_timeout`.
- `log_file` (string, optional)
  - Neovim log file path (`NVIM_LOG_FILE`). Relative paths are resolved from the case directory.
- `log_level` (string, optional)
  - Neovim log level (`NVIM_LOG_LEVEL`).
- `data_home` / `config_home` (string, optional)
  - XDG data/config home. Relative paths are resolved from the case directory.
- `rtp` (string or string[], optional)
  - Runtimepath entries to prepend. `${CASE}` and `${ROOT}` placeholders are supported.

## Schema

See `snapcase.schema.json` for the formal JSON Schema.
