package tui

import (
	"fmt"
	"strings"
	"time"

	"Karazhan/internal/auth"
	"Karazhan/internal/config"
	"Karazhan/internal/mux"
	"Karazhan/internal/server"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const kerberosCheckInterval = 30 * time.Minute

type screen int

const (
	screenLogin screen = iota
	screenServers
)

// focusArea is the part of the split screen that receives keys.
type focusArea int

const (
	focusList focusArea = iota
	focusTerm
)

// frameInterval coalesces bursts of terminal output into one redraw.
const frameInterval = 16 * time.Millisecond

type appModel struct {
	cfg       *config.Config
	screen    screen
	width     int
	height    int
	principal string
	krbOk     bool

	login      loginModel
	serverList serverListModel
	muxMgr     *mux.Manager

	focus         focusArea
	listCollapsed bool // Ctrl+A e: terminal takes the whole width
	listPrefix    bool // Ctrl+A pressed while the list has focus
}

func NewApp(cfg *config.Config) appModel {
	login := newLoginModel()
	login.mock = cfg.KerberosMock
	sl := newServerListModel()
	sl.cfg = cfg
	return appModel{
		cfg:        cfg,
		login:      login,
		serverList: sl,
		muxMgr:     mux.NewManager(cfg.ServersMock),
	}
}

func (m appModel) Init() tea.Cmd {
	if m.cfg.KerberosMock {
		return tea.Batch(
			func() tea.Msg {
				return kerberosCheckedMsg{status: auth.KerberosStatus{Authenticated: false}}
			},
			tea.WindowSize(),
			waitTermUpdate(m.muxMgr),
		)
	}
	return tea.Batch(
		checkKerberos(),
		tea.WindowSize(),
		waitTermUpdate(m.muxMgr),
	)
}

// --- Kerberos messages ---

type kerberosCheckedMsg struct{ status auth.KerberosStatus }
type kerberosTickMsg struct{}

func checkKerberos() tea.Cmd {
	return func() tea.Msg {
		return kerberosCheckedMsg{status: auth.CheckKerberos()}
	}
}

func scheduleKerberosCheck() tea.Cmd {
	return tea.Tick(kerberosCheckInterval, func(t time.Time) tea.Msg {
		return kerberosTickMsg{}
	})
}

// --- Terminal messages ---

// termUpdateMsg means a session's screen changed or a session ended.
type termUpdateMsg struct{}

func waitTermUpdate(mgr *mux.Manager) tea.Cmd {
	return func() tea.Msg {
		<-mgr.Updates()
		time.Sleep(frameInterval)
		return termUpdateMsg{}
	}
}

// --- Layout ---

// listWidth is the server list panel width (0 when collapsed).
func (m appModel) listWidth() int {
	if m.listCollapsed && m.muxMgr.TabCount() > 0 {
		return 0
	}
	w := m.width * 32 / 100
	if w < 34 {
		w = 34
	}
	if w > 60 {
		w = 60
	}
	if m.width-w-1 < 40 {
		w = m.width / 2
	}
	return w
}

// bodyHeight is the height between the status bar and the help line.
func (m appModel) bodyHeight() int {
	if h := m.height - 2; h > 3 {
		return h
	}
	return 3
}

// termWidth is the width of the terminal area right of the list.
func (m appModel) termWidth() int {
	lw := m.listWidth()
	if lw == 0 {
		return max(m.width, 10)
	}
	return max(m.width-lw-1, 10)
}

// --- Update ---

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	if nm, ok := next.(appModel); ok {
		if nm.muxMgr.TabCount() == 0 {
			nm.focus = focusList
			nm.listCollapsed = false
		}
		nm.muxMgr.SetSize(nm.termWidth(), nm.bodyHeight())
		return nm, cmd
	}
	return next, cmd
}

func (m appModel) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case termUpdateMsg:
		m.muxMgr.Prune()
		return m, waitTermUpdate(m.muxMgr)

	case tea.KeyMsg:
		if m.screen == screenServers && m.focus == focusTerm {
			return m.handleTermKey(msg)
		}
		if m.screen == screenServers && m.listPrefix {
			m.listPrefix = false
			return m.handleListPrefix(msg)
		}
		// Let search mode handle its own keys first
		if m.screen == screenServers && m.serverList.searching {
			var searchCmd tea.Cmd
			m.serverList, searchCmd = m.serverList.Update(msg)
			return m, searchCmd
		}
		switch msg.String() {
		case "ctrl+c":
			m.muxMgr.CloseAll()
			return m, tea.Quit
		case "ctrl+a":
			if m.screen == screenServers {
				m.listPrefix = true
				return m, nil
			}
		case "q":
			if m.screen == screenServers {
				if m.serverList.filter != "" {
					// esc clears filter, q should not quit while filtered
					break
				}
				m.muxMgr.CloseAll()
				return m, tea.Quit
			}
		case "esc":
			if m.screen == screenServers && m.serverList.filter == "" && m.muxMgr.TabCount() > 0 {
				m.focus = focusTerm
				return m, nil
			}
		}

	case kerberosCheckedMsg:
		if msg.status.Authenticated {
			m.principal = msg.status.Principal
			m.krbOk = true
			if m.screen == screenLogin {
				m.screen = screenServers
				return m, tea.Batch(m.loadServers(), scheduleKerberosCheck())
			}
			return m, scheduleKerberosCheck()
		}
		m.krbOk = false
		if m.screen != screenLogin {
			m.screen = screenLogin
			m.login = newLoginModel()
			m.login.mock = m.cfg.KerberosMock
			m.login.err = "Kerberos session expired. Please re-login."
			return m, m.login.passwordInput.Focus()
		}
		return m, m.login.passwordInput.Focus()

	case kerberosTickMsg:
		if m.cfg.KerberosMock {
			return m, scheduleKerberosCheck()
		}
		return m, checkKerberos()

	case loginSuccessMsg:
		m.principal = msg.principal
		m.krbOk = true
		m.screen = screenServers
		return m, tea.Batch(m.loadServers(), scheduleKerberosCheck())
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenLogin:
		m.login, cmd = m.login.Update(msg)
		return m, cmd

	case screenServers:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if !m.serverList.searching {
				switch msg.String() {
				case "r":
					m.serverList.loading = true
					return m, tea.Batch(m.refreshServers(), m.serverList.spinner.Tick)
				case "m":
					if m.muxMgr.TabCount() > 0 {
						m.focus = focusTerm
						return m, nil
					}
				}
			}
		case serverSelectedMsg:
			connectCmd := m.connectSSH(msg.server, msg.mode)
			return m, connectCmd
		}

		m.serverList, cmd = m.serverList.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleTermKey forwards a key to the multiplexer while the terminal area
// has focus; Ctrl+A commands may hand focus back to the list.
func (m appModel) handleTermKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	data := keyToBytes(msg, m.muxMgr.AppCursor())
	switch m.muxMgr.HandleInput(data) {
	case mux.ActionFocusList:
		m.focus = focusList
		m.listCollapsed = false
	case mux.ActionToggleList:
		m.listCollapsed = !m.listCollapsed
	}
	return m, nil
}

// handleListPrefix handles the key after Ctrl+A while the list has focus.
func (m appModel) handleListPrefix(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.muxMgr.TabCount() == 0 {
		return m, nil
	}
	switch msg.String() {
	case "e":
		m.listCollapsed = true
		m.focus = focusTerm
	case "right", "l", "d":
		m.focus = focusTerm
	}
	return m, nil
}

// --- View ---

func (m appModel) View() string {
	switch m.screen {
	case screenLogin:
		return m.login.View()

	case screenServers:
		return m.renderStatusBar() + "\n" + m.renderBody() + "\n" + m.renderHelpLine()
	}

	return "Loading..."
}

// renderBody places the server list and the terminal area side by side.
func (m appModel) renderBody() string {
	height := m.bodyHeight()
	term := strings.Split(m.muxMgr.View(m.focus == focusTerm), "\n")
	lw := m.listWidth()
	if lw == 0 {
		return strings.Join(term, "\n")
	}

	m.serverList.tabCount = m.muxMgr.TabCount()
	m.serverList.openUsers = m.muxMgr.OpenUsers()
	list := strings.Split(m.serverList.View(lw, height), "\n")

	sepStyle := dimStyle
	if m.focus == focusList {
		sepStyle = selectedStyle
	}
	sep := sepStyle.Render("│")

	var b strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(lineAt(list, i))
		b.WriteString(sep)
		b.WriteString(lineAt(term, i))
	}
	return b.String()
}

func (m appModel) renderStatusBar() string {
	krbStatus := krbOkStyle.Render("● authenticated")
	if !m.krbOk {
		krbStatus = krbExpiredStyle.Render("● expired")
	}
	left := statusBarStyle.Render(fmt.Sprintf("🔑 %s  %s", m.principal, krbStatus))
	right := dimStyle.Render(m.serverList.cacheInfo)
	if m.serverList.toast != "" {
		right = successStyle.Render("✓ "+m.serverList.toast) + "  " + right
	}
	gap := m.width - ansi.StringWidth(left) - ansi.StringWidth(right) - 1
	if gap < 1 {
		return fitLine(left, m.width)
	}
	return fitLine(left+strings.Repeat(" ", gap)+right, m.width)
}

// renderHelpLine shows the focused area and its keys on the last line.
func (m appModel) renderHelpLine() string {
	var badge, help string
	if m.focus == focusTerm {
		badge = activeTabStyle.Render("TERM")
		help = "^A ?: help • ^A d/←: server list • ^A e: hide/show list • ^A u: irteam↔irteamsu • ^A c: duplicate • ^A Space: layout • ^A z: zoom • ^A b: broadcast"
	} else {
		m.serverList.tabCount = m.muxMgr.TabCount()
		badge = activeTabStyle.Render("LIST")
		help = m.serverList.helpText()
	}
	return fitLine(badge+" "+helpStyle.Render(help), m.width)
}

// --- Server loading ---

func (m appModel) loadServers() tea.Cmd {
	if m.cfg.ServersMock {
		return func() tea.Msg {
			return serversLoadedMsg{nodes: mockServerData(), cacheInfo: "(mock data)"}
		}
	}
	return func() tea.Msg {
		cachePath := m.cfg.ServerCachePath()
		cached, err := server.LoadCache(cachePath)
		if err == nil && cached != nil {
			info := fmt.Sprintf("(cached: %s)", cached.FetchedAt.Format("2006-01-02 15:04"))
			return serversLoadedMsg{nodes: cached.Nodes, cacheInfo: info}
		}

		if len(m.cfg.API.ESMCodes) == 0 {
			return serversErrorMsg{err: fmt.Errorf("ESM codes not configured. Set api.esm_codes in ~/.karazhan/config.yaml")}
		}

		nodes, err := server.FetchAllServers(m.cfg.API.BaseURL, m.cfg.API.ESMCodes)
		if err != nil {
			return serversErrorMsg{err: err}
		}

		_ = server.SaveCache(cachePath, &server.CachedData{
			FetchedAt: time.Now(),
			ESMCodes:  m.cfg.API.ESMCodes,
			Nodes:     nodes,
		})

		return serversLoadedMsg{nodes: nodes, cacheInfo: "(just fetched)"}
	}
}

func (m appModel) refreshServers() tea.Cmd {
	if m.cfg.ServersMock {
		return func() tea.Msg {
			return serversLoadedMsg{nodes: mockServerData(), cacheInfo: "(mock data)"}
		}
	}
	return func() tea.Msg {
		if len(m.cfg.API.ESMCodes) == 0 {
			return serversErrorMsg{err: fmt.Errorf("ESM codes not configured")}
		}

		nodes, err := server.FetchAllServers(m.cfg.API.BaseURL, m.cfg.API.ESMCodes)
		if err != nil {
			return serversErrorMsg{err: err}
		}

		cachePath := m.cfg.ServerCachePath()
		_ = server.SaveCache(cachePath, &server.CachedData{
			FetchedAt: time.Now(),
			ESMCodes:  m.cfg.API.ESMCodes,
			Nodes:     nodes,
		})

		info := fmt.Sprintf("(refreshed: %s)", time.Now().Format("15:04:05"))
		return serversLoadedMsg{nodes: nodes, cacheInfo: info}
	}
}

// --- SSH + Multiplexer ---

// connectSSH opens the selected server in the multiplexer and focuses the
// terminal. A server that already has a tab is focused instead of opening
// another connection, unless a new tab is requested.
func (m *appModel) connectSSH(srv server.CIServer, mode connectMode) tea.Cmd {
	defaultUser := m.cfg.SSH.DefaultUser
	if defaultUser == "" {
		defaultUser = "irteam"
	}

	var err error
	switch mode {
	case connectNewTab:
		err = m.muxMgr.OpenTab(srv.HostName, srv.HostName, defaultUser)
	case connectSu:
		if m.muxMgr.FocusServer(srv.HostName) {
			err = m.muxMgr.SwitchUser("irteamsu")
		} else {
			err = m.muxMgr.OpenTab(srv.HostName, srv.HostName, "irteamsu")
		}
	default:
		if !m.muxMgr.FocusServer(srv.HostName) {
			err = m.muxMgr.OpenTab(srv.HostName, srv.HostName, defaultUser)
		}
	}
	if err != nil {
		return m.serverList.setToast("✗ connect failed: " + err.Error())
	}
	m.focus = focusTerm
	return nil
}
