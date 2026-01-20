package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/kyoh86/nvim-snap/internal/collector"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	if value == "" {
		return nil
	}
	*s = append(*s, value)
	return nil
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
	var (
		scenario   string
		nvimPath   string
		width      int
		height     int
		waitMS     int
		postWaitMS int
		waitDone   bool
		doneWaitMS int
		timeoutMS  int
		dataHome   string
		configHome string
		logFile    string
		logLevel   string
		workDir    string
		outPath    string
	)
	var rtp stringList

	flag.StringVar(&scenario, "scenario", "", "Scenario script path (required)")
	flag.StringVar(&nvimPath, "nvim", "nvim", "Neovim binary")
	flag.IntVar(&width, "width", 80, "UI columns")
	flag.IntVar(&height, "height", 24, "UI lines")
	flag.IntVar(&waitMS, "wait", 200, "Wait for redraw flush (ms)")
	flag.IntVar(&postWaitMS, "post-wait", 0, "Wait after scenario execution (ms)")
	flag.BoolVar(&waitDone, "wait-done", false, "Wait for scenario completion notification")
	flag.IntVar(&doneWaitMS, "done-timeout", 5000, "Wait timeout for scenario completion (ms)")
	flag.IntVar(&timeoutMS, "rpc-timeout", 2000, "RPC timeout (ms)")
	flag.StringVar(&dataHome, "data-home", "", "XDG data home")
	flag.StringVar(&configHome, "config-home", "", "XDG config home")
	flag.StringVar(&logFile, "log-file", "", "NVIM_LOG_FILE path")
	flag.StringVar(&logLevel, "log-level", "", "NVIM_LOG_LEVEL")
	flag.StringVar(&workDir, "workdir", "", "Working directory")
	flag.StringVar(&outPath, "out", "-", "Output snapshot JSON path ('-' for stdout)")
	flag.Var(&rtp, "rtp", "Runtimepath entry (repeatable)")
	flag.Parse()

	if scenario == "" {
		fmt.Fprintln(os.Stderr, "-scenario is required")
		os.Exit(2)
	}

	result, err := collector.Collect(collector.Options{
		Scenario:      scenario,
		NvimPath:      nvimPath,
		Width:         width,
		Height:        height,
		WaitMS:        waitMS,
		PostWaitMS:    postWaitMS,
		WaitDone:      waitDone,
		DoneTimeoutMS: doneWaitMS,
		RPCTimeoutMS:  timeoutMS,
		DataHome:      dataHome,
		ConfigHome:    configHome,
		LogFile:       logFile,
		LogLevel:      logLevel,
		WorkDir:       workDir,
		RTP:           rtp,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect failed: %v\n", err)
		os.Exit(1)
	}

	payload, err := json.MarshalIndent(result.Snapshot, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode snapshot: %v\n", err)
		os.Exit(1)
	}
	if err := writeOutput(outPath, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}
