package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// keyToBytes turns a Bubble Tea key event back into the bytes a terminal
// would send, for forwarding to a PTY. appCursor selects application-mode
// arrow keys (ESC O A), which full-screen programs like vim switch on.
func keyToBytes(k tea.KeyMsg, appCursor bool) []byte {
	var s string
	switch k.Type {
	case tea.KeyRunes:
		s = string(k.Runes)
		if k.Paste {
			// Terminals send pasted line breaks as carriage returns.
			s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\r"), "\n", "\r")
		}
	case tea.KeySpace:
		s = " "
	case tea.KeyUp:
		s = cursorKey('A', appCursor)
	case tea.KeyDown:
		s = cursorKey('B', appCursor)
	case tea.KeyRight:
		s = cursorKey('C', appCursor)
	case tea.KeyLeft:
		s = cursorKey('D', appCursor)
	case tea.KeyHome:
		s = cursorKey('H', appCursor)
	case tea.KeyEnd:
		s = cursorKey('F', appCursor)
	case tea.KeyShiftUp:
		s = "\x1b[1;2A"
	case tea.KeyShiftDown:
		s = "\x1b[1;2B"
	case tea.KeyShiftRight:
		s = "\x1b[1;2C"
	case tea.KeyShiftLeft:
		s = "\x1b[1;2D"
	case tea.KeyCtrlUp:
		s = "\x1b[1;5A"
	case tea.KeyCtrlDown:
		s = "\x1b[1;5B"
	case tea.KeyCtrlRight:
		s = "\x1b[1;5C"
	case tea.KeyCtrlLeft:
		s = "\x1b[1;5D"
	case tea.KeyCtrlShiftUp:
		s = "\x1b[1;6A"
	case tea.KeyCtrlShiftDown:
		s = "\x1b[1;6B"
	case tea.KeyCtrlShiftRight:
		s = "\x1b[1;6C"
	case tea.KeyCtrlShiftLeft:
		s = "\x1b[1;6D"
	case tea.KeyShiftHome:
		s = "\x1b[1;2H"
	case tea.KeyShiftEnd:
		s = "\x1b[1;2F"
	case tea.KeyCtrlHome:
		s = "\x1b[1;5H"
	case tea.KeyCtrlEnd:
		s = "\x1b[1;5F"
	case tea.KeyShiftTab:
		s = "\x1b[Z"
	case tea.KeyInsert:
		s = "\x1b[2~"
	case tea.KeyDelete:
		s = "\x1b[3~"
	case tea.KeyPgUp:
		s = "\x1b[5~"
	case tea.KeyPgDown:
		s = "\x1b[6~"
	case tea.KeyCtrlPgUp:
		s = "\x1b[5;5~"
	case tea.KeyCtrlPgDown:
		s = "\x1b[6;5~"
	case tea.KeyF1:
		s = "\x1bOP"
	case tea.KeyF2:
		s = "\x1bOQ"
	case tea.KeyF3:
		s = "\x1bOR"
	case tea.KeyF4:
		s = "\x1bOS"
	case tea.KeyF5:
		s = "\x1b[15~"
	case tea.KeyF6:
		s = "\x1b[17~"
	case tea.KeyF7:
		s = "\x1b[18~"
	case tea.KeyF8:
		s = "\x1b[19~"
	case tea.KeyF9:
		s = "\x1b[20~"
	case tea.KeyF10:
		s = "\x1b[21~"
	case tea.KeyF11:
		s = "\x1b[23~"
	case tea.KeyF12:
		s = "\x1b[24~"
	default:
		// Control characters (Ctrl+A…Ctrl+Z, Enter, Tab, Esc, Backspace) are
		// their own byte values.
		if (k.Type >= 0 && k.Type <= 31) || k.Type == 127 {
			s = string([]byte{byte(k.Type)})
		}
	}
	if s == "" {
		return nil
	}
	if k.Alt {
		s = "\x1b" + s
	}
	return []byte(s)
}

func cursorKey(final byte, appCursor bool) string {
	if appCursor {
		return "\x1bO" + string(final)
	}
	return "\x1b[" + string(final)
}
