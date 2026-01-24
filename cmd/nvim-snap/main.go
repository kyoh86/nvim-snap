package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	case "capture":
		cmdCapture(os.Args[2:])
	case "normalize":
		cmdNormalize(os.Args[2:])
	case "compare":
		cmdCompare(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	case "init":
		cmdInit(os.Args[2:])
	case "regression":
		cmdRegression(os.Args[2:])
	case "golden":
		cmdGolden(os.Args[2:])
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
	fmt.Println("  capture     capture a snapshot from a scenario")
	fmt.Println("  normalize   normalize a snapshot JSON")
	fmt.Println("  compare     compare two snapshot JSON files")
	fmt.Println("  list        list test cases")
	fmt.Println("  init        scaffold CI workflow")
	fmt.Println("  regression  regression commands (new/save/test)")
	fmt.Println("  golden      golden commands (new/test)")
}

func cmdCapture(args []string) {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	scenario := fs.String("scenario", "", "Scenario file")
	outDir := fs.String("out", "", "Output directory")
	format := fs.String("format", "json", "Output formats (json,ansi,html)")
	width := fs.Int("width", 0, "UI width")
	height := fs.Int("height", 0, "UI height")
	wait := fs.Int("wait", 0, "Wait before capture (ms)")
	postWait := fs.Int("post-wait", 0, "Wait after scenario execution (ms)")
	waitDone := fs.Bool("wait-done", false, "Wait for scenario completion notification")
	doneTimeout := fs.Int("done-timeout", 0, "Scenario completion timeout (ms)")
	rpcTimeout := fs.Int("rpc-timeout", 0, "RPC timeout (ms)")
	nvimPath := fs.String("nvim", "", "Neovim executable path")
	dataHome := fs.String("data-home", "", "XDG data home")
	configHome := fs.String("config-home", "", "XDG config home")
	logFile := fs.String("log-file", "", "Neovim log file path")
	logLevel := fs.String("log-level", "", "Neovim log level")
	workDir := fs.String("workdir", ".", "Working directory")
	var rtp stringList
	fs.Var(&rtp, "rtp", "Runtime path entry (comma separated or repeat)")
	_ = fs.Parse(args)

	if *scenario == "" {
		fmt.Fprintln(os.Stderr, "--scenario is required")
		os.Exit(2)
	}
	if *outDir == "" {
		fmt.Fprintln(os.Stderr, "--out is required")
		os.Exit(2)
	}

	formats := parseFormats(*format, map[string]bool{"json": true})
	for key := range formats {
		if key != "json" && key != "ansi" && key != "html" {
			fmt.Fprintf(os.Stderr, "unsupported format: %s\n", key)
			os.Exit(2)
		}
	}

	res, err := collector.Collect(collector.Options{
		Scenario:      *scenario,
		NvimPath:      *nvimPath,
		Width:         *width,
		Height:        *height,
		WaitMS:        *wait,
		PostWaitMS:    *postWait,
		WaitDone:      *waitDone,
		DoneTimeoutMS: *doneTimeout,
		RPCTimeoutMS:  *rpcTimeout,
		DataHome:      *dataHome,
		ConfigHome:    *configHome,
		LogFile:       *logFile,
		LogLevel:      *logLevel,
		WorkDir:       mustAbs(*workDir),
		RTP:           rtp,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *waitDone && !res.WaitedDone {
		fmt.Fprintln(os.Stderr, "wait_done timeout (possible input wait; prefer vim.api.nvim_cmd)")
	}
	if err := writeSnapshotOutputs(*outDir, "snapshot", res.Snapshot, formats); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok")
}

func cmdNormalize(args []string) {
	fs := flag.NewFlagSet("normalize", flag.ExitOnError)
	inPath := fs.String("in", "-", "Input snapshot JSON ('-' for stdin)")
	outPath := fs.String("out", "-", "Output path ('-' for stdout)")
	_ = fs.Parse(args)

	var data []byte
	if *inPath == "-" {
		payload, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		data = payload
	} else {
		payload, err := os.ReadFile(*inPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		data = payload
	}

	var snapshot snapshots.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	normalized := snapshots.Normalize(snapshot)
	payload, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *outPath == "-" {
		if _, err := os.Stdout.Write(payload); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if _, err := os.Stdout.Write([]byte("\n")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, payload, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func cmdCompare(args []string) {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	expectedPath := fs.String("expected", "", "Expected snapshot JSON path")
	actualPath := fs.String("actual", "", "Actual snapshot JSON path")
	format := fs.String("format", "text", "Diff format (text,ansi,html,png)")
	outPath := fs.String("out", "-", "Diff output path ('-' for stdout)")
	ctxLen := fs.Int("context", 3, "Unified diff context lines")
	_ = fs.Parse(args)

	if *expectedPath == "" || *actualPath == "" {
		fmt.Fprintln(os.Stderr, "--expected and --actual are required")
		os.Exit(2)
	}

	expected, err := readSnapshot(*expectedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read expected: %v\n", err)
		os.Exit(1)
	}
	actual, err := readSnapshot(*actualPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read actual: %v\n", err)
		os.Exit(1)
	}

	normExpected := snapshots.Normalize(expected)
	normActual := snapshots.Normalize(actual)
	if equalSnapshot(normExpected, normActual) {
		fmt.Println("no_diff")
		return
	}

	switch *format {
	case "html":
		html := htmldiff.RenderHTML(normExpected, normActual, "unified", "expected", "actual")
		if err := writeCompareOutput(*outPath, []byte(html), true); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write diff: %v\n", err)
			os.Exit(1)
		}
	case "png":
		if *outPath == "-" {
			fmt.Fprintln(os.Stderr, "png output requires a file path")
			os.Exit(2)
		}
		html := htmldiff.RenderHTML(normExpected, normActual, "overlay", "expected", "actual")
		if err := pngutil.WritePNGFromHTML(html, *outPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write diff: %v\n", err)
			os.Exit(1)
		}
	case "ansi", "text":
		var expectedText, actualText string
		if *format == "ansi" {
			expectedText = snapshots.RenderANSI(normExpected)
			actualText = snapshots.RenderANSI(normActual)
		} else {
			expectedText = snapshots.RenderText(normExpected)
			actualText = snapshots.RenderText(normActual)
		}
		diff, err := unifiedDiffTextContext("expected", "actual", expectedText, actualText, *ctxLen)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create diff: %v\n", err)
			os.Exit(1)
		}
		if err := writeCompareOutput(*outPath, []byte(strings.TrimRight(diff, "\n")), false); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write diff: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unsupported format: %s\n", *format)
		os.Exit(2)
	}

	fmt.Println("diff")
	os.Exit(1)
}

func usageRegression() {
	fmt.Println("usage:")
	fmt.Println("  nvim-snap regression <command> [options]")
	fmt.Println("")
	fmt.Println("commands:")
	fmt.Println("  new   scaffold a regression test case")
	fmt.Println("  save  run scenarios and save snapshots by id")
	fmt.Println("  test  compare saved snapshots by id")
}

func usageGolden() {
	fmt.Println("usage:")
	fmt.Println("  nvim-snap golden <command> [options]")
	fmt.Println("")
	fmt.Println("commands:")
	fmt.Println("  new   scaffold a golden test case")
	fmt.Println("  test  run golden+target scenarios and compare")
}

func cmdRegression(args []string) {
	if len(args) == 0 {
		usageRegression()
		os.Exit(2)
	}
	switch args[0] {
	case "new":
		cmdRegressionNew(args[1:])
	case "save":
		cmdRegressionSave(args[1:])
	case "test":
		cmdRegressionTest(args[1:])
	default:
		usageRegression()
		os.Exit(2)
	}
}

func cmdGolden(args []string) {
	if len(args) == 0 {
		usageGolden()
		os.Exit(2)
	}
	switch args[0] {
	case "new":
		cmdGoldenNew(args[1:])
	case "test":
		cmdGoldenTest(args[1:])
	default:
		usageGolden()
		os.Exit(2)
	}
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

type runConfig struct {
	absRoot     string
	resultsRoot string
	formats     map[string]bool
	overrides   waitOverrides
}

func resolveCasesRoot(absRoot, casesDir string) string {
	if casesDir == "" {
		casesDir = "snapcase"
	}
	if filepath.IsAbs(casesDir) {
		return casesDir
	}
	return filepath.Join(absRoot, casesDir)
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

func gitHead(absRoot string) (string, error) {
	cmd := exec.Command("git", "-C", absRoot, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitDirty(absRoot string) (bool, error) {
	cmd := exec.Command("git", "-C", absRoot, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

func resolveCommitID(absRoot, id string) (string, error) {
	if id != "" {
		return id, nil
	}
	dirty, err := gitDirty(absRoot)
	if err != nil {
		return "", err
	}
	if dirty {
		return "", fmt.Errorf("working tree is dirty")
	}
	return gitHead(absRoot)
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

func unifiedDiffTextContext(fromLabel, toLabel, expected, actual string, context int) (string, error) {
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expected),
		B:        difflib.SplitLines(actual),
		FromFile: fromLabel,
		ToFile:   toLabel,
		Context:  context,
	}
	return difflib.GetUnifiedDiffString(d)
}

func writeCompareOutput(path string, data []byte, raw bool) error {
	if path == "" || path == "-" {
		if _, err := os.Stdout.Write(data); err != nil {
			return err
		}
		if raw {
			_, err := os.Stdout.Write([]byte("\n"))
			return err
		}
		_, err := os.Stdout.Write([]byte("\n"))
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
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

func cmdRegressionSave(args []string) {
	fs := flag.NewFlagSet("regression save", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	saveID := fs.String("id", "", "Snapshot id (default: current git commit)")
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
	resultsRoot := resolveResultsRoot(absRoot, *casesDir)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := filterByKind(casefile.Filter(cases, tags, names), "regression")

	id := *saveID
	if id == "" {
		var err error
		id, err = resolveCommitID(absRoot, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to resolve id: %v\n", err)
			os.Exit(2)
		}
	}

	cfg := runConfig{
		absRoot:     absRoot,
		resultsRoot: resultsRoot,
		formats:     formats,
		overrides: waitOverrides{
			postWait:    postWait,
			waitDone:    waitDone,
			doneTimeout: doneTimeout,
		},
	}

	failed := len(errs) > 0
	for _, c := range filtered {
		res, err := collectSnapshot(c, c.Scenario, cfg, "save", "", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}
		resultDir := filepath.Join(resultsRoot, "regression", c.Name)
		if err := os.MkdirAll(resultDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}
		if err := writeSnapshotOutputs(resultDir, "snapshot-"+id, res.Snapshot, formats); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}
		fmt.Printf("%s\tok\n", c.Name)
	}

	if failed {
		os.Exit(1)
	}
}

func cmdRegressionTest(args []string) {
	fs := flag.NewFlagSet("regression test", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	baseID := fs.String("base", "", "Base snapshot id")
	targetID := fs.String("target", "", "Target snapshot id (default: current git commit)")
	diffFormat := fs.String("diff-format", "text", "Diff formats (text,ansi,html,png)")
	diffAlways := fs.Bool("diff-always", false, "Write diffs even if no difference")
	output := fs.String("output", "summary", "Output format (summary,diff,json)")
	var tags stringList
	var names stringList
	fs.Var(&tags, "tag", "Tag filter")
	fs.Var(&names, "case", "Case name filter")
	_ = fs.Parse(args)

	if *baseID == "" {
		fmt.Fprintln(os.Stderr, "--base is required")
		os.Exit(2)
	}

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
	resultsRoot := resolveResultsRoot(absRoot, *casesDir)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := filterByKind(casefile.Filter(cases, tags, names), "regression")

	target := *targetID
	if target == "" {
		var err error
		target, err = resolveCommitID(absRoot, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to resolve target id: %v\n", err)
			os.Exit(2)
		}
	}

	compareItems := make([]compareCase, 0, len(filtered))
	for _, c := range filtered {
		resultDir := filepath.Join(resultsRoot, "regression", c.Name)
		diffDir := filepath.Join(resultDir, "diff")
		compareItems = append(compareItems, compareCase{
			Name:          c.Name,
			Title:         c.Title,
			Kind:          c.Kind,
			Tags:          c.Tags,
			ExpectedPath:  filepath.Join(resultDir, "snapshot-"+*baseID+".json"),
			ActualPath:    filepath.Join(resultDir, "snapshot-"+target+".json"),
			ExpectedLabel: *baseID,
			ActualLabel:   target,
			DiffDir:       diffDir,
		})
	}

	results, summary, failed, hasDiff := compareCases(compareItems, formats, *diffAlways, *output == "diff")
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

func cmdGoldenTest(args []string) {
	fs := flag.NewFlagSet("golden test", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	format := fs.String("format", "json", "Snapshot formats (json,ansi,html)")
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

	formats := parseFormats(*format, map[string]bool{"json": true})
	for key := range formats {
		if key != "json" && key != "ansi" && key != "html" {
			fmt.Fprintf(os.Stderr, "unsupported format: %s\n", key)
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
	resultsRoot := resolveResultsRoot(absRoot, *casesDir)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := filterByKind(casefile.Filter(cases, tags, names), "golden")

	cfg := runConfig{
		absRoot:     absRoot,
		resultsRoot: resultsRoot,
		formats:     formats,
		overrides: waitOverrides{
			postWait:    postWait,
			waitDone:    waitDone,
			doneTimeout: doneTimeout,
		},
	}

	failed := len(errs) > 0
	compareItems := make([]compareCase, 0, len(filtered))
	for _, c := range filtered {
		resultDir := filepath.Join(resultsRoot, "golden", c.Name)
		baselineDir := filepath.Join(resultDir, "baseline")
		actualDir := filepath.Join(resultDir, "actual")

		goldenDataHome := filepath.Join(c.DataHome, "golden")
		goldenConfigHome := filepath.Join(c.ConfigHome, "golden")
		targetDataHome := filepath.Join(c.DataHome, "target")
		targetConfigHome := filepath.Join(c.ConfigHome, "target")
		goldenRes, err := collectSnapshot(c, c.Golden, cfg, "golden", goldenDataHome, goldenConfigHome)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: golden: %v\n", c.Name, err)
			failed = true
			continue
		}
		if err := writeSnapshotOutputs(baselineDir, "snapshot", goldenRes.Snapshot, formats); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}

		targetRes, err := collectSnapshot(c, c.Target, cfg, "target", targetDataHome, targetConfigHome)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: target: %v\n", c.Name, err)
			failed = true
			continue
		}
		if err := writeSnapshotOutputs(actualDir, "snapshot", targetRes.Snapshot, formats); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}

		diffDir := filepath.Join(resultDir, "diff")
		compareItems = append(compareItems, compareCase{
			Name:          c.Name,
			Title:         c.Title,
			Kind:          c.Kind,
			Tags:          c.Tags,
			ExpectedPath:  filepath.Join(baselineDir, "snapshot.json"),
			ActualPath:    filepath.Join(actualDir, "snapshot.json"),
			ExpectedLabel: "baseline",
			ActualLabel:   "actual",
			DiffDir:       diffDir,
		})
	}

	results, summary, compareFailed, hasDiff := compareCases(compareItems, diffFormats, *diffAlways, *output == "diff")
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

	if failed {
		os.Exit(2)
	}
	if hasDiff {
		os.Exit(1)
	}
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
		"      - name: Install Neovim (nightly)",
		"        uses: rhysd/action-setup-vim@v1",
		"        with:",
		"          neovim: true",
		"          version: nightly",
		"      - uses: kyoh86/setup-nvim-snap-action@main",
		"        id: setup",
		"      - uses: actions/cache@v4",
		"        with:",
		"          path: " + filepath.ToSlash(filepath.Join(root, casesDir, ".result")),
		"          key: nvim-snap-${{ runner.os }}-${{ github.sha }}",
		"          restore-keys: |",
		"            nvim-snap-${{ runner.os }}-",
		"      - uses: kyoh86/nvim-snap-action@main",
		"        with:",
		"          nvim-snap-path: ${{ steps.setup.outputs.nvim-snap-path }}",
		"          root: " + root + "",
		"          cases-dir: " + casesDir + "",
		"          diff-format: " + diffFormat + "",
		"          diff-always: true",
		"      - name: Upload results",
		"        if: always()",
		"        uses: actions/upload-artifact@v4",
		"        with:",
		"          name: nvim-snap-result",
		"          path: |",
		"            " + filepath.ToSlash(filepath.Join(root, casesDir, ".result")) + "/**",
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

func cmdRegressionNew(args []string) {
	cmdNewByKind(args, "regression")
}

func cmdGoldenNew(args []string) {
	cmdNewByKind(args, "golden")
}

func cmdNewByKind(args []string, kind string) {
	fs := flag.NewFlagSet(kind+" new", flag.ExitOnError)
	root := fs.String("root", ".", "Root directory")
	casesDir := fs.String("cases-dir", "snapcase", "Cases directory under root")
	name := fs.String("name", "", "Case name")
	title := fs.String("title", "", "Case title")
	force := fs.Bool("force", false, "Overwrite existing files")
	var tags stringList
	fs.Var(&tags, "tag", "Tag")
	_ = fs.Parse(args)

	absRoot := mustAbs(*root)
	casesRoot := resolveCasesRoot(absRoot, *casesDir)
	if err := os.MkdirAll(casesRoot, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := ensureResultsGitignore(casesRoot); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	caseName := *name
	if caseName == "" {
		caseName = randomName(8)
	}
	caseDir := filepath.Join(casesRoot, kind, caseName)
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

	scenario := ""
	if kind == "regression" {
		scenario = "scenario.lua"
	}
	if err := writeSnapcaseJSON(filepath.Join(caseDir, "snapcase.json"), *title, kind, tags, scenario, *force); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if kind == "regression" {
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
		if idx < 0 {
			idx = -idx
		}
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

func writeSnapcaseJSON(path, title, kind string, tags []string, scenario string, force bool) error {
	payload := map[string]any{
		"version":     1,
		"data_home":   ".nvim-data",
		"config_home": ".nvim-config",
		"rtp":         []string{"${ROOT}"},
	}
	if kind != "" {
		payload["kind"] = kind
	}
	if scenario != "" {
		payload["scenario"] = scenario
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

func ensureResultsGitignore(casesRoot string) error {
	path := filepath.Join(casesRoot, ".gitignore")
	entry := ".result/"
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(path, []byte(entry+"\n"), 0o644)
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}
	lines = append(lines, entry)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
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
