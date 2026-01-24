package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/kyoh86/nvim-snap/internal/casefile"
	"github.com/kyoh86/nvim-snap/internal/collector"
	"github.com/kyoh86/nvim-snap/internal/htmldiff"
	"github.com/kyoh86/nvim-snap/internal/pngutil"
	"github.com/kyoh86/nvim-snap/internal/snapshots"
	"github.com/pmezard/go-difflib/difflib"
)

func splitCSV(value string) []string {
	out := []string{}
	for item := range strings.SplitSeq(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseFormats(value string, fallback map[string]bool) map[string]bool {
	if value == "" {
		return fallback
	}
	formats := map[string]bool{}
	for _, item := range splitCSV(value) {
		formats[item] = true
	}
	if len(formats) == 0 {
		return fallback
	}
	return formats
}

type waitOverrides struct {
	postWait    optionalInt
	waitDone    optionalBool
	doneTimeout optionalInt
}

func resolveWaits(c casefile.Case, overrides waitOverrides) (int, bool, int) {
	casePostWait := c.PostWait
	if overrides.postWait.set {
		casePostWait = overrides.postWait.value
	}
	caseWaitDone := c.WaitDone
	if overrides.waitDone.set {
		caseWaitDone = overrides.waitDone.value
	}
	caseDoneTimeout := c.DoneTimeout
	if overrides.doneTimeout.set {
		caseDoneTimeout = overrides.doneTimeout.value
	}
	return casePostWait, caseWaitDone, caseDoneTimeout
}

func resolveResultsRoot(absRoot, casesDir string) string {
	return filepath.Join(resolveCasesRoot(absRoot, casesDir), ".result")
}

func filterByKind(cases []casefile.Case, kind string) []casefile.Case {
	out := make([]casefile.Case, 0, len(cases))
	for _, c := range cases {
		if c.Kind == kind {
			out = append(out, c)
		}
	}
	return out
}

func collectSnapshot(c casefile.Case, scenario string, cfg runConfig, stage, dataHomeOverride, configHomeOverride string) (collector.Result, error) {
	if _, err := os.Stat(scenario); err != nil {
		return collector.Result{}, fmt.Errorf("scenario not found: %w", err)
	}
	dataHome := c.DataHome
	if dataHomeOverride != "" {
		dataHome = dataHomeOverride
	}
	configHome := c.ConfigHome
	if configHomeOverride != "" {
		configHome = configHomeOverride
	}
	casePostWait, caseWaitDone, caseDoneTimeout := resolveWaits(c, cfg.overrides)
	res, err := collector.Collect(collector.Options{
		Scenario:      scenario,
		Width:         c.Width,
		Height:        c.Height,
		WaitMS:        c.Wait,
		PostWaitMS:    casePostWait,
		WaitDone:      caseWaitDone,
		DoneTimeoutMS: caseDoneTimeout,
		RPCTimeoutMS:  c.RPCTimeout,
		DataHome:      dataHome,
		ConfigHome:    configHome,
		LogFile:       c.LogFile,
		LogLevel:      c.LogLevel,
		WorkDir:       cfg.absRoot,
		RTP:           c.RTP,
	})
	if err != nil {
		return collector.Result{}, err
	}
	if caseWaitDone && !res.WaitedDone {
		fmt.Fprintf(os.Stderr, "%s: wait_done timeout (possible input wait; prefer vim.api.nvim_cmd)\n", c.Name)
	}
	if len(res.Logs) > 0 {
		if err := writeScenarioLogs(cfg.resultsRoot, c.Name, stage, res.Logs); err != nil {
			return collector.Result{}, err
		}
	}
	return res, nil
}

func writeSnapshotOutputs(dir, base string, snapshot snapshots.Snapshot, formats map[string]bool) error {
	if formats["json"] {
		if err := writeJSON(filepath.Join(dir, base+".json"), snapshot); err != nil {
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

func writeScenarioLogs(resultsRoot, caseName, stage string, logs []string) error {
	if len(logs) == 0 {
		return nil
	}
	destDir := filepath.Join(resultsRoot, "logs")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(destDir, fmt.Sprintf("%s.%s.log", caseName, stage))
	return writeText(path, strings.Join(logs, "\n")+"\n")
}

type compareCase struct {
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

func compareCases(cases []compareCase, formats map[string]bool, diffAlways bool, wantDiffText bool) ([]map[string]any, map[string]int, bool, bool) {
	failed := false
	hasDiff := false
	results := []map[string]any{}
	summary := map[string]int{
		"total":   0,
		"no_diff": 0,
		"diff":    0,
		"error":   0,
	}
	for _, c := range cases {
		summary["total"]++
		entry := map[string]any{
			"name":          c.Name,
			"title":         c.Title,
			"kind":          c.Kind,
			"tags":          c.Tags,
			"expected_path": c.ExpectedPath,
			"actual_path":   c.ActualPath,
			"result":        "error",
			"diff_paths":    nil,
			"error_reason":  nil,
		}
		expected, err := readSnapshot(c.ExpectedPath)
		if err != nil {
			reason := err.Error()
			entry["error_reason"] = reason
			results = append(results, entry)
			fmt.Fprintf(os.Stderr, "%s: %s not found: %v\n", c.Name, c.ExpectedLabel, err)
			summary["error"]++
			failed = true
			continue
		}
		actual, err := readSnapshot(c.ActualPath)
		if err != nil {
			reason := err.Error()
			entry["error_reason"] = reason
			results = append(results, entry)
			fmt.Fprintf(os.Stderr, "%s: %s not found: %v\n", c.Name, c.ActualLabel, err)
			summary["error"]++
			failed = true
			continue
		}
		normExpected := snapshots.Normalize(expected)
		normActual := snapshots.Normalize(actual)
		equal := equalSnapshot(normExpected, normActual)
		if equal {
			entry["result"] = "no_diff"
			summary["no_diff"]++
		} else {
			entry["result"] = "diff"
			summary["diff"]++
			hasDiff = true
		}

		if wantDiffText && entry["result"] == "diff" {
			diff, err := unifiedDiffText(c.ExpectedLabel, c.ActualLabel, snapshots.RenderText(normExpected), snapshots.RenderText(normActual))
			if err != nil {
				entry["result"] = "error"
				entry["error_reason"] = err.Error()
				summary["error"]++
				failed = true
			} else {
				entry["diff_text"] = diff
			}
		}

		if entry["result"] == "diff" || diffAlways {
			diffPaths, err := writeDiffOutputs(c.DiffDir, c.ExpectedLabel, c.ActualLabel, normExpected, normActual, formats)
			if err != nil {
				entry["result"] = "error"
				entry["error_reason"] = err.Error()
				summary["error"]++
				failed = true
			} else {
				entry["diff_paths"] = diffPaths
			}
		}

		results = append(results, entry)
	}
	return results, summary, failed, hasDiff
}

func unifiedDiffText(fromLabel, toLabel, expected, actual string) (string, error) {
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expected),
		B:        difflib.SplitLines(actual),
		FromFile: fromLabel,
		ToFile:   toLabel,
		Context:  3,
	}
	return difflib.GetUnifiedDiffString(d)
}

func writeDiffOutputs(diffDir, expectedLabel, actualLabel string, expected, actual snapshots.Snapshot, formats map[string]bool) (map[string]string, error) {
	if err := os.MkdirAll(diffDir, 0o755); err != nil {
		return nil, err
	}
	paths := map[string]string{}
	if formats["text"] {
		diff, err := unifiedDiffText(expectedLabel, actualLabel, snapshots.RenderText(expected), snapshots.RenderText(actual))
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
		diff, err := unifiedDiffText(expectedLabel, actualLabel, snapshots.RenderANSI(expected), snapshots.RenderANSI(actual))
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

func printCompareText(results []map[string]any, diffHeader string, singleFormat bool) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(tw, "name\ttitle\tkind\ttags\tresult\t%s\terror_reason\n", diffHeader)
	for _, entry := range results {
		name, _ := entry["name"].(string)
		title, _ := entry["title"].(string)
		kind, _ := entry["kind"].(string)
		result, _ := entry["result"].(string)
		errorReason := ""
		if value, ok := entry["error_reason"]; ok && value != nil {
			errorReason, _ = value.(string)
		}
		tags := ""
		if list, ok := entry["tags"].([]string); ok {
			tags = strings.Join(list, ",")
		}
		diffPaths := ""
		if raw, ok := entry["diff_paths"].(map[string]string); ok {
			keys := make([]string, 0, len(raw))
			for key := range raw {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			items := make([]string, 0, len(keys))
			for _, key := range keys {
				if singleFormat {
					items = append(items, raw[key])
					continue
				}
				items = append(items, key+"="+raw[key])
			}
			diffPaths = strings.Join(items, ",")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", name, title, kind, tags, result, diffPaths, errorReason)
	}
	_ = tw.Flush()
}

func printCompareDiff(results []map[string]any) {
	for _, entry := range results {
		result, _ := entry["result"].(string)
		if result == "error" {
			name, _ := entry["name"].(string)
			title, _ := entry["title"].(string)
			kind, _ := entry["kind"].(string)
			tags := ""
			if list, ok := entry["tags"].([]string); ok {
				tags = strings.Join(list, ",")
			}
			errorReason := ""
			if value, ok := entry["error_reason"]; ok && value != nil {
				errorReason, _ = value.(string)
			}
			fmt.Fprintf(os.Stderr, "%s\t%s\t%s\t%s\n", name, title, kind, tags)
			if errorReason != "" {
				fmt.Fprintf(os.Stderr, "error: %s\n", errorReason)
			}
			continue
		}
		if result != "diff" {
			continue
		}
		name, _ := entry["name"].(string)
		title, _ := entry["title"].(string)
		kind, _ := entry["kind"].(string)
		tags := ""
		if list, ok := entry["tags"].([]string); ok {
			tags = strings.Join(list, ",")
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", name, title, kind, tags)
		diffText, _ := entry["diff_text"].(string)
		if diffText != "" {
			fmt.Println(diffText)
		}
	}
}

func writeJSON(path string, snapshot snapshots.Snapshot) error {
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func writeText(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func writeFile(path, contents string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file exists: %s", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}
