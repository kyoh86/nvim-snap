// Package htmldiff renders HTML diffs between snapshots.
package htmldiff

import (
	"fmt"
	"html"
	"strings"

	"github.com/kyoh86/nvim-snap/internal/snapshots"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type alignedPair struct {
	expected int
	actual   int
	kind     string
}

type rendered struct {
	bg   string
	fg   string
	html string
}

func RenderHTML(expected, actual snapshots.Snapshot, defaultView, expectedLabel, actualLabel string) string {
	if expectedLabel == "" {
		expectedLabel = "expected"
	}
	if actualLabel == "" {
		actualLabel = "actual"
	}
	expectedLines := snapshots.TextLines(expected)
	actualLines := snapshots.TextLines(actual)
	pairs, expectedLineKinds, actualLineKinds, expectedCells, actualCells := alignLines(expectedLines, actualLines)

	expectedPlain := renderAligned(expected, pairs, nil, nil, "", true)
	actualPlain := renderAligned(actual, pairs, nil, nil, "", true)
	expectedAligned := renderAligned(expected, pairs, expectedLineKinds, expectedCells, "removed", false)
	actualAligned := renderAligned(actual, pairs, actualLineKinds, actualCells, "added", false)

	expectedText := snapshots.RenderText(expected)
	actualText := snapshots.RenderText(actual)
	unified := unifiedHTML(expectedText, actualText, expectedLabel, actualLabel)

	view := defaultView
	if view == "" {
		view = "overlay"
	}

	return wrapHTML(unified, expectedPlain, actualPlain, expectedAligned, actualAligned, view, expectedLabel, actualLabel)
}

func unifiedHTML(expectedText, actualText, expectedLabel, actualLabel string) string {
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expectedText),
		B:        difflib.SplitLines(actualText),
		FromFile: expectedLabel,
		ToFile:   actualLabel,
		Context:  3,
	}
	raw, _ := difflib.GetUnifiedDiffString(d)
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	var out strings.Builder
	for _, line := range lines {
		class := "line"
		if strings.HasPrefix(line, "@@") {
			class = "line hunk"
		} else if strings.HasPrefix(line, "+") {
			class = "line added"
		} else if strings.HasPrefix(line, "-") {
			class = "line removed"
		}
		out.WriteString("<div class=\"")
		out.WriteString(class)
		out.WriteString("\">")
		out.WriteString(html.EscapeString(line))
		out.WriteString("</div>")
	}
	return out.String()
}

func alignLines(expectedLines, actualLines []string) ([]alignedPair, []string, []string, map[int]map[int]bool, map[int]map[int]bool) {
	matcher := difflib.NewMatcher(expectedLines, actualLines)
	opcodes := matcher.GetOpCodes()
	pairs := []alignedPair{}
	expectedLineKinds := []string{}
	actualLineKinds := []string{}
	expectedCells := map[int]map[int]bool{}
	actualCells := map[int]map[int]bool{}

	for _, op := range opcodes {
		eCount := op.I2 - op.I1
		aCount := op.J2 - op.J1
		switch op.Tag {
		case 'e':
			for i := range eCount {
				pairs = append(pairs, alignedPair{expected: op.I1 + i, actual: op.J1 + i})
				expectedLineKinds = append(expectedLineKinds, "")
				actualLineKinds = append(actualLineKinds, "")
			}
		case 'd':
			for i := range eCount {
				pairs = append(pairs, alignedPair{expected: op.I1 + i, actual: -1, kind: "removed"})
				expectedLineKinds = append(expectedLineKinds, "removed")
				actualLineKinds = append(actualLineKinds, "")
			}
		case 'i':
			for i := range aCount {
				pairs = append(pairs, alignedPair{expected: -1, actual: op.J1 + i, kind: "added"})
				expectedLineKinds = append(expectedLineKinds, "")
				actualLineKinds = append(actualLineKinds, "added")
			}
		case 'r':
			max := eCount
			if aCount > max {
				max = aCount
			}
			for i := range max {
				eIdx := -1
				aIdx := -1
				if op.I1+i < op.I2 {
					eIdx = op.I1 + i
				}
				if op.J1+i < op.J2 {
					aIdx = op.J1 + i
				}
				kind := "changed"
				if eIdx == -1 {
					kind = "added"
				} else if aIdx == -1 {
					kind = "removed"
				}
				pairs = append(pairs, alignedPair{expected: eIdx, actual: aIdx, kind: kind})
				if eIdx >= 0 && aIdx >= 0 {
					expectedLineKinds = append(expectedLineKinds, "changed")
					actualLineKinds = append(actualLineKinds, "changed")
					markCellDiffs(expectedCells, actualCells, eIdx, aIdx, expectedLines[eIdx], actualLines[aIdx])
				} else if eIdx >= 0 {
					expectedLineKinds = append(expectedLineKinds, "removed")
					actualLineKinds = append(actualLineKinds, "")
				} else {
					expectedLineKinds = append(expectedLineKinds, "")
					actualLineKinds = append(actualLineKinds, "added")
				}
			}
		}
	}

	return pairs, expectedLineKinds, actualLineKinds, expectedCells, actualCells
}

func markCellDiffs(expectedCells, actualCells map[int]map[int]bool, eIdx, aIdx int, expected, actual string) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expected, actual, false)
	ePos := 0
	aPos := 0
	for _, d := range diffs {
		length := len([]rune(d.Text))
		switch d.Type {
		case diffmatchpatch.DiffDelete:
			markColumns(expectedCells, eIdx, ePos, length)
			ePos += length
		case diffmatchpatch.DiffInsert:
			markColumns(actualCells, aIdx, aPos, length)
			aPos += length
		default:
			ePos += length
			aPos += length
		}
	}
}

func markColumns(target map[int]map[int]bool, row, start, length int) {
	if row < 0 || length <= 0 {
		return
	}
	rowMap := target[row]
	if rowMap == nil {
		rowMap = map[int]bool{}
		target[row] = rowMap
	}
	for i := range length {
		rowMap[start+i] = true
	}
}

func renderAligned(snapshot snapshots.Snapshot, pairs []alignedPair, lineKinds []string, cellDiff map[int]map[int]bool, diffKind string, withStyle bool) rendered {
	grid := snapshots.GridForRender(snapshot)
	if grid == nil {
		return rendered{bg: "#000000", fg: "#ffffff", html: ""}
	}
	attrMap, defaultFg, defaultBg := buildAttrMap(snapshot)
	bg := toHexColor(defaultBg)
	if bg == "" {
		bg = "#000000"
	}
	fg := toHexColor(defaultFg)
	if fg == "" {
		fg = "#ffffff"
	}

	var out strings.Builder
	for idx, pair := range pairs {
		lineClass := "line"
		kind := ""
		if lineKinds != nil && idx < len(lineKinds) {
			kind = lineKinds[idx]
		}
		if kind != "" {
			lineClass = lineClass + " diff " + kind
		}
		out.WriteString("<div class=\"")
		out.WriteString(lineClass)
		out.WriteString("\">")
		rowIdx := pair.expected
		if diffKind == "added" {
			rowIdx = pair.actual
		}
		for c := 0; c < grid.Cols; c++ {
			cell := snapshots.Cell{Text: " ", HlID: 0}
			if rowIdx >= 0 && rowIdx < len(grid.Cells) {
				row := grid.Cells[rowIdx]
				if c < len(row) {
					cell = row[c]
					if cell.Text == "" {
						cell.Text = " "
					}
				}
			}
			style := styleFrom(attrMap, defaultFg, defaultBg, cell.HlID)
			css := styleToCSS(style)
			cellClass := "cell"
			if rowIdx >= 0 && cellDiff != nil {
				if rowCells, ok := cellDiff[rowIdx]; ok && rowCells[c] {
					cellClass = cellClass + " diff " + diffKind
				}
			}
			out.WriteString("<span class=\"")
			out.WriteString(cellClass)
			out.WriteString("\"")
			if withStyle && css != "" {
				out.WriteString(" style=\"")
				out.WriteString(css)
				out.WriteString("\"")
			}
			out.WriteString(">")
			out.WriteString(html.EscapeString(cell.Text))
			out.WriteString("</span>")
		}
		out.WriteString("</div>")
	}

	return rendered{bg: bg, fg: fg, html: out.String()}
}

func buildAttrMap(snapshot snapshots.Snapshot) (map[int]map[string]any, *int, *int) {
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

type style struct {
	fg, bg    *int
	bold      bool
	italic    bool
	underline bool
	strike    bool
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
	}
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

func toHexColor(value *int) string {
	if value == nil {
		return ""
	}
	if *value < 0 {
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

func wrapHTML(unified string, expectedPlain, actualPlain, expectedAligned, actualAligned rendered, defaultView, expectedLabel, actualLabel string) string {
	return fmt.Sprintf(`<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>nvim-snap diff</title>
  <style>
    :root {
      --bg: #f4f5f7;
      --panel: #ffffff;
      --border: #d8dfe6;
      --shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
      --diff-add: #dff6e7;
      --diff-add-strong: #2fa24f;
      --diff-del: #fde2e1;
      --diff-del-strong: #d45757;
      --diff-change: #fff2cc;
      --diff-add-soft: rgba(47, 162, 79, 0.18);
      --diff-del-soft: rgba(212, 87, 87, 0.18);
      --diff-change-soft: rgba(245, 201, 74, 0.2);
    }
    body {
      margin: 0;
      padding: 24px;
      background: var(--bg);
      color: #1f2933;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    }
    .controls {
      display: inline-flex;
      align-items: center;
      gap: 12px;
      background: var(--panel);
      padding: 10px 14px;
      border-radius: 12px;
      border: 1px solid var(--border);
      box-shadow: var(--shadow);
    }
    .controls .label {
      font-weight: 700;
      text-transform: uppercase;
      font-size: 0.9rem;
      letter-spacing: 0.1em;
      padding: 4px 10px;
      border-radius: 999px;
      border: 1px solid var(--border);
      background: #fff;
    }
    .controls button {
      border: none;
      background: none;
      font: inherit;
      padding: 6px 10px;
      border-radius: 8px;
      cursor: pointer;
    }
    .controls button.active {
      background: #2fa24f;
      color: white;
      font-weight: 700;
    }
    .panel {
      background: var(--panel);
      border-radius: 10px;
      border: 1px solid var(--border);
      box-shadow: var(--shadow);
      margin-top: 20px;
      overflow: hidden;
    }
    .panel .title {
      padding: 10px 16px;
      font-weight: 700;
      border-bottom: 1px solid var(--border);
    }
    .unified {
      padding: 0;
      font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      font-size: 14px;
      line-height: 1.4;
    }
    .unified .line { padding: 0 16px; white-space: pre; }
    .unified .line.added { background: var(--diff-add); }
    .unified .line.removed { background: var(--diff-del); }
    .unified .line.hunk { color: #2457c5; }
    .grid-wrap {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 16px;
      padding: 16px;
    }
    .overlay-stack {
      position: relative;
      min-height: 1px;
    }
    .overlay-stack .layer.base {
      position: relative;
      z-index: 1;
    }
    .overlay-stack .layer.overlay {
      position: absolute;
      inset: 0;
      z-index: 2;
      pointer-events: none;
      color: transparent;
    }
    .overlay-stack .layer.overlay .cell {
      background: transparent !important;
    }
    .overlay-stack .layer.overlay .cell.diff.removed { background: var(--diff-del-soft) !important; }
    .overlay-stack .layer.overlay .cell.diff.added { background: var(--diff-add-soft) !important; }
    .overlay-stack .layer.overlay .cell.diff.changed { background: var(--diff-change-soft) !important; }
    .overlay-stack .layer.overlay .line.diff.removed { background: var(--diff-del-soft) !important; }
    .overlay-stack .layer.overlay .line.diff.added { background: var(--diff-add-soft) !important; }
    .overlay-stack .layer.overlay .line.diff.changed { background: var(--diff-change-soft) !important; }
    .grid {
      border: 1px solid var(--border);
      border-radius: 8px;
      overflow: hidden;
    }
    .grid .title {
      padding: 8px 12px;
      font-weight: 700;
      border-bottom: 1px solid var(--border);
      background: #fafafa;
    }
    .grid .content {
      padding: 8px 12px;
      font-size: 14px;
      line-height: 1.3;
      overflow: auto;
    }
    .line {
      display: flex;
      gap: 0;
    }
    .cell {
      display: inline-block;
      min-width: 0.6ch;
      white-space: pre;
    }
    .line.diff.removed { background: var(--diff-del); }
    .line.diff.added { background: var(--diff-add); }
    .line.diff.changed { background: var(--diff-change); }
    .cell.diff.removed { background: var(--diff-del-soft); }
    .cell.diff.added { background: var(--diff-add-soft); }
    .cell.diff.changed { background: var(--diff-change-soft); }
    .hidden { display: none; }
  </style>
</head>
<body>
  <div class="controls">
    <div class="label">view</div>
    <button data-view="unified">unified</button>
    <button data-view="side">side</button>
    <button data-view="overlay">overlay</button>
  </div>

  <div class="panel unified" data-view="unified">
    <div class="title">unified diff (text)</div>
    <div class="content">%s</div>
  </div>

  <div class="panel" data-view="side">
    <div class="grid-wrap">
      <div class="grid">
        <div class="title">%s</div>
        <div class="content" style="background:%s;color:%s">%s</div>
      </div>
      <div class="grid">
        <div class="title">%s</div>
        <div class="content" style="background:%s;color:%s">%s</div>
      </div>
    </div>
  </div>

  <div class="panel" data-view="overlay">
    <div class="grid-wrap">
      <div class="grid">
        <div class="title">%s</div>
        <div class="content" style="background:%s;color:%s">
          <div class="overlay-stack">
            <div class="layer base">%s</div>
            <div class="layer overlay">%s</div>
          </div>
        </div>
      </div>
      <div class="grid">
        <div class="title">%s</div>
        <div class="content" style="background:%s;color:%s">
          <div class="overlay-stack">
            <div class="layer base">%s</div>
            <div class="layer overlay">%s</div>
          </div>
        </div>
      </div>
    </div>
  </div>

  <script>
    const panels = document.querySelectorAll('.panel[data-view]');
    const buttons = document.querySelectorAll('.controls button[data-view]');
    function setView(view) {
      panels.forEach(panel => {
        panel.classList.toggle('hidden', panel.dataset.view !== view);
      });
      buttons.forEach(button => {
        button.classList.toggle('active', button.dataset.view === view);
      });
    }
    buttons.forEach(button => {
      button.addEventListener('click', () => setView(button.dataset.view));
    });
    setView('%s');
  </script>
</body>
</html>`,
		unified,
		expectedLabel, expectedPlain.bg, expectedPlain.fg, expectedPlain.html,
		actualLabel, actualPlain.bg, actualPlain.fg, actualPlain.html,
		expectedLabel, expectedPlain.bg, expectedPlain.fg, expectedPlain.html, expectedAligned.html,
		actualLabel, actualPlain.bg, actualPlain.fg, actualPlain.html, actualAligned.html,
		defaultView,
	)
}
