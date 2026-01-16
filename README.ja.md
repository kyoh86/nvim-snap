# nvim-snap

NeovimのUI描画のスナップショットを取得するためのPoCです。
スナップショットはJSON/ANSI/HTMLで出力できます。

## 使い方

`snapcase-example/scenario.lua` を同梱しているので、そのまま呼び出して実際の動作を確認できます。

```sh
nvim --headless -u NONE -i NONE -l snap.lua capture \
  --scenario snapcase-example/scenario.lua \
  --out-dir snapcase-example/.out \
  --data-home snapcase-example/.nvim-data \
  --config-home snapcase-example/.nvim-config \
  --json --ansi --html
```

生成物は `snapcase-example/.out/` に出力されます。

## コマンド

- `capture` スナップショット生成
- `normalize` スナップショットJSONの正規化
- `compare` スナップショットJSONの比較

## 典型ワークフロー

1. スナップショットを生成  
   `nvim --headless -u NONE -i NONE -l snap.lua capture --scenario snapcase-example/scenario.lua --out-dir snapcase-example/.out --data-home snapcase-example/.nvim-data --config-home snapcase-example/.nvim-config --json`
2. 期待値を作成  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --actual snapcase-example/.out/snapshot.json --expected snapcase-example/.expected/snapshot.json --update --pretty`
3. 比較（CI向け）  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --actual snapcase-example/.out/snapshot.json --expected snapcase-example/.expected/snapshot.json --diff`
4. 人間向けdiff（HTML）  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --actual snapcase-example/.out/snapshot.json --expected snapcase-example/.expected/snapshot.json --diff --diff-format html --diff-out snapcase-example/.out/diff.html`

## 差分の例（行追加/行変更/行削除）

```sh
nvim --headless -u NONE -i NONE -l snap.lua capture \
  --scenario snapcase-example/scenario_baseline.lua \
  --out-dir snapcase-example/.out/base \
  --json
nvim --headless -u NONE -i NONE -l snap.lua capture \
  --scenario snapcase-example/scenario_alt.lua \
  --out-dir snapcase-example/.out/alt \
  --json
```

```sh
nvim --headless -u NONE -i NONE -l snap.lua compare \
  --actual snapcase-example/.out/alt/snapshot.json \
  --expected snapcase-example/.out/base/snapshot.json \
  --diff --diff-format html --diff-out snapcase-example/.out/diff-alt.html
```

## サンプル構成

- `snapcase-example/snapcase.json` 高レイヤー用のケース定義（予定）
- `snapcase-example/snapcase.schema.json` JSON Schema
- `snapcase-example/scenario.lua` 操作シナリオ

## 注意点

- シナリオの中でプラグインが必要な場合は、 `vim.pack.add()` を使うのがおすすめです。
- `vim.pack.add()` を使う場合は `--data-home` / `--config-home` を明示的に設定してください。
