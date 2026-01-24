package report

import (
	"path/filepath"
	"testing"

	"github.com/kyoh86/nvim-snap/internal/snapshots"
)

func writeSnapshot(t *testing.T, path string, text string) {
	t.Helper()
	s := snapshots.Snapshot{
		Size: map[string]int{"columns": 1, "lines": 1},
		Grids: []snapshots.Grid{{
			ID:   1,
			Rows: 1,
			Cols: 1,
			Cells: [][]snapshots.Cell{{
				{Text: text},
			}},
		}},
	}
	if err := snapshots.WriteJSON(path, s); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
}

func TestCompareCases(t *testing.T) {
	dir := t.TempDir()
	expected := filepath.Join(dir, "expected.json")
	actual := filepath.Join(dir, "actual.json")
	writeSnapshot(t, expected, "a")
	writeSnapshot(t, actual, "a")

	cases := []CompareCase{{
		Name:          "case",
		Title:         "Case",
		Kind:          "regression",
		ExpectedPath:  expected,
		ActualPath:    actual,
		ExpectedLabel: "base",
		ActualLabel:   "target",
		DiffDir:       filepath.Join(dir, "diff"),
	}}
	results, summary, failed, hasDiff := CompareCases(cases, map[string]bool{}, false, false)
	if failed || hasDiff {
		t.Fatalf("failed=%v hasDiff=%v", failed, hasDiff)
	}
	if summary.NoDiff != 1 || summary.Total != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	if results[0].Result != "no_diff" {
		t.Fatalf("result = %q", results[0].Result)
	}
}

func TestCompareCasesDiff(t *testing.T) {
	dir := t.TempDir()
	expected := filepath.Join(dir, "expected.json")
	actual := filepath.Join(dir, "actual.json")
	writeSnapshot(t, expected, "a")
	writeSnapshot(t, actual, "b")

	cases := []CompareCase{{
		Name:          "case",
		Title:         "Case",
		Kind:          "regression",
		ExpectedPath:  expected,
		ActualPath:    actual,
		ExpectedLabel: "base",
		ActualLabel:   "target",
		DiffDir:       filepath.Join(dir, "diff"),
	}}
	results, summary, failed, hasDiff := CompareCases(cases, map[string]bool{}, false, false)
	if failed || !hasDiff {
		t.Fatalf("failed=%v hasDiff=%v", failed, hasDiff)
	}
	if summary.Diff != 1 || summary.Total != 1 {
		t.Fatalf("summary = %+v", summary)
	}
	if results[0].Result != "diff" {
		t.Fatalf("result = %q", results[0].Result)
	}
}

func TestCompareCasesMissing(t *testing.T) {
	dir := t.TempDir()
	actual := filepath.Join(dir, "actual.json")
	writeSnapshot(t, actual, "a")

	cases := []CompareCase{{
		Name:          "case",
		Title:         "Case",
		Kind:          "regression",
		ExpectedPath:  filepath.Join(dir, "missing.json"),
		ActualPath:    actual,
		ExpectedLabel: "base",
		ActualLabel:   "target",
		DiffDir:       filepath.Join(dir, "diff"),
	}}
	_, summary, failed, hasDiff := CompareCases(cases, map[string]bool{}, false, false)
	if !failed || hasDiff {
		t.Fatalf("failed=%v hasDiff=%v", failed, hasDiff)
	}
	if summary.Error != 1 || summary.Total != 1 {
		t.Fatalf("summary = %+v", summary)
	}
}
