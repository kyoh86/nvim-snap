# nvim-snap

nvim-snapは、NeovimのUIスナップショットに基づくテストを実行できるようにするためのツールです。

## コンセプト

シナリオを実行して現在のUIスナップショットを生成し、保存済みの期待値と比較することで表示の一致を検証する。
テストケース単位でシナリオと期待値を管理し、一括実行やタグでの絞り込みができる。

## 例

![diff overlayの例](docs/diff-overlay.png)

`snapcase-example` を同梱しているので、そのままケースを実行して動作を確認できます。

## インストール

簡易インストーラを使う場合:

```sh
git clone https://github.com/kyoh86/nvim-snap.git
cd nvim-snap
./install.sh
```

`PREFIX` を指定するとインストール先を変更できます（既定は `~/.local`）。
`~/.local/bin` がPATHに入っていない場合は追加してください。

単一ファイルの `nvim-snap` を `~/.local/bin` に配置します。
`dist/nvim-snap` が存在しない場合は `scripts/bundle.sh` を実行します。

### 依存関係

- Neovim（`nvim`）
- 単一ファイル配布の生成に `luabundler`（Node.js、開発時のみ）
- PNG出力に `google-chrome`/`chromium`/`msedge`/`wkhtmltoimage`

### 単一ファイル配布

`luabundler` を使って `dist/nvim-snap` を生成できます。

```sh
scripts/bundle.sh
```

生成された `dist/nvim-snap` は単一ファイルで配布できます。
リリースページに添付されるため、そこから直接ダウンロードして使うこともできます。

### mise での利用

```sh
mise use 'github:kyoh86/nvim-snap[asset_pattern=nvim-snap]'
```

## コマンド

- `list` テストケース一覧
- `new` テストケース雛形の作成
- `ci init` CI向けワークフロー雛形の作成
- `run` テストケース実行（スナップショット生成）
- `compare` テストケース比較
- `update-expected` expected更新

### list

テストケースを一覧表示します。

```sh
nvim-snap list --root .
nvim-snap list --tag ui --tag regression
nvim-snap list --case basic-regression
nvim-snap list --json
```

### new

テストケースの雛形を作成します。

```sh
nvim-snap new --root tests/cases --id basic-regression --kind regression
nvim-snap new --dir tests/cases/sample --kind golden --name "Sample Golden"
```

#### 雛形の進め方

- リグレッション（`scenario.lua`）:
  - `scenario.lua` に操作手順を書く
  - `nvim-snap run` で `actual/` を生成
  - `nvim-snap update-expected` で `expected/` を更新
  - `nvim-snap compare` で差分確認
- ゴールデン（`golden.lua`/`target.lua`）:
  - `golden.lua` に期待表示を作る手順を書く
  - `target.lua` に実装側の表示を作る手順を書く
  - `nvim-snap run` で `actual/` を生成
  - `nvim-snap update-expected` で `expected/` を更新（`golden.lua` を実行）
  - `nvim-snap compare` で差分確認

### run

テストケースを実行して `actual/` にスナップショットを生成します。

```sh
nvim-snap run --root tests/cases --format json,html
nvim-snap run --tag ui
```

### compare

`expected/` と `actual/` を比較し、差分がある場合は `diff/` に出力します。

```sh
nvim-snap compare --root tests/cases --format text
nvim-snap compare --root tests/cases --format html --diff-always
nvim-snap compare --root tests/cases --format png --diff-always
```

### update-expected

期待値を更新します。リグレッションは `actual` を採用し、ゴールデンは `golden.lua` を実行します。

```sh
nvim-snap update-expected --root tests/cases
nvim-snap update-expected --root tests/cases --dry-run
nvim-snap update-expected --root tests/cases --no-confirm
```

### ci init

CI向けワークフロー雛形を作成します。

```sh
nvim-snap ci init --path .github/workflows/nvim-snap.yml
```

## 典型ワークフロー

1. スナップショットを生成  
   `nvim-snap run --root snapcase-example --format json`
2. 比較（CI向け）  
   `nvim-snap compare --root snapcase-example --format text`
3. 人間向けdiff（HTML）  
   `nvim-snap compare --root snapcase-example --format html`
4. 期待値を更新  
   `nvim-snap update-expected --root snapcase-example`

## サンプル構成

- `snapcase-example/case.json` 高レイヤー用のケース定義
- `snapcase-example/case.schema.json` JSON Schema
- `snapcase-example/scenario.lua` 操作シナリオ

## 注意点

- シナリオの中でプラグインが必要な場合は、 `vim.pack.add()` を使うのがおすすめです。
- `vim.pack.add()` を使う場合は `--data-home` / `--config-home` を明示的に設定してください。
