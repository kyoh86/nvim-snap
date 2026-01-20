package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/kyoh86/nvim-snap/internal/casefile"
	"github.com/kyoh86/nvim-snap/internal/collector"
	"github.com/kyoh86/nvim-snap/internal/htmldiff"
	"github.com/kyoh86/nvim-snap/internal/pngutil"
	"github.com/kyoh86/nvim-snap/internal/snapshots"
	"github.com/pmezard/go-difflib/difflib"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	if value == "" {
		return nil
	}
	for item := range strings.SplitSeq(value, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			*s = append(*s, trimmed)
		}
	}
	return nil
}

type optionalInt struct {
	value int
	set   bool
}

func (o *optionalInt) String() string {
	if !o.set {
		return ""
	}
	return strconv.Itoa(o.value)
}

func (o *optionalInt) Set(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

type optionalBool struct {
	value bool
	set   bool
}

func (o *optionalBool) String() string {
	if !o.set {
		return ""
	}
	if o.value {
		return "true"
	}
	return "false"
}

func (o *optionalBool) Set(value string) error {
	if value == "" {
		o.value = true
		o.set = true
		return nil
	}
	v, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	o.value = v
	o.set = true
	return nil
}

func (o *optionalBool) IsBoolFlag() bool {
	return true
}

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

func displayWidth(value string) int {
	return utf8.RuneCountInString(value)
}

func formatPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if absCwd, err := filepath.Abs(cwd); err == nil && absCwd == path {
		return "."
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil || rel == "" || strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "list":
		cmdList(os.Args[2:])
	case "run":
		cmdRun(os.Args[2:])
	case "compare":
		cmdCompare(os.Args[2:])
	case "accept":
		cmdAccept(os.Args[2:])
	case "golden":
		cmdGolden(os.Args[2:])
	case "report":
		cmdReport(os.Args[2:])
	case "new":
		cmdNew(os.Args[2:])
	case "init":
		cmdInit(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Println("usage:")
	fmt.Println("  nvim-snap <command> [options]")
	fmt.Println("")
	fmt.Println("commands:")
	fmt.Println("  list    list test cases")
	fmt.Println("  run     run test cases (generate snapshots)")
	fmt.Println("  compare compare snapshots")
	fmt.Println("  accept  accept regression snapshots")
	fmt.Println("  golden  generate golden snapshots")
	fmt.Println("  report run golden, run, compare, collect diffs")
	fmt.Println("  new     scaffold a test case")
	fmt.Println("  init    scaffold CI workflow")
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	jsonOut := fs.Bool("json", false, "Output JSON")
	var tags stringList
	var names stringList
	fs.Var(&tags, "tag", "Tag filter")
	fs.Var(&names, "case", "Case name filter")
	_ = fs.Parse(args)

	absRoot := mustAbs(*root)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := casefile.Filter(cases, tags, names)

	if *jsonOut {
		out := map[string]any{
			"root":  absRoot,
			"cases": []map[string]any{},
		}
		for _, c := range filtered {
			out["cases"] = append(out["cases"].([]map[string]any), map[string]any{
				"name":  c.Name,
				"title": c.Title,
				"kind":  c.Kind,
				"tags":  c.Tags,
				"path":  c.Path,
			})
		}
		payload, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(payload))
	} else {
		rows := [][]string{{"name", "title", "kind", "tags", "path"}}
		for _, c := range filtered {
			rows = append(rows, []string{
				c.Name,
				c.Title,
				c.Kind,
				strings.Join(c.Tags, ","),
				formatPath(c.Path),
			})
		}
		widths := make([]int, len(rows[0]))
		for _, row := range rows {
			for i, value := range row {
				if w := displayWidth(value); w > widths[i] {
					widths[i] = w
				}
			}
		}
		for _, row := range rows {
			parts := make([]string, 0, len(row))
			for i, value := range row {
				padding := max(0, widths[i]-displayWidth(value))
				parts = append(parts, value+strings.Repeat(" ", padding))
			}
			fmt.Println(strings.Join(parts, "  "))
		}
	}
	if len(errs) > 0 {
		os.Exit(2)
	}
}

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	format := fs.String("format", "json", "Output formats (json,ansi,html)")
	var postWait optionalInt
	var waitDone optionalBool
	var doneTimeout optionalInt
	fs.Var(&postWait, "post-wait", "Wait after scenario execution (ms)")
	fs.Var(&waitDone, "wait-done", "Wait for scenario completion notification")
	fs.Var(&doneTimeout, "done-timeout", "Scenario completion timeout (ms)")
	var tags stringList
	var names stringList
	fs.Var(&tags, "tag", "Tag filter")
	fs.Var(&names, "case", "Case name filter")
	_ = fs.Parse(args)

	formats := parseFormats(*format, map[string]bool{"json": true})
	for key := range formats {
		if key != "json" && key != "ansi" && key != "html" {
			fmt.Fprintf(os.Stderr, "unsupported format: %s\n", key)
			os.Exit(2)
		}
	}

	absRoot := mustAbs(*root)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := casefile.Filter(cases, tags, names)

	failed := len(errs) > 0
	if runFailed, _ := runCases(filtered, runConfig{
		absRoot: absRoot,
		formats: formats,
		overrides: waitOverrides{
			postWait:    postWait,
			waitDone:    waitDone,
			doneTimeout: doneTimeout,
		},
	}); runFailed {
		failed = true
	}
	if failed {
		os.Exit(1)
	}
}

type runConfig struct {
	absRoot   string
	formats   map[string]bool
	overrides waitOverrides
}

func runCases(cases []casefile.Case, cfg runConfig) (bool, map[string][]string) {
	failed := false
	logs := map[string][]string{}
	for _, c := range cases {
		scenario := c.Scenario
		if c.Kind == "golden" {
			scenario = c.Target
		}
		if _, err := os.Stat(scenario); err != nil {
			fmt.Fprintf(os.Stderr, "%s: scenario not found: %v\n", c.Name, err)
			failed = true
			continue
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
			DataHome:      c.DataHome,
			ConfigHome:    c.ConfigHome,
			LogFile:       c.LogFile,
			LogLevel:      c.LogLevel,
			WorkDir:       cfg.absRoot,
			RTP:           c.RTP,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}
		if caseWaitDone && !res.WaitedDone {
			fmt.Fprintf(os.Stderr, "%s: wait_done timeout (possible input wait; prefer vim.api.nvim_cmd)\n", c.Name)
		}
		if len(res.Logs) > 0 {
			logs[c.Name] = append(logs[c.Name], res.Logs...)
			if err := writeScenarioLogs(cfg.absRoot, c.Name, "run", res.Logs); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
				failed = true
				continue
			}
		}
		actualDir := filepath.Dir(c.Actual)
		if cfg.formats["json"] {
			if err := writeJSON(c.Actual, res.Snapshot); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
				failed = true
				continue
			}
		}
		if cfg.formats["ansi"] {
			ansi := snapshots.RenderANSI(res.Snapshot)
			if err := writeText(filepath.Join(actualDir, "snapshot.ansi"), ansi); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
				failed = true
				continue
			}
		}
		if cfg.formats["html"] {
			html := snapshots.RenderHTML(res.Snapshot)
			if err := writeText(filepath.Join(actualDir, "snapshot.html"), html); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
				failed = true
				continue
			}
		}
		fmt.Printf("%s\tok\n", c.Name)
	}
	return failed, logs
}

func cmdCompare(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	diffFormat := fs.String("diff-format", "text", "Diff formats (text,ansi,html,png)")
	diffAlways := fs.Bool("diff-always", false, "Write diffs even if no difference")
	output := fs.String("output", "summary", "Output format (summary,diff,json)")
	var tags stringList
	var names stringList
	fs.Var(&tags, "tag", "Tag filter")
	fs.Var(&names, "case", "Case name filter")
	_ = fs.Parse(args)

	formats := parseFormats(*diffFormat, map[string]bool{"text": true})
	for key := range formats {
		if key != "text" && key != "ansi" && key != "html" && key != "png" {
			fmt.Fprintf(os.Stderr, "unsupported format: %s\n", key)
			os.Exit(2)
		}
	}
	if *output != "summary" && *output != "diff" && *output != "json" {
		fmt.Fprintf(os.Stderr, "unsupported output: %s\n", *output)
		os.Exit(2)
	}

	absRoot := mustAbs(*root)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := casefile.Filter(cases, tags, names)

	results, summary, failed, hasDiff := compareCases(filtered, formats, *diffAlways, *output == "diff")
	if len(errs) > 0 {
		failed = true
	}

	if *output == "json" {
		out := map[string]any{
			"root":    absRoot,
			"summary": summary,
			"cases":   results,
		}
		payload, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Println(string(payload))
	} else if *output == "diff" {
		printCompareDiff(results)
	} else {
		diffHeader := "diff_paths"
		if len(formats) == 1 {
			for key := range formats {
				diffHeader = "diff_path(" + key + ")"
			}
		}
		printCompareText(results, diffHeader, len(formats) == 1)
	}

	if failed {
		os.Exit(2)
	}
	if hasDiff {
		os.Exit(1)
	}
}

func compareCases(cases []casefile.Case, formats map[string]bool, diffAlways bool, wantDiffText bool) ([]map[string]any, map[string]int, bool, bool) {
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
			"name":         c.Name,
			"title":        c.Title,
			"kind":         c.Kind,
			"tags":         c.Tags,
			"result":       "error",
			"diff_paths":   nil,
			"error_reason": nil,
		}
		expected, err := readSnapshot(c.Expected)
		if err != nil {
			reason := err.Error()
			entry["error_reason"] = reason
			results = append(results, entry)
			fmt.Fprintf(os.Stderr, "%s: %s not found: %v\n", c.Name, c.ExpectedLabel, err)
			summary["error"]++
			failed = true
			continue
		}
		actual, err := readSnapshot(c.Actual)
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
			diffPaths, err := writeDiffOutputs(c, normExpected, normActual, formats)
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

func collectDiffArtifacts(cases []casefile.Case, destDir string) error {
	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	for _, c := range cases {
		entries, err := os.ReadDir(c.DiffDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			src := filepath.Join(c.DiffDir, entry.Name())
			ext := filepath.Ext(entry.Name())
			if ext == "" {
				ext = ".diff"
			}
			dest := filepath.Join(destDir, c.Name+ext)
			if err := copyFile(src, dest); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeScenarioLogs(root, caseName, stage string, logs []string) error {
	if len(logs) == 0 {
		return nil
	}
	destDir := filepath.Join(root, ".nvim-snap-log")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(destDir, fmt.Sprintf("%s.%s.log", caseName, stage))
	return writeText(path, strings.Join(logs, "\n")+"\n")
}

func cmdReport(args []string) {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	snapshotFormat := fs.String("snapshot-format", "json", "Snapshot formats (json,ansi,html)")
	diffFormat := fs.String("diff-format", "text", "Diff formats (text,ansi,html,png)")
	diffAlways := fs.Bool("diff-always", false, "Write diffs even if no difference")
	output := fs.String("output", "summary", "Output format (summary,diff,json)")
	var postWait optionalInt
	var waitDone optionalBool
	var doneTimeout optionalInt
	fs.Var(&postWait, "post-wait", "Wait after scenario execution (ms)")
	fs.Var(&waitDone, "wait-done", "Wait for scenario completion notification")
	fs.Var(&doneTimeout, "done-timeout", "Scenario completion timeout (ms)")
	var tags stringList
	var names stringList
	fs.Var(&tags, "tag", "Tag filter")
	fs.Var(&names, "case", "Case name filter")
	_ = fs.Parse(args)

	snapFormats := parseFormats(*snapshotFormat, map[string]bool{"json": true})
	for key := range snapFormats {
		if key != "json" && key != "ansi" && key != "html" {
			fmt.Fprintf(os.Stderr, "unsupported snapshot format: %s\n", key)
			os.Exit(2)
		}
	}
	diffFormats := parseFormats(*diffFormat, map[string]bool{"text": true})
	for key := range diffFormats {
		if key != "text" && key != "ansi" && key != "html" && key != "png" {
			fmt.Fprintf(os.Stderr, "unsupported diff format: %s\n", key)
			os.Exit(2)
		}
	}
	if *output != "summary" && *output != "diff" && *output != "json" {
		fmt.Fprintf(os.Stderr, "unsupported output: %s\n", *output)
		os.Exit(2)
	}

	absRoot := mustAbs(*root)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := casefile.Filter(cases, tags, names)

	failed := len(errs) > 0
	goldenFailed, goldenLogs := goldenCases(filtered, goldenConfig{
		absRoot: absRoot,
		overrides: waitOverrides{
			postWait:    postWait,
			waitDone:    waitDone,
			doneTimeout: doneTimeout,
		},
	})
	if goldenFailed {
		failed = true
	}

	runFailed, runLogs := runCases(filtered, runConfig{
		absRoot: absRoot,
		formats: snapFormats,
		overrides: waitOverrides{
			postWait:    postWait,
			waitDone:    waitDone,
			doneTimeout: doneTimeout,
		},
	})
	if runFailed {
		failed = true
	}

	results, summary, compareFailed, hasDiff := compareCases(filtered, diffFormats, *diffAlways, *output == "diff")
	for _, entry := range results {
		name, _ := entry["name"].(string)
		logEntry := map[string][]string{}
		if logs, ok := goldenLogs[name]; ok && len(logs) > 0 {
			logEntry["golden"] = append([]string{}, logs...)
		}
		if logs, ok := runLogs[name]; ok && len(logs) > 0 {
			logEntry["run"] = append([]string{}, logs...)
		}
		if len(logEntry) > 0 {
			entry["logs"] = logEntry
		}
	}
	if compareFailed {
		failed = true
	}

	if *output == "json" {
		out := map[string]any{
			"root":    absRoot,
			"summary": summary,
			"cases":   results,
		}
		payload, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Println(string(payload))
	} else if *output == "diff" {
		printCompareDiff(results)
	} else {
		diffHeader := "diff_paths"
		if len(diffFormats) == 1 {
			for key := range diffFormats {
				diffHeader = "diff_path(" + key + ")"
			}
		}
		printCompareText(results, diffHeader, len(diffFormats) == 1)
	}

	if err := collectDiffArtifacts(filtered, filepath.Join(absRoot, ".nvim-snap-diff")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}

	if failed {
		os.Exit(2)
	}
	if hasDiff {
		os.Exit(1)
	}
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

func writeDiffOutputs(c casefile.Case, expected, actual snapshots.Snapshot, formats map[string]bool) (map[string]string, error) {
	if err := os.MkdirAll(c.DiffDir, 0o755); err != nil {
		return nil, err
	}
	paths := map[string]string{}
	if formats["text"] {
		diff, err := unifiedDiffText(c.ExpectedLabel, c.ActualLabel, snapshots.RenderText(expected), snapshots.RenderText(actual))
		if err != nil {
			return nil, err
		}
		path := filepath.Join(c.DiffDir, "diff.txt")
		if err := writeText(path, diff); err != nil {
			return nil, err
		}
		paths["text"] = path
	}
	if formats["ansi"] {
		diff, err := unifiedDiffText(c.ExpectedLabel, c.ActualLabel, snapshots.RenderANSI(expected), snapshots.RenderANSI(actual))
		if err != nil {
			return nil, err
		}
		path := filepath.Join(c.DiffDir, "diff.ansi")
		if err := writeText(path, diff); err != nil {
			return nil, err
		}
		paths["ansi"] = path
	}
	if formats["html"] {
		html := htmldiff.RenderHTML(expected, actual, "unified", c.ExpectedLabel, c.ActualLabel)
		path := filepath.Join(c.DiffDir, "diff.html")
		if err := writeText(path, html); err != nil {
			return nil, err
		}
		paths["html"] = path
	}
	if formats["png"] {
		html := htmldiff.RenderHTML(expected, actual, "overlay", c.ExpectedLabel, c.ActualLabel)
		path := filepath.Join(c.DiffDir, "diff.png")
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

func confirmActions(prompt string, count int) bool {
	fmt.Fprintf(os.Stderr, prompt, count)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && answer == "" {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func cmdAccept(args []string) {
	fs := flag.NewFlagSet("accept", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	dryRun := fs.Bool("dry-run", false, "Show updates without writing")
	noConfirm := fs.Bool("no-confirm", false, "Skip confirmation prompt")
	yes := fs.Bool("yes", false, "Skip confirmation prompt")
	var tags stringList
	var names stringList
	fs.Var(&tags, "tag", "Tag filter")
	fs.Var(&names, "case", "Case name filter")
	_ = fs.Parse(args)

	confirm := !*noConfirm && !*yes

	absRoot := mustAbs(*root)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := casefile.Filter(cases, tags, names)

	failed := len(errs) > 0
	actions := []map[string]any{}
	for _, c := range filtered {
		if c.Kind != "regression" {
			continue
		}
		if _, err := os.Stat(c.Actual); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s not found: %v\n", c.Name, c.ActualLabel, err)
			failed = true
			continue
		}
		actions = append(actions, map[string]any{
			"case": c,
			"src":  c.Actual,
			"dst":  c.Expected,
		})
	}

	if len(actions) == 0 {
		if failed {
			os.Exit(1)
		}
		return
	}

	if *dryRun {
		for _, action := range actions {
			c := action["case"].(casefile.Case)
			fmt.Printf("%s\t%s -> %s\n", c.Name, action["src"], action["dst"])
		}
		if failed {
			os.Exit(1)
		}
		return
	}

	if confirm {
		if !confirmActions("Accept current snapshots for %d case(s)? [y/N]: ", len(actions)) {
			fmt.Fprintln(os.Stderr, "aborted")
			os.Exit(1)
		}
	}

	for _, action := range actions {
		c := action["case"].(casefile.Case)
		if err := copyFile(action["src"].(string), action["dst"].(string)); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}
		fmt.Printf("%s\taccepted\n", c.Name)
	}
	if failed {
		os.Exit(1)
	}
}

func cmdGolden(args []string) {
	fs := flag.NewFlagSet("golden", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	var postWait optionalInt
	var waitDone optionalBool
	var doneTimeout optionalInt
	fs.Var(&postWait, "post-wait", "Wait after scenario execution (ms)")
	fs.Var(&waitDone, "wait-done", "Wait for scenario completion notification")
	fs.Var(&doneTimeout, "done-timeout", "Scenario completion timeout (ms)")
	dryRun := fs.Bool("dry-run", false, "Show updates without writing")
	var tags stringList
	var names stringList
	fs.Var(&tags, "tag", "Tag filter")
	fs.Var(&names, "case", "Case name filter")
	_ = fs.Parse(args)

	absRoot := mustAbs(*root)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := casefile.Filter(cases, tags, names)

	failed := len(errs) > 0
	if *dryRun {
		for _, c := range filtered {
			if c.Kind != "golden" {
				continue
			}
			fmt.Printf("%s\t%s -> %s\n", c.Name, c.Golden, c.Expected)
		}
		if failed {
			os.Exit(1)
		}
		return
	}

	if goldenFailed, _ := goldenCases(filtered, goldenConfig{
		absRoot: absRoot,
		overrides: waitOverrides{
			postWait:    postWait,
			waitDone:    waitDone,
			doneTimeout: doneTimeout,
		},
	}); goldenFailed {
		failed = true
	}
	if failed {
		os.Exit(1)
	}
}

type goldenConfig struct {
	absRoot   string
	overrides waitOverrides
}

func goldenCases(cases []casefile.Case, cfg goldenConfig) (bool, map[string][]string) {
	failed := false
	logs := map[string][]string{}
	actions := []casefile.Case{}
	for _, c := range cases {
		if c.Kind != "golden" {
			continue
		}
		if _, err := os.Stat(c.Golden); err != nil {
			fmt.Fprintf(os.Stderr, "%s: golden scenario not found: %v\n", c.Name, err)
			failed = true
			continue
		}
		actions = append(actions, c)
	}

	for _, c := range actions {
		casePostWait, caseWaitDone, caseDoneTimeout := resolveWaits(c, cfg.overrides)
		res, err := collector.Collect(collector.Options{
			Scenario:      c.Golden,
			Width:         c.Width,
			Height:        c.Height,
			WaitMS:        c.Wait,
			PostWaitMS:    casePostWait,
			WaitDone:      caseWaitDone,
			DoneTimeoutMS: caseDoneTimeout,
			RPCTimeoutMS:  c.RPCTimeout,
			DataHome:      c.DataHome,
			ConfigHome:    c.ConfigHome,
			LogFile:       c.LogFile,
			LogLevel:      c.LogLevel,
			WorkDir:       cfg.absRoot,
			RTP:           c.RTP,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}
		if caseWaitDone && !res.WaitedDone {
			fmt.Fprintf(os.Stderr, "%s: wait_done timeout (possible input wait; prefer vim.api.nvim_cmd)\n", c.Name)
		}
		if len(res.Logs) > 0 {
			logs[c.Name] = append(logs[c.Name], res.Logs...)
			if err := writeScenarioLogs(cfg.absRoot, c.Name, "golden", res.Logs); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
				failed = true
				continue
			}
		}
		if err := writeJSON(c.Expected, res.Snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}
		fmt.Printf("%s\tgenerated\n", c.Name)
	}
	return failed, logs
}

func workflowYAML(name, root, casesDir, diffFormat string) string {
	lines := []string{
		"name: " + name,
		"",
		"on:",
		"  push:",
		"  pull_request:",
		"",
		"jobs:",
		"  snap:",
		"    runs-on: ubuntu-latest",
		"    steps:",
		"      - uses: actions/checkout@v4",
		"      - name: Install Neovim",
		"        run: sudo apt-get update && sudo apt-get install -y neovim",
		"      - name: Install nvim-snap",
		"        run: |",
		"          curl -sSL https://github.com/kyoh86/nvim-snap/releases/latest/download/nvim-snap -o nvim-snap",
		"          chmod +x nvim-snap",
		"      - name: Report snapshots",
		"        run: |",
		"          ./nvim-snap report --root " + root + " --cases-dir " + casesDir + " --output diff --diff-format " + diffFormat + " --diff-always",
		"      - name: Upload diffs",
		"        if: always()",
		"        uses: actions/upload-artifact@v4",
		"        with:",
		"          name: nvim-snap-diff",
		"          path: |",
		"            .nvim-snap-diff/*",
		"      - name: Upload logs",
		"        if: always()",
		"        uses: actions/upload-artifact@v4",
		"        with:",
		"          name: nvim-snap-log",
		"          path: |",
		"            .nvim-snap-log/*",
		"",
	}
	return strings.Join(lines, "\n")
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	path := fs.String("path", ".github/workflows/nvim-snap.yml", "Workflow path")
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	diffFormat := fs.String("diff-format", "html", "Diff formats (text,ansi,html,png)")
	name := fs.String("name", "nvim-snap", "Workflow name")
	force := fs.Bool("force", false, "Overwrite existing workflow")
	_ = fs.Parse(args)

	if err := writeFile(*path, workflowYAML(*name, *root, *casesDir, *diffFormat), *force); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(*path)
}

func cmdNew(args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	name := fs.String("name", "", "Case name")
	title := fs.String("title", "", "Case title")
	kind := fs.String("kind", "regression", "regression|golden")
	force := fs.Bool("force", false, "Overwrite existing files")
	var tags stringList
	fs.Var(&tags, "tag", "Tag")
	_ = fs.Parse(args)

	if *kind != "regression" && *kind != "golden" {
		fmt.Fprintln(os.Stderr, "--kind must be regression or golden")
		os.Exit(2)
	}

	absRoot := mustAbs(*root)
	casesRoot := filepath.Join(absRoot, *casesDir)
	if err := os.MkdirAll(casesRoot, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	caseName := *name
	if caseName == "" {
		caseName = randomName(8)
	}
	caseDir := filepath.Join(casesRoot, caseName)
	if !*force {
		if _, err := os.Stat(caseDir); err == nil {
			fmt.Fprintf(os.Stderr, "case directory already exists: %s\n", caseDir)
			os.Exit(1)
		}
	}
	if err := os.MkdirAll(caseDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := writeCaseGitignore(filepath.Join(caseDir, ".gitignore"), *force); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writeSnapcaseJSON(filepath.Join(caseDir, "snapcase.json"), *title, *kind, tags, *force); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *kind == "golden" {
		_ = os.MkdirAll(filepath.Join(caseDir, "baseline"), 0o755)
		_ = os.MkdirAll(filepath.Join(caseDir, "actual"), 0o755)
	} else {
		_ = os.MkdirAll(filepath.Join(caseDir, "accepted"), 0o755)
		_ = os.MkdirAll(filepath.Join(caseDir, "current"), 0o755)
	}
	_ = os.MkdirAll(filepath.Join(caseDir, "diff"), 0o755)

	if *kind == "regression" {
		if err := writeFile(filepath.Join(caseDir, "scenario.lua"), regressionScenario(caseName), *force); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		if err := writeFile(filepath.Join(caseDir, "golden.lua"), goldenScenario(caseName, "golden"), *force); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := writeFile(filepath.Join(caseDir, "target.lua"), goldenScenario(caseName, "target"), *force); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	fmt.Printf("created %s\n", caseDir)
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

func readSnapshot(path string) (snapshots.Snapshot, error) {
	var out snapshots.Snapshot
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

func equalSnapshot(a, b snapshots.Snapshot) bool {
	return reflect.DeepEqual(a, b)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func randomName(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, length)
	seed := int64(os.Getpid()) + timeNowSeed()
	for i := range out {
		seed = seed*1664525 + 1013904223
		idx := seed % int64(len(chars))
		out[i] = chars[idx]
	}
	return string(out)
}

func timeNowSeed() int64 {
	return timeNow().UnixNano()
}

var timeNow = func() time.Time {
	return time.Now()
}

func writeCaseGitignore(path string, force bool) error {
	contents := strings.Join([]string{
		".nvim-data/",
		".nvim-config/",
		".out/",
		"current/",
		"baseline/",
		"actual/",
		"diff/",
		"",
	}, "\n")
	return writeFile(path, contents, force)
}

func writeSnapcaseJSON(path, title, kind string, tags []string, force bool) error {
	payload := map[string]any{
		"version":     1,
		"kind":        kind,
		"scenario":    "scenario.lua",
		"out_dir":     ".out",
		"data_home":   ".nvim-data",
		"config_home": ".nvim-config",
		"outputs": map[string]any{
			"json": "snapshot.json",
			"ansi": "snapshot.ansi",
			"html": "snapshot.html",
		},
		"rtp": []string{"."},
	}
	if title != "" {
		payload["title"] = title
	}
	if len(tags) > 0 {
		payload["tags"] = tags
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(path, string(data), force)
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

func regressionScenario(name string) string {
	return strings.Join([]string{
		"vim.cmd(\"enew\")",
		"vim.fn.setline(1, {",
		"  \"case: " + name + "\",",
		"  \"edit this scenario\",",
		"})",
		"",
	}, "\n")
}

func goldenScenario(name, label string) string {
	return strings.Join([]string{
		"vim.cmd(\"enew\")",
		"vim.fn.setline(1, {",
		"  \"" + label + " view for " + name + "\",",
		"  \"edit this scenario\",",
		"})",
		"",
	}, "\n")
}
