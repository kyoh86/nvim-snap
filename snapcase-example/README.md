# snapcase-example

This README describes the minimal structure and how to run the sample cases.

This is the minimal setup for snapshot generation using `nvim-snap`.

## Usage

```sh
nvim-snap run --root snapcase-example --cases-dir cases --case diff-example --format json,ansi,html
```

## Layout

- `snapcase.schema.json` JSON Schema for case definition and capture settings
- `cases/` sample cases (regression/golden mixed)
  - `diff-example/` regression case with diff-oriented scenarios

## Notes

`snapcase.json` is used by the CLI to configure capture settings per case.
