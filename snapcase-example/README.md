# snapcase-example

This is the minimal setup for snapshot generation using `nvim-snap`.

## Usage

```console
$ nvim-snap golden test --root snapcase-example --cases-dir snapcase --output diff --diff-format text
```

## Layout

- `snapcase/` sample cases
  - `regression/` regression cases
    - `diff-example/` regression case with diff-oriented scenarios
    - `layout-splits/` regression case for split layout
  - `golden/` golden cases
    - `floating-golden/` golden case for floating windows
    - `qlean-hidden/` golden case that installs `qlean.nvim` via `vim.pack.add()`
    - `qlean-quickfix/` golden case that installs `qlean.nvim` via `vim.pack.add()`
    - `quickfix-golden/` golden case for quickfix view
- `failure/` cases that intentionally produce diffs (for preview/testing)
  - `golden/` golden cases
    - `diff-golden/` golden case that always produces a diff

## Failure cases

Failure cases live under `failure/` and are meant to produce diffs.

```console
$ nvim-snap golden test --root snapcase-example --cases-dir failure --output diff --diff-format text
```

## Notes

`qlean-*` cases require `git`, network access, and Neovim with `vim.pack` (0.12+).
