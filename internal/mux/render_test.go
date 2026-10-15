package mux

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestWideFilterPadsCJKOutsideEscapes(t *testing.T) {
	var f wideFilter
	// Title OSC with Korean must stay untouched; the visible "한a" gets a pad.
	in := "\x1b]0;제목\x07\x1b[1;31m한a"
	got := string(f.apply([]byte(in)))
	want := "\x1b]0;제목\x07\x1b[1;31m한" + string(widePad) + "a"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWideFilterSplitUTF8AcrossChunks(t *testing.T) {
	var f wideFilter
	b := []byte("가")
	out := append(f.apply(b[:1]), f.apply(b[1:])...)
	if string(out) != "가"+string(widePad) {
		t.Fatalf("got %q", out)
	}
}

func TestFrameRowsKeepWidthWithWideChars(t *testing.T) {
	f := newFrame(10, 2)
	f.text(0, 0, 10, "한글abcdefgh", sgrPlain)
	f.text(1, 8, 10, "가나", sgrPlain) // only "가" fits
	for i, line := range strings.Split(f.String(), "\n") {
		if w := ansi.StringWidth(line); w != 10 {
			t.Fatalf("row %d width = %d, want 10: %q", i, w, line)
		}
	}
}

func TestViewEveryRowHasAreaWidth(t *testing.T) {
	t.Setenv("SHELL", "/bin/cat")
	m := NewManager(true)
	defer m.CloseAll()
	m.SetSize(70, 20)
	for _, n := range []string{"h1", "h2", "h3", "h4"} {
		if err := m.OpenTab(n, n, "irteam"); err != nil {
			t.Fatal(err)
		}
	}
	for _, keys := range [][]byte{nil, []byte(" "), []byte(" "), []byte(" "), []byte(" "), []byte("?")} {
		if keys != nil {
			m.HandleInput([]byte{prefixKey})
			m.HandleInput(keys)
		}
		lines := strings.Split(m.View(true), "\n")
		if len(lines) != 20 {
			t.Fatalf("layout %v: %d rows, want 20", m.split, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w != 70 {
				t.Fatalf("layout %v row %d width %d, want 70", m.split, i, w)
			}
		}
	}
}

func TestPrefixActions(t *testing.T) {
	m := NewManager(true)
	press := func(k string) Action {
		m.HandleInput([]byte{prefixKey})
		return m.HandleInput([]byte(k))
	}
	if press("d") != ActionFocusList || press("e") != ActionToggleList {
		t.Fatalf("Ctrl+A d / e should ask for list focus / toggle")
	}
	if press("\x1b[D") != ActionFocusList {
		t.Fatalf("Ctrl+A ← in single mode should focus the list")
	}
}
