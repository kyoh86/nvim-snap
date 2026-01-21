# snapcase-example

This README describes the minimal structure and how to run the sample cases.

This is the minimal setup for snapshot generation using `nvim-snap`.

## Usage

```sh
nvim-snap golden test --root snapcase-example --cases-dir snapcase --output diff --diff-format text
```

## Layout

- `snapcase.schema.json` JSON Schema for case definition and capture settings
- `snapcase/` sample cases
  - `regression/` regression cases
    - `diff-example/` regression case with diff-oriented scenarios
  - `golden/` golden cases
    - `qlean-hidden/` golden case that installs `qlean.nvim` via `vim.pack.add()`
    - `qlean-quickfix/` golden case that installs `qlean.nvim` via `vim.pack.add()`

## Notes

`snapcase.json` is used by the CLI to configure capture settings per case.
`qlean-*` cases require `git`, network access, and Neovim with `vim.pack` (0.12+).
