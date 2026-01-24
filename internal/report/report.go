// Package report formats and writes compare outputs.
package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kyoh86/nvim-snap/internal/htmldiff"
	"github.com/kyoh86/nvim-snap/internal/pngutil"
	"github.com/kyoh86/nvim-snap/internal/snapshots"
	"github.com/pmezard/go-difflib/difflib"
)

type CompareCase struct {
	Name          string
	Title         string
	Kind          string
	Tags          []string
	ExpectedPath  string
	ActualPath    string
	ExpectedLabel string
	ActualLabel   string
	DiffDir       string
}

type CaseResult struct {
	Name         string            `json:"name"`
	Title        string            `json:"title"`
	Kind         string            `json:"kind"`
	Tags         []string          `json:"tags"`
	ExpectedPath string            `json:"expected_path"`
	ActualPath   string            `json:"actual_path"`
	Result       string            `json:"result"`
	DiffPaths    map[string]string `json:"diff_paths,omitempty"`
	ErrorReason  string            `json:"error_reason,omitempty"`
	DiffText     string            `json:"diff_text,omitempty"`
}

type Summary struct {
	Total  int `json:"total"`
	NoDiff int `json:"no_diff"`
	Diff   int `json:"diff"`
	Error  int `json:"error"`
}

func CompareCases(cases []CompareCase, formats map[string]bool, diffAlways bool, wantDiffText bool) ([]CaseResult, Summary, bool, bool) {
	failed := false
	hasDiff := false
	results := []CaseResult{}
	summary := Summary{}
	for _, c := range cases {
		summary.Total++
		entry := CaseResult{
			Name:         c.Name,
			Title:        c.Title,
			Kind:         c.Kind,
			Tags:         c.Tags,
			ExpectedPath: c.ExpectedPath,
			ActualPath:   c.ActualPath,
			Result:       "error",
		}
		expected, err := snapshots.ReadJSON(c.ExpectedPath)
		if err != nil {
			entry.ErrorReason = err.Error()
			results = append(results, entry)
			fmt.Fprintf(os.Stderr, "%s: %s not found: %v\n", c.Name, c.ExpectedLabel, err)
			summary.Error++
			failed = true
			continue
		}
		actual, err := snapshots.ReadJSON(c.ActualPath)
		if err != nil {
			entry.ErrorReason = err.Error()
			results = append(results, entry)
			fmt.Fprintf(os.Stderr, "%s: %s not found: %v\n", c.Name, c.ActualLabel, err)
			summary.Error++
			failed = true
			continue
		}
		normExpected := snapshots.Normalize(expected)
		normActual := snapshots.Normalize(actual)
		equal := snapshots.Equal(normExpected, normActual)
		if equal {
			entry.Result = "no_diff"
			summary.NoDiff++
		} else {
			entry.Result = "diff"
			summary.Diff++
			hasDiff = true
		}

		if wantDiffText && entry.Result == "diff" {
			diff, err := UnifiedDiffText(c.ExpectedLabel, c.ActualLabel, snapshots.RenderText(normExpected), snapshots.RenderText(normActual), 3)
			if err != nil {
				entry.Result = "error"
				entry.ErrorReason = err.Error()
				summary.Error++
				failed = true
			} else {
				entry.DiffText = diff
			}
		}

		if entry.Result == "diff" || diffAlways {
			diffPaths, err := WriteDiffOutputs(c.DiffDir, c.ExpectedLabel, c.ActualLabel, normExpected, normActual, formats)
			if err != nil {
				entry.Result = "error"
				entry.ErrorReason = err.Error()
				summary.Error++
				failed = true
			} else {
				entry.DiffPaths = diffPaths
			}
		}

		results = append(results, entry)
	}
	return results, summary, failed, hasDiff
}

func UnifiedDiffText(fromLabel, toLabel, expected, actual string, context int) (string, error) {
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expected),
		B:        difflib.SplitLines(actual),
		FromFile: fromLabel,
		ToFile:   toLabel,
		Context:  context,
	}
	return difflib.GetUnifiedDiffString(d)
}

func WriteDiffOutputs(diffDir, expectedLabel, actualLabel string, expected, actual snapshots.Snapshot, formats map[string]bool) (map[string]string, error) {
	if err := os.MkdirAll(diffDir, 0o755); err != nil {
		return nil, err
	}
	paths := map[string]string{}
	if formats["text"] {
		diff, err := UnifiedDiffText(expectedLabel, actualLabel, snapshots.RenderText(expected), snapshots.RenderText(actual), 3)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(diffDir, "diff.txt")
		if err := writeText(path, diff); err != nil {
			return nil, err
		}
		paths["text"] = path
	}
	if formats["ansi"] {
		diff, err := UnifiedDiffText(expectedLabel, actualLabel, snapshots.RenderANSI(expected), snapshots.RenderANSI(actual), 3)
		if err != nil {
			return nil, err
		}
		path := filepath.Join(diffDir, "diff.ansi")
		if err := writeText(path, diff); err != nil {
			return nil, err
		}
		paths["ansi"] = path
	}
	if formats["html"] {
		html := htmldiff.RenderHTML(expected, actual, "unified", expectedLabel, actualLabel)
		path := filepath.Join(diffDir, "diff.html")
		if err := writeText(path, html); err != nil {
			return nil, err
		}
		paths["html"] = path
	}
	if formats["png"] {
		html := htmldiff.RenderHTML(expected, actual, "overlay", expectedLabel, actualLabel)
		path := filepath.Join(diffDir, "diff.png")
		if err := pngutil.WritePNGFromHTML(html, path); err != nil {
			return nil, err
		}
		paths["png"] = path
	}
	return paths, nil
}

func WriteSnapshotOutputs(dir, base string, snapshot snapshots.Snapshot, formats map[string]bool) error {
	if formats["json"] {
		if err := snapshots.WriteJSON(filepath.Join(dir, base+".json"), snapshot); err != nil {
			return err
		}
	}
	if formats["ansi"] {
		ansi := snapshots.RenderANSI(snapshot)
		if err := writeText(filepath.Join(dir, base+".ansi"), ansi); err != nil {
			return err
		}
	}
	if formats["html"] {
		html := snapshots.RenderHTML(snapshot)
		if err := writeText(filepath.Join(dir, base+".html"), html); err != nil {
			return err
		}
	}
	return nil
}

func PrintCompareText(results []CaseResult, diffHeader string, singleFormat bool, out io.Writer) {
	tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "name\ttitle\tkind\ttags\tresult\t%s\terror_reason\n", diffHeader)
	for _, entry := range results {
		tags := strings.Join(entry.Tags, ",")
		diffPaths := ""
		if len(entry.DiffPaths) > 0 {
			keys := make([]string, 0, len(entry.DiffPaths))
			for key := range entry.DiffPaths {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			items := make([]string, 0, len(keys))
			for _, key := range keys {
				if singleFormat {
					items = append(items, entry.DiffPaths[key])
					continue
				}
				items = append(items, key+"="+entry.DiffPaths[key])
			}
			diffPaths = strings.Join(items, ",")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", entry.Name, entry.Title, entry.Kind, tags, entry.Result, diffPaths, entry.ErrorReason)
	}
	_ = tw.Flush()
}

func PrintCompareDiff(results []CaseResult, out io.Writer, errOut io.Writer) {
	for _, entry := range results {
		if entry.Result == "error" {
			tags := strings.Join(entry.Tags, ",")
			fmt.Fprintf(errOut, "%s\t%s\t%s\t%s\n", entry.Name, entry.Title, entry.Kind, tags)
			if entry.ErrorReason != "" {
				fmt.Fprintf(errOut, "error: %s\n", entry.ErrorReason)
			}
			continue
		}
		if entry.Result != "diff" {
			continue
		}
		tags := strings.Join(entry.Tags, ",")
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", entry.Name, entry.Title, entry.Kind, tags)
		if entry.DiffText != "" {
			fmt.Fprintln(out, entry.DiffText)
		}
	}
}

func writeText(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}
