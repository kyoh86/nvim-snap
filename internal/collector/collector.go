// Package collector runs a scenario and captures a UI snapshot from Neovim.
package collector

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/kyoh86/nvim-snap/internal/snapshots"
	"github.com/neovim/go-client/msgpack/rpc"
	"github.com/neovim/go-client/nvim"
)

type Options struct {
	Scenario      string
	NvimPath      string
	Width         int
	Height        int
	WaitMS        int
	FlushRetry    int
	PostWaitMS    int
	WaitDone      bool
	DoneTimeoutMS int
	RPCTimeoutMS  int
	DataHome      string
	ConfigHome    string
	LogFile       string
	LogLevel      string
	WorkDir       string
	RTP           []string
}

type Result struct {
	Snapshot    snapshots.Snapshot
	GotFlush    bool
	WaitedFlush bool
	WaitedDone  bool
}

type GridState struct {
	Rows  int
	Cols  int
	Cells [][]snapshots.Cell
}

type State struct {
	Grids         map[int]*GridState
	HLAttrs       map[int]snapshots.HLAttr
	HLGroups      map[string]int
	DefaultColors snapshots.DefaultColors
	GotFlush      int32
}

func resetFlush(state *State, flushCh chan struct{}) {
	atomic.StoreInt32(&state.GotFlush, 0)
	for {
		select {
		case <-flushCh:
		default:
			return
		}
	}
}

func Collect(opts Options) (Result, error) {
	if opts.Scenario == "" {
		return Result{}, errors.New("scenario is required")
	}

	log := func(format string, args ...any) {
		if opts.LogFile == "" {
			return
		}
		msg := fmt.Sprintf(format, args...)
		line := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339Nano), msg)
		f, err := os.OpenFile(opts.LogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
		if err != nil {
			return
		}
		defer f.Close()
		_, _ = f.WriteString(line)
	}

	absScenario, err := filepath.Abs(opts.Scenario)
	if err != nil {
		return Result{}, fmt.Errorf("failed to resolve scenario: %w", err)
	}

	env := os.Environ()
	if opts.DataHome != "" {
		env = append(env, "XDG_DATA_HOME="+opts.DataHome)
	}
	if opts.ConfigHome != "" {
		env = append(env, "XDG_CONFIG_HOME="+opts.ConfigHome)
	}
	if opts.LogFile != "" {
		if err := ensureFile(opts.LogFile); err != nil {
			return Result{}, fmt.Errorf("failed to prepare log file: %w", err)
		}
		env = append(env, "NVIM_LOG_FILE="+opts.LogFile)
	}
	if opts.LogLevel != "" {
		env = append(env, "NVIM_LOG_LEVEL="+opts.LogLevel)
	}

	args := []string{"--embed", "--headless", "-u", "NONE", "-i", "NONE", "-n"}
	copts := []nvim.ChildProcessOption{
		nvim.ChildProcessCommand(defaultString(opts.NvimPath, "nvim")),
		nvim.ChildProcessArgs(args...),
		nvim.ChildProcessEnv(env),
	}
	if opts.WorkDir != "" {
		copts = append(copts, nvim.ChildProcessDir(opts.WorkDir))
	}

	rpcTimeout := defaultInt(opts.RPCTimeoutMS, 2000)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpcTimeout)*time.Millisecond)
	defer cancel()
	copts = append(copts, nvim.ChildProcessContext(ctx))

	v, err := nvim.NewChildProcess(copts...)
	if err != nil {
		return Result{}, fmt.Errorf("failed to start nvim: %w", err)
	}
	closed := false
	closeNvim := func() error {
		if closed {
			return nil
		}
		closed = true
		return v.Close()
	}
	defer closeNvim()

	state := &State{
		Grids:    map[int]*GridState{},
		HLAttrs:  map[int]snapshots.HLAttr{},
		HLGroups: map[string]int{},
	}
	flushCh := make(chan struct{}, 1)
	doneCh := make(chan struct{}, 1)
	if err := v.RegisterHandler("redraw", func(updates ...[]any) {
		for _, update := range updates {
			if len(update) == 0 {
				continue
			}
			name, ok := update[0].(string)
			if !ok {
				continue
			}
			if name == "flush" {
				handleEvent(state, name, nil, flushCh)
				continue
			}
			for i := 1; i < len(update); i++ {
				args, ok := update[i].([]any)
				if !ok {
					continue
				}
				handleEvent(state, name, args, flushCh)
			}
		}
	}); err != nil {
		log("register redraw failed: %v", err)
		if isSessionClosed(err) {
			return Result{}, wrapClosed(err, closeNvim)
		}
		return Result{}, fmt.Errorf("failed to register redraw handler: %w", err)
	}
	if err := v.RegisterHandler("snap_done", func(_ ...any) {
		select {
		case doneCh <- struct{}{}:
		default:
		}
	}); err != nil {
		log("register snap_done failed: %v", err)
		if isSessionClosed(err) {
			return Result{}, wrapClosed(err, closeNvim)
		}
		return Result{}, fmt.Errorf("failed to register done handler: %w", err)
	}

	uiOpts := map[string]any{
		"ext_linegrid":  true,
		"ext_hlstate":   true,
		"ext_multigrid": false,
	}
	if err := v.AttachUI(defaultInt(opts.Width, 80), defaultInt(opts.Height, 24), uiOpts); err != nil {
		log("attach UI failed: %v", err)
		if isSessionClosed(err) {
			return Result{}, wrapClosed(err, closeNvim)
		}
		return Result{}, fmt.Errorf("failed to attach UI: %w", err)
	}

	if len(opts.RTP) > 0 {
		log("set rtp start")
		if err := v.ExecLua(`local paths = ...
for i = #paths, 1, -1 do
  vim.opt.rtp:prepend(paths[i])
end`, nil, opts.RTP); err != nil {
			log("set rtp failed: %v", err)
			if isSessionClosed(err) {
				return Result{}, wrapClosed(err, closeNvim)
			}
			return Result{}, fmt.Errorf("failed to set rtp: %w", err)
		}
		log("set rtp ok")
	}

	channelID := v.ChannelID()
	if opts.WaitDone {
		log("define snap_done helper")
		if err := v.ExecLua(`local chan = ...
_G.snap_done = function()
  vim.rpcnotify(chan, "snap_done")
end`, nil, channelID); err != nil {
			log("define snap_done helper failed: %v", err)
			if isSessionClosed(err) {
				return Result{}, wrapClosed(err, closeNvim)
			}
			return Result{}, fmt.Errorf("failed to set done helper: %w", err)
		}
	} else {
		if err := v.ExecLua(`_G.snap_done = function() end`, nil); err != nil {
			log("define snap_done noop failed: %v", err)
			if isSessionClosed(err) {
				return Result{}, wrapClosed(err, closeNvim)
			}
			return Result{}, fmt.Errorf("failed to set done helper: %w", err)
		}
	}

	log("run scenario start: %s", absScenario)
	if err := v.ExecLua(`local p = ...; dofile(p)`, nil, absScenario); err != nil {
		log("run scenario failed: %v", err)
		if isSessionClosed(err) {
			return Result{}, wrapClosed(err, closeNvim)
		}
		return Result{}, fmt.Errorf("failed to run scenario: %w", err)
	}
	log("run scenario ok")
	if opts.PostWaitMS > 0 {
		log("post wait start: %dms", opts.PostWaitMS)
		if err := v.ExecLua(`vim.wait(...)`, nil, opts.PostWaitMS); err != nil {
			log("post wait failed: %v", err)
			if isSessionClosed(err) {
				return Result{}, wrapClosed(err, closeNvim)
			}
			return Result{}, fmt.Errorf("failed to wait after scenario: %w", err)
		}
		log("post wait ok")
	}

	result := Result{}
	if opts.WaitDone {
		doneWait := defaultInt(opts.DoneTimeoutMS, 5000)
		log("wait done start: %dms", doneWait)
		select {
		case <-doneCh:
			result.WaitedDone = true
			log("wait done ok")
		case <-time.After(time.Duration(doneWait) * time.Millisecond):
			result.WaitedDone = false
			log("wait done timeout")
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log("wait done rpc timeout")
				return Result{}, errors.New("rpc timeout while waiting done (headless input wait? prefer vim.api.nvim_cmd)")
			}
			log("wait done ctx error: %v", ctx.Err())
			return Result{}, ctx.Err()
		}
	}

	resetFlush(state, flushCh)
	waitMS := defaultInt(opts.WaitMS, 200)
	retries := defaultInt(opts.FlushRetry, 3)
	if retries < 1 {
		retries = 1
	}
	for attempt := 1; attempt <= retries; attempt++ {
		log("redraw attempt %d/%d", attempt, retries)
		if err := v.Command("redraw"); err != nil {
			log("redraw failed: %v", err)
			if isSessionClosed(err) {
				return Result{}, wrapClosed(err, closeNvim)
			}
			return Result{}, fmt.Errorf("failed to redraw: %w", err)
		}
		if err := v.Command("redrawstatus"); err != nil {
			log("redrawstatus failed: %v", err)
			if isSessionClosed(err) {
				return Result{}, wrapClosed(err, closeNvim)
			}
			return Result{}, fmt.Errorf("failed to redrawstatus: %w", err)
		}
		log("wait flush start: %dms", waitMS)
		select {
		case <-flushCh:
			result.GotFlush = true
			result.WaitedFlush = true
			log("wait flush ok")
			attempt = retries
			break
		case <-time.After(time.Duration(waitMS) * time.Millisecond):
			result.GotFlush = false
			log("wait flush timeout")
			resetFlush(state, flushCh)
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				log("wait flush rpc timeout")
				return Result{}, errors.New("rpc timeout (headless input wait? prefer vim.api.nvim_cmd)")
			}
			log("wait flush ctx error: %v", ctx.Err())
			return Result{}, ctx.Err()
		}
	}

	snapshot := snapshotFromState(state, defaultInt(opts.Width, 80), defaultInt(opts.Height, 24))
	result.Snapshot = snapshot
	_ = v.Command("qa!")

	return result, nil
}

func isSessionClosed(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, rpc.ErrClosed) {
		return true
	}
	return strings.Contains(err.Error(), "msgpack/rpc: session closed")
}

func wrapClosed(err error, closeNvim func() error) error {
	closeErr := closeNvim()
	if closeErr == nil {
		return err
	}
	var exitErr *exec.ExitError
	if errors.As(closeErr, &exitErr) {
		return fmt.Errorf("%w (nvim exited: %s; headless input wait? prefer vim.api.nvim_cmd)", err, exitErr.ProcessState.String())
	}
	return fmt.Errorf("%w (close error: %v)", err, closeErr)
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

func allocRow(cols int) []snapshots.Cell {
	row := make([]snapshots.Cell, cols)
	for i := range cols {
		row[i] = snapshots.Cell{Text: " ", HlID: 0}
	}
	return row
}

func ensureGrid(grid *GridState, rows, cols int) {
	grid.Rows = rows
	grid.Cols = cols
	if grid.Cells == nil {
		grid.Cells = make([][]snapshots.Cell, 0, rows)
	}
	for r := range rows {
		if r >= len(grid.Cells) || grid.Cells[r] == nil {
			grid.Cells = append(grid.Cells, allocRow(cols))
			continue
		}
		row := grid.Cells[r]
		if len(row) < cols {
			row = append(row, make([]snapshots.Cell, cols-len(row))...)
			for c := len(row) - (cols - len(row)); c < cols; c++ {
				row[c] = snapshots.Cell{Text: " ", HlID: 0}
			}
			grid.Cells[r] = row
		} else if len(row) > cols {
			grid.Cells[r] = row[:cols]
		}
	}
	if len(grid.Cells) > rows {
		grid.Cells = grid.Cells[:rows]
	}
}

func clearGrid(grid *GridState) {
	if grid.Rows == 0 || grid.Cols == 0 {
		return
	}
	for r := range grid.Rows {
		row := grid.Cells[r]
		if row == nil || len(row) != grid.Cols {
			row = allocRow(grid.Cols)
			grid.Cells[r] = row
			continue
		}
		for c := range grid.Cols {
			row[c] = snapshots.Cell{Text: " ", HlID: 0}
		}
	}
}

func copyCell(cell snapshots.Cell) snapshots.Cell {
	return snapshots.Cell{Text: cell.Text, HlID: cell.HlID}
}

func scrollGrid(grid *GridState, top, bot, left, right, rows int) {
	if rows == 0 || grid.Rows == 0 || grid.Cols == 0 {
		return
	}
	topR := top + 1
	botR := bot
	leftC := left + 1
	rightC := right
	if rows > 0 {
		for r := topR; r <= botR-rows; r++ {
			src := grid.Cells[r+rows-1]
			dst := grid.Cells[r-1]
			for c := leftC; c <= rightC; c++ {
				dst[c-1] = copyCell(src[c-1])
			}
		}
		for r := botR - rows + 1; r <= botR; r++ {
			row := grid.Cells[r-1]
			for c := leftC; c <= rightC; c++ {
				row[c-1] = snapshots.Cell{Text: " ", HlID: 0}
			}
		}
		return
	}
	offset := -rows
	for r := botR; r >= topR+offset; r-- {
		src := grid.Cells[r-offset-1]
		dst := grid.Cells[r-1]
		for c := leftC; c <= rightC; c++ {
			dst[c-1] = copyCell(src[c-1])
		}
	}
	for r := topR; r <= topR+offset-1; r++ {
		row := grid.Cells[r-1]
		for c := leftC; c <= rightC; c++ {
			row[c-1] = snapshots.Cell{Text: " ", HlID: 0}
		}
	}
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case uint64:
		return int(v), true
	default:
		return 0, false
	}
}

func toString(value any) (string, bool) {
	v, ok := value.(string)
	return v, ok
}

func toMapStringAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	switch m := value.(type) {
	case map[string]any:
		return m
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, v := range m {
			ks, ok := k.(string)
			if !ok {
				continue
			}
			out[ks] = v
		}
		return out
	default:
		return nil
	}
}

func handleEvent(state *State, name string, args []any, flushCh chan<- struct{}) {
	switch name {
	case "grid_resize":
		if len(args) < 3 {
			return
		}
		gridID, ok1 := toInt(args[0])
		width, ok2 := toInt(args[1])
		height, ok3 := toInt(args[2])
		if !ok1 || !ok2 || !ok3 {
			return
		}
		grid := state.Grids[gridID]
		if grid == nil {
			grid = &GridState{}
			state.Grids[gridID] = grid
		}
		ensureGrid(grid, height, width)
	case "grid_clear":
		if len(args) < 1 {
			return
		}
		gridID, ok := toInt(args[0])
		if !ok {
			return
		}
		grid := state.Grids[gridID]
		if grid == nil {
			return
		}
		clearGrid(grid)
	case "grid_destroy":
		if len(args) < 1 {
			return
		}
		gridID, ok := toInt(args[0])
		if !ok {
			return
		}
		delete(state.Grids, gridID)
	case "grid_line":
		if len(args) < 4 {
			return
		}
		gridID, ok1 := toInt(args[0])
		row, ok2 := toInt(args[1])
		colStart, ok3 := toInt(args[2])
		cells, ok4 := args[3].([]any)
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return
		}
		grid := state.Grids[gridID]
		if grid == nil {
			return
		}
		if row < 0 || row >= grid.Rows {
			return
		}
		currentHL := 0
		col := colStart
		rowCells := grid.Cells[row]
		for _, cellRaw := range cells {
			cellItems, ok := cellRaw.([]any)
			if !ok || len(cellItems) == 0 {
				continue
			}
			text, ok := toString(cellItems[0])
			if !ok {
				text = fmt.Sprint(cellItems[0])
			}
			if len(cellItems) > 1 && cellItems[1] != nil {
				if hl, ok := toInt(cellItems[1]); ok {
					currentHL = hl
				}
			}
			repeat := 1
			if len(cellItems) > 2 && cellItems[2] != nil {
				if rep, ok := toInt(cellItems[2]); ok {
					repeat = rep
				}
			}
			for range repeat {
				if col >= 0 && col < grid.Cols {
					rowCells[col] = snapshots.Cell{Text: text, HlID: currentHL}
				}
				col++
			}
		}
	case "grid_scroll":
		if len(args) < 6 {
			return
		}
		gridID, ok1 := toInt(args[0])
		top, ok2 := toInt(args[1])
		bot, ok3 := toInt(args[2])
		left, ok4 := toInt(args[3])
		right, ok5 := toInt(args[4])
		rows, ok6 := toInt(args[5])
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 {
			return
		}
		grid := state.Grids[gridID]
		if grid == nil {
			return
		}
		scrollGrid(grid, top, bot, left, right, rows)
	case "default_colors_set":
		if len(args) < 5 {
			return
		}
		if val, ok := toInt(args[0]); ok {
			state.DefaultColors.RGBFg = &val
		}
		if val, ok := toInt(args[1]); ok {
			state.DefaultColors.RGBBg = &val
		}
		if val, ok := toInt(args[2]); ok {
			state.DefaultColors.RGBSp = &val
		}
		if val, ok := toInt(args[3]); ok {
			state.DefaultColors.CtermFg = &val
		}
		if val, ok := toInt(args[4]); ok {
			state.DefaultColors.CtermBg = &val
		}
	case "hl_attr_define":
		if len(args) < 4 {
			return
		}
		id, ok := toInt(args[0])
		if !ok {
			return
		}
		state.HLAttrs[id] = snapshots.HLAttr{
			ID:        id,
			RGBAttr:   toMapStringAny(args[1]),
			CtermAttr: toMapStringAny(args[2]),
			Info:      toMapStringAny(args[3]),
		}
	case "hl_group_set":
		if len(args) < 2 {
			return
		}
		name, ok := toString(args[0])
		if !ok {
			return
		}
		hlID, ok := toInt(args[1])
		if !ok {
			return
		}
		state.HLGroups[name] = hlID
	case "flush":
		if atomic.CompareAndSwapInt32(&state.GotFlush, 0, 1) {
			select {
			case flushCh <- struct{}{}:
			default:
			}
		}
	}
}

func snapshotFromState(state *State, width, height int) snapshots.Snapshot {
	grids := make([]snapshots.Grid, 0, len(state.Grids))
	ids := make([]int, 0, len(state.Grids))
	for id := range state.Grids {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		g := state.Grids[id]
		if g == nil {
			continue
		}
		cells := make([][]snapshots.Cell, len(g.Cells))
		for i, row := range g.Cells {
			if row == nil {
				cells[i] = nil
				continue
			}
			rowCopy := make([]snapshots.Cell, len(row))
			copy(rowCopy, row)
			cells[i] = rowCopy
		}
		grids = append(grids, snapshots.Grid{ID: id, Rows: g.Rows, Cols: g.Cols, Cells: cells})
	}

	hlAttrs := make([]snapshots.HLAttr, 0, len(state.HLAttrs))
	attrIDs := make([]int, 0, len(state.HLAttrs))
	for id := range state.HLAttrs {
		attrIDs = append(attrIDs, id)
	}
	sort.Ints(attrIDs)
	for _, id := range attrIDs {
		hlAttrs = append(hlAttrs, state.HLAttrs[id])
	}

	hlGroups := make([]snapshots.HLGroup, 0, len(state.HLGroups))
	groupNames := make([]string, 0, len(state.HLGroups))
	for name := range state.HLGroups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)
	for _, name := range groupNames {
		hlGroups = append(hlGroups, snapshots.HLGroup{Name: name, HlID: state.HLGroups[name]})
	}

	return snapshots.Snapshot{
		Size: map[string]int{
			"columns": width,
			"lines":   height,
		},
		DefaultColors: state.DefaultColors,
		HLAttrs:       hlAttrs,
		HLGroups:      hlGroups,
		Grids:         grids,
	}
}

func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
