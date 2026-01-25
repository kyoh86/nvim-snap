# snapcase-example

`nvim-snap` を使ったスナップショット生成の最小構成です。

## 使い方

```console
$ nvim-snap golden test --root snapcase-example --cases-dir snapcase --output diff --diff-format text
```

## 構成

- `snapcase/` サンプルケース
  - `regression/` リグレッションケース
    - `diff-example/` 差分確認向けのリグレッションケース
    - `layout-splits/` 分割レイアウト向けのリグレッションケース
  - `golden/` ゴールデンケース
    - `floating-golden/` 浮動ウインドウ向けのゴールデンケース
    - `qlean-hidden/` `vim.pack.add()` で `qlean.nvim` を取得するゴールデンケース
    - `qlean-quickfix/` `vim.pack.add()` で `qlean.nvim` を取得するゴールデンケース
    - `quickfix-golden/` quickfix向けのゴールデンケース
- `failure/` 意図的にdiffが出るケース（プレビュー/検証用）
  - `golden/` ゴールデンケース
    - `diff-golden/` 必ずdiffが出るゴールデンケース

## 失敗ケース

goldenで比較したときにdiffが出るケースを別のディレクトリ(`failure`)に用意しています

```console
$ nvim-snap golden test --root snapcase-example --cases-dir failure --output diff --diff-format text
```

## 補足

`qlean-*` は `git` とネットワークアクセス、`vim.pack` が使えるNeovim（0.12+）が必要です。
