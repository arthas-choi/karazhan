package mux

import (
	"fmt"
	"sync"
)

const prefixKey = 0x01 // Ctrl+A

// Action tells the host UI what a key handled by the multiplexer asks for.
type Action int

const (
	ActionNone       Action = iota
	ActionFocusList         // Ctrl+A d, Ctrl+A ← at the left edge, last tab closed
	ActionToggleList        // Ctrl+A e: collapse / expand the server list
)

// Manager manages server tabs, each holding one session per SSH user, and
// renders them into a text frame the host UI places on screen. It does no
// terminal I/O itself.
type Manager struct {
	mu        sync.Mutex
	tabs      []*Tab
	activeIdx int // focused tab
	nextID    int
	mock      bool

	// Split state. In split mode activeIdx is always panes[paneFocus].
	split     SplitMode
	panes     []int // tab index shown in each pane of the preset
	paneFocus int
	zoomed    bool // focused pane temporarily fills the area
	broadcast bool // input goes to every pane of the preset

	// Size of the area the multiplexer draws in, tab bar included.
	cols, rows int

	prefix    bool // Ctrl+A pressed, waiting for the command key
	helpShown bool

	// updates is signalled (coalesced) whenever a session's screen changes
	// or a session ends.
	updates chan struct{}

	// clips holds clipboard text set by sessions (OSC 52) that the host UI
	// has not yet forwarded to the real terminal.
	clips []string
}

func NewManager(mock bool) *Manager {
	return &Manager{
		mock:    mock,
		cols:    80,
		rows:    24,
		updates: make(chan struct{}, 1),
	}
}

// Updates returns a channel that receives a value when the frame should be redrawn.
func (m *Manager) Updates() <-chan struct{} {
	return m.updates
}

func (m *Manager) notify() {
	select {
	case m.updates <- struct{}{}:
	default:
	}
}

// startSessionLocked opens a session and starts its output reader.
// Caller must hold m.mu.
func (m *Manager) startSessionLocked(name, user, host string) (*Session, error) {
	cols, rows := m.contentSizeLocked()
	s, err := newSession(m.nextID, name, user, host, cols, rows, m.mock)
	if err != nil {
		return nil, err
	}
	m.nextID++
	go m.readLoop(s)
	return s, nil
}

// contentSizeLocked is the terminal size of a full-area session.
func (m *Manager) contentSizeLocked() (cols, rows int) {
	rows = m.rows - 1 // tab bar
	if rows < 1 {
		rows = 1
	}
	cols = m.cols
	if cols < 1 {
		cols = 1
	}
	return cols, rows
}

// OpenTab always opens a new tab for the server, connected as user.
// A server that already has tabs gets a "#N" duplicate tab.
func (m *Manager) OpenTab(name, host, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, err := m.startSessionLocked(name, user, host)
	if err != nil {
		return err
	}
	dup := 1
	for _, t := range m.tabs {
		if t.Name == name && t.Dup >= dup {
			dup = t.Dup + 1
		}
	}
	t := &Tab{Name: name, Host: host, Dup: dup}
	t.add(s)
	m.tabs = append(m.tabs, t)
	m.focusLocked(len(m.tabs) - 1)
	m.relayoutLocked(false)
	return nil
}

// FocusServer focuses the first tab open to the server, keeping its user.
// Returns false if the server has no tab.
func (m *Manager) FocusServer(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.tabs {
		if t.Name == name {
			m.focusLocked(i)
			m.relayoutLocked(false)
			return true
		}
	}
	return false
}

// SwitchUser shows the focused tab's session for user, opening a new SSH
// session to the same server if the tab has none for that user yet.
func (m *Manager) SwitchUser(user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.switchUserLocked(user)
}

func (m *Manager) switchUserLocked(user string) error {
	t := m.focusedTabLocked()
	if t == nil {
		return fmt.Errorf("no tab")
	}
	if !t.activate(user) {
		s, err := m.startSessionLocked(t.Name, user, t.Host)
		if err != nil {
			return err
		}
		t.add(s)
	}
	m.relayoutLocked(false)
	return nil
}

// OpenUsers maps each server with an open tab to the users connected to it.
func (m *Manager) OpenUsers() map[string][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string][]string)
	for _, t := range m.tabs {
		for _, u := range t.Users() {
			if !containsString(out[t.Name], u) {
				out[t.Name] = append(out[t.Name], u)
			}
		}
	}
	return out
}

// TakeClipboard returns and clears the clipboard texts sessions have set
// since the last call, oldest first.
func (m *Manager) TakeClipboard() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	clips := m.clips
	m.clips = nil
	return clips
}

func (m *Manager) TabCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tabs)
}

// SetSize sets the drawing area (tab bar included) and resizes sessions.
func (m *Manager) SetSize(cols, rows int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cols == m.cols && rows == m.rows {
		return
	}
	m.cols, m.rows = cols, rows
	m.relayoutLocked(false)
}

// AppCursor reports whether the focused program wants application-mode arrow keys.
func (m *Manager) AppCursor() bool {
	m.mu.Lock()
	s := m.tabSessionLocked(m.activeIdx)
	m.mu.Unlock()
	return s != nil && s.AppCursor()
}

// focusLocked makes tab idx the focused tab. In split mode the tab is shown
// in the focused pane; if another pane already shows it, the two swap.
func (m *Manager) focusLocked(idx int) {
	if m.split != SplitNone && m.paneFocus < len(m.panes) {
		for i, t := range m.panes {
			if t == idx && i != m.paneFocus {
				m.panes[i] = m.panes[m.paneFocus]
			}
		}
		m.panes[m.paneFocus] = idx
	}
	m.activeIdx = idx
}

func (m *Manager) focusedTabLocked() *Tab {
	if m.activeIdx < 0 || m.activeIdx >= len(m.tabs) {
		return nil
	}
	return m.tabs[m.activeIdx]
}

// tabSessionLocked returns the session shown by tab idx, or nil.
func (m *Manager) tabSessionLocked(idx int) *Session {
	if idx < 0 || idx >= len(m.tabs) {
		return nil
	}
	return m.tabs[idx].Current()
}

// singleLocked reports whether one tab fills the area (no split, or zoomed).
func (m *Manager) singleLocked() bool {
	return m.split == SplitNone || m.zoomed
}

// smallerPreset is the fallback when there are too few tabs for a preset.
func smallerPreset(s SplitMode) SplitMode {
	switch s {
	case SplitGrid:
		return SplitMainStack
	case SplitMainStack:
		return SplitVertical
	default:
		return SplitNone
	}
}

// fillPanesLocked makes panes match the preset: the focused tab stays in the
// focused pane, other panes keep their tab when still valid, and empty panes
// take tabs that are not shown yet. Presets that need more tabs than exist
// shrink. Returns true if the preset changed.
func (m *Manager) fillPanesLocked() bool {
	before := m.split
	for m.split != SplitNone && m.split.PaneCount() > len(m.tabs) {
		m.split = smallerPreset(m.split)
	}
	if m.split == SplitNone {
		m.panes = nil
		m.paneFocus = 0
		m.zoomed = false
		m.broadcast = false
		return before != m.split
	}

	n := m.split.PaneCount()
	if m.paneFocus >= n {
		m.paneFocus = 0
	}
	out := make([]int, n)
	used := make(map[int]bool)
	for i := range out {
		out[i] = -1
	}
	if m.activeIdx >= 0 && m.activeIdx < len(m.tabs) {
		out[m.paneFocus] = m.activeIdx
		used[m.activeIdx] = true
	}
	for i := range out {
		if out[i] == -1 && i < len(m.panes) {
			if t := m.panes[i]; t >= 0 && t < len(m.tabs) && !used[t] {
				out[i] = t
				used[t] = true
			}
		}
	}
	for i := range out {
		if out[i] != -1 {
			continue
		}
		for t := range m.tabs {
			if !used[t] {
				out[i] = t
				used[t] = true
				break
			}
		}
	}
	m.panes = out
	m.activeIdx = out[m.paneFocus]
	return before != m.split
}

// setPresetLocked switches the pane layout. The focused tab moves to the first pane.
func (m *Manager) setPresetLocked(p SplitMode) {
	if p.PaneCount() > len(m.tabs) || p == m.split {
		return
	}
	m.split = p
	m.paneFocus = 0
	m.zoomed = false
	m.fillPanesLocked()
	m.relayoutLocked(true)
}

// paneLayoutLocked computes the pane layout below the tab bar (row 1 is the
// tab bar, so pane rows are shifted down by one).
func (m *Manager) paneLayoutLocked() ([]Pane, []Separator) {
	split := m.split
	if m.zoomed {
		split = SplitNone
	}
	panes, seps := Layout(split, m.paneFocus, m.panes, m.cols, m.rows)
	if split == SplitNone {
		panes[0].TabIdx = m.activeIdx
	}
	for i := range panes {
		panes[i].Row++
	}
	for i := range seps {
		seps[i].Row++
	}
	return panes, seps
}

// relayoutLocked resizes sessions to the panes they are shown in. clearVT
// drops their screen content (used when the layout itself changes, where
// old content at the old size would be garbled).
func (m *Manager) relayoutLocked(clearVT bool) {
	resize := func(s *Session, rows, cols int) {
		if clearVT {
			s.ResizeWithClear(uint16(rows), uint16(cols))
		} else {
			s.Resize(uint16(rows), uint16(cols))
		}
	}
	if m.split == SplitNone {
		cols, rows := m.contentSizeLocked()
		for _, t := range m.tabs {
			for _, s := range t.sessions {
				resize(s, rows, cols)
			}
		}
		return
	}
	panes, _ := m.paneLayoutLocked()
	for _, p := range panes {
		if p.TabIdx < 0 || p.TabIdx >= len(m.tabs) {
			continue
		}
		// Size every user session of the pane's tab, so switching user keeps the layout.
		for _, s := range m.tabs[p.TabIdx].sessions {
			if m.zoomed {
				// Zoom is a quick look: keep content, only resize.
				s.Resize(uint16(p.Height), uint16(p.Width))
			} else {
				resize(s, p.Height, p.Width)
			}
		}
	}
}

// readLoop pumps a session's PTY output into its VT buffer for the session's
// whole lifetime and asks for a redraw.
func (m *Manager) readLoop(s *Session) {
	buf := make([]byte, 4096)
	for {
		n, err := s.Read(buf)
		if n > 0 {
			if clips := s.WriteToVT(buf[:n]); len(clips) > 0 {
				m.mu.Lock()
				m.clips = append(m.clips, clips...)
				m.mu.Unlock()
			}
			m.notify()
		}
		if err != nil {
			s.markDone()
			m.notify()
			return
		}
	}
}

// HandleInput processes one key's bytes typed while the terminal area has
// focus: Ctrl+A commands, or input forwarded to the focused session (or to
// every pane while broadcasting).
func (m *Manager) HandleInput(data []byte) Action {
	if len(data) == 0 {
		return ActionNone
	}
	m.mu.Lock()
	if m.helpShown {
		m.helpShown = false
		m.mu.Unlock()
		return ActionNone
	}
	if m.prefix {
		m.prefix = false
		defer m.mu.Unlock()
		// Arrow keys: ESC [ X, ESC O X, or ESC [ 1 ; mod X
		if len(data) >= 3 && data[0] == 0x1b && (data[1] == '[' || data[1] == 'O') {
			return m.handleArrowLocked(data[len(data)-1])
		}
		return m.handlePrefixKeyLocked(data[0])
	}
	if len(data) == 1 && data[0] == prefixKey {
		m.prefix = true
		m.mu.Unlock()
		return ActionNone
	}
	m.mu.Unlock()
	m.writeInput(data)
	return ActionNone
}

// handleArrowLocked moves pane focus in the arrow's direction; going left
// from the leftmost pane hands focus to the server list.
func (m *Manager) handleArrowLocked(dir byte) Action {
	if m.singleLocked() {
		if dir == 'D' {
			return ActionFocusList
		}
		return ActionNone
	}
	panes, _ := m.paneLayoutLocked()
	next := neighbor(panes, m.paneFocus, dir)
	if next == m.paneFocus && dir == 'D' {
		return ActionFocusList
	}
	m.focusPaneLocked(next)
	return ActionNone
}

func (m *Manager) focusPaneLocked(i int) {
	if i < 0 || i >= len(m.panes) {
		return
	}
	m.paneFocus = i
	m.activeIdx = m.panes[i]
}

func (m *Manager) handlePrefixKeyLocked(b byte) Action {
	switch b {
	case 'd', 'D':
		return ActionFocusList
	case 'e', 'E':
		return ActionToggleList

	case 'c', 'C':
		// Duplicate the focused tab: new tab to the same server as the same user.
		t := m.focusedTabLocked()
		if t == nil {
			return ActionNone
		}
		name, host, user := t.Name, t.Host, t.CurrentUser()
		m.mu.Unlock()
		_ = m.OpenTab(name, host, user)
		m.mu.Lock()

	case 'u', 'U':
		// Switch the focused tab between irteam and irteamsu. Both sessions
		// stay open, so switching back returns to the same shell.
		if t := m.focusedTabLocked(); t != nil {
			_ = m.switchUserLocked(otherUser(t.CurrentUser()))
		}

	case 'x', 'X':
		// Close the focused tab's current user session; the tab closes with its last session.
		shrank := m.closeFocusedLocked()
		if len(m.tabs) == 0 {
			return ActionFocusList
		}
		m.relayoutLocked(shrank)

	case 'n', ']':
		if len(m.tabs) > 0 {
			m.focusLocked((m.activeIdx + 1) % len(m.tabs))
			m.relayoutLocked(false)
		}
	case 'p', '[':
		if len(m.tabs) > 0 {
			m.focusLocked((m.activeIdx - 1 + len(m.tabs)) % len(m.tabs))
			m.relayoutLocked(false)
		}

	// Split presets
	case ' ':
		m.setPresetLocked(nextPreset(m.split, len(m.tabs)))
	case '-':
		m.setPresetLocked(SplitHorizontal)
	case '|', '\\':
		m.setPresetLocked(SplitVertical)
	case '!':
		m.setPresetLocked(SplitNone)

	case 'z', 'Z':
		// Toggle zoom: the focused pane fills the area, the layout is kept.
		if m.split != SplitNone {
			m.zoomed = !m.zoomed
			m.relayoutLocked(false)
		}

	case 'b', 'B':
		// Toggle broadcast: typed input goes to every pane of the layout.
		if m.split != SplitNone {
			m.broadcast = !m.broadcast
		}

	case '\t':
		if m.split != SplitNone && !m.zoomed && len(m.panes) > 0 {
			m.focusPaneLocked((m.paneFocus + 1) % len(m.panes))
		}

	case '?':
		m.helpShown = true

	case prefixKey:
		m.mu.Unlock()
		m.writeInput([]byte{prefixKey})
		m.mu.Lock()

	default:
		if b >= '1' && b <= '9' {
			if idx := int(b - '1'); idx < len(m.tabs) {
				m.focusLocked(idx)
				m.relayoutLocked(false)
			}
		}
	}
	return ActionNone
}

// Prune drops finished sessions and the tabs they leave empty.
func (m *Manager) Prune() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pruneDoneLocked() {
		m.relayoutLocked(true)
	}
}

// pruneDoneLocked drops finished sessions and the tabs they leave empty.
// Returns true if the split preset shrank as a result.
func (m *Manager) pruneDoneLocked() bool {
	changed := false
	for i := len(m.tabs) - 1; i >= 0; i-- {
		t := m.tabs[i]
		for _, s := range append([]*Session(nil), t.sessions...) {
			if s.IsDone() {
				s.Close()
				t.remove(s)
			}
		}
		if len(t.sessions) == 0 {
			if m.removeTabLocked(i) {
				changed = true
			}
		}
	}
	return changed
}

// removeTabLocked removes tab idx, shifts the tab indices that follow it
// and refills the panes. Returns true if the split preset shrank.
func (m *Manager) removeTabLocked(idx int) bool {
	m.tabs = append(m.tabs[:idx], m.tabs[idx+1:]...)
	fix := func(i int) int {
		switch {
		case i == idx:
			return -1
		case i > idx:
			return i - 1
		}
		return i
	}
	for p := range m.panes {
		m.panes[p] = fix(m.panes[p])
	}
	m.activeIdx = fix(m.activeIdx)
	if m.activeIdx < 0 || m.activeIdx >= len(m.tabs) {
		// The focused tab closed: focus its neighbour.
		m.activeIdx = idx
		if m.activeIdx >= len(m.tabs) {
			m.activeIdx = len(m.tabs) - 1
		}
		if m.activeIdx < 0 {
			m.activeIdx = 0
		}
		// Keep an already visible tab where it is instead of swapping it in.
		for p, t := range m.panes {
			if t == m.activeIdx && p != m.paneFocus && m.paneFocus < len(m.panes) && m.panes[m.paneFocus] == -1 {
				m.paneFocus = p
			}
		}
	}
	return m.fillPanesLocked()
}

// writeInput sends typed input to the focused session, or to the shown
// session of every pane while broadcasting.
func (m *Manager) writeInput(data []byte) {
	m.mu.Lock()
	var targets []*Session
	if m.broadcast && m.split != SplitNone {
		for _, idx := range m.panes {
			if s := m.tabSessionLocked(idx); s != nil {
				targets = append(targets, s)
			}
		}
	} else if s := m.tabSessionLocked(m.activeIdx); s != nil {
		targets = append(targets, s)
	}
	m.mu.Unlock()
	for _, s := range targets {
		if !s.IsDone() {
			s.Write(data)
		}
	}
}

// closeFocusedLocked closes the focused tab's current session, and the tab if
// it was the last one. Returns true if the split preset shrank.
func (m *Manager) closeFocusedLocked() bool {
	t := m.focusedTabLocked()
	if t == nil {
		return false
	}
	if s := t.Current(); s != nil {
		s.Close()
		t.remove(s)
	}
	if len(t.sessions) == 0 {
		return m.removeTabLocked(m.activeIdx)
	}
	return false
}

// CloseAll terminates all sessions.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tabs {
		for _, s := range t.sessions {
			s.Close()
		}
	}
	m.tabs = nil
	m.split = SplitNone
	m.panes = nil
	m.zoomed = false
	m.broadcast = false
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
