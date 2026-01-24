package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/kyoh86/nvim-snap/internal/casefile"
)

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
