package mux

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

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
	vt   vt10x.Terminal // virtual terminal for split-pane rendering

	mu   sync.Mutex
	done bool
	err  error
}

func newSession(id int, name, user, ip string, cols, rows int, mock bool) (*Session, error) {
	var cmd *exec.Cmd
	if mock {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh"
		}
		cmd = exec.Command(shell)
	} else {
		cmd = exec.Command("ssh",
			"-o", "GSSAPIAuthentication=yes",
			"-o", "GSSAPIDelegateCredentials=yes",
			"-o", "StrictHostKeyChecking=no",
			"-l", user,
			ip,
		)
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	s := &Session{
		ID:   id,
		Name: name,
		User: user,
		IP:   ip,
		cmd:  cmd,
		ptmx: ptmx,
		vt:   vt10x.New(vt10x.WithSize(cols, rows)),
	}

	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		s.done = true
		s.err = err
		s.mu.Unlock()
	}()

	return s, nil
}

func (s *Session) IsDone() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

func (s *Session) Resize(rows, cols uint16) {
	_ = pty.Setsize(s.ptmx, &pty.Winsize{Rows: rows, Cols: cols})
	s.mu.Lock()
	s.vt.Resize(int(cols), int(rows))
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
	s.vt.Write(data)
	s.mu.Unlock()
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
