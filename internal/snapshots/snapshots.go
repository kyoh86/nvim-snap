// Package snapshots provides snapshot structures and helpers.
package snapshots

import (
	"fmt"
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
	ID        int            `json:"id"`
	RGBAttr   map[string]any `json:"rgb_attr"`
	CtermAttr map[string]any `json:"cterm_attr"`
	Info      any            `json:"info"`
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

type style struct {
	fg, bg    *int
	bold      bool
	italic    bool
	underline bool
	strike    bool
	reverse   bool
}

func buildAttrMap(snapshot Snapshot) (map[int]map[string]any, *int, *int) {
	attrMap := map[int]map[string]any{}
	for _, attr := range snapshot.HLAttrs {
		if attr.RGBAttr != nil {
			attrMap[attr.ID] = attr.RGBAttr
		} else {
			attrMap[attr.ID] = map[string]any{}
		}
	}
	return attrMap, snapshot.DefaultColors.RGBFg, snapshot.DefaultColors.RGBBg
}

func styleFrom(attrMap map[int]map[string]any, defaultFg, defaultBg *int, hlID int) style {
	attr := attrMap[hlID]
	if attr == nil {
		attr = map[string]any{}
	}
	fg := toColor(attr["foreground"], defaultFg)
	bg := toColor(attr["background"], defaultBg)
	reverse := toBool(attr["reverse"])
	if reverse {
		fg, bg = bg, fg
	}
	return style{
		fg:        fg,
		bg:        bg,
		bold:      toBool(attr["bold"]),
		italic:    toBool(attr["italic"]),
		underline: toBool(attr["underline"]) || toBool(attr["undercurl"]) || toBool(attr["underdouble"]) || toBool(attr["underdotted"]) || toBool(attr["underdashed"]),
		strike:    toBool(attr["strikethrough"]),
		reverse:   reverse,
	}
}

func styleEqual(a, b style) bool {
	return a.fg == b.fg &&
		a.bg == b.bg &&
		a.bold == b.bold &&
		a.italic == b.italic &&
		a.underline == b.underline &&
		a.strike == b.strike &&
		a.reverse == b.reverse
}

func toColor(value any, fallback *int) *int {
	if value == nil {
		return fallback
	}
	switch v := value.(type) {
	case int:
		return &v
	case int64:
		tmp := int(v)
		return &tmp
	case float64:
		tmp := int(v)
		return &tmp
	default:
		return fallback
	}
}

func toBool(value any) bool {
	if value == nil {
		return false
	}
	if v, ok := value.(bool); ok {
		return v
	}
	return false
}

func rgbToANSI(color *int, isBG bool) string {
	if color == nil || *color < 0 {
		return ""
	}
	r := (*color / 65536) % 256
	g := (*color / 256) % 256
	b := *color % 256
	if isBG {
		return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func styleToANSI(s style) string {
	codes := []string{"\x1b[0m"}
	if s.bold {
		codes = append(codes, "\x1b[1m")
	}
	if s.italic {
		codes = append(codes, "\x1b[3m")
	}
	if s.underline {
		codes = append(codes, "\x1b[4m")
	}
	if s.strike {
		codes = append(codes, "\x1b[9m")
	}
	if fg := rgbToANSI(s.fg, false); fg != "" {
		codes = append(codes, fg)
	}
	if bg := rgbToANSI(s.bg, true); bg != "" {
		codes = append(codes, bg)
	}
	return strings.Join(codes, "")
}

// RenderANSI renders the snapshot as ANSI colored text.
func RenderANSI(s Snapshot) string {
	grid := GridForRender(s)
	if grid == nil {
		return ""
	}
	attrMap, defaultFg, defaultBg := buildAttrMap(s)

	lines := make([]string, 0, grid.Rows)
	for r := 0; r < grid.Rows; r++ {
		row := grid.Cells[r]
		if row == nil {
			row = make([]Cell, grid.Cols)
		}
		var b strings.Builder
		current := style{}
		for c := 0; c < grid.Cols; c++ {
			cell := Cell{Text: " ", HlID: 0}
			if c < len(row) {
				cell = row[c]
				if cell.Text == "" {
					cell.Text = " "
				}
			}
			st := styleFrom(attrMap, defaultFg, defaultBg, cell.HlID)
			if !styleEqual(st, current) {
				b.WriteString(styleToANSI(st))
				current = st
			}
			b.WriteString(cell.Text)
		}
		b.WriteString("\x1b[0m")
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func toHexColor(value *int) string {
	if value == nil || *value < 0 {
		return ""
	}
	return fmt.Sprintf("#%06x", *value)
}

func styleToCSS(s style) string {
	parts := []string{}
	if s.fg != nil {
		parts = append(parts, "color:"+toHexColor(s.fg))
	}
	if s.bg != nil {
		parts = append(parts, "background-color:"+toHexColor(s.bg))
	}
	if s.bold {
		parts = append(parts, "font-weight:700")
	}
	if s.italic {
		parts = append(parts, "font-style:italic")
	}
	decorations := []string{}
	if s.underline {
		decorations = append(decorations, "underline")
	}
	if s.strike {
		decorations = append(decorations, "line-through")
	}
	if len(decorations) > 0 {
		parts = append(parts, "text-decoration:"+strings.Join(decorations, " "))
	}
	return strings.Join(parts, ";")
}

func escapeHTML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&#39;",
	)
	return replacer.Replace(text)
}

type htmlFragment struct {
	bg   string
	fg   string
	html string
}

func renderHTMLFragment(s Snapshot) htmlFragment {
	grid := GridForRender(s)
	if grid == nil {
		return htmlFragment{bg: "#000000", fg: "#ffffff", html: ""}
	}
	attrMap, defaultFg, defaultBg := buildAttrMap(s)

	lines := make([]string, 0, grid.Rows)
	for r := 0; r < grid.Rows; r++ {
		row := grid.Cells[r]
		if row == nil {
			row = make([]Cell, grid.Cols)
		}
		current := style{}
		var line strings.Builder
		var chunk strings.Builder
		flush := func() {
			if chunk.Len() == 0 {
				return
			}
			css := styleToCSS(current)
			if css != "" {
				line.WriteString(`<span style="`)
				line.WriteString(css)
				line.WriteString(`">`)
				line.WriteString(escapeHTML(chunk.String()))
				line.WriteString(`</span>`)
			} else {
				line.WriteString(escapeHTML(chunk.String()))
			}
			chunk.Reset()
		}
		for c := 0; c < grid.Cols; c++ {
			cell := Cell{Text: " ", HlID: 0}
			if c < len(row) {
				cell = row[c]
				if cell.Text == "" {
					cell.Text = " "
				}
			}
			st := styleFrom(attrMap, defaultFg, defaultBg, cell.HlID)
			st.reverse = false
			if !styleEqual(st, current) {
				flush()
				current = st
			}
			chunk.WriteString(cell.Text)
		}
		flush()
		lines = append(lines, line.String())
	}

	bg := toHexColor(defaultBg)
	if bg == "" {
		bg = "#000000"
	}
	fg := toHexColor(defaultFg)
	if fg == "" {
		fg = "#ffffff"
	}
	return htmlFragment{
		bg:   bg,
		fg:   fg,
		html: "<pre>" + strings.Join(lines, "\n") + "</pre>",
	}
}

// RenderHTML renders a snapshot as standalone HTML.
func RenderHTML(s Snapshot) string {
	fragment := renderHTMLFragment(s)
	return strings.Join([]string{
		"<!doctype html>",
		"<html>",
		"<head>",
		`  <meta charset="utf-8" />`,
		"  <title>Neovim UI Snapshot</title>",
		"  <style>",
		"    body {",
		"      margin: 0;",
		"      background: " + fragment.bg + ";",
		"      color: " + fragment.fg + ";",
		"      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;",
		"      line-height: 1.2;",
		"    }",
		"    pre {",
		"      margin: 0;",
		"      padding: 16px;",
		"      white-space: pre;",
		"    }",
		"  </style>",
		"</head>",
		"<body>",
		"  " + fragment.html,
		"</body>",
		"</html>",
	}, "\n")
}
