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

- `case.schema.json` JSON Schema
- `cases/` sample cases (regression/golden mixed)
  - `diff-example/` regression case with diff-oriented scenarios

## Notes

`case.json` is for high-level case management, so `core capture` does not read it.
