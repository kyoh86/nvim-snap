package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/kyoh86/nvim-snap/internal/snapshots"
	"io"
	"os"
	"path/filepath"
)

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
