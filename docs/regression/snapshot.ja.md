# リグレッションケースのスナップショットの取り扱い

リグレッションケースでは、保存されたスナップショットをコミット間などで比較することで、
変更を検出することを目的にしています。

そのため、リグレッションケースの実行結果は、スナップショットとして保存しておく必要があります。
nvim-snapは、リグレッションケースの実行結果を以下のディレクトリに保存します。

`<root>/<cases-dir>/.result/regression/<case-name>/snapshot-<id>.json`

## CIの場合

`nvim-snap init`で生成するGitHub Workflow定義ファイル`nvim-snap.yml`で採用している通り、
[`actions/cache`](https://github.com/actions/cache)を使用する方法を推奨しています。

```yaml
- uses: actions/cache@v4
  with:
    path: snapcase-example/snapcase/.result
    key: nvim-snap-${{ runner.os }}-${{ github.sha }}
    restore-keys: |
      nvim-snap-${{ runner.os }}-
```

## ローカルの場合

スナップショットはコミットしたり外部保存せず、一時的なファイルとして利用します。
スナップショットは肥大化しやすいため、プラグインのリポジトリなどに含めると、インストール時間の長大などを招きます。
`nvim-snap init`では.gitignoreを生成していますが、同様にignoreしておくことを推奨します。

例:

```console
$ git switch base-branch
$ nvim-snap regression save
diff-example    ok
layout-splits   ok

$ git switch feature-branch
$ nvim-snap regression save
diff-example    ok
layout-splits   ok

$ nvim-snap regression test --base (git show --format='%H' --no-patch base-branch) --target (git show --format='%H' --no-patch feature-branch)
```
