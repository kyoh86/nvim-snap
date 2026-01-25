# snapcase.json

`snapcase.json` はテストケース定義ファイルです。

## 最小構成

```json
{
  "version": 1,
  "title": "Basic Regression",
  "tags": ["ui", "regression"]
}
```

## フィールド

- `version`（integer, 必須）
  - スキーマのバージョン。現在は `1`。
- `title`（string, 任意）
  - 表示名。省略時はケースのディレクトリ名。
- `tags`（string[], 任意）
  - `list --tag` などの絞り込みに使うタグ。
- `width` / `height`（integer, 任意）
  - 取得時の UI サイズ。
- `wait`（integer, 任意）
  - redraw flush の待機時間 (ms)。
- `post_wait`（integer, 任意）
  - シナリオ実行後の待機時間 (ms)。
- `wait_done`（boolean, 任意）
  - シナリオ完了通知を待機する。
- `done_timeout`（integer, 任意）
  - `wait_done` のタイムアウト (ms)。
- `rpc_timeout`（integer, 任意）
  - RPC タイムアウト (ms)。`wait_done` 有効時は `done_timeout` より長くなるよう調整される。
- `log_file`（string, 任意）
  - Neovim のログ出力先 (`NVIM_LOG_FILE`)。相対パスはケースディレクトリ基準。
- `log_level`（string, 任意）
  - Neovim のログレベル (`NVIM_LOG_LEVEL`)。
- `data_home` / `config_home`（string, 任意）
  - XDG data/config home。相対パスはケースディレクトリ基準。
- `rtp`（string または string[], 任意）
  - runtimepath に追加するパス。`${CASE}` と `${ROOT}` が使える。

## スキーマ

正式な JSON Schema は `snapcase.schema.json` を参照してください。
