package mux

import (
	"fmt"
	"strings"

	"github.com/hinshun/vt10x"
)

// cell is one column of the frame. cont marks the right half of a
// double-width character drawn in the cell before it.
type cell struct {
	ch   rune
	sgr  string
	cont bool
}

// frame is a cols×rows grid the multiplexer paints into, then serializes.
type frame struct {
	cols, rows int
	cells      [][]cell
}

const sgrPlain = "\033[0m"

func newFrame(cols, rows int) *frame {
	f := &frame{cols: cols, rows: rows, cells: make([][]cell, rows)}
	for r := range f.cells {
		f.cells[r] = make([]cell, cols)
		for c := range f.cells[r] {
			f.cells[r][c] = cell{ch: ' ', sgr: sgrPlain}
		}
	}
	return f
}

// set draws r at (row, col) (0-based) and returns the columns it used.
func (f *frame) set(row, col int, r rune, sgr string) int {
	if row < 0 || row >= f.rows || col < 0 || col >= f.cols {
		return 0
	}
	w := cellWidth(r)
	if w < 1 {
		r, w = ' ', 1
	}
	if col+w > f.cols {
		f.cells[row][col] = cell{ch: ' ', sgr: sgr}
		return 1
	}
	f.cells[row][col] = cell{ch: r, sgr: sgr}
	if w == 2 {
		f.cells[row][col+1] = cell{cont: true, sgr: sgr}
	}
	return w
}

// text draws s from (row, col), clipped at maxCol (exclusive); returns the next column.
func (f *frame) text(row, col, maxCol int, s, sgr string) int {
	if maxCol > f.cols {
		maxCol = f.cols
	}
	for _, r := range s {
		w := cellWidth(r)
		if w < 1 {
			continue
		}
		if col+w > maxCol {
			break
		}
		col += f.set(row, col, r, sgr)
	}
	return col
}

// fill paints cols [col, maxCol) of row with spaces in sgr.
func (f *frame) fill(row, col, maxCol int, sgr string) {
	for ; col < maxCol && col < f.cols; col++ {
		f.set(row, col, ' ', sgr)
	}
}

func (f *frame) String() string {
	var b strings.Builder
	b.Grow(f.rows * f.cols * 4)
	for r, line := range f.cells {
		if r > 0 {
			b.WriteByte('\n')
		}
		last := ""
		for _, c := range line {
			if c.cont {
				continue
			}
			if c.sgr != last {
				b.WriteString(c.sgr)
				last = c.sgr
			}
			b.WriteRune(c.ch)
		}
		b.WriteString(sgrPlain)
	}
	return b.String()
}

// View renders the tab bar and panes. focused says whether the terminal area
// has keyboard focus (draws the cursor and the focus accents).
func (m *Manager) View(focused bool) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	f := newFrame(m.cols, m.rows)
	m.drawTabBarLocked(f, focused)

	if len(m.tabs) == 0 {
		msg := "No sessions — pick a server on the left and press enter"
		f.text(m.rows/2, max(0, (m.cols-len(msg))/2), m.cols, msg, "\033[0;90m")
		return f.String()
	}

	panes, seps := m.paneLayoutLocked()
	split := !m.singleLocked()
	for i, p := range panes {
		s := m.tabSessionLocked(p.TabIdx)
		if split {
			drawPaneHeader(f, i, s, p, m.broadcast)
		}
		if s == nil {
			continue
		}
		drawPane(f, s, p, focused && p.Focused)
	}
	if split {
		drawSeparators(f, seps, m.broadcast)
	}
	if m.helpShown {
		drawHelp(f)
	}
	return f.String()
}

// drawPane copies the session's VT screen into the pane (1-based Row/Col).
func drawPane(f *frame, s *Session, p Pane, showCursor bool) {
	vtCols, vtRows := s.VTSize()
	for row := 0; row < p.Height && row < vtRows; row++ {
		y := p.Row - 1 + row
		x := p.Col - 1
		end := p.Col - 1 + p.Width
		for col := 0; col < vtCols && x < end; col++ {
			g := s.CellAt(col, row)
			if g.Char == widePad {
				continue
			}
			ch := g.Char
			if ch == 0 {
				ch = ' '
			}
			w := cellWidth(ch)
			if w < 1 {
				ch, w = ' ', 1
			}
			if x+w > end {
				break
			}
			x += f.set(y, x, ch, sgrFromGlyph(g))
		}
	}

	if !showCursor || !s.CursorVisible() {
		return
	}
	cx, cy := s.CursorPos()
	if cx >= p.Width || cy >= p.Height {
		return
	}
	y, x := p.Row-1+cy, p.Col-1+cx
	if y < f.rows && x < f.cols && !f.cells[y][x].cont {
		// Bubble Tea hides the real cursor: draw it as a reversed cell.
		f.cells[y][x].sgr = "\033[0;7m"
	}
}

// drawPaneHeader draws the one-line header above a pane:
// "P1 user@host", purple when focused, yellow for irteamsu, red while broadcasting.
func drawPaneHeader(f *frame, i int, s *Session, p Pane, broadcast bool) {
	style := "\033[0;97;100m" // unfocused: white on grey
	switch {
	case broadcast && p.Focused:
		style = "\033[0;1;97;41m"
	case broadcast:
		style = "\033[0;97;41m"
	case s != nil && s.User == "irteamsu" && p.Focused:
		style = "\033[0;1;30;43m"
	case s != nil && s.User == "irteamsu":
		style = "\033[0;30;43m"
	case p.Focused:
		style = "\033[0;1;97;45m"
	}

	label := fmt.Sprintf(" P%d ", i+1)
	if s != nil {
		label += s.Label()
	}
	if broadcast {
		label += "  ● BROADCAST"
	}
	if p.Focused {
		label += "  ◂"
	}
	row, col, end := p.Row-2, p.Col-1, p.Col-1+p.Width
	next := f.text(row, col, end, label, style)
	f.fill(row, next, end, style)
}

func drawSeparators(f *frame, seps []Separator, broadcast bool) {
	style := "\033[0;90m"
	if broadcast {
		style = "\033[0;91m"
	}
	for _, sep := range seps {
		for i := 0; i < sep.Len; i++ {
			if sep.Vertical {
				f.set(sep.Row-1+i, sep.Col-1, '│', style)
			} else {
				f.set(sep.Row-1, sep.Col-1+i, '─', style)
			}
		}
	}
}

// sgrFromGlyph returns the SGR sequence for a VT cell's attributes.
func sgrFromGlyph(g vt10x.Glyph) string {
	// Always reset first to prevent stale attributes from leaking.
	// vt10x attr bits: reverse=1, underline=2, bold=4, gfx=8, italic=16, blink=32
	parts := []string{"0"}

	if g.Mode&4 != 0 { // attrBold
		parts = append(parts, "1")
	}
	if g.Mode&16 != 0 { // attrItalic
		parts = append(parts, "3")
	}
	if g.Mode&2 != 0 { // attrUnderline
		parts = append(parts, "4")
	}
	if g.Mode&32 != 0 { // attrBlink
		parts = append(parts, "5")
	}
	if g.Mode&1 != 0 { // attrReverse
		parts = append(parts, "7")
	}

	parts = append(parts, colorSGR(g.FG, 30, 90, 38, 39))
	parts = append(parts, colorSGR(g.BG, 40, 100, 48, 49))
	return "\033[" + strings.Join(parts, ";") + "m"
}

// colorSGR encodes a vt10x color: 0-7 basic, 8-15 bright, 16-255 palette,
// below 1<<24 24-bit RGB, otherwise the default color.
func colorSGR(c vt10x.Color, base, bright, ext, def int) string {
	switch {
	case c <= 7:
		return fmt.Sprintf("%d", base+int(c))
	case c <= 15:
		return fmt.Sprintf("%d", bright+int(c)-8)
	case c <= 255:
		return fmt.Sprintf("%d;5;%d", ext, c)
	case c < 1<<24:
		return fmt.Sprintf("%d;2;%d;%d;%d", ext, (c>>16)&0xff, (c>>8)&0xff, c&0xff)
	default:
		return fmt.Sprintf("%d", def)
	}
}
