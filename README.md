# nvim-snap

nvim-snap is a tool for running tests based on Neovim UI snapshots.

## Concept

Run Neovim scenarios, capture UI snapshots, and compare results.
Regression compares snapshots saved per commit, while golden compares the golden and target scenarios executed in the same run.

## Example

![diff overlay example](docs/diff-overlay.png)

This repository includes `snapcase-example`, so you can run the cases as-is to confirm behavior.
By default, `--root` is the current directory and `--cases-dir` is `snapcase/`.
Cases live under `<root>/<cases-dir>/regression/<case-name>/snapcase.json` or
`<root>/<cases-dir>/golden/<case-name>/snapcase.json`.
Outputs are stored under `<root>/<cases-dir>/.result/`.

## Installation

### Requirements

- Neovim (`nvim`)
- `google-chrome`/`chromium`/`msedge`/`wkhtmltoimage` for PNG output

### Download from releases

Download `nvim-snap` from the release page and put it on your PATH.

### Build from source

```sh
go build -o nvim-snap ./cmd/nvim-snap
```

### Using mise

```sh
mise use github:kyoh86/nvim-snap
```

## Commands

Cases live under `snapcase/regression/<case-name>/snapcase.json` or `snapcase/golden/<case-name>/snapcase.json`.
Case names come from the directory name, and `snapcase.json` holds the case metadata and capture settings.
`snapcase.json` accepts `rtp` (string or list) to prepend runtimepath entries.
Use `${CASE}` or `${ROOT}` placeholders to target the case directory or the `--root` path.

- `list` list test cases
- `init` scaffold a CI workflow
- `capture` capture a snapshot from a scenario
- `normalize` normalize a snapshot JSON
- `compare` compare two snapshot JSON files
- `regression new` scaffold a regression case
- `regression save` save regression snapshots by id
- `regression test` compare regression snapshots by id
- `golden new` scaffold a golden case
- `golden test` run golden and target scenarios and compare

### list

List test cases.

```sh
nvim-snap list
nvim-snap list --tag ui --tag regression
nvim-snap list --case basic-regression
nvim-snap list --json
```

### capture

Capture a snapshot from a scenario.

```sh
nvim-snap capture --scenario scenario.lua --out ./out --format json,ansi,html
```

### normalize

Normalize a snapshot JSON.

```sh
nvim-snap normalize --in snapshot.json --out normalized.json
nvim-snap normalize --in snapshot.json
```

### compare

Compare two snapshot JSON files.

```sh
nvim-snap compare --expected expected.json --actual actual.json --format text
nvim-snap compare --expected expected.json --actual actual.json --format html --out diff.html
```

### regression new

Create a regression case scaffold.

```sh
nvim-snap regression new --name basic-regression
```

When `--name` is omitted, `nvim-snap` generates a random case name under
`<root>/<cases-dir>/regression/`.

### regression save

Save snapshots for the current commit (default) into `.result/`.

```sh
nvim-snap regression save
nvim-snap regression save --id abcdef1234
nvim-snap regression save --tag ui
```

### regression test

Compare saved snapshots by id and output diffs.
`--target` defaults to the current commit id.

```sh
nvim-snap regression test --base abcdef1234 --target 0123456789
nvim-snap regression test --base abcdef1234 --output diff --diff-format text
```

### golden new

Create a golden case scaffold.

```sh
nvim-snap golden new --name sample-golden --title "Sample Golden"
```

### golden test

Run golden and target scenarios, then compare results.

```sh
nvim-snap golden test --output summary --diff-format html
nvim-snap golden test --output diff --diff-format text --diff-always
```

### init

Scaffold a CI workflow.

```sh
nvim-snap init --path .github/workflows/nvim-snap.yml
```

## Practical Flow

### Regression flow

1. Create a regression case
   `nvim-snap regression new --name my-case`
2. Write the scenario
   `scenario.lua`
3. Save snapshots at the base commit
   `nvim-snap regression save`
4. Save snapshots at the target commit
   `nvim-snap regression save`
5. Compare
   `nvim-snap regression test --base <base-id> --target <target-id>`

### Golden flow

1. Create a golden case
   `nvim-snap golden new --name my-case`
2. Write scenarios
   `golden.lua` / `target.lua`
3. Run and compare
   `nvim-snap golden test --output summary --diff-format html`

## Test Types

- Regression: compare snapshots saved per commit from the same scenario
- Golden: compare golden and target scenario outputs in the same run

## Notes

- Outputs are stored under `<root>/<cases-dir>/.result/`.
- `regression save` defaults to the current git commit id and fails on dirty trees.
- If your scenario needs plugins, using `vim.pack.add()` is recommended.
- When using `vim.pack.add()`, set `data_home` / `config_home` in `snapcase.json`.
- Golden runs isolate `data_home` / `config_home` per scenario under the configured paths.
- In headless runs, commands that may prompt for input can block. Prefer `vim.api.nvim_cmd` to `vim.cmd`.
- When using `wait_done`, call `require("nvim_snap").done()` in your scenario.
