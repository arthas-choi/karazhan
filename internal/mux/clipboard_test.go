package mux

import (
	"encoding/base64"
	"testing"
	"time"
)

func osc52(text string) string {
	return "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
}

func TestClipFilterExtractsOSC52AndKeepsTheRest(t *testing.T) {
	var f clipFilter
	in := "abc\x1b[1;31m" + osc52("hello") + "\x1b]0;title\x07def"
	out, clips := f.apply([]byte(in))
	if want := "abc\x1b[1;31m\x1b]0;title\x07def"; string(out) != want {
		t.Fatalf("stream = %q, want %q", out, want)
	}
	if len(clips) != 1 || clips[0] != "hello" {
		t.Fatalf("clips = %q, want [hello]", clips)
	}
}

func TestClipFilterAcrossChunksAndSTTerminator(t *testing.T) {
	var f clipFilter
	seq := "\x1b]52;p;" + base64.StdEncoding.EncodeToString([]byte("안녕 world")) + "\x1b\\" + "tail"
	var out []byte
	var clips []string
	for i := 0; i < len(seq); i++ { // one byte at a time
		o, c := f.apply([]byte{seq[i]})
		out = append(out, o...)
		clips = append(clips, c...)
	}
	if string(out) != "tail" {
		t.Fatalf("stream = %q, want %q", out, "tail")
	}
	if len(clips) != 1 || clips[0] != "안녕 world" {
		t.Fatalf("clips = %q", clips)
	}
}

func TestClipFilterSwallowsQueriesAndBadBase64(t *testing.T) {
	var f clipFilter
	out, clips := f.apply([]byte("\x1b]52;c;?\x07\x1b]52;c;***\x07x"))
	if string(out) != "x" || len(clips) != 0 {
		t.Fatalf("stream = %q clips = %q", out, clips)
	}
}

func TestClipFilterLeavesOtherStringsAlone(t *testing.T) {
	var f clipFilter
	in := "\x1bP+q544e\x1b\\" + "\x1b]8;;http://x\x07link\x1b]8;;\x07" + "\x1b(B" + "한"
	out, clips := f.apply([]byte(in))
	if string(out) != in || len(clips) != 0 {
		t.Fatalf("stream changed: %q clips %q", out, clips)
	}
}

func TestOSC52RoundTrip(t *testing.T) {
	var f clipFilter
	_, clips := f.apply([]byte(OSC52("round\ntrip")))
	if len(clips) != 1 || clips[0] != "round\ntrip" {
		t.Fatalf("clips = %q", clips)
	}
}

func TestManagerCollectsClipboardFromSession(t *testing.T) {
	t.Setenv("SHELL", "/bin/cat")
	m := NewManager(true)
	defer m.CloseAll()
	m.SetSize(60, 10)
	if err := m.OpenTab("h1", "h1", "irteam"); err != nil {
		t.Fatal(err)
	}
	// cat echoes the sequence back through the PTY, as a remote vim would emit it.
	m.HandleInput([]byte(osc52("from vim") + "\r")) // Enter: the line discipline hands the line to cat

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		for _, c := range m.TakeClipboard() {
			if c == "from vim" {
				if len(m.TakeClipboard()) != 0 {
					t.Fatal("TakeClipboard should drain")
				}
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("clipboard text never arrived")
}
