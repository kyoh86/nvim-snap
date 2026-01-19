package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/neovim/go-client/nvim"
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

func ensureFile(path string) error {
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

func main() {
	var (
		scenario   string
		nvimPath   string
		width      int
		height     int
		waitMS     int
		timeoutMS  int
		dataHome   string
		configHome string
		logFile    string
		logLevel   string
		workDir    string
	)
	var rtp stringList

	flag.StringVar(&scenario, "scenario", "", "Scenario script path (required)")
	flag.StringVar(&nvimPath, "nvim", "nvim", "Neovim binary")
	flag.IntVar(&width, "width", 80, "UI columns")
	flag.IntVar(&height, "height", 24, "UI lines")
	flag.IntVar(&waitMS, "wait", 200, "Wait for redraw flush (ms)")
	flag.IntVar(&timeoutMS, "rpc-timeout", 2000, "RPC timeout (ms)")
	flag.StringVar(&dataHome, "data-home", "", "XDG data home")
	flag.StringVar(&configHome, "config-home", "", "XDG config home")
	flag.StringVar(&logFile, "log-file", "", "NVIM_LOG_FILE path")
	flag.StringVar(&logLevel, "log-level", "", "NVIM_LOG_LEVEL")
	flag.StringVar(&workDir, "workdir", "", "Working directory")
	flag.Var(&rtp, "rtp", "Runtimepath entry (repeatable)")
	flag.Parse()

	if scenario == "" {
		fmt.Fprintln(os.Stderr, "-scenario is required")
		os.Exit(2)
	}

	absScenario, err := filepath.Abs(scenario)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to resolve scenario: %v\n", err)
		os.Exit(2)
	}

	env := os.Environ()
	if dataHome != "" {
		env = append(env, "XDG_DATA_HOME="+dataHome)
	}
	if configHome != "" {
		env = append(env, "XDG_CONFIG_HOME="+configHome)
	}
	if logFile != "" {
		if err := ensureFile(logFile); err != nil {
			fmt.Fprintf(os.Stderr, "failed to prepare log file: %v\n", err)
			os.Exit(2)
		}
		env = append(env, "NVIM_LOG_FILE="+logFile)
	}
	if logLevel != "" {
		env = append(env, "NVIM_LOG_LEVEL="+logLevel)
	}

	args := []string{"--embed", "--headless", "-u", "NONE", "-i", "NONE", "-n"}
	opts := []nvim.ChildProcessOption{
		nvim.ChildProcessCommand(nvimPath),
		nvim.ChildProcessArgs(args...),
		nvim.ChildProcessEnv(env),
	}
	if workDir != "" {
		opts = append(opts, nvim.ChildProcessDir(workDir))
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMS)*time.Millisecond)
	defer cancel()
	opts = append(opts, nvim.ChildProcessContext(ctx))

	v, err := nvim.NewChildProcess(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start nvim: %v\n", err)
		os.Exit(1)
	}
	defer v.Close()

	var gotFlush int32
	flushCh := make(chan struct{}, 1)
	if err := v.RegisterHandler("redraw", func(updates ...[]interface{}) {
		for _, update := range updates {
			if len(update) == 0 {
				continue
			}
			name, ok := update[0].(string)
			if ok && name == "flush" {
				if atomic.CompareAndSwapInt32(&gotFlush, 0, 1) {
					select {
					case flushCh <- struct{}{}:
					default:
					}
				}
			}
		}
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to register redraw handler: %v\n", err)
		os.Exit(1)
	}

	uiOpts := map[string]interface{}{
		"ext_linegrid":  true,
		"ext_hlstate":   true,
		"ext_multigrid": false,
	}
	if err := v.AttachUI(width, height, uiOpts); err != nil {
		fmt.Fprintf(os.Stderr, "failed to attach UI: %v\n", err)
		os.Exit(1)
	}

	if len(rtp) > 0 {
		if err := v.ExecLua(`local paths = ...
for i = #paths, 1, -1 do
  vim.opt.rtp:prepend(paths[i])
end`, nil, []string(rtp)); err != nil {
			fmt.Fprintf(os.Stderr, "failed to set rtp: %v\n", err)
			os.Exit(1)
		}
	}

	if err := v.ExecLua(`local p = ...; dofile(p)`, nil, absScenario); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run scenario: %v\n", err)
		os.Exit(1)
	}

	if err := v.Command("redraw"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to redraw: %v\n", err)
		os.Exit(1)
	}

	select {
	case <-flushCh:
		fmt.Println("got redraw flush")
	case <-time.After(time.Duration(waitMS) * time.Millisecond):
		fmt.Println("flush not received within wait")
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			fmt.Println("rpc timeout")
		} else {
			fmt.Printf("context error: %v\n", ctx.Err())
		}
	}

	_ = v.Command("qa!")
}
