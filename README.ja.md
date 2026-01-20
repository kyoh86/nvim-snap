# nvim-snap

nvim-snapは、NeovimのUIスナップショットに基づくテストを実行できるようにするためのツールです。

## コンセプト

シナリオを実行して現在のUIスナップショットを生成し、保存済みの期待値と比較することで表示の一致を検証する。
テストケース単位でシナリオと期待値を管理し、一括実行やタグでの絞り込みができる。

## 例

![diff overlayの例](docs/diff-overlay.png)

このリポジトリ内に `snapcase-example` を同梱しているので、そのままケースを実行して動作を確認できます。
既定では `--root` はカレントディレクトリ、`--cases-dir` は `snapcase/` です。
ケースは `<root>/<cases-dir>/<case-name>/snapcase.json` に置きます。

## インストール

### 依存関係

- Neovim（`nvim`）
- PNG出力に `google-chrome`/`chromium`/`msedge`/`wkhtmltoimage`

### リリースから取得

リリースページの `nvim-snap` をダウンロードして PATH に置いてください。

### ソースからビルド

```sh
go build -o nvim-snap ./cmd/nvim-snap
```

### mise での利用

```sh
mise use github:kyoh86/nvim-snap
```


## コマンド

以下の説明は、既定のオプションで `snapcase/<case-name>/snapcase.json` のように配置されている想定です。
ケース名はディレクトリ名で決まり、`snapcase.json` でケース情報と capture の設定をまとめて管理します。
`snapcase.json` の `rtp` は文字列または配列で、runtimepathに追加するパスを指定します。
`${CASE}`（ケースディレクトリ）と `${ROOT}`（`--root` のパス）のプレースホルダが使えます。

- `list` テストケース一覧
- `new` テストケース雛形の作成
- `init` CI向けワークフロー雛形の作成
- `run` テストケース実行（スナップショット生成）
- `compare` テストケース比較
- `accept` リグレッションのexpected更新
- `golden` ゴールデンexpected生成

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

`--name` を省略するとランダムなケース名を生成し、`<root>/<cases-dir>/` 配下にディレクトリを作成します。


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

### accept

リグレッションのexpectedをactualから更新します。

```sh
nvim-snap accept
nvim-snap accept --dry-run
nvim-snap accept --no-confirm
```

### golden

`golden.lua` を実行してexpectedを生成します。

```sh
nvim-snap golden
nvim-snap golden --dry-run
```

### init

CI向けワークフロー雛形を作成します。

```sh
nvim-snap init --path .github/workflows/nvim-snap.yml
```

## 実用時の流れ

### リグレッションの流れ

1. リグレッションケースを作る  
   `nvim-snap new --name my-case --kind regression`
2. シナリオを書く  
   `scenario.lua`
3. actualを生成  
   `nvim-snap run`
4. 差分を確認  
   `nvim-snap compare --format html`
5. 判断  
   - バグ修正して再実行  
   - 仕様変更なら受け入れ: `nvim-snap accept`

### ゴールデンの流れ

1. ゴールデンケースを作る  
   `nvim-snap new --name my-case --kind golden`
2. シナリオを書く  
   `golden.lua` / `target.lua`
3. expectedを生成  
   `nvim-snap golden`
4. actualを生成  
   `nvim-snap run`
5. 差分を確認  
   `nvim-snap compare --format html`

## テストの種類

- リグレッション: 同一シナリオの結果が過去の期待値と一致するか確認する
- ゴールデン: 期待表示（golden）と実装結果（target）の一致を確認する

## 注意点

- シナリオの中でプラグインが必要な場合は、 `vim.pack.add()` を使うのがおすすめです。
- `vim.pack.add()` を使う場合は `--data-home` / `--config-home` を明示的に設定してください。
