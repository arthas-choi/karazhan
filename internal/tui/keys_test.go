package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestKeyToBytes(t *testing.T) {
	cases := []struct {
		name      string
		key       tea.KeyMsg
		appCursor bool
		want      string
	}{
		{"rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}, false, "a"},
		{"korean", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("한")}, false, "한"},
		{"alt+b", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b"), Alt: true}, false, "\x1bb"},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}, false, "\r"},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}, false, "\x03"},
		{"ctrl+a", tea.KeyMsg{Type: tea.KeyCtrlA}, false, "\x01"},
		{"backspace", tea.KeyMsg{Type: tea.KeyBackspace}, false, "\x7f"},
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}, false, "\x1b"},
		{"up", tea.KeyMsg{Type: tea.KeyUp}, false, "\x1b[A"},
		{"up app", tea.KeyMsg{Type: tea.KeyUp}, true, "\x1bOA"},
		{"ctrl+left", tea.KeyMsg{Type: tea.KeyCtrlLeft}, false, "\x1b[1;5D"},
		{"pgdown", tea.KeyMsg{Type: tea.KeyPgDown}, false, "\x1b[6~"},
		{"f5", tea.KeyMsg{Type: tea.KeyF5}, false, "\x1b[15~"},
		{"paste", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a\nb"), Paste: true}, false, "a\rb"},
	}
	for _, c := range cases {
		if got := string(keyToBytes(c.key, c.appCursor)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
