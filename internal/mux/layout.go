package mux

// SplitMode determines how panes are arranged.
type SplitMode int

const (
	SplitNone       SplitMode = iota // single pane (tab mode)
	SplitHorizontal                  // top / bottom
	SplitVertical                    // left / right
)

func (s SplitMode) String() string {
	switch s {
	case SplitHorizontal:
		return "horizontal"
	case SplitVertical:
		return "vertical"
	default:
		return "none"
	}
}

// Pane represents a visible area showing a session.
type Pane struct {
	SessionIdx int
	Row, Col   int // top-left position (1-based)
	Width      int
	Height     int
	Focused    bool
}

// Layout calculates pane positions for the given terminal size.
// tabBarRow is reserved for the tab bar (excluded from pane area).
func Layout(split SplitMode, focusIdx int, sessionIdxA, sessionIdxB int, cols, rows int) []Pane {
	// usable area: rows-1 (tab bar at bottom)
	usable := rows - 1
	if usable < 2 {
		usable = 2
	}

	switch split {
	case SplitHorizontal:
		topH := usable / 2
		botH := usable - topH - 1 // -1 for separator
		return []Pane{
			{SessionIdx: sessionIdxA, Row: 1, Col: 1, Width: cols, Height: topH, Focused: focusIdx == 0},
			{SessionIdx: sessionIdxB, Row: 1 + topH + 1, Col: 1, Width: cols, Height: botH, Focused: focusIdx == 1},
		}

	case SplitVertical:
		leftW := cols / 2
		rightW := cols - leftW - 1 // -1 for separator
		return []Pane{
			{SessionIdx: sessionIdxA, Row: 1, Col: 1, Width: leftW, Height: usable, Focused: focusIdx == 0},
			{SessionIdx: sessionIdxB, Row: 1, Col: 1 + leftW + 1, Width: rightW, Height: usable, Focused: focusIdx == 1},
		}

	default:
		// Single pane
		return []Pane{
			{SessionIdx: sessionIdxA, Row: 1, Col: 1, Width: cols, Height: usable, Focused: true},
		}
	}
}

// SeparatorRow returns the row for horizontal separator (1-based), or 0 if none.
func SeparatorRow(split SplitMode, cols, rows int) int {
	if split != SplitHorizontal {
		return 0
	}
	usable := rows - 1
	topH := usable / 2
	return topH + 1
}

// SeparatorCol returns the col for vertical separator (1-based), or 0 if none.
func SeparatorCol(split SplitMode, cols, rows int) int {
	if split != SplitVertical {
		return 0
	}
	return cols/2 + 1
}
