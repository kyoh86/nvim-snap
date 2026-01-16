# snapcase-example

This README describes the minimal structure and how to run the sample cases.

This is the minimal setup for snapshot generation using `snap.lua`.

## Usage

```sh
nvim --headless -u NONE -i NONE -l snap.lua \
  core capture --scenario snapcase-example/scenario.lua \
  --out-dir snapcase-example/.out \
  --data-home snapcase-example/.nvim-data \
  --config-home snapcase-example/.nvim-config \
  --json --ansi --html
```

## Layout

- `case.json` case definition for high-level usage
- `case.schema.json` JSON Schema
- `scenario.lua` scenario
- `scenario_baseline.lua` baseline sample (lines and content)
- `scenario_alt.lua` sample with added/changed/removed lines
- `cases/` multiple sample cases (regression/golden mixed)

## Notes

`case.json` is for high-level case management, so `core capture` does not read it.
