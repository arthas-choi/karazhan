package mux

// SplitMode is a pane layout preset.
type SplitMode int

const (
	SplitNone       SplitMode = iota // single pane (tab mode)
	SplitVertical                    // left | right
	SplitHorizontal                  // top / bottom
	SplitMainStack                   // main on the left, two stacked on the right
	SplitGrid                        // 2×2
)

// presetCycle is the order Ctrl+A Space steps through.
var presetCycle = []SplitMode{SplitNone, SplitVertical, SplitHorizontal, SplitMainStack, SplitGrid}

func (s SplitMode) String() string {
	switch s {
	case SplitVertical:
		return "vertical"
	case SplitHorizontal:
		return "horizontal"
	case SplitMainStack:
		return "main+stack"
	case SplitGrid:
		return "2x2"
	default:
		return "none"
	}
}

// PaneCount is the number of panes the preset shows.
func (s SplitMode) PaneCount() int {
	switch s {
	case SplitVertical, SplitHorizontal:
		return 2
	case SplitMainStack:
		return 3
	case SplitGrid:
		return 4
	default:
		return 1
	}
}

// nextPreset returns the preset after cur that fits in tabCount tabs.
func nextPreset(cur SplitMode, tabCount int) SplitMode {
	start := 0
	for i, p := range presetCycle {
		if p == cur {
			start = i
		}
	}
	for step := 1; step <= len(presetCycle); step++ {
		p := presetCycle[(start+step)%len(presetCycle)]
		if p.PaneCount() <= tabCount {
			return p
		}
	}
	return SplitNone
}

// Pane is a visible area showing one tab. Row/Col/Width/Height (1-based)
// are the terminal content area; in split mode a one-line header sits on
// the row above it.
type Pane struct {
	TabIdx  int
	Row     int
	Col     int
	Width   int
	Height  int
	Focused bool
}

// Separator is a line drawn between panes.
type Separator struct {
	Vertical bool
	Row, Col int // start (1-based)
	Len      int
}

// rect is a pane area including its header row.
type rect struct{ row, col, w, h int }

// Layout places the preset's panes in the area above the tab bar (the last
// row). tabIdx[i] is the tab shown in pane i; focus is the focused pane.
func Layout(split SplitMode, focus int, tabIdx []int, cols, rows int) ([]Pane, []Separator) {
	usable := rows - 1
	if usable < 4 {
		usable = 4
	}

	if split == SplitNone {
		idx := 0
		if len(tabIdx) > 0 && focus >= 0 && focus < len(tabIdx) {
			idx = tabIdx[focus]
		}
		return []Pane{{TabIdx: idx, Row: 1, Col: 1, Width: cols, Height: usable, Focused: true}}, nil
	}

	leftW := cols / 2
	rightW := cols - leftW - 1
	topH := usable / 2
	botH := usable - topH - 1

	var rects []rect
	var seps []Separator
	switch split {
	case SplitVertical:
		rects = []rect{{1, 1, leftW, usable}, {1, leftW + 2, rightW, usable}}
		seps = []Separator{{Vertical: true, Row: 1, Col: leftW + 1, Len: usable}}
	case SplitHorizontal:
		rects = []rect{{1, 1, cols, topH}, {topH + 2, 1, cols, botH}}
		seps = []Separator{{Row: topH + 1, Col: 1, Len: cols}}
	case SplitMainStack:
		rects = []rect{{1, 1, leftW, usable}, {1, leftW + 2, rightW, topH}, {topH + 2, leftW + 2, rightW, botH}}
		seps = []Separator{
			{Vertical: true, Row: 1, Col: leftW + 1, Len: usable},
			{Row: topH + 1, Col: leftW + 2, Len: rightW},
		}
	case SplitGrid:
		rects = []rect{
			{1, 1, leftW, topH}, {1, leftW + 2, rightW, topH},
			{topH + 2, 1, leftW, botH}, {topH + 2, leftW + 2, rightW, botH},
		}
		seps = []Separator{
			{Vertical: true, Row: 1, Col: leftW + 1, Len: usable},
			{Row: topH + 1, Col: 1, Len: cols},
		}
	}

	panes := make([]Pane, 0, len(rects))
	for i, r := range rects {
		idx := -1
		if i < len(tabIdx) {
			idx = tabIdx[i]
		}
		h := r.h - 1 // header row
		if h < 1 {
			h = 1
		}
		panes = append(panes, Pane{TabIdx: idx, Row: r.row + 1, Col: r.col, Width: r.w, Height: h, Focused: i == focus})
	}
	return panes, seps
}

// neighbor returns the pane nearest to from in direction dir ('A' up,
// 'B' down, 'C' right, 'D' left), or from if there is none.
func neighbor(panes []Pane, from int, dir byte) int {
	if from < 0 || from >= len(panes) {
		return from
	}
	cur := panes[from]
	cx, cy := cur.Col*2+cur.Width, cur.Row*2+cur.Height // doubled centers
	best, bestDist := from, -1
	for i, p := range panes {
		if i == from {
			continue
		}
		px, py := p.Col*2+p.Width, p.Row*2+p.Height
		var ok bool
		switch dir {
		case 'A':
			ok = p.Row+p.Height <= cur.Row
		case 'B':
			ok = p.Row >= cur.Row+cur.Height
		case 'C':
			ok = p.Col >= cur.Col+cur.Width
		case 'D':
			ok = p.Col+p.Width <= cur.Col
		}
		if !ok {
			continue
		}
		dist := abs(px-cx) + abs(py-cy)
		if bestDist < 0 || dist < bestDist {
			best, bestDist = i, dist
		}
	}
	return best
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
