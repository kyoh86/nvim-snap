# snapcase-example

このREADMEはサンプルケースの最小構成と実行方法に限定した説明です。

`nvim-snap` を使ったスナップショット生成の最小構成です。

## 使い方

```sh
nvim-snap core capture --scenario snapcase-example/cases/diff-example/scenario.lua \
  --out-dir snapcase-example/cases/diff-example/.out \
  --data-home snapcase-example/cases/diff-example/.nvim-data \
  --config-home snapcase-example/cases/diff-example/.nvim-config \
  --json --ansi --html
```

## 構成

- `case.schema.json` JSON Schema
- `cases/` 複数ケースのサンプル（regression/golden混在）
  - `diff-example/` 差分確認向けのリグレッションケース

## 補足

`case.json` は高レイヤーのcase管理向けのため、`core capture` は参照しません。
