package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kyoh86/nvim-snap/internal/casefile"
	"os"
	"path/filepath"
)

func usageGolden() {
	fmt.Println("usage:")
	fmt.Println("  nvim-snap golden <command> [options]")
	fmt.Println("")
	fmt.Println("commands:")
	fmt.Println("  new   scaffold a golden test case")
	fmt.Println("  test  run golden+target scenarios and compare")
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
