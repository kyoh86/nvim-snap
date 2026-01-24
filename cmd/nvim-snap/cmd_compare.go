package main

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/kyoh86/nvim-snap/internal/htmldiff"
	"github.com/kyoh86/nvim-snap/internal/pngutil"
	"github.com/kyoh86/nvim-snap/internal/snapshots"
)

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

func equalSnapshot(a, b snapshots.Snapshot) bool {
	return reflect.DeepEqual(a, b)
}
