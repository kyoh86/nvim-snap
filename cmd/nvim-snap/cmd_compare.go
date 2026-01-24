package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyoh86/nvim-snap/internal/htmldiff"
	"github.com/kyoh86/nvim-snap/internal/pngutil"
	"github.com/kyoh86/nvim-snap/internal/report"
	"github.com/kyoh86/nvim-snap/internal/snapshots"
)

func cmdCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	expectedPath := fs.String("expected", "", "Expected snapshot JSON path")
	actualPath := fs.String("actual", "", "Actual snapshot JSON path")
	format := fs.String("format", "text", "Diff format (text,ansi,html,png)")
	outPath := fs.String("out", "-", "Diff output path ('-' for stdout)")
	ctxLen := fs.Int("context", 3, "Unified diff context lines")
	_ = fs.Parse(args)

	if *expectedPath == "" || *actualPath == "" {
		return exitErrorf(2, "--expected and --actual are required")
	}

	expected, err := snapshots.ReadJSON(*expectedPath)
	if err != nil {
		return exitErrorf(1, "failed to read expected: %v", err)
	}
	actual, err := snapshots.ReadJSON(*actualPath)
	if err != nil {
		return exitErrorf(1, "failed to read actual: %v", err)
	}

	normExpected := snapshots.Normalize(expected)
	normActual := snapshots.Normalize(actual)
	if snapshots.Equal(normExpected, normActual) {
		fmt.Println("no_diff")
		return nil
	}

	switch *format {
	case "html":
		html := htmldiff.RenderHTML(normExpected, normActual, "unified", "expected", "actual")
		if err := writeCompareOutput(*outPath, []byte(html), true); err != nil {
			return exitErrorf(1, "failed to write diff: %v", err)
		}
	case "png":
		if *outPath == "-" {
			return exitErrorf(2, "png output requires a file path")
		}
		html := htmldiff.RenderHTML(normExpected, normActual, "overlay", "expected", "actual")
		if err := pngutil.WritePNGFromHTML(html, *outPath); err != nil {
			return exitErrorf(1, "failed to write diff: %v", err)
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
		diff, err := report.UnifiedDiffText("expected", "actual", expectedText, actualText, *ctxLen)
		if err != nil {
			return exitErrorf(1, "failed to create diff: %v", err)
		}
		if err := writeCompareOutput(*outPath, []byte(strings.TrimRight(diff, "\n")), false); err != nil {
			return exitErrorf(1, "failed to write diff: %v", err)
		}
	default:
		return exitErrorf(2, "unsupported format: %s", *format)
	}

	fmt.Println("diff")
	return ExitError{Code: 1, Silent: true}
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
