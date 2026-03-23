package mux

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/term"
)

const prefixKey = 0x01 // Ctrl+A

// DetachReason indicates why the multiplexer returned control.
type DetachReason int

const (
	DetachNormal DetachReason = iota // Ctrl+A d
	DetachNewTab                     // Ctrl+A c
	DetachEmpty                      // All sessions closed
)

// Manager manages multiple terminal sessions.
type Manager struct {
	mu        sync.Mutex
	sessions  []*Session
	activeIdx int
	nextID    int
	mock      bool

	// Split state
	split      SplitMode
	splitPanes [2]int // session indices for pane 0 and pane 1
	splitFocus int    // 0 or 1

	// Session death notification
	deathCh chan int

	LastDetach DetachReason
}

// Runner implements tea.ExecCommand for integration with Bubble Tea.
type Runner struct {
	Mgr    *Manager
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (r *Runner) Run() error              { return r.Mgr.Run() }
func (r *Runner) SetStdin(in io.Reader)    { r.stdin = in }
func (r *Runner) SetStdout(out io.Writer)  { r.stdout = out }
func (r *Runner) SetStderr(errw io.Writer) { r.stderr = errw }

func NewManager(mock bool) *Manager {
	return &Manager{
		mock:    mock,
		deathCh: make(chan int, 16),
	}
}

func (m *Manager) AddSession(name, user, ip string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, err := newSession(m.nextID, name, user, ip, 80, 24, m.mock)
	if err != nil {
		return err
	}
	m.nextID++
	m.sessions = append(m.sessions, s)
	m.activeIdx = len(m.sessions) - 1
	return nil
}

func (m *Manager) SessionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

func (m *Manager) Runner() *Runner {
	return &Runner{Mgr: m}
}

func (m *Manager) Run() error {
	fd := int(os.Stdin.Fd())

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	cols, rows, err := term.GetSize(fd)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, "\033[2J\033[H")
	defer func() {
		fmt.Fprint(os.Stdout, "\033[r")
		fmt.Fprint(os.Stdout, "\033[2J\033[H")
	}()

	m.applyLayout(rows, cols)

	// Handle SIGWINCH
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	var running atomic.Bool
	running.Store(true)

	go func() {
		for range sigCh {
			if !running.Load() {
				return
			}
			newCols, newRows, err := term.GetSize(fd)
			if err != nil {
				continue
			}
			cols, rows = newCols, newRows
			m.applyLayout(rows, cols)
		}
	}()

	// Output goroutines
	for i := range m.sessions {
		go m.relayOutput(i, &running)
	}

	// Render ticker for split mode (~10fps, non-blocking)
	renderTick := time.NewTicker(100 * time.Millisecond)
	defer renderTick.Stop()

	go func() {
		for range renderTick.C {
			if !running.Load() {
				return
			}
			// Snapshot split state without holding lock during render
			m.mu.Lock()
			isSplit := m.split != SplitNone
			m.mu.Unlock()

			if isSplit {
				// renderSplit reads session data via Session.mu (not Manager.mu)
				m.renderSplit(rows, cols)
				m.drawTabBar(rows, cols)
			}
		}
	}()

	// Session death handler
	go func() {
		for range m.deathCh {
			if !running.Load() {
				return
			}
			m.handleSessionDeath(rows, cols, &running)
		}
	}()

	if m.split == SplitNone {
		m.switchToActive(rows, cols)
	}

	// Input loop: read in chunks (handles multi-byte sequences like ESC [ A)
	inputBuf := make([]byte, 256)
	prefixMode := false
	helpShown := false

	for running.Load() {
		n, err := os.Stdin.Read(inputBuf)
		if err != nil {
			break
		}

		for i := 0; i < n; i++ {
			b := inputBuf[i]

			if helpShown {
				helpShown = false
				m.applyLayout(rows, cols)
				continue
			}

			if prefixMode {
				prefixMode = false

				// Check for ESC [ X arrow sequence
				if b == 0x1b && i+2 < n && inputBuf[i+1] == '[' {
					m.handleArrow(inputBuf[i+2])
					i += 2 // skip [ and direction byte
					continue
				}

				m.handlePrefixKey(b, &running, &helpShown, rows, cols)
				continue
			}

			if b == prefixKey {
				prefixMode = true
				continue
			}

			// Forward to focused session
			m.writeToFocused(inputBuf[i : i+1])
		}
	}

	return nil
}

func (m *Manager) handleArrow(dir byte) {
	if m.split == SplitNone {
		return
	}
	switch dir {
	case 'A', 'B': // Up/Down
		if m.split == SplitHorizontal {
			m.splitFocus = 1 - m.splitFocus
			m.mu.Lock()
			m.activeIdx = m.splitPanes[m.splitFocus]
			m.mu.Unlock()
		}
	case 'C', 'D': // Right/Left
		if m.split == SplitVertical {
			m.splitFocus = 1 - m.splitFocus
			m.mu.Lock()
			m.activeIdx = m.splitPanes[m.splitFocus]
			m.mu.Unlock()
		}
	}
}

func (m *Manager) handlePrefixKey(b byte, running *atomic.Bool, helpShown *bool, rows, cols int) {
	switch b {
	case 'd', 'D':
		m.LastDetach = DetachNormal
		running.Store(false)
	case 'c', 'C':
		m.LastDetach = DetachNewTab
		running.Store(false)
	case 'x', 'X':
		m.closeFocused()
		if len(m.sessions) == 0 {
			m.LastDetach = DetachEmpty
			running.Store(false)
			return
		}
		if m.split != SplitNone && len(m.sessions) < 2 {
			m.split = SplitNone
		}
		m.applyLayout(rows, cols)
	case 'n', ']':
		m.mu.Lock()
		m.activeIdx = (m.activeIdx + 1) % len(m.sessions)
		if m.split != SplitNone {
			m.splitPanes[m.splitFocus] = m.activeIdx
		}
		m.mu.Unlock()
		if m.split == SplitNone {
			m.applyLayout(rows, cols)
		}
	case 'p', '[':
		m.mu.Lock()
		m.activeIdx = (m.activeIdx - 1 + len(m.sessions)) % len(m.sessions)
		if m.split != SplitNone {
			m.splitPanes[m.splitFocus] = m.activeIdx
		}
		m.mu.Unlock()
		if m.split == SplitNone {
			m.applyLayout(rows, cols)
		}

	// Split commands
	case '-':
		m.enterSplit(SplitHorizontal, rows, cols)
	case '|', '\\':
		m.enterSplit(SplitVertical, rows, cols)
	case 'z', 'Z':
		if m.split != SplitNone {
			m.split = SplitNone
			m.applyLayout(rows, cols)
		}

	// Focus switch
	case '\t':
		if m.split != SplitNone {
			m.splitFocus = 1 - m.splitFocus
			m.mu.Lock()
			m.activeIdx = m.splitPanes[m.splitFocus]
			m.mu.Unlock()
		}

	case '?':
		*helpShown = true
		m.drawHelp(rows-1, cols)

	case prefixKey:
		m.writeToFocused([]byte{prefixKey})

	default:
		if b >= '1' && b <= '9' {
			idx := int(b - '1')
			m.mu.Lock()
			if idx < len(m.sessions) {
				m.activeIdx = idx
				if m.split != SplitNone {
					m.splitPanes[m.splitFocus] = idx
				}
			}
			m.mu.Unlock()
			if m.split == SplitNone {
				m.applyLayout(rows, cols)
			}
		}
	}
}

func (m *Manager) handleSessionDeath(rows, cols int, running *atomic.Bool) {
	m.mu.Lock()

	// Remove dead sessions
	var alive []*Session
	for _, s := range m.sessions {
		if s.IsDone() {
			s.Close()
		} else {
			alive = append(alive, s)
		}
	}
	m.sessions = alive

	if len(m.sessions) == 0 {
		m.mu.Unlock()
		m.LastDetach = DetachEmpty
		running.Store(false)
		return
	}

	// Fix indices
	if m.activeIdx >= len(m.sessions) {
		m.activeIdx = len(m.sessions) - 1
	}
	for i := range m.splitPanes {
		if m.splitPanes[i] >= len(m.sessions) {
			m.splitPanes[i] = len(m.sessions) - 1
		}
	}
	if m.split != SplitNone && len(m.sessions) < 2 {
		m.split = SplitNone
	}
	m.mu.Unlock()

	m.applyLayout(rows, cols)
}

func (m *Manager) enterSplit(mode SplitMode, rows, cols int) {
	m.mu.Lock()
	if len(m.sessions) < 2 {
		m.mu.Unlock()
		return
	}
	m.split = mode
	m.splitFocus = 0
	m.splitPanes[0] = m.activeIdx
	m.splitPanes[1] = (m.activeIdx + 1) % len(m.sessions)
	m.mu.Unlock()
	m.applyLayout(rows, cols)
}

func (m *Manager) applyLayout(rows, cols int) {
	if m.split == SplitNone {
		fmt.Fprintf(os.Stdout, "\033[1;%dr", rows-1)
		m.resizeAll(uint16(rows-1), uint16(cols))
		m.drawTabBar(rows, cols)
		m.switchToActive(rows, cols)
	} else {
		fmt.Fprint(os.Stdout, "\033[r")
		panes := Layout(m.split, m.splitFocus, m.splitPanes[0], m.splitPanes[1], cols, rows)
		m.mu.Lock()
		for _, p := range panes {
			if p.SessionIdx >= 0 && p.SessionIdx < len(m.sessions) {
				m.sessions[p.SessionIdx].Resize(uint16(p.Height), uint16(p.Width))
			}
		}
		m.mu.Unlock()
		m.drawTabBar(rows, cols)
	}
}

// relayOutput reads from session PTY and routes to VT + stdout.
func (m *Manager) relayOutput(idx int, running *atomic.Bool) {
	buf := make([]byte, 4096)
	s := m.sessions[idx]
	for running.Load() {
		n, err := s.Read(buf)
		if err != nil {
			// Session died → notify main loop
			select {
			case m.deathCh <- idx:
			default:
			}
			return
		}

		data := buf[:n]
		s.WriteToVT(data)

		m.mu.Lock()
		isSplit := m.split != SplitNone
		isActive := m.activeIdx == idx
		m.mu.Unlock()

		if !isSplit && isActive {
			os.Stdout.Write(data)
		}
	}
}

func (m *Manager) writeToFocused(data []byte) {
	m.mu.Lock()
	var idx int
	if m.split != SplitNone {
		idx = m.splitPanes[m.splitFocus]
	} else {
		idx = m.activeIdx
	}
	if idx >= 0 && idx < len(m.sessions) && !m.sessions[idx].IsDone() {
		s := m.sessions[idx]
		m.mu.Unlock()
		s.Write(data)
		return
	}
	m.mu.Unlock()
}

func (m *Manager) switchToActive(rows, cols int) {
	m.mu.Lock()
	if m.activeIdx >= 0 && m.activeIdx < len(m.sessions) {
		s := m.sessions[m.activeIdx]
		m.mu.Unlock()
		fmt.Fprintf(os.Stdout, "\033[2J\033[H")
		s.Resize(uint16(rows-1), uint16(cols))
		m.drawTabBar(rows, cols)
		return
	}
	m.mu.Unlock()
}

func (m *Manager) resizeAll(rows, cols uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		s.Resize(rows, cols)
	}
}

func (m *Manager) closeFocused() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var idx int
	if m.split != SplitNone {
		idx = m.splitPanes[m.splitFocus]
	} else {
		idx = m.activeIdx
	}

	if idx >= len(m.sessions) {
		return
	}
	s := m.sessions[idx]
	s.Close()
	m.sessions = append(m.sessions[:idx], m.sessions[idx+1:]...)

	if m.activeIdx >= len(m.sessions) && len(m.sessions) > 0 {
		m.activeIdx = len(m.sessions) - 1
	}
	for i := range m.splitPanes {
		if m.splitPanes[i] >= len(m.sessions) && len(m.sessions) > 0 {
			m.splitPanes[i] = len(m.sessions) - 1
		}
	}
}

// CleanDone removes finished sessions.
func (m *Manager) CleanDone() {
	m.mu.Lock()
	defer m.mu.Unlock()
	var alive []*Session
	for _, s := range m.sessions {
		if s.IsDone() {
			s.Close()
		} else {
			alive = append(alive, s)
		}
	}
	m.sessions = alive
	if m.activeIdx >= len(m.sessions) && len(m.sessions) > 0 {
		m.activeIdx = len(m.sessions) - 1
	}
	if m.split != SplitNone && len(m.sessions) < 2 {
		m.split = SplitNone
	}
}

// CloseAll terminates all sessions.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		s.Close()
	}
	m.sessions = nil
	m.split = SplitNone
}
