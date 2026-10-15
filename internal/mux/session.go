package mux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/hinshun/vt10x"
)

// Session represents a single SSH connection wrapped in a PTY.
type Session struct {
	ID   int
	Name string // short display name (hostname)
	User string
	IP   string

	cmd  *exec.Cmd
	ptmx *os.File
	vt   vt10x.Terminal // virtual terminal the multiplexer renders from
	wide wideFilter     // pads double-width characters for vt

	mu   sync.Mutex
	done bool
	err  error
}

func newSession(id int, name, user, ipOrHostName string, cols, rows int, mock bool) (*Session, error) {
	newCmd := func() *exec.Cmd {
		if mock {
			shell := os.Getenv("SHELL")
			if shell == "" {
				shell = "/bin/sh"
			}
			return exec.Command(shell)
		}
		return exec.Command("ssh",
			"-o", "GSSAPIAuthentication=yes",
			"-o", "GSSAPIDelegateCredentials=yes",
			"-o", "StrictHostKeyChecking=no",
			"-l", user,
			ipOrHostName,
		)
	}

	// macOS occasionally fails to open a just-recycled pty with ENXIO when
	// several are opened and closed in quick succession; retry briefly.
	var cmd *exec.Cmd
	var ptmx *os.File
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		cmd = newCmd()
		ptmx, err = pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
		if !errors.Is(err, syscall.ENXIO) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	s := &Session{
		ID:   id,
		Name: name,
		User: user,
		IP:   ipOrHostName,
		cmd:  cmd,
		ptmx: ptmx,
	}
	s.vt = s.newVT(cols, rows)

	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		s.done = true
		s.err = err
		s.mu.Unlock()
	}()

	return s, nil
}

// markDone flags the session as finished (e.g. its PTY returned an error
// before cmd.Wait noticed the exit).
func (s *Session) markDone() {
	s.mu.Lock()
	s.done = true
	s.mu.Unlock()
}

func (s *Session) IsDone() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

// Resize sets the PTY and VT size; a no-op when the size is unchanged, so
// relayouts don't send the remote program needless SIGWINCHs.
func (s *Session) Resize(rows, cols uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, r := s.vt.Size(); c == int(cols) && r == int(rows) {
		return
	}
	_ = pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
	s.vt.Resize(int(cols), int(rows))
}

// newVT creates the virtual terminal; it answers the program's status
// queries (cursor position etc.) through the PTY like a real terminal would.
func (s *Session) newVT(cols, rows int) vt10x.Terminal {
	return vt10x.New(vt10x.WithSize(cols, rows), vt10x.WithWriter(s.ptmx))
}

// ResizeWithClear resizes and clears the VT buffer.
// Used when transitioning between layouts so stale content is flushed.
func (s *Session) ResizeWithClear(rows, cols uint16) {
	_ = pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
	s.mu.Lock()
	s.vt = s.newVT(int(cols), int(rows))
	s.mu.Unlock()
}

func (s *Session) Read(buf []byte) (int, error) {
	return s.ptmx.Read(buf)
}

func (s *Session) Write(data []byte) (int, error) {
	return s.ptmx.Write(data)
}

// WriteToVT writes data to the virtual terminal buffer.
func (s *Session) WriteToVT(data []byte) {
	s.mu.Lock()
	s.vt.Write(s.wide.apply(data))
	s.mu.Unlock()
}

// AppCursor reports whether the program switched arrow keys to application
// mode (ESC O A instead of ESC [ A), as vim and less do.
func (s *Session) AppCursor() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vt.Mode()&vt10x.ModeAppCursor != 0
}

// CursorVisible reports whether the program shows the cursor.
func (s *Session) CursorVisible() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vt.CursorVisible()
}

// VTSize returns the current VT terminal dimensions (cols, rows).
func (s *Session) VTSize() (cols, rows int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vt.Size()
}

// CellAt returns the glyph at (col, row) from the virtual terminal.
func (s *Session) CellAt(col, row int) vt10x.Glyph {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.vt.Cell(col, row)
}

// CursorPos returns the cursor position from the virtual terminal.
func (s *Session) CursorPos() (col, row int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur := s.vt.Cursor()
	return cur.X, cur.Y
}

func (s *Session) Close() {
	s.ptmx.Close()
	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
}

func (s *Session) Label() string {
	return fmt.Sprintf("%s@%s", s.User, s.Name)
}
