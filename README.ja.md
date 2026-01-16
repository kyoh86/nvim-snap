# nvim-snap

NeovimのUI描画のスナップショットを取得するためのPoCです。
スナップショットはJSON/ANSI/HTMLで出力できます。

## 使い方

`snapcase-example/scenario.lua` を同梱しているので、そのまま呼び出して実際の動作を確認できます。

```sh
nvim --headless -u NONE -i NONE -l snap.lua core capture \
  --scenario snapcase-example/scenario.lua \
  --out-dir snapcase-example/.out \
  --data-home snapcase-example/.nvim-data \
  --config-home snapcase-example/.nvim-config \
  --json --ansi --html
```

生成物は `snapcase-example/.out/` に出力されます。

## コマンド

- `list` テストケース一覧（高レイヤー）
- `run` テストケース実行（高レイヤー）
- `compare` テストケース比較（高レイヤー）
- `update-expected` expected更新（高レイヤー）
- `core capture` スナップショット生成（低レイヤー）
- `core normalize` スナップショットJSONの正規化（低レイヤー）
- `core compare` スナップショットJSONの比較（低レイヤー）

## 典型ワークフロー

1. actualを生成  
   `nvim --headless -u NONE -i NONE -l snap.lua run --root snapcase-example --format json`
2. 比較（CI向け）  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --root snapcase-example --format text`
3. 人間向けdiff（HTML）  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --root snapcase-example --format html`
4. expectedを更新  
   `nvim --headless -u NONE -i NONE -l snap.lua update-expected --root snapcase-example`

## 差分の例（行追加/行変更/行削除）

```sh
nvim --headless -u NONE -i NONE -l snap.lua core capture \
  --scenario snapcase-example/scenario_baseline.lua \
  --out-dir snapcase-example/.out/base \
  --json
nvim --headless -u NONE -i NONE -l snap.lua core capture \
  --scenario snapcase-example/scenario_alt.lua \
  --out-dir snapcase-example/.out/alt \
  --json
```

```sh
nvim --headless -u NONE -i NONE -l snap.lua core compare \
  --actual snapcase-example/.out/alt/snapshot.json \
  --expected snapcase-example/.out/base/snapshot.json \
  --diff --diff-format html --diff-out snapcase-example/.out/diff-alt.html
```

## サンプル構成

- `snapcase-example/case.json` 高レイヤー用のケース定義
- `snapcase-example/case.schema.json` JSON Schema
- `snapcase-example/scenario.lua` 操作シナリオ

## 注意点

- シナリオの中でプラグインが必要な場合は、 `vim.pack.add()` を使うのがおすすめです。
- `vim.pack.add()` を使う場合は `--data-home` / `--config-home` を明示的に設定してください。
