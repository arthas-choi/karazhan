package tui

import (
	"os/user"
	"strings"

	"Karazhan/internal/auth"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type loginModel struct {
	passwordInput textinput.Model
	err           string
	loggingIn     bool
	mock          bool
}

type loginSuccessMsg struct{ principal string }
type loginFailMsg struct{ err error }

func newLoginModel() loginModel {
	pw := textinput.New()
	pw.Placeholder = "password"
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '•'
	pw.CharLimit = 128
	pw.Focus()

	return loginModel{
		passwordInput: pw,
	}
}

func (m loginModel) Update(msg tea.Msg) (loginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.loggingIn {
				return m, nil
			}
			password := m.passwordInput.Value()
			if password == "" {
				m.err = "Password is required"
				return m, nil
			}
			m.loggingIn = true
			m.err = ""
			if m.mock {
				return m, doMockLogin()
			}
			return m, doLogin(password)
		}

	case loginSuccessMsg:
		m.loggingIn = false
		return m, nil

	case loginFailMsg:
		m.loggingIn = false
		m.err = msg.err.Error()
		m.passwordInput.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.passwordInput, cmd = m.passwordInput.Update(msg)
	return m, cmd
}

func (m loginModel) View() string {
	var b strings.Builder

	title := "🔐 Kerberos Login"
	if m.mock {
		title = "🔐 Kerberos Login (mock)"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	b.WriteString("  Password:\n")
	b.WriteString("  " + m.passwordInput.View() + "\n\n")

	if m.err != "" {
		b.WriteString("  " + errorStyle.Render("✗ "+m.err) + "\n\n")
	}

	if m.loggingIn {
		b.WriteString("  " + dimStyle.Render("Logging in...") + "\n")
	} else {
		b.WriteString(helpStyle.Render("  enter: login • q: quit"))
	}

	return b.String()
}

func doLogin(password string) tea.Cmd {
	return func() tea.Msg {
		if err := auth.Kinit(password); err != nil {
			return loginFailMsg{err: err}
		}
		status := auth.CheckKerberos()
		return loginSuccessMsg{principal: status.Principal}
	}
}

func doMockLogin() tea.Cmd {
	return func() tea.Msg {
		principal := "mock@MOCK.COM"
		if u, err := user.Current(); err == nil {
			principal = u.Username + "@MOCK.COM"
		}
		return loginSuccessMsg{principal: principal}
	}
}
