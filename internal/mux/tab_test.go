package mux

import (
	"strings"
	"testing"
	"time"
)

func TestTabRemoveKeepsActiveOnRemainingSession(t *testing.T) {
	a := &Session{User: "irteam"}
	b := &Session{User: "irteamsu"}
	tab := &Tab{Name: "host"}
	tab.add(a)
	tab.add(b)

	if tab.CurrentUser() != "irteamsu" {
		t.Fatalf("current = %q, want irteamsu", tab.CurrentUser())
	}
	tab.remove(b)
	if tab.Current() != a {
		t.Fatalf("after removing irteamsu, current = %v, want irteam session", tab.Current())
	}
	tab.remove(a)
	if tab.Current() != nil {
		t.Fatalf("empty tab should have no current session")
	}
}

func TestTabActivate(t *testing.T) {
	tab := &Tab{Name: "host"}
	tab.add(&Session{User: "irteam"})
	tab.add(&Session{User: "irteamsu"})

	if !tab.activate("irteam") || tab.CurrentUser() != "irteam" {
		t.Fatalf("activate irteam failed, current = %q", tab.CurrentUser())
	}
	if tab.activate("root") {
		t.Fatalf("activate should fail for a user without a session")
	}
	if got := otherUser(tab.CurrentUser()); got != "irteamsu" {
		t.Fatalf("otherUser(irteam) = %q", got)
	}
}

func TestDisplayName(t *testing.T) {
	if got := (&Tab{Name: "h", Dup: 1}).DisplayName(); got != "h" {
		t.Fatalf("got %q", got)
	}
	if got := (&Tab{Name: "h", Dup: 2}).DisplayName(); got != "h #2" {
		t.Fatalf("got %q", got)
	}
}

func namedTabs(m *Manager, names ...string) {
	for _, n := range names {
		m.tabs = append(m.tabs, &Tab{Name: n})
	}
}

func paneNames(m *Manager) []string {
	var out []string
	for _, idx := range m.panes {
		out = append(out, m.tabs[idx].Name)
	}
	return out
}

func TestRemoveTabShiftsIndices(t *testing.T) {
	m := NewManager(true)
	namedTabs(m, "a", "b", "c", "d")
	m.split = SplitVertical
	m.panes = []int{1, 3}
	m.paneFocus = 1
	m.activeIdx = 3

	m.removeTabLocked(0)

	if m.tabs[m.activeIdx].Name != "d" {
		t.Fatalf("active tab = %s, want d", m.tabs[m.activeIdx].Name)
	}
	if got := paneNames(m); got[0] != "b" || got[1] != "d" {
		t.Fatalf("panes = %v, want [b d]", got)
	}
}

func TestRemoveVisibleTabRefillsPane(t *testing.T) {
	m := NewManager(true)
	namedTabs(m, "a", "b", "c", "d", "e")
	m.split = SplitGrid
	m.panes = []int{0, 1, 2, 3}
	m.paneFocus = 2
	m.activeIdx = 2

	if shrank := m.removeTabLocked(2); shrank {
		t.Fatalf("grid should stay with 4 tabs left")
	}
	got := paneNames(m)
	want := map[string]bool{"a": true, "b": true, "d": true, "e": true}
	for _, n := range got {
		if !want[n] {
			t.Fatalf("panes = %v, want a,b,d,e in some order", got)
		}
		delete(want, n)
	}
	if m.panes[m.paneFocus] != m.activeIdx {
		t.Fatalf("focused pane shows %d, active is %d", m.panes[m.paneFocus], m.activeIdx)
	}
}

func TestPresetShrinksWhenTabsClose(t *testing.T) {
	m := NewManager(true)
	namedTabs(m, "a", "b", "c", "d")
	m.split = SplitGrid
	m.panes = []int{0, 1, 2, 3}

	if !m.removeTabLocked(3) || m.split != SplitMainStack {
		t.Fatalf("split = %v, want main+stack after closing to 3 tabs", m.split)
	}
	if !m.removeTabLocked(2) || m.split != SplitVertical {
		t.Fatalf("split = %v, want vertical after closing to 2 tabs", m.split)
	}
	m.broadcast = true
	if !m.removeTabLocked(1) || m.split != SplitNone || m.broadcast {
		t.Fatalf("split = %v broadcast = %v, want single and broadcast off", m.split, m.broadcast)
	}
}

func TestFocusSwapsVisibleTab(t *testing.T) {
	m := NewManager(true)
	namedTabs(m, "a", "b", "c")
	m.split = SplitVertical
	m.panes = []int{0, 1}
	m.paneFocus = 0
	m.activeIdx = 0

	m.focusLocked(1) // b is already in pane 2: swap
	if got := paneNames(m); got[0] != "b" || got[1] != "a" {
		t.Fatalf("panes = %v, want [b a]", got)
	}
	m.focusLocked(2) // c is hidden: replaces the focused pane
	if got := paneNames(m); got[0] != "c" || got[1] != "a" {
		t.Fatalf("panes = %v, want [c a]", got)
	}
}

func TestNextPresetSkipsPresetsNeedingMoreTabs(t *testing.T) {
	seq := []SplitMode{}
	p := SplitNone
	for i := 0; i < 4; i++ {
		p = nextPreset(p, 3)
		seq = append(seq, p)
	}
	want := []SplitMode{SplitVertical, SplitHorizontal, SplitMainStack, SplitNone}
	for i := range want {
		if seq[i] != want[i] {
			t.Fatalf("cycle with 3 tabs = %v, want %v", seq, want)
		}
	}
	if nextPreset(SplitNone, 1) != SplitNone {
		t.Fatalf("one tab cannot split")
	}
}

func TestLayoutGridAndNeighbors(t *testing.T) {
	panes, seps := Layout(SplitGrid, 0, []int{0, 1, 2, 3}, 100, 41)
	if len(panes) != 4 || len(seps) != 2 {
		t.Fatalf("got %d panes, %d separators", len(panes), len(seps))
	}
	// Panes must not overlap and must leave room for their header row.
	for i, a := range panes {
		if a.Row < 2 || a.Width < 1 || a.Height < 1 {
			t.Fatalf("pane %d has bad geometry %+v", i, a)
		}
		for j, b := range panes {
			if i < j && a.Col < b.Col+b.Width && b.Col < a.Col+a.Width &&
				a.Row-1 < b.Row+b.Height && b.Row-1 < a.Row+a.Height {
				t.Fatalf("panes %d and %d overlap: %+v %+v", i, j, a, b)
			}
		}
	}
	if got := neighbor(panes, 0, 'C'); got != 1 {
		t.Fatalf("right of pane 0 = %d, want 1", got)
	}
	if got := neighbor(panes, 0, 'B'); got != 2 {
		t.Fatalf("below pane 0 = %d, want 2", got)
	}
	if got := neighbor(panes, 0, 'A'); got != 0 {
		t.Fatalf("nothing above pane 0, got %d", got)
	}
	ms, _ := Layout(SplitMainStack, 0, []int{0, 1, 2}, 100, 41)
	if got := neighbor(ms, 2, 'D'); got != 0 {
		t.Fatalf("left of bottom stack pane = %d, want main pane 0", got)
	}
}

func TestOpenTabDuplicatesAndFocusServer(t *testing.T) {
	t.Setenv("SHELL", "/bin/cat")
	m := NewManager(true)
	defer m.CloseAll()

	if err := m.OpenTab("h1", "h1", "irteam"); err != nil {
		t.Fatal(err)
	}
	if err := m.OpenTab("h2", "h2", "irteam"); err != nil {
		t.Fatal(err)
	}
	if err := m.OpenTab("h1", "h1", "irteam"); err != nil {
		t.Fatal(err)
	}
	if m.tabs[2].Dup != 2 {
		t.Fatalf("duplicate tab Dup = %d, want 2", m.tabs[2].Dup)
	}

	if !m.FocusServer("h1") || m.activeIdx != 0 {
		t.Fatalf("FocusServer(h1) should focus tab 0, got %d", m.activeIdx)
	}
	if m.FocusServer("nope") {
		t.Fatalf("FocusServer on unknown server should fail")
	}

	if err := m.SwitchUser("irteamsu"); err != nil {
		t.Fatal(err)
	}
	if err := m.SwitchUser("irteam"); err != nil {
		t.Fatal(err)
	}
	if n := len(m.tabs[0].sessions); n != 2 {
		t.Fatalf("switching back should reuse the irteam session; sessions = %d", n)
	}
	if got := m.OpenUsers()["h1"]; len(got) != 2 {
		t.Fatalf("OpenUsers(h1) = %v", got)
	}
}

func TestBroadcastWritesToEveryPane(t *testing.T) {
	t.Setenv("SHELL", "/bin/cat")
	m := NewManager(true)
	defer m.CloseAll()
	for _, n := range []string{"h1", "h2", "h3"} {
		if err := m.OpenTab(n, n, "irteam"); err != nil {
			t.Fatal(err)
		}
	}
	m.HandleInput([]byte{prefixKey})
	m.HandleInput([]byte("|")) // Ctrl+A |: split left/right
	m.HandleInput([]byte{prefixKey})
	m.HandleInput([]byte("b")) // Ctrl+A b: broadcast on
	m.mu.Lock()
	if m.split != SplitVertical || !m.broadcast {
		m.mu.Unlock()
		t.Fatalf("split = %v broadcast = %v after Ctrl+A | and Ctrl+A b", m.split, m.broadcast)
	}
	visible := []*Session{m.tabSessionLocked(m.panes[0]), m.tabSessionLocked(m.panes[1])}
	m.mu.Unlock()

	m.HandleInput([]byte("ping\n"))

	deadline := time.Now().Add(3 * time.Second)
	for _, s := range visible {
		for !vtContains(s, "ping") {
			if time.Now().After(deadline) {
				t.Fatalf("%s never received broadcast input", s.Label())
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func vtContains(s *Session, text string) bool {
	cols, rows := s.VTSize()
	for r := 0; r < rows; r++ {
		line := make([]rune, 0, cols)
		for c := 0; c < cols; c++ {
			line = append(line, s.CellAt(c, r).Char)
		}
		if strings.Contains(string(line), text) {
			return true
		}
	}
	return false
}
