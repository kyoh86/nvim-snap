package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kyoh86/nvim-snap/internal/collector"
	"github.com/kyoh86/nvim-snap/internal/report"
)

func cmdCapture(args []string) {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	scenario := fs.String("scenario", "", "Scenario file")
	outDir := fs.String("out", "", "Output directory")
	format := fs.String("format", "json", "Output formats (json,ansi,html)")
	width := fs.Int("width", 0, "UI width")
	height := fs.Int("height", 0, "UI height")
	wait := fs.Int("wait", 0, "Wait before capture (ms)")
	postWait := fs.Int("post-wait", 0, "Wait after scenario execution (ms)")
	waitDone := fs.Bool("wait-done", false, "Wait for scenario completion notification")
	doneTimeout := fs.Int("done-timeout", 0, "Scenario completion timeout (ms)")
	rpcTimeout := fs.Int("rpc-timeout", 0, "RPC timeout (ms)")
	nvimPath := fs.String("nvim", "", "Neovim executable path")
	dataHome := fs.String("data-home", "", "XDG data home")
	configHome := fs.String("config-home", "", "XDG config home")
	logFile := fs.String("log-file", "", "Neovim log file path")
	logLevel := fs.String("log-level", "", "Neovim log level")
	workDir := fs.String("workdir", ".", "Working directory")
	var rtp stringList
	fs.Var(&rtp, "rtp", "Runtime path entry (comma separated or repeat)")
	_ = fs.Parse(args)

	if *scenario == "" {
		fmt.Fprintln(os.Stderr, "--scenario is required")
		os.Exit(2)
	}
	if *outDir == "" {
		fmt.Fprintln(os.Stderr, "--out is required")
		os.Exit(2)
	}

	formats := parseFormats(*format, map[string]bool{"json": true})
	for key := range formats {
		if key != "json" && key != "ansi" && key != "html" {
			fmt.Fprintf(os.Stderr, "unsupported format: %s\n", key)
			os.Exit(2)
		}
	}

	res, err := collector.Collect(collector.Options{
		Scenario:      *scenario,
		NvimPath:      *nvimPath,
		Width:         *width,
		Height:        *height,
		WaitMS:        *wait,
		PostWaitMS:    *postWait,
		WaitDone:      *waitDone,
		DoneTimeoutMS: *doneTimeout,
		RPCTimeoutMS:  *rpcTimeout,
		DataHome:      *dataHome,
		ConfigHome:    *configHome,
		LogFile:       *logFile,
		LogLevel:      *logLevel,
		WorkDir:       mustAbs(*workDir),
		RTP:           rtp,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *waitDone && !res.WaitedDone {
		fmt.Fprintln(os.Stderr, "wait_done timeout (possible input wait; prefer vim.api.nvim_cmd)")
	}
	if err := report.WriteSnapshotOutputs(*outDir, "snapshot", res.Snapshot, formats); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok")
}
