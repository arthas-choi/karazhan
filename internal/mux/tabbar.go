package mux

import (
	"fmt"
	"os"
	"strings"
)

// drawTabBar renders the tab bar at the bottom row of the terminal.
func (m *Manager) drawTabBar(row, cols int) {
	// Save cursor position
	fmt.Fprint(os.Stdout, "\0337")

	// Move to last row, column 1
	fmt.Fprintf(os.Stdout, "\033[%d;1H", row)

	// Clear the line
	fmt.Fprint(os.Stdout, "\033[2K")

	// Build tab bar content
	var tabs []string
	m.mu.Lock()
	for i, s := range m.sessions {
		label := fmt.Sprintf(" %d:%s ", i+1, s.Name)
		if s.IsDone() {
			label += "(x) "
		}

		isActive := i == m.activeIdx
		// In split mode, highlight both pane sessions
		if m.split != SplitNone {
			isActive = i == m.splitPanes[0] || i == m.splitPanes[1]
		}

		if isActive {
			if m.split != SplitNone && i == m.splitPanes[m.splitFocus] {
				// Focused pane: bold white on purple
				tabs = append(tabs, fmt.Sprintf("\033[1;97;45m%s\033[0m", label))
			} else if m.split != SplitNone {
				// Other visible pane: white on dark blue
				tabs = append(tabs, fmt.Sprintf("\033[97;44m%s\033[0m", label))
			} else {
				tabs = append(tabs, fmt.Sprintf("\033[1;97;45m%s\033[0m", label))
			}
		} else {
			tabs = append(tabs, fmt.Sprintf("\033[37;40m%s\033[0m", label))
		}
	}
	splitLabel := ""
	if m.split != SplitNone {
		splitLabel = fmt.Sprintf(" [%s] ", m.split)
	}
	m.mu.Unlock()

	bar := strings.Join(tabs, "")

	// Hint
	hint := " ^A ? "
	if splitLabel != "" {
		hint = splitLabel + hint
	}
	hintStyled := fmt.Sprintf("\033[90;40m%s\033[0m", hint)

	// Fill background
	fmt.Fprintf(os.Stdout, "\033[40m%-*s\033[0m", cols, "")
	fmt.Fprintf(os.Stdout, "\033[%d;1H", row)
	fmt.Fprint(os.Stdout, bar)

	hintPos := cols - len(hint) + 1
	if hintPos > 0 {
		fmt.Fprintf(os.Stdout, "\033[%d;%dH%s", row, hintPos, hintStyled)
	}

	// Restore cursor position
	fmt.Fprint(os.Stdout, "\0338")
}

// drawHelp shows a help overlay.
func (m *Manager) drawHelp(rows, cols int) {
	lines := []string{
		"",
		"  Karazhan Multiplexer",
		"  ════════════════════",
		"",
		"  Ctrl+A d       Detach (back to server list)",
		"  Ctrl+A c       New tab (select server)",
		"  Ctrl+A x       Close current tab/pane",
		"",
		"  Ctrl+A n / ]   Next tab",
		"  Ctrl+A p / [   Previous tab",
		"  Ctrl+A 1-9     Switch to tab N",
		"",
		"  Ctrl+A -       Split horizontal (top/bottom)",
		"  Ctrl+A |       Split vertical (left/right)",
		"  Ctrl+A Tab     Switch focus between panes",
		"  Ctrl+A Arrow   Switch focus (directional)",
		"  Ctrl+A z       Zoom (unsplit / single pane)",
		"",
		"  Ctrl+A Ctrl+A  Send literal Ctrl+A",
		"  Ctrl+A ?       This help",
		"",
		"  Press any key to close",
		"",
	}

	boxW := 50
	startRow := (rows-len(lines))/2 - 1
	startCol := (cols - boxW) / 2

	if startRow < 1 {
		startRow = 1
	}
	if startCol < 1 {
		startCol = 1
	}

	for i, line := range lines {
		padded := line
		if len(padded) < boxW {
			padded += strings.Repeat(" ", boxW-len(padded))
		}
		fmt.Fprintf(os.Stdout, "\033[%d;%dH\033[97;44m%s\033[0m", startRow+i, startCol, padded)
	}
}
