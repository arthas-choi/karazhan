package tui

import (
	"strings"
	"testing"

	"Karazhan/internal/config"
	"Karazhan/internal/mux"

	"github.com/charmbracelet/x/ansi"
)

func TestViewCarriesPendingClipboardWithoutChangingLayout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.KerberosMock, cfg.ServersMock = true, true
	m := NewApp(cfg)
	m.screen = screenServers
	m.width, m.height = 80, 24

	plain := m.View()
	m.pendingClip = "yanked text"
	m.clipSeq = 1
	withClip := m.View()

	want := mux.OSC52("yanked text")
	if !strings.HasPrefix(withClip, want) {
		t.Fatalf("view should start with the OSC 52 sequence")
	}
	if strings.TrimPrefix(withClip, want) != plain {
		t.Fatalf("clipboard sequence changed the visible view")
	}
	first := strings.SplitN(withClip, "\n", 2)[0]
	if ansi.StringWidth(first) != 80 {
		t.Fatalf("first line width %d, want 80", ansi.StringWidth(first))
	}
	// Bubble Tea truncates lines to the terminal width; the sequence must survive.
	if !strings.Contains(ansi.Truncate(first, 80, ""), want) {
		t.Fatalf("renderer truncation would drop the clipboard sequence")
	}

	// A stale clear is ignored; the matching one removes the sequence.
	next, _ := m.update(clipClearMsg{seq: 0})
	m = next.(appModel)
	if m.pendingClip == "" {
		t.Fatal("stale clipClearMsg cleared the clipboard")
	}
	next, _ = m.update(clipClearMsg{seq: 1})
	m = next.(appModel)
	if m.pendingClip != "" || m.View() != plain {
		t.Fatal("clipClearMsg should drop the sequence from the view")
	}
}
