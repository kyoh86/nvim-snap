# nvim-snap

nvim-snap is a tool for running tests based on Neovim UI snapshots.

## Concept

Run a scenario to generate the current UI snapshot, then compare it with stored baselines to verify display correctness.
Test cases manage scenarios and baselines, and can be executed in batch or filtered by tags.

## Example

![diff overlay example](docs/diff-overlay.png)

This repository includes `snapcase-example`, so you can run the cases as-is to confirm behavior.
By default, `--root` is the current directory and `--cases-dir` is `snapcase/`.
Cases live under `<root>/<cases-dir>/<case-name>/snapcase.json`.

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

The examples below assume cases are under `snapcase/<case-name>/snapcase.json` with the default options.
Case names come from the directory name, and `snapcase.json` holds the case metadata and capture settings.
`snapcase.json` accepts `rtp` (string or list) to prepend runtimepath entries.
Use `${CASE}` or `${ROOT}` placeholders to target the case directory or the `--root` path.

- `list` list test cases
- `new` scaffold a test case
- `init` scaffold a CI workflow
- `run` run test cases (generate snapshots)
- `compare` compare test cases
- `accept` accept regression snapshots
- `golden` generate golden snapshots

### list

List test cases.

```sh
nvim-snap list
nvim-snap list --tag ui --tag regression
nvim-snap list --case basic-regression
nvim-snap list --json
```

### new

Create a case scaffold.

```sh
nvim-snap new --name basic-regression --kind regression
nvim-snap new --name sample --kind golden --title "Sample Golden"
```

When `--name` is omitted, `nvim-snap` generates a random case name under `<root>/<cases-dir>/`.


### run

Run cases and write snapshots under `current/` (regression) or `actual/` (golden).

```sh
nvim-snap run --format json,html
nvim-snap run --tag ui
```

### compare

Compare `accepted/` and `current/` (regression) or `baseline/` and `actual/` (golden), writing diffs under `diff/` when needed.

```sh
nvim-snap compare --format text
nvim-snap compare --format html --diff-always
nvim-snap compare --format png --diff-always
```

### accept

Accept regression snapshots from `current/`.

```sh
nvim-snap accept
nvim-snap accept --dry-run
nvim-snap accept --no-confirm
```

### golden

Generate golden baselines by running `golden.lua`.

```sh
nvim-snap golden
nvim-snap golden --dry-run
```

### init

Scaffold a CI workflow.

```sh
nvim-snap init --path .github/workflows/nvim-snap.yml
```

## Practical Flow

### Regression flow

1. Create a regression case
   `nvim-snap new --name my-case --kind regression`
2. Write the scenario
   `scenario.lua`
3. Run current
   `nvim-snap run`
4. Review diffs
   `nvim-snap compare --format html`
5. Decide
   - Fix the scenario/plugin and re-run
   - Or accept the new output: `nvim-snap accept`

### Golden flow

1. Create a golden case
   `nvim-snap new --name my-case --kind golden`
2. Write scenarios
   `golden.lua` / `target.lua`
3. Generate baseline
   `nvim-snap golden`
4. Run actual
   `nvim-snap run`
5. Review diffs
   `nvim-snap compare --format html`

## Test Types

- Regression: verify the same scenario matches the stored accepted snapshot
- Golden: compare the baseline (golden) output with the implementation result (target)

## Notes

- If your scenario needs plugins, using `vim.pack.add()` is recommended.
- When using `vim.pack.add()`, set `--data-home` / `--config-home` explicitly.
