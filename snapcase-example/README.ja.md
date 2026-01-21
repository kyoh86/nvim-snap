# snapcase-example

このREADMEはサンプルケースの最小構成と実行方法に限定した説明です。

`nvim-snap` を使ったスナップショット生成の最小構成です。

## 使い方

```sh
nvim-snap golden test --root snapcase-example --cases-dir snapcase --output diff --diff-format text
```

## 構成

- `snapcase.schema.json` ケース定義と取得設定用のJSON Schema
- `snapcase/` サンプルケース
  - `regression/` リグレッションケース
    - `diff-example/` 差分確認向けのリグレッションケース
  - `golden/` ゴールデンケース
    - `qlean-hidden/` `vim.pack.add()` で `qlean.nvim` を取得するゴールデンケース
    - `qlean-quickfix/` `vim.pack.add()` で `qlean.nvim` を取得するゴールデンケース

## 補足

`snapcase.json` はケースごとの capture 設定に使います。  
`qlean-*` は `git` とネットワークアクセス、`vim.pack` が使えるNeovim（0.12+）が必要です。
