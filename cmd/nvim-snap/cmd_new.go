package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

func resolveCasesRoot(absRoot, casesDir string) string {
	if casesDir == "" {
		casesDir = "snapcase"
	}
	if filepath.IsAbs(casesDir) {
		return casesDir
	}
	return filepath.Join(absRoot, casesDir)
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
