package mux

import (
	"fmt"
	"strings"
)

// ANSI styles for the tab bar.
const (
	styleBar         = "\033[0;37;40m"    // bar background
	styleBarDim      = "\033[0;90;40m"    // hints
	styleTabFocused  = "\033[0;1;97;45m"  // focused tab: bold white on purple
	styleTabBlurred  = "\033[0;1;97;100m" // focused tab while the server list has focus
	styleTabVisible  = "\033[0;97;44m"    // other visible split pane: white on blue
	styleTabIdle     = "\033[0;37;40m"    // background tab
	styleUserCurrent = "\033[0;1;30;47m"  // shown user: black on white
	styleUserSu      = "\033[0;1;30;43m"  // shown irteamsu: black on yellow (privileged)
	styleUserSuIdle  = "\033[0;33;40m"    // irteamsu shown in a background tab: yellow text
	styleUserOther   = "\033[0;90;40m"    // user with an open but hidden session
	styleBroadcast   = "\033[0;1;97;41m"
	stylePrefix      = "\033[0;1;30;43m"
)

type segment struct{ text, sgr string }

func segmentsWidth(segs []segment) int {
	w := 0
	for _, s := range segs {
		w += stringWidth(s.text)
	}
	return w
}

// drawTabBarLocked renders the tab bar on the frame's first row: the tabs on
// the left (scrolled so the focused one is visible), layout and hints on the right.
func (m *Manager) drawTabBarLocked(f *frame, focused bool) {
	f.fill(0, 0, f.cols, styleBar)

	// Right side: prefix indicator, broadcast, layout, help hint.
	var right []segment
	if m.prefix {
		right = append(right, segment{" ^A … ", stylePrefix}, segment{" ", styleBar})
	}
	if m.broadcast && m.split != SplitNone {
		right = append(right, segment{" BROADCAST ", styleBroadcast}, segment{" ", styleBar})
	}
	if m.split != SplitNone {
		zoom := ""
		if m.zoomed {
			zoom = " zoom"
		}
		right = append(right, segment{fmt.Sprintf("[%s%s] ", m.split, zoom), styleBarDim})
	}
	right = append(right, segment{"^A ? ", styleBarDim})
	rightW := segmentsWidth(right)
	avail := f.cols - rightW - 1

	tabs := make([][]segment, len(m.tabs))
	for i, t := range m.tabs {
		tabs[i] = m.tabSegmentsLocked(i, t, focused)
	}
	// Drop tabs from the left until the focused one fits.
	start := 0
	for start < m.activeIdx {
		w := 0
		for i := start; i <= m.activeIdx && i < len(tabs); i++ {
			w += segmentsWidth(tabs[i])
		}
		if w <= avail {
			break
		}
		start++
	}
	col := 0
	if start > 0 {
		col = f.text(0, col, avail, "‹", styleBarDim)
	}
	for i := start; i < len(tabs); i++ {
		for _, s := range tabs[i] {
			col = f.text(0, col, avail, s.text, s.sgr)
		}
	}

	col = f.cols - rightW
	for _, s := range right {
		col = f.text(0, col, f.cols, s.text, s.sgr)
	}
}

// tabSegmentsLocked renders one tab as " P1 N:host #d " followed by a chip
// per user session; the shown user's chip is highlighted, irteamsu in yellow.
func (m *Manager) tabSegmentsLocked(i int, t *Tab, focused bool) []segment {
	base := styleTabIdle
	visible := false
	pane := -1
	if !m.singleLocked() {
		for p, idx := range m.panes {
			if idx == i {
				pane = p
			}
		}
	}
	switch {
	case i == m.activeIdx && focused:
		base, visible = styleTabFocused, true
	case i == m.activeIdx:
		base, visible = styleTabBlurred, true
	case pane >= 0:
		base, visible = styleTabVisible, true
	}

	label := fmt.Sprintf(" %d:%s ", i+1, t.DisplayName())
	if pane >= 0 {
		label = fmt.Sprintf(" P%d%s", pane+1, label)
	}
	segs := []segment{{label, base}}
	current := t.Current()
	for _, s := range t.sessions {
		style := styleUserOther
		switch {
		case s == current && s.User == "irteamsu" && visible:
			style = styleUserSu
		case s == current && s.User == "irteamsu":
			style = styleUserSuIdle
		case s == current && visible:
			style = styleUserCurrent
		case s == current:
			style = base
		}
		label := s.User
		if s.IsDone() {
			label += "(x)"
		}
		segs = append(segs, segment{" " + label + " ", style})
	}
	return append(segs, segment{" ", base}, segment{" ", styleBar})
}

var helpLines = []string{
	"",
	"  Karazhan Multiplexer",
	"  ════════════════════",
	"",
	"  Ctrl+A d / ←   Focus server list",
	"  Ctrl+A e       Hide / show server list",
	"  Ctrl+A c       Duplicate tab (same server/user)",
	"  Ctrl+A u       Switch user in tab (irteam ↔ irteamsu)",
	"  Ctrl+A x       Close current user session",
	"",
	"  Ctrl+A n / ]   Next tab",
	"  Ctrl+A p / [   Previous tab",
	"  Ctrl+A 1-9     Show tab N (in focused pane)",
	"",
	"  Ctrl+A Space   Next layout (1 / 2 / 2 / 1+2 / 2x2)",
	"  Ctrl+A - / |   Split top-bottom / left-right",
	"  Ctrl+A !       Unsplit (single pane)",
	"  Ctrl+A Tab     Next pane",
	"  Ctrl+A Arrow   Move focus to pane in direction",
	"  Ctrl+A z       Zoom focused pane (toggle)",
	"  Ctrl+A b       Broadcast input to all panes",
	"",
	"  Ctrl+A Ctrl+A  Send literal Ctrl+A",
	"  Ctrl+A ?       This help",
	"",
	"  Press any key to close",
	"",
}

// drawHelp paints the help box centered over the frame.
func drawHelp(f *frame) {
	const boxW = 60
	const style = "\033[0;97;44m"
	startRow := (f.rows - len(helpLines)) / 2
	startCol := (f.cols - boxW) / 2
	if startRow < 1 {
		startRow = 1
	}
	if startCol < 0 {
		startCol = 0
	}
	for i, line := range helpLines {
		row := startRow + i
		end := startCol + boxW
		next := f.text(row, startCol, end, strings.TrimRight(line, " "), style)
		f.fill(row, next, end, style)
	}
}
