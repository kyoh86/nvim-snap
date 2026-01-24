package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kyoh86/nvim-snap/internal/casefile"
	"github.com/kyoh86/nvim-snap/internal/paths"
	"github.com/kyoh86/nvim-snap/internal/report"
	"github.com/kyoh86/nvim-snap/internal/runner"
)

func usageRegression() {
	fmt.Println("usage:")
	fmt.Println("  nvim-snap regression <command> [options]")
	fmt.Println("")
	fmt.Println("commands:")
	fmt.Println("  new   scaffold a regression test case")
	fmt.Println("  save  run scenarios and save snapshots by id")
	fmt.Println("  test  compare saved snapshots by id")
}

func cmdRegression(args []string) error {
	if len(args) == 0 {
		usageRegression()
		return usageError()
	}
	switch args[0] {
	case "new":
		return cmdRegressionNew(args[1:])
	case "save":
		return cmdRegressionSave(args[1:])
	case "test":
		return cmdRegressionTest(args[1:])
	default:
		usageRegression()
		return usageError()
	}
}

func cmdRegressionSave(args []string) error {
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
			return exitErrorf(2, "unsupported format: %s", key)
		}
	}

	absRoot := mustAbs(*root)
	resultsRoot := paths.ResolveResultsRoot(absRoot, *casesDir)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := casefile.FilterByKind(casefile.Filter(cases, tags, names), "regression")

	id := *saveID
	if id == "" {
		var err error
		id, err = resolveCommitID(absRoot, "")
		if err != nil {
			return exitErrorf(2, "failed to resolve id: %v", err)
		}
	}

	cfg := runner.Config{
		Root:        absRoot,
		ResultsRoot: resultsRoot,
		Overrides: runner.WaitOverrides{
			PostWait:    optionalIntPtr(postWait),
			WaitDone:    optionalBoolPtr(waitDone),
			DoneTimeout: optionalIntPtr(doneTimeout),
		},
	}

	failed := len(errs) > 0
	for _, c := range filtered {
		res, err := runner.CollectCase(c, c.Scenario, cfg, "save", "", "")
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
		if err := report.WriteSnapshotOutputs(resultDir, "snapshot-"+id, res.Snapshot, formats); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", c.Name, err)
			failed = true
			continue
		}
		fmt.Printf("%s\tok\n", c.Name)
	}

	if failed {
		return ExitError{Code: 1, Silent: true}
	}
	return nil
}

func cmdRegressionTest(args []string) error {
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
		return exitErrorf(2, "--base is required")
	}

	formats := parseFormats(*diffFormat, map[string]bool{"text": true})
	for key := range formats {
		if key != "text" && key != "ansi" && key != "html" && key != "png" {
			return exitErrorf(2, "unsupported format: %s", key)
		}
	}
	if *output != "summary" && *output != "diff" && *output != "json" {
		return exitErrorf(2, "unsupported output: %s", *output)
	}

	absRoot := mustAbs(*root)
	resultsRoot := paths.ResolveResultsRoot(absRoot, *casesDir)
	cases, errs := casefile.Find(absRoot, *casesDir)
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, err)
	}
	filtered := casefile.FilterByKind(casefile.Filter(cases, tags, names), "regression")

	target := *targetID
	if target == "" {
		var err error
		target, err = resolveCommitID(absRoot, "")
		if err != nil {
			return exitErrorf(2, "failed to resolve target id: %v", err)
		}
	}

	compareItems := make([]report.CompareCase, 0, len(filtered))
	for _, c := range filtered {
		resultDir := filepath.Join(resultsRoot, "regression", c.Name)
		diffDir := filepath.Join(resultDir, "diff")
		compareItems = append(compareItems, report.CompareCase{
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

	results, summary, failed, hasDiff := report.CompareCases(compareItems, formats, *diffAlways, *output == "diff")
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
			return exitError(2, err)
		}
		fmt.Println(string(payload))
	} else if *output == "diff" {
		report.PrintCompareDiff(results, os.Stdout, os.Stderr)
	} else {
		diffHeader := "diff_paths"
		if len(formats) == 1 {
			for key := range formats {
				diffHeader = "diff_path(" + key + ")"
			}
		}
		report.PrintCompareText(results, diffHeader, len(formats) == 1, os.Stdout)
	}

	if failed {
		return ExitError{Code: 2, Silent: true}
	}
	if hasDiff {
		return ExitError{Code: 1, Silent: true}
	}
	return nil
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
