# snapcase-example

このREADMEはサンプルケースの最小構成と実行方法に限定した説明です。

`snap.lua` を使ったスナップショット生成の最小構成です。

## 使い方

```sh
nvim --headless -u NONE -i NONE -l snap.lua \
  capture --scenario snapcase-example/scenario.lua \
  --out-dir snapcase-example/.out \
  --data-home snapcase-example/.nvim-data \
  --config-home snapcase-example/.nvim-config \
  --json --ansi --html
```

## 構成

- `snapcase.json` 高レイヤー用のケース定義（予定）
- `snapcase.schema.json` JSON Schema
- `scenario.lua` 操作シナリオ
- `scenario_baseline.lua` 行数と内容が基準のサンプル
- `scenario_alt.lua` 行追加/行変更/行削除が入ったサンプル

## 補足

`snapcase.json` は高レイヤーのcase管理向けに予約しているため、現状の `capture` は参照しません。
