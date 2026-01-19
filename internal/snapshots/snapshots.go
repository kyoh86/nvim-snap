package snapshots

import (
	"sort"
	"strings"
)

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

func Normalize(s Snapshot) Snapshot {
	out := s
	out.Grids = append([]Grid(nil), s.Grids...)
	sort.Slice(out.Grids, func(i, j int) bool {
		return out.Grids[i].ID < out.Grids[j].ID
	})
	out.HLAttrs = append([]HLAttr(nil), s.HLAttrs...)
	sort.Slice(out.HLAttrs, func(i, j int) bool {
		return out.HLAttrs[i].ID < out.HLAttrs[j].ID
	})
	out.HLGroups = append([]HLGroup(nil), s.HLGroups...)
	sort.Slice(out.HLGroups, func(i, j int) bool {
		return out.HLGroups[i].Name < out.HLGroups[j].Name
	})
	return out
}

func RenderText(s Snapshot) string {
	grid := GridForRender(s)
	if grid == nil {
		return ""
	}
	lines := make([]string, 0, grid.Rows)
	for r := 0; r < grid.Rows; r++ {
		row := grid.Cells[r]
		if row == nil {
			row = make([]Cell, grid.Cols)
		}
		var b strings.Builder
		b.Grow(grid.Cols)
		for c := 0; c < grid.Cols; c++ {
			text := " "
			if c < len(row) && row[c].Text != "" {
				text = row[c].Text
			}
			b.WriteString(text)
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func TextLines(s Snapshot) []string {
	grid := GridForRender(s)
	if grid == nil {
		return []string{}
	}
	lines := make([]string, 0, grid.Rows)
	for r := 0; r < grid.Rows; r++ {
		row := grid.Cells[r]
		if row == nil {
			row = make([]Cell, grid.Cols)
		}
		var b strings.Builder
		b.Grow(grid.Cols)
		for c := 0; c < grid.Cols; c++ {
			text := " "
			if c < len(row) && row[c].Text != "" {
				text = row[c].Text
			}
			b.WriteString(text)
		}
		lines = append(lines, b.String())
	}
	return lines
}

func GridForRender(s Snapshot) *Grid {
	for i := range s.Grids {
		if s.Grids[i].ID == 1 {
			return &s.Grids[i]
		}
	}
	if len(s.Grids) > 0 {
		return &s.Grids[0]
	}
	return nil
}
