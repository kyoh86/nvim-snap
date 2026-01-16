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

Set `NVIM_SNAP_LUA` to point at `snap.lua` explicitly.
Set `NVIM` to choose which Neovim binary to run.

## Commands

- `list` list test cases
- `new` scaffold a test case
- `ci init` scaffold a CI workflow
- `run` run test cases (generate snapshots)
- `compare` compare test cases
- `update-expected` update expected snapshots

## Example

![diff overlay example](docs/diff-overlay.png)

## Typical Workflow

1. Generate snapshots
   `nvim --headless -u NONE -i NONE -l snap.lua run --root snapcase-example --format json`
2. Compare (CI)
   `nvim --headless -u NONE -i NONE -l snap.lua compare --root snapcase-example --format text`
3. Human-friendly diff (HTML)
   `nvim --headless -u NONE -i NONE -l snap.lua compare --root snapcase-example --format html`
4. Update expected
   `nvim --headless -u NONE -i NONE -l snap.lua update-expected --root snapcase-example`

## Sample Layout

- `snapcase-example/case.json` case definition for high-level usage
- `snapcase-example/case.schema.json` JSON Schema
- `snapcase-example/scenario.lua` scenario

## Notes

- If your scenario needs plugins, using `vim.pack.add()` is recommended.
- When using `vim.pack.add()`, set `--data-home` / `--config-home` explicitly.
- PNG output requires one of: `google-chrome`/`chromium`/`msedge`/`wkhtmltoimage`.
  - On Ubuntu, `apt install chromium` installs the snap build which may not work here; prefer the deb `google-chrome` or `wkhtmltoimage`.
