# snapcase-example

このREADMEはサンプルケースの最小構成と実行方法に限定した説明です。

`nvim-snap` を使ったスナップショット生成の最小構成です。

## 使い方

```sh
nvim-snap run --root snapcase-example --cases-dir cases --case diff-example --format json,ansi,html
```

## 構成

- `snapcase.schema.json` ケース定義と取得設定用のJSON Schema
- `cases/` 複数ケースのサンプル（regression/golden混在）
  - `diff-example/` 差分確認向けのリグレッションケース
  - `qlean-hidden/` `vim.pack.add()` で `qlean.nvim` を取得するゴールデンケース
  - `qlean-quickfix/` `vim.pack.add()` で `qlean.nvim` を取得するゴールデンケース

## 補足

`snapcase.json` はケースごとの capture 設定に使います。  
`qlean-*` は `git` とネットワークアクセスが必要です。
