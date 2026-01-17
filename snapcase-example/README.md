# snapcase-example

This README describes the minimal structure and how to run the sample cases.

This is the minimal setup for snapshot generation using `nvim-snap`.

## Usage

```sh
nvim-snap core capture --scenario snapcase-example/cases/diff-example/scenario.lua \
  --out-dir snapcase-example/cases/diff-example/.out \
  --data-home snapcase-example/cases/diff-example/.nvim-data \
  --config-home snapcase-example/cases/diff-example/.nvim-config \
  --json --ansi --html
```

## Layout

- `snapcase.schema.json` JSON Schema for case definition and capture settings
- `cases/` sample cases (regression/golden mixed)
  - `diff-example/` regression case with diff-oriented scenarios

## Notes

`snapcase.json` is used by both the high-level commands and `core capture`.
