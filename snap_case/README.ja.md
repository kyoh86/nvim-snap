# snap case

`snap.lua` をケースディレクトリとして実行するための最小構成です。

## 使い方

```sh
nvim --headless -u NONE -i NONE -l snap.lua \
  --case snap_case
```

## 構成

- `case.json` ケース定義
- `case.schema.json` JSON Schema
- `scenario.lua` 操作シナリオ

## case.json の主なキー

- `scenario` シナリオのパス（デフォルト: `scenario.lua`）
- `width` / `height` UIサイズ
- `data_home` / `config_home` XDGパス
- `out_dir` 出力先ディレクトリ
- `outputs` 出力ファイル名や無効化設定

`outputs` は `"none"` または `false` で無効化できます。
