# nvim-snap

nvim-snapは、NeovimのUIスナップショットに基づくテストを実行できるようにするためのツールです。

## コンセプト

シナリオを実行して現在のUIスナップショットを生成し、保存済みの期待値と比較することで表示の一致を検証する。
テストケース単位でシナリオと期待値を管理し、一括実行やタグでの絞り込みができる。

## 例

![diff overlayの例](docs/diff-overlay.png)

このリポジトリ内に `snapcase-example` を同梱しているので、そのままケースを実行して動作を確認できます。
通常は `snapcase/` 配下にケースを作成します。

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

以下の説明は、テストケースが `snapcase/<case-name>/snapcase.json` のように配置されている前提です。
各ケースディレクトリに `snapcase.json` を置き、ケースの情報と `core capture` の設定をまとめて管理します。
`snapcase.json` の `rtp` は文字列または配列で、runtimepathに追加するパスを指定します。

- `list` テストケース一覧
- `new` テストケース雛形の作成
- `init` CI向けワークフロー雛形の作成
- `run` テストケース実行（スナップショット生成）
- `compare` テストケース比較
- `update-expected` expected更新

### list

テストケースを一覧表示します。

```sh
nvim-snap list
nvim-snap list --tag ui --tag regression
nvim-snap list --case basic-regression
nvim-snap list --json
```

### new

テストケースの雛形を作成します。

```sh
nvim-snap new --name basic-regression --kind regression
nvim-snap new --name sample --kind golden --title "Sample Golden"
```

`--name` を省略するとランダムなケース名を生成し、`snapcase/` 配下にディレクトリを作成します。


### run

テストケースを実行して `actual/` にスナップショットを生成します。

```sh
nvim-snap run --format json,html
nvim-snap run --tag ui
```

### compare

`expected/` と `actual/` を比較し、差分がある場合は `diff/` に出力します。

```sh
nvim-snap compare --format text
nvim-snap compare --format html --diff-always
nvim-snap compare --format png --diff-always
```

### update-expected

期待値を更新します。リグレッションは `actual` を採用し、ゴールデンは `golden.lua` を実行します。

```sh
nvim-snap update-expected
nvim-snap update-expected --dry-run
nvim-snap update-expected --no-confirm
```

### init

CI向けワークフロー雛形を作成します。

```sh
nvim-snap init --path .github/workflows/nvim-snap.yml
```

## 実用時の流れ

1. ケースを作る  
   `nvim-snap new --name my-case --kind regression`
2. シナリオを書く  
   - リグレッションは `scenario.lua`  
   - ゴールデンは `golden.lua` / `target.lua`
3. actualを生成  
   `nvim-snap run`
4. expectedを更新  
   `nvim-snap update-expected`
5. 差分を確認  
   `nvim-snap compare --format html`

## テストの種類

- リグレッション: 同一シナリオの結果が過去の期待値と一致するか確認する
- ゴールデン: 期待表示（golden）と実装結果（target）の一致を確認する

## 注意点

- シナリオの中でプラグインが必要な場合は、 `vim.pack.add()` を使うのがおすすめです。
- `vim.pack.add()` を使う場合は `--data-home` / `--config-home` を明示的に設定してください。
