# nvim-snap

nvim-snapは、NeovimのUIスナップショットに基づくテストを実行できるようにするためのツールです。

## コンセプト

シナリオを実行してUIスナップショットを生成し、結果を比較します。
リグレッションはコミット単位で保存した結果同士を比較し、ゴールデンは同一実行内でgolden/targetを比較します。

## 例

![diff overlayの例](docs/diff-overlay.png)

このリポジトリ内に `snapcase-example` を同梱しているので、そのままケースを実行して動作を確認できます。
詳しい配置ルールは「テストケースの配置」、実行方法は `list` と `regression` / `golden` の項を参照してください。

例:

```sh
nvim-snap list --root snapcase-example
nvim-snap golden test --root snapcase-example
```

## インストール

### 依存関係

- Neovim（`nvim`）
- PNG出力を使用する場合はさらに `google-chrome`/`chromium`/`msedge`/`wkhtmltoimage` のいずれか

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

## テストケースの配置

ケースは以下のいずれかの`snapcase.json`として配置します。

- `snapcase/`
    - `regression/`
        - `<case-name>/`
            - `snapcase.json`
            - `scenario.lua`
    - `golden/`
        - `<case-name>/`
            - `snapcase.json`
            - `golden.lua`
            - `target.lua`

ケースの名前はディレクトリ名で決まり、`snapcase.json` にメタデータと取得設定を記述します。
`snapcase.json` の詳細は「snapcase.json」の項を参照してください。

## テストの種類

- リグレッション: 同一シナリオの結果をコミット単位で保存して比較します
- ゴールデン: golden/target を同一実行で比較します

## 実用時の流れ

### リグレッションの流れ

1. リグレッションケースを作る  
   `nvim-snap regression new --name my-case`
2. シナリオを書く  
   `scenario.lua`
3. ベースコミットでスナップショットを作り保存  
   `nvim-snap regression save`
4. ターゲットコミットでスナップショットを作り保存  
   `nvim-snap regression save`
5. 比較  
   `nvim-snap regression test --base <base-id> --target <target-id>`

`regression test` は保存済みのスナップショットのみ比較します。CIでは `.result/` をキャッシュしてベース側を用意してください。

### ゴールデンの流れ

1. ゴールデンケースを作る  
   `nvim-snap golden new --name my-case`
2. シナリオを書く  
   `golden.lua` / `target.lua`
3. 実行と比較  
   `nvim-snap golden test --output summary --diff-format html`

## コマンド

用意したテストケースに対して、以下のコマンドを使用できます。
それぞれの詳細を後述しますが、より詳しいフラグなどの説明は `--help` フラグでの出力を確認してください。

- `list` テストケース一覧
- `init` CI向けワークフロー雛形の作成
- `capture` シナリオからスナップショットを取得
- `normalize` スナップショットJSONを正規化
- `compare` スナップショットJSONを比較
- `regression new` リグレッションケース雛形の作成
- `regression save` コミット単位で保存
- `regression test` 保存済みスナップショットの比較
- `golden new` ゴールデンケース雛形の作成
- `golden test` ゴールデン/ターゲットの比較

### list

テストケースを一覧表示します。

```sh
nvim-snap list
nvim-snap list --tag ui --tag regression
nvim-snap list --case basic-regression
nvim-snap list --json
```

### capture

シナリオからスナップショットを取得します。

```sh
nvim-snap capture --scenario scenario.lua --out ./out --format json,ansi,html
```

### normalize

スナップショットJSONを正規化します。

```sh
nvim-snap normalize --in snapshot.json --out normalized.json
nvim-snap normalize --in snapshot.json
```

### compare

スナップショットJSONを比較します。

```sh
nvim-snap compare --expected expected.json --actual actual.json --format text
nvim-snap compare --expected expected.json --actual actual.json --format html --out diff.html
```

### regression new

リグレッションケースの雛形を作成します。

```sh
nvim-snap regression new --name basic-regression
```

`--name` を省略するとランダムなケース名を生成し、`<root>/<cases-dir>/regression/` 配下にディレクトリを作成します。

### regression save

現在のコミット（既定）でスナップショットを保存します。

```sh
nvim-snap regression save
nvim-snap regression save --id abcdef1234
nvim-snap regression save --tag ui
```

### regression test

保存済みスナップショットを比較します。
`--target` を省略すると現在のコミットIDを使います。

```sh
nvim-snap regression test --base abcdef1234 --target 0123456789
nvim-snap regression test --base abcdef1234 --output diff --diff-format text
```

### golden new

ゴールデンケースの雛形を作成します。

```sh
nvim-snap golden new --name sample-golden --title "Sample Golden"
```

### golden test

golden/targetを実行して比較します。

```sh
nvim-snap golden test --output summary --diff-format html
nvim-snap golden test --output diff --diff-format text --diff-always
```

### init

CI向けワークフロー雛形を作成します。

```sh
nvim-snap init --path .github/workflows/nvim-snap.yml
```

## snapcase.json

`snapcase.json` はケース定義ファイルです。最低限 `version` を指定し、必要に応じて `title` / `tags` を設定します。
実行ファイル名はリグレッションでは `scenario.lua`、ゴールデンでは `golden.lua` / `target.lua` で固定です。
詳細な項目は `snapcase.schema.json` を参照してください。

## 補足と注意点

- テスト結果の出力は `<root>/<cases-dir>/.result/` に保存されます。
- `regression save` は既定で現在のGitコミットIDを使い、作業ツリーがdirtyだとエラーになります。
    - Dirtyでもsaveしたい場合は、`--id`フラグで適切なIDを明示してください。
- `regression test` はスナップショットを生成せず、保存済みの結果だけを比較します。
- シナリオの中でプラグインが必要な場合は、 `vim.pack.add()` を使うのがおすすめです。
    - `vim.pack.add()` を使う場合は `snapcase.json` の `data_home` / `config_home` を明示的に設定してください。
    - `data_home`や`config_home`を指定しない場合Neovim環境やテストケースと相互に干渉します。
- テストケースの実行にはNeovimのheadlessモードを使用しています。
    - 入力待ちが発生する処理が入力待ちのまま止まることがあります。
    - 一部のコマンドは標準コマンドを含めて入力待ちを生じさせやすいです。`vim.cmd` より `vim.api.nvim_cmd` の使用を推奨します。
- 非同期処理を呼び出す場合は、シナリオの終了を知らせる必要があります。
    - `snapcase.json` の `wait_done` を `true` にします。
    - シナリオの終了時（キャプチャを取っていいタイミング）に `require("nvim_snap").done()` を呼び出してください。
