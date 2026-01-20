# nvim-snap 設計メモ

## 目的

nvim-snapは、NeovimのUIスナップショットに基づくテストを実行できるようにするための仕組みを提供する。

## 要件

### テストケースの種類

個別のテストケースは次の2種類を前提とする

- リグレッションテスト
    - 現在の表示結果（current）が、前回承認した基準（accepted）と一致することを確認する
- ゴールデンテスト
    - 実装したものの表示（actual）が同一環境で生成した基準（baseline）と一致することを確認する
    - baselineの生成のために、ゴールデンシナリオは分離されている

### テストケース管理の方針

テストケースは一覧で管理でき、一括で実行できるようにする。
管理で必要な処理は次のものである。

- テストケースの表示
- テストケースの一括実行
    - current/actualのスナップショットを生成する
    - accepted/baseline と合うことを確認して、合わなければ適切にレポートする
- 基準の更新
    - リグレッションテストなら、今の結果（current）をacceptedとして採用する
    - ゴールデンテストなら、用意されたシナリオからbaselineを生成する

必要に応じて処理対象のテストケースをタグで絞り込みを行えるようにする。
また、個別のテストケースを直接指定して実行できるようにする。

レポートは標準出力でのサマリを基本とし、失敗したテストケースについては
結果とHTML差分のパス、簡単なエラー理由を出力する。

### 低/高レイヤーの責務

低レイヤーは `capture/normalize/compare` をパス指定で完結するプリミティブとして提供する。
高レイヤーはケース単位の運用を担い、一覧化・一括実行・期待値の扱いなどを受け持つ。

## 設計

### 低レイヤーのコマンド

- `capture`: シナリオと設定に従ってスナップショットを出力する
  - 入力: シナリオファイル、出力ディレクトリ、出力形式、UIサイズ、nvim実行パス、data/config home
  - 出力: `snapshot.json` / `snapshot.ansi` / `snapshot.html`（指定された形式のみ）
- `normalize`: スナップショットJSONを正規化する
  - 入力: スナップショットJSON（ファイルまたは標準入力）
  - 出力: 正規化済みJSON（ファイルまたは標準出力）
- `compare`: スナップショットJSON同士を比較する
  - 入力: baseline/actual または accepted/current のスナップショットJSON
  - 出力: 一致/不一致の判定と、必要に応じたdiff出力（text/ansi/html）
  - 期待値の更新は高レイヤーで行う

CLIでは `nvim-snap <command>` で呼び出す。

### 高レイヤーのコマンド

ケース探索の起点は `--root`、その配下に `--cases-dir`（既定は `snapcase/`）を掘る。
ケースは `<root>/<cases-dir>/<case-name>/snapcase.json` に配置する。
`list/run/compare/accept/golden` は `--tag` / `--case` で絞り込みできる。

- `list`: テストケースの一覧を表示する
  - 入力: `--root`（既定は `.`）、`--cases-dir`（既定は `snapcase`）
  - 対象: `<root>/<cases-dir>/*/snapcase.json`
  - 出力: 一覧（name/title/kind/tags/path）
  - 形式: 既定はテキスト、`--json` でJSON出力
  - JSON出力:
    - `root` ルートディレクトリ
    - `cases[]` ケース配列
      - `name` / `title` / `kind` / `tags` / `path`
    - 例:
      ```json
      {
        "root": ".",
        "cases": [
          {
            "name": "case-name",
            "title": "Case Title",
            "kind": "regression",
            "tags": [
              "ui",
              "regression"
            ],
            "path": "tests/cases/case-name"
          }
        ]
      }
      ```
  - 終了コード: 成功=0、失敗=1
- `new`: テストケースの雛形を作成する
  - 入力: `--root`（既定は `.`）、`--cases-dir`（既定は `snapcase`）、`--name`（省略時は自動生成）、`--title`、`--kind`、`--tag`
  - 出力: ケースディレクトリ配下の`snapcase.json`とシナリオ雛形、`accepted/current/diff`（regression）または`baseline/actual/diff`（golden）の作成
  - `--force` で既存ファイルを上書きできる
- `init`: GitHub Actions向けのワークフロー雛形を作成する
  - 入力: `--path`（既定は `.github/workflows/nvim-snap.yml`）、`--root`（既定は `.`）、`--cases-dir`（既定は `snapcase`）、`--diff-format`
  - 出力: ワークフローYAML
  - `--force` で既存ファイルを上書きできる
- `run`: テストケースを実行してcurrent/actualを生成する
  - 入力: `--root`（既定は `.`）、`--cases-dir`（既定は `snapcase`）
  - 対象: `<root>/<cases-dir>/*/snapcase.json`
  - 出力: regressionは`current/`、goldenは`actual/`配下にスナップショット
  - 出力形式: `--format=json,ansi,html`（既定は `json`）
  - 終了コード: 成功=0、失敗=1
- `compare`: accepted/current または baseline/actual を比較し、結果をサマリ出力する
  - 入力: `--root`（既定は `.`）、`--cases-dir`（既定は `snapcase`）
  - 対象: `<root>/<cases-dir>/*/snapcase.json`
  - 出力: 標準出力のサマリ/差分/JSON
  - 出力形式: `--output=summary|diff|json`（既定は `summary`）
  - diff出力形式: `--diff-format=text,ansi,html,png`（既定は `text`）
  - HTML差分は不一致時のみ生成し、`--diff-always` で常時生成できる
  - サマリ項目: name/title/kind/tags/result/diff_paths/error_reason
  - resultは `no_diff` / `diff` / `error`
  - headless実行で入力待ちが起きる可能性があるため、シナリオでは `vim.api.nvim_cmd` の利用を推奨する
  - JSON出力:
    - `root` ルートディレクトリ
    - `summary` 集計
      - `total` / `no_diff` / `diff` / `error`
    - `cases[]` ケース配列
      - `name` / `title` / `kind` / `tags` / `result` / `diff_paths` / `error_reason`
      - `diff_paths` は出力形式ごとのパス
    - 例:
      ```json
      {
        "root": ".",
        "summary": {
          "total": 1,
          "no_diff": 0,
          "diff": 1,
          "error": 0
        },
        "cases": [
          {
            "name": "case-name",
            "title": "Case Title",
            "kind": "golden",
            "tags": [
              "ui",
              "golden"
            ],
            "result": "diff",
            "diff_paths": {
              "text": "tests/cases/case-name/diff/diff.txt",
              "ansi": "tests/cases/case-name/diff/diff.ansi",
              "html": "tests/cases/case-name/diff/diff.html"
            },
            "error_reason": null
          }
        ]
      }
      ```
  - 終了コード: 成功=0、差分あり=1、エラーあり=2
- `accept`: リグレッションのacceptedを更新する
  - 入力: `--root`（既定は `.`）、`--cases-dir`（既定は `snapcase`）
  - 対象: `<root>/<cases-dir>/*/snapcase.json`
  - 出力: `accepted/`配下の更新（currentをacceptedとして採用）
  - `--dry-run` で更新内容を表示のみ
  - 規定は対話確認
  - `--no-confirm`（または `--yes`）で対話を省略して実行
- `golden`: ゴールデンのbaselineを生成する
  - 入力: `--root`（既定は `.`）、`--cases-dir`（既定は `snapcase`）
  - 対象: `<root>/<cases-dir>/*/snapcase.json`
  - 出力: `baseline/`配下の更新（`golden.lua` を実行して生成）
  - `--dry-run` で更新内容を表示のみ
  - 終了コード: 成功=0、失敗=1

### テストケース定義

テストケースは1ケース=1ディレクトリで管理する。
各ディレクトリにケース定義（JSON）とシナリオと成果物を配置する。

#### ケースディレクトリ構成

- `snapcase.json` ケース定義ファイル（後述）
- regression: `accepted/`（基準）と`current/`（実行結果）
- golden: `baseline/`（基準）と`actual/`（実行結果）
- `diff/` 比較結果（HTMLなど、任意）
- `scenario.lua` リグレッションテスト用のシナリオ
- `golden.lua` ゴールデンテスト用のゴールデンシナリオ
- `target.lua` ゴールデンテスト用の実装結果シナリオ

#### ケース定義（`snapcase.json`）

- `version` 定義のバージョン
- `title` 表示名（任意、既定はケース名）
- `kind` `regression` または `golden`
- `tags` タグ配列（任意）
  - ケース名はディレクトリ名で決まり、`snapcase.json` には書かない

例:
```json
{
  "version": 1,
  "title": "Basic Regression",
  "kind": "regression",
  "tags": [
    "ui",
    "regression"
  ]
}
```

#### スキーマ

`snapcase.json` 用のJSON Schemaを用意する。
