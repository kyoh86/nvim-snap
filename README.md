# nvim-snap

nvim-snap is a tool for running tests based on Neovim UI snapshots.

## Concept

Run a scenario to generate the current UI snapshot, then compare it with stored expected snapshots to verify display correctness.
Test cases manage scenarios and expected snapshots, and can be executed in batch or filtered by tags.

## Usage

`snapcase-example` is included, so you can run the cases as-is to confirm behavior.

## Installation

Use the simple installer:

```sh
git clone https://github.com/kyoh86/nvim-snap.git
cd nvim-snap
./install.sh
```

Set `PREFIX` to change the install location (default: `~/.local`).
Ensure `~/.local/bin` is on your PATH.

This installs the single-file `nvim-snap` binary into `~/.local/bin`.
If `dist/nvim-snap` is missing, it runs `scripts/bundle.sh`.

### Requirements

- Neovim (`nvim`)
- `luabundler` (Node.js, dev-only) for single-file bundle generation
- `google-chrome`/`chromium`/`msedge`/`wkhtmltoimage` for PNG output

### Single-file bundle

Use `luabundler` to generate `dist/nvim-snap`.

```sh
scripts/bundle.sh
```

The generated `dist/nvim-snap` can be distributed as a single file.
It is also attached to releases, so you can download it directly from the release page.

### Using mise

```sh
mise use 'github:kyoh86/nvim-snap[asset_pattern=nvim-snap]'
```

## Example

![diff overlay example](docs/diff-overlay.png)

## Commands

- `list` list test cases
- `new` scaffold a test case
- `ci init` scaffold a CI workflow
- `run` run test cases (generate snapshots)
- `compare` compare test cases
- `update-expected` update expected snapshots

### list

List test cases.

```sh
nvim-snap list --root .
nvim-snap list --tag ui --tag regression
nvim-snap list --case basic-regression
nvim-snap list --json
```

### new

Create a case scaffold.

```sh
nvim-snap new --root tests/cases --id basic-regression --kind regression
nvim-snap new --dir tests/cases/sample --kind golden --name "Sample Golden"
```

### run

Run cases and write snapshots under `actual/`.

```sh
nvim-snap run --root tests/cases --format json,html
nvim-snap run --tag ui
```

### compare

Compare `expected/` and `actual/`, writing diffs under `diff/` when needed.

```sh
nvim-snap compare --root tests/cases --format text
nvim-snap compare --root tests/cases --format html --diff-always
nvim-snap compare --root tests/cases --format png --diff-always
```

### update-expected

Update expected snapshots. Regression accepts `actual`, golden runs `golden.lua`.

```sh
nvim-snap update-expected --root tests/cases
nvim-snap update-expected --root tests/cases --dry-run
nvim-snap update-expected --root tests/cases --no-confirm
```

### ci init

Scaffold a CI workflow.

```sh
nvim-snap ci init --path .github/workflows/nvim-snap.yml
```

## Typical Workflow

1. Generate snapshots
   `nvim-snap run --root snapcase-example --format json`
2. Compare (CI)
   `nvim-snap compare --root snapcase-example --format text`
3. Human-friendly diff (HTML)
   `nvim-snap compare --root snapcase-example --format html`
4. Update expected
   `nvim-snap update-expected --root snapcase-example`

## Sample Layout

- `snapcase-example/case.json` case definition for high-level usage
- `snapcase-example/case.schema.json` JSON Schema
- `snapcase-example/scenario.lua` scenario

## Notes

- If your scenario needs plugins, using `vim.pack.add()` is recommended.
- When using `vim.pack.add()`, set `--data-home` / `--config-home` explicitly.
