# nvim-snap

nvim-snapは、NeovimのUIスナップショットに基づくテストを実行できるようにするためのツールです。

## コンセプト

シナリオを実行して現在のUIスナップショットを生成し、保存済みの期待値と比較することで表示の一致を検証する。
テストケース単位でシナリオと期待値を管理し、一括実行やタグでの絞り込みができる。

## 使い方

`snapcase-example` を同梱しているので、そのままケースを実行して動作を確認できます。

## コマンド

- `list` テストケース一覧
- `new` テストケース雛形の作成
- `run` テストケース実行（スナップショット生成）
- `compare` テストケース比較
- `update-expected` expected更新

## 例

![diff overlayの例](docs/diff-overlay.png)

## 典型ワークフロー

1. スナップショットを生成  
   `nvim --headless -u NONE -i NONE -l snap.lua run --root snapcase-example --format json`
2. 比較（CI向け）  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --root snapcase-example --format text`
3. 人間向けdiff（HTML）  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --root snapcase-example --format html`
4. 期待値を更新  
   `nvim --headless -u NONE -i NONE -l snap.lua update-expected --root snapcase-example`

## サンプル構成

- `snapcase-example/case.json` 高レイヤー用のケース定義
- `snapcase-example/case.schema.json` JSON Schema
- `snapcase-example/scenario.lua` 操作シナリオ

## 注意点

- シナリオの中でプラグインが必要な場合は、 `vim.pack.add()` を使うのがおすすめです。
- `vim.pack.add()` を使う場合は `--data-home` / `--config-home` を明示的に設定してください。
- PNG出力には `google-chrome`/`chromium`/`msedge`/`wkhtmltoimage` のいずれかが必要です。
  - Ubuntuでは `apt install chromium` がsnap版になり動かない場合があるため、`google-chrome` のdeb版や `wkhtmltoimage` を利用してください。
