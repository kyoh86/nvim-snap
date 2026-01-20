package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/kyoh86/nvim-snap/internal/htmldiff"
	"github.com/kyoh86/nvim-snap/internal/pngutil"
	"github.com/kyoh86/nvim-snap/internal/snapshots"
	"github.com/pmezard/go-difflib/difflib"
)

type diffFormat string

const (
	formatNone diffFormat = ""
	formatText diffFormat = "text"
	formatANSI diffFormat = "ansi"
	formatHTML diffFormat = "html"
	formatPNG  diffFormat = "png"
)

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

func writeOutput(path string, data []byte) error {
	if path == "" || path == "-" {
		_, err := os.Stdout.Write(data)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write([]byte("\n"))
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func main() {
	var expectedPath string
	var actualPath string
	var format string
	var outPath string
	var ctxLen int

	flag.StringVar(&expectedPath, "expected", "", "Expected snapshot JSON path")
	flag.StringVar(&actualPath, "actual", "", "Actual snapshot JSON path")
	flag.StringVar(&format, "format", string(formatNone), "Diff format (text,ansi,html,png)")
	flag.StringVar(&outPath, "out", "-", "Diff output path ('-' for stdout)")
	flag.IntVar(&ctxLen, "context", 3, "Unified diff context lines")
	flag.Parse()

	if expectedPath == "" || actualPath == "" {
		fmt.Fprintln(os.Stderr, "-expected and -actual are required")
		os.Exit(2)
	}

	expected, err := readSnapshot(expectedPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read expected: %v\n", err)
		os.Exit(1)
	}
	actual, err := readSnapshot(actualPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read actual: %v\n", err)
		os.Exit(1)
	}

	normExpected := snapshots.Normalize(expected)
	normActual := snapshots.Normalize(actual)

	if reflect.DeepEqual(normExpected, normActual) {
		fmt.Println("no_diff")
		return
	}

	switch diffFormat(format) {
	case formatHTML:
		html := htmldiff.RenderHTML(normExpected, normActual, "unified", "expected", "actual")
		if err := writeOutput(outPath, []byte(html)); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write diff: %v\n", err)
			os.Exit(1)
		}
	case formatPNG:
		if outPath == "-" {
			fmt.Fprintln(os.Stderr, "png output requires a file path")
			os.Exit(2)
		}
		html := htmldiff.RenderHTML(normExpected, normActual, "overlay", "expected", "actual")
		if err := pngutil.WritePNGFromHTML(html, outPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write diff: %v\n", err)
			os.Exit(1)
		}
	case formatANSI, formatText:
		var expectedText, actualText string
		if diffFormat(format) == formatANSI {
			expectedText = snapshots.RenderANSI(normExpected)
			actualText = snapshots.RenderANSI(normActual)
		} else {
			expectedText = snapshots.RenderText(normExpected)
			actualText = snapshots.RenderText(normActual)
		}
		d := difflib.UnifiedDiff{
			A:        difflib.SplitLines(expectedText),
			B:        difflib.SplitLines(actualText),
			FromFile: "expected",
			ToFile:   "actual",
			Context:  ctxLen,
		}
		out, err := difflib.GetUnifiedDiffString(d)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create diff: %v\n", err)
			os.Exit(1)
		}
		out = strings.TrimRight(out, "\n")
		if err := writeOutput(outPath, []byte(out)); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write diff: %v\n", err)
			os.Exit(1)
		}
	case formatNone:
		// no diff output
	default:
		fmt.Fprintf(os.Stderr, "unsupported format: %s\n", format)
		os.Exit(2)
	}

	fmt.Println("diff")
	os.Exit(1)
}
