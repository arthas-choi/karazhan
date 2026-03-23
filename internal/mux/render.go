package mux

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/hinshun/vt10x"
)

// renderSplit composites all panes into a single buffer, then writes once.
func (m *Manager) renderSplit(rows, cols int) {
	// Snapshot state under lock
	m.mu.Lock()
	split := m.split
	if split == SplitNone {
		m.mu.Unlock()
		return
	}
	paneA := m.splitPanes[0]
	paneB := m.splitPanes[1]
	focus := m.splitFocus
	sessCount := len(m.sessions)
	m.mu.Unlock()

	if sessCount < 2 {
		return
	}

	panes := Layout(split, focus, paneA, paneB, cols, rows)

	var buf bytes.Buffer
	buf.Grow(rows * cols * 3)

	buf.WriteString("\033[?25l")

	for _, p := range panes {
		if p.SessionIdx < 0 || p.SessionIdx >= sessCount {
			continue
		}
		m.mu.Lock()
		s := m.sessions[p.SessionIdx]
		m.mu.Unlock()
		renderPaneToBuf(&buf, s, p)
	}

	renderSeparatorToBuf(&buf, split, rows, cols)

	// Cursor at focused pane
	for _, p := range panes {
		if p.Focused && p.SessionIdx >= 0 && p.SessionIdx < sessCount {
			m.mu.Lock()
			s := m.sessions[p.SessionIdx]
			m.mu.Unlock()
			curCol, curRow := s.CursorPos()
			absRow := p.Row + curRow
			absCol := p.Col + curCol
			if absRow >= p.Row && absRow < p.Row+p.Height && absCol >= p.Col && absCol < p.Col+p.Width {
				fmt.Fprintf(&buf, "\033[%d;%dH", absRow, absCol)
			}
		}
	}

	buf.WriteString("\033[?25h")

	os.Stdout.Write(buf.Bytes())
}

func renderPaneToBuf(buf *bytes.Buffer, s *Session, p Pane) {
	var lastFG, lastBG vt10x.Color
	lastFG = 0xFFFFFFFF
	lastBG = 0xFFFFFFFF

	for row := 0; row < p.Height; row++ {
		fmt.Fprintf(buf, "\033[%d;%dH", p.Row+row, p.Col)

		for col := 0; col < p.Width; col++ {
			cell := s.CellAt(col, row)
			ch := cell.Char
			if ch == 0 {
				ch = ' '
			}

			if cell.FG != lastFG || cell.BG != lastBG {
				buf.WriteString(sgrFromGlyph(cell))
				lastFG = cell.FG
				lastBG = cell.BG
			}

			buf.WriteRune(ch)
		}
	}
	buf.WriteString("\033[0m")
}

func renderSeparatorToBuf(buf *bytes.Buffer, split SplitMode, rows, cols int) {
	if split == SplitHorizontal {
		sepRow := SeparatorRow(split, cols, rows)
		if sepRow > 0 {
			fmt.Fprintf(buf, "\033[%d;1H\033[90;40m", sepRow)
			buf.WriteString(strings.Repeat("─", cols))
			buf.WriteString("\033[0m")
		}
	} else if split == SplitVertical {
		sepCol := SeparatorCol(split, cols, rows)
		usable := rows - 1
		if sepCol > 0 {
			for row := 1; row <= usable; row++ {
				fmt.Fprintf(buf, "\033[%d;%dH\033[90;40m│\033[0m", row, sepCol)
			}
		}
	}
}

func sgrFromGlyph(g vt10x.Glyph) string {
	var parts []string

	if g.Mode&1 != 0 {
		parts = append(parts, "1")
	}
	if g.Mode&4 != 0 {
		parts = append(parts, "4")
	}

	fg := g.FG
	if fg <= 7 {
		parts = append(parts, fmt.Sprintf("%d", 30+fg))
	} else if fg <= 15 {
		parts = append(parts, fmt.Sprintf("%d", 90+fg-8))
	} else if fg <= 255 {
		parts = append(parts, fmt.Sprintf("38;5;%d", fg))
	} else {
		parts = append(parts, "39")
	}

	bg := g.BG
	if bg <= 7 {
		parts = append(parts, fmt.Sprintf("%d", 40+bg))
	} else if bg <= 15 {
		parts = append(parts, fmt.Sprintf("%d", 100+bg-8))
	} else if bg <= 255 {
		parts = append(parts, fmt.Sprintf("48;5;%d", bg))
	} else {
		parts = append(parts, "49")
	}

	if len(parts) == 0 {
		return "\033[0m"
	}
	return "\033[" + strings.Join(parts, ";") + "m"
}
