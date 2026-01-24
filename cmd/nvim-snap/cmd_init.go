package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
