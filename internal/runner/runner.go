// Package runner executes scenarios and collects snapshots/logs.
package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyoh86/nvim-snap/internal/casefile"
	"github.com/kyoh86/nvim-snap/internal/collector"
)

type WaitOverrides struct {
	PostWait    *int
	WaitDone    *bool
	DoneTimeout *int
}

type Config struct {
	Root        string
	ResultsRoot string
	Overrides   WaitOverrides
}

func resolveWaits(c casefile.Case, overrides WaitOverrides) (int, bool, int) {
	casePostWait := c.PostWait
	if overrides.PostWait != nil {
		casePostWait = *overrides.PostWait
	}
	caseWaitDone := c.WaitDone
	if overrides.WaitDone != nil {
		caseWaitDone = *overrides.WaitDone
	}
	caseDoneTimeout := c.DoneTimeout
	if overrides.DoneTimeout != nil {
		caseDoneTimeout = *overrides.DoneTimeout
	}
	return casePostWait, caseWaitDone, caseDoneTimeout
}

func CollectCase(c casefile.Case, scenario string, cfg Config, stage, dataHomeOverride, configHomeOverride string) (collector.Result, error) {
	if _, err := os.Stat(scenario); err != nil {
		return collector.Result{}, fmt.Errorf("scenario not found: %w", err)
	}
	dataHome := c.DataHome
	if dataHomeOverride != "" {
		dataHome = dataHomeOverride
	}
	configHome := c.ConfigHome
	if configHomeOverride != "" {
		configHome = configHomeOverride
	}
	casePostWait, caseWaitDone, caseDoneTimeout := resolveWaits(c, cfg.Overrides)
	res, err := collector.Collect(collector.Options{
		Scenario:      scenario,
		Width:         c.Width,
		Height:        c.Height,
		WaitMS:        c.Wait,
		PostWaitMS:    casePostWait,
		WaitDone:      caseWaitDone,
		DoneTimeoutMS: caseDoneTimeout,
		RPCTimeoutMS:  c.RPCTimeout,
		DataHome:      dataHome,
		ConfigHome:    configHome,
		LogFile:       c.LogFile,
		LogLevel:      c.LogLevel,
		WorkDir:       cfg.Root,
		RTP:           c.RTP,
	})
	if err != nil {
		return collector.Result{}, err
	}
	if caseWaitDone && !res.WaitedDone {
		fmt.Fprintf(os.Stderr, "%s: wait_done timeout (possible input wait; prefer vim.api.nvim_cmd)\n", c.Name)
	}
	if err := WriteScenarioLogs(cfg.ResultsRoot, c.Name, stage, res.Logs); err != nil {
		return collector.Result{}, err
	}
	return res, nil
}

func WriteScenarioLogs(resultsRoot, caseName, stage string, logs []string) error {
	if len(logs) == 0 {
		return nil
	}
	destDir := filepath.Join(resultsRoot, "logs")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(destDir, fmt.Sprintf("%s.%s.log", caseName, stage))
	return writeText(path, strings.Join(logs, "\n")+"\n")
}

func writeText(path, contents string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(contents), 0o644)
}
