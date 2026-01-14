# nvim-snap

NeovimのUI描画スナップショットを取得するためのPoCです。ケースディレクトリで
シナリオ/設定/出力をまとめて扱います。

## 使い方

```sh
nvim --headless -u NONE -i NONE -l snap.lua capture --case snapcase-example
```

生成物は `snapcase-example/.out/` に出力されます（`capture` は省略可）。

## コマンド

- `capture` スナップショット生成（デフォルト）
- `normalize` スナップショットJSONの正規化
- `compare` スナップショットJSONの比較

## ケース構成

- `snapcase-example/snapcase.json` ケース定義
- `snapcase-example/snapcase.schema.json` JSON Schema
- `snapcase-example/scenario.lua` 操作シナリオ

## 開発メモ

- `vim.pack.add()` を使う場合は `--case` で `data_home` / `config_home`
  をケース内に設定するのが前提です。
