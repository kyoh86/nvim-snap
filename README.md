# nvim-snap

NeovimのUI描画スナップショットを取得するためのPoCです。ケースディレクトリで
シナリオ/設定/出力をまとめて扱います。

## 使い方

```sh
nvim --headless -u NONE -i NONE -l snap.lua --case snap_case
```

生成物は `snap_case/.out/` に出力されます。

## ケース構成

- `snap_case/case.json` ケース定義
- `snap_case/case.schema.json` JSON Schema
- `snap_case/scenario.lua` 操作シナリオ

## 開発メモ

- `vim.pack.add()` を使う場合は `--case` で `data_home` / `config_home`
  をケース内に設定するのが前提です。
