# nvim-snap

NeovimのUI描画スナップショットを取得するためのPoCです。

## コンセプト

- NeovimのUI描画をスナップショット化してJSON/ANSI/HTMLで出力する
- caseというディレクトリの単位でスナップショット取得までのシナリオ、設定、出力を集約する
- 低レイヤー（スナップショット生成、比較・正規化）と高レイヤーのコマンドを分離する

## 使い方

snapcase-exampleというcaseを同梱しているので、そのまま呼び出して実際の動作を確認できます。

```sh
nvim --headless -u NONE -i NONE -l snap.lua capture --case snapcase-example
```

生成物は `snapcase-example/.out/` に出力されます。

## コマンド

- `capture` スナップショット生成
- `normalize` スナップショットJSONの正規化
- `compare` スナップショットJSONの比較

## 典型ワークフロー

1. ケースを実行してスナップショットを生成  
   `nvim --headless -u NONE -i NONE -l snap.lua capture --case snapcase-example`
2. 期待値を作成  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --case snapcase-example --expected snapcase-example/.expected/snapshot.json --update --pretty`
3. 比較（CI向け）  
   `nvim --headless -u NONE -i NONE -l snap.lua compare --case snapcase-example --expected snapcase-example/.expected/snapshot.json --diff`

## ケース構成

- `snapcase-example/snapcase.json` ケース定義
- `snapcase-example/snapcase.schema.json` JSON Schema
- `snapcase-example/scenario.lua` 操作シナリオ

## 注意点

- シナリオの中でプラグインが必要な場合は、 `vim.pack.add()` を使うのがおすすめです。
- `vim.pack.add()` を使う場合は `--case` で `data_home` / `config_home` をcaseディレクトリ内に設定してください。
