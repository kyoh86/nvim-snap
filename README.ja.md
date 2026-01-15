# nvim-snap

このREADMEはプロジェクト全体の概要と使い方をまとめたものです。

NeovimのUI描画スナップショットを取得するためのPoCです。ケースディレクトリで
シナリオ/設定/出力をまとめて扱います。

## コンセプト

- NeovimのUI描画をスナップショット化してJSON/ANSI/HTMLで出力する
- ケースディレクトリにシナリオ/設定/出力を集約する
- 低レイヤー（スナップショット生成）と高レイヤー（比較・正規化）を分離する

## 使い方

```sh
nvim --headless -u NONE -i NONE -l snap.lua capture --case snapcase-example
```

生成物は `snapcase-example/.out/` に出力されます（`capture` は省略可）。

## コマンド

- `capture` スナップショット生成（デフォルト）
- `normalize` スナップショットJSONの正規化
- `compare` スナップショットJSONの比較

## 典型ワークフロー

1. ケースを実行してスナップショットを生成  
   `nvim --headless -u NONE -i NONE -l snap.lua capture --case snapcase-example`
2. 期待値を作成  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --case snapcase-example --expected snapcase-example/.expected/snapshot.json --update --pretty`
3. 比較（CI向け）  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --case snapcase-example --expected snapcase-example/.expected/snapshot.json --diff`

## ケース構成

- `snapcase-example/snapcase.json` ケース定義
- `snapcase-example/snapcase.schema.json` JSON Schema
- `snapcase-example/scenario.lua` 操作シナリオ

## 開発メモ

- `vim.pack.add()` を使う場合は `--case` で `data_home` / `config_home`
  をケース内に設定するのが前提です。
