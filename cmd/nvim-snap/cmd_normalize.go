package main

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/kyoh86/nvim-snap/internal/snapshots"
)

func cmdNormalize(args []string) error {
	fs := flag.NewFlagSet("normalize", flag.ExitOnError)
	inPath := fs.String("in", "-", "Input snapshot JSON ('-' for stdin)")
	outPath := fs.String("out", "-", "Output path ('-' for stdout)")
	_ = fs.Parse(args)

	var data []byte
	if *inPath == "-" {
		payload, err := io.ReadAll(os.Stdin)
		if err != nil {
			return exitError(1, err)
		}
		data = payload
	} else {
		payload, err := os.ReadFile(*inPath)
		if err != nil {
			return exitError(1, err)
		}
		data = payload
	}

	var snapshot snapshots.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return exitError(1, err)
	}
	normalized := snapshots.Normalize(snapshot)
	payload, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return exitError(1, err)
	}

	if *outPath == "-" {
		if _, err := os.Stdout.Write(payload); err != nil {
			return exitError(1, err)
		}
		if _, err := os.Stdout.Write([]byte("\n")); err != nil {
			return exitError(1, err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		return exitError(1, err)
	}
	if err := os.WriteFile(*outPath, payload, 0o644); err != nil {
		return exitError(1, err)
	}
	return nil
}
