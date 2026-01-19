package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

type Cell struct {
	Text string `json:"text"`
	HlID int    `json:"hl_id"`
}

type Grid struct {
	ID    int      `json:"id"`
	Rows  int      `json:"rows"`
	Cols  int      `json:"cols"`
	Cells [][]Cell `json:"cells"`
}

type HLAttr struct {
	ID        int                    `json:"id"`
	RGBAttr   map[string]interface{} `json:"rgb_attr"`
	CtermAttr map[string]interface{} `json:"cterm_attr"`
	Info      map[string]interface{} `json:"info"`
}

type HLGroup struct {
	Name string `json:"name"`
	HlID int    `json:"hl_id"`
}

type DefaultColors struct {
	RGBFg   *int `json:"rgb_fg,omitempty"`
	RGBBg   *int `json:"rgb_bg,omitempty"`
	RGBSp   *int `json:"rgb_sp,omitempty"`
	CtermFg *int `json:"cterm_fg,omitempty"`
	CtermBg *int `json:"cterm_bg,omitempty"`
}

type Snapshot struct {
	Size          map[string]int `json:"size"`
	DefaultColors DefaultColors  `json:"default_colors"`
	HLAttrs       []HLAttr       `json:"hl_attrs"`
	HLGroups      []HLGroup      `json:"hl_groups"`
	Grids         []Grid         `json:"grids"`
}

type GridState struct {
	Rows  int
	Cols  int
	Cells [][]Cell
}

type State struct {
	Grids         map[int]*GridState
	HLAttrs       map[int]HLAttr
	HLGroups      map[string]int
	DefaultColors DefaultColors
	GotFlush      int32
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

func allocRow(cols int) []Cell {
	row := make([]Cell, cols)
	for i := 0; i < cols; i++ {
		row[i] = Cell{Text: " ", HlID: 0}
	}
	return row
}

func ensureGrid(grid *GridState, rows, cols int) {
	grid.Rows = rows
	grid.Cols = cols
	if grid.Cells == nil {
		grid.Cells = make([][]Cell, 0, rows)
	}
	for r := 0; r < rows; r++ {
		if r >= len(grid.Cells) || grid.Cells[r] == nil {
			grid.Cells = append(grid.Cells, allocRow(cols))
			continue
		}
		row := grid.Cells[r]
		if len(row) < cols {
			row = append(row, make([]Cell, cols-len(row))...)
			for c := len(row) - (cols - len(row)); c < cols; c++ {
				row[c] = Cell{Text: " ", HlID: 0}
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
	for r := 0; r < grid.Rows; r++ {
		row := grid.Cells[r]
		if row == nil || len(row) != grid.Cols {
			row = allocRow(grid.Cols)
			grid.Cells[r] = row
			continue
		}
		for c := 0; c < grid.Cols; c++ {
			row[c] = Cell{Text: " ", HlID: 0}
		}
	}
}

func copyCell(cell Cell) Cell {
	return Cell{Text: cell.Text, HlID: cell.HlID}
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
				row[c-1] = Cell{Text: " ", HlID: 0}
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
			row[c-1] = Cell{Text: " ", HlID: 0}
		}
	}
}

func toInt(value interface{}) (int, bool) {
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

func toString(value interface{}) (string, bool) {
	v, ok := value.(string)
	return v, ok
}

func toMapStringInterface(value interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	switch m := value.(type) {
	case map[string]interface{}:
		return m
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(m))
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

func handleEvent(state *State, name string, args []interface{}, flushCh chan<- struct{}) {
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
		cells, ok4 := args[3].([]interface{})
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
			cellItems, ok := cellRaw.([]interface{})
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
			for i := 0; i < repeat; i++ {
				if col >= 0 && col < grid.Cols {
					rowCells[col] = Cell{Text: text, HlID: currentHL}
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
		state.HLAttrs[id] = HLAttr{
			ID:        id,
			RGBAttr:   toMapStringInterface(args[1]),
			CtermAttr: toMapStringInterface(args[2]),
			Info:      toMapStringInterface(args[3]),
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

func snapshotFromState(state *State, width, height int) Snapshot {
	grids := make([]Grid, 0, len(state.Grids))
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
		cells := make([][]Cell, len(g.Cells))
		for i, row := range g.Cells {
			if row == nil {
				cells[i] = nil
				continue
			}
			rowCopy := make([]Cell, len(row))
			copy(rowCopy, row)
			cells[i] = rowCopy
		}
		grids = append(grids, Grid{ID: id, Rows: g.Rows, Cols: g.Cols, Cells: cells})
	}

	hlAttrs := make([]HLAttr, 0, len(state.HLAttrs))
	attrIDs := make([]int, 0, len(state.HLAttrs))
	for id := range state.HLAttrs {
		attrIDs = append(attrIDs, id)
	}
	sort.Ints(attrIDs)
	for _, id := range attrIDs {
		hlAttrs = append(hlAttrs, state.HLAttrs[id])
	}

	hlGroups := make([]HLGroup, 0, len(state.HLGroups))
	groupNames := make([]string, 0, len(state.HLGroups))
	for name := range state.HLGroups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)
	for _, name := range groupNames {
		hlGroups = append(hlGroups, HLGroup{Name: name, HlID: state.HLGroups[name]})
	}

	return Snapshot{
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

func writeOutput(path string, data []byte) error {
	if path == "" || path == "-" {
		_, err := os.Stdout.Write(data)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write([]byte("\n"))
		return err
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
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

	state := &State{
		Grids:    map[int]*GridState{},
		HLAttrs:  map[int]HLAttr{},
		HLGroups: map[string]int{},
	}
	flushCh := make(chan struct{}, 1)
	if err := v.RegisterHandler("redraw", func(updates ...[]interface{}) {
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
				args, ok := update[i].([]interface{})
				if !ok {
					continue
				}
				handleEvent(state, name, args, flushCh)
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

	snapshot := snapshotFromState(state, width, height)
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode snapshot: %v\n", err)
		os.Exit(1)
	}
	if err := writeOutput(outPath, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}

	_ = v.Command("qa!")
}
