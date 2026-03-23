package tui

import (
	"fmt"
	"time"

	"Karazhan/internal/auth"
	"Karazhan/internal/config"
	"Karazhan/internal/mux"
	"Karazhan/internal/server"

	tea "github.com/charmbracelet/bubbletea"
)

const kerberosCheckInterval = 30 * time.Minute

type screen int

const (
	screenLogin screen = iota
	screenServers
	screenUserSelect
)

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

	selectedServer *server.CIServer
	sshUser        string
}

func NewApp(cfg *config.Config) appModel {
	login := newLoginModel()
	login.mock = cfg.KerberosMock
	return appModel{
		cfg:        cfg,
		login:      login,
		serverList: newServerListModel(),
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
		)
	}
	return tea.Batch(
		checkKerberos(),
		tea.WindowSize(),
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

// --- Mux messages ---

type muxDoneMsg struct{}

// --- Update ---

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.muxMgr.CloseAll()
			return m, tea.Quit
		case "q":
			if m.screen == screenServers {
				m.muxMgr.CloseAll()
				return m, tea.Quit
			}
		case "esc":
			switch m.screen {
			case screenUserSelect:
				m.screen = screenServers
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

	case muxDoneMsg:
		// Returned from multiplexer
		m.muxMgr.CleanDone()
		m.screen = screenServers
		return m, nil
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenLogin:
		m.login, cmd = m.login.Update(msg)
		return m, cmd

	case screenServers:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "r":
				m.serverList.loading = true
				return m, tea.Batch(m.refreshServers(), m.serverList.spinner.Tick)
			case "m":
				// Re-enter multiplexer if sessions exist
				if m.muxMgr.SessionCount() > 0 {
					return m, m.enterMux()
				}
			}
		case serverSelectedMsg:
			m.selectedServer = &msg.server
			m.screen = screenUserSelect
			return m, nil
		}

		m.serverList, cmd = m.serverList.Update(msg)
		return m, cmd

	case screenUserSelect:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "1":
				m.sshUser = "irteam"
				return m, m.connectSSH()
			case "2":
				m.sshUser = "irteamsu"
				return m, m.connectSSH()
			case "q":
				m.screen = screenServers
				return m, nil
			}
		}
	}

	return m, nil
}

// --- View ---

func (m appModel) View() string {
	switch m.screen {
	case screenLogin:
		return m.login.View()

	case screenServers:
		header := m.renderStatusBar()
		return header + "\n\n" + m.serverList.View(m.width, m.height)

	case screenUserSelect:
		if m.selectedServer == nil {
			return "No server selected"
		}
		srv := m.selectedServer
		return fmt.Sprintf(
			"%s\n\n"+
				"  Server: %s\n"+
				"  IP:     %s\n"+
				"  IDC:    %s\n"+
				"  OS:     %s\n\n"+
				"  Select user:\n"+
				"    %s  irteam\n"+
				"    %s  irteamsu\n\n"+
				"%s",
			titleStyle.Render("👤 Select User"),
			selectedStyle.Render(srv.HostName),
			srv.PrimaryIP(),
			srv.IDCName,
			srv.OSName,
			selectedStyle.Render("[1]"),
			selectedStyle.Render("[2]"),
			helpStyle.Render("  1/2: select user • esc/q: back"),
		)
	}

	return "Loading..."
}

func (m appModel) renderStatusBar() string {
	krbStatus := krbOkStyle.Render("● authenticated")
	if !m.krbOk {
		krbStatus = krbExpiredStyle.Render("● expired")
	}
	sessInfo := ""
	if n := m.muxMgr.SessionCount(); n > 0 {
		sessInfo = fmt.Sprintf("  [%d sessions: m to attach]", n)
	}
	return statusBarStyle.Render(fmt.Sprintf(" 🔑 %s  %s%s ", m.principal, krbStatus, sessInfo))
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

func (m appModel) connectSSH() tea.Cmd {
	if m.selectedServer == nil {
		return nil
	}
	srv := m.selectedServer

	// Add session to multiplexer
	err := m.muxMgr.AddSession(srv.HostName, m.sshUser, srv.PrimaryIP())
	if err != nil {
		return func() tea.Msg {
			return muxDoneMsg{}
		}
	}

	return m.enterMux()
}

func (m appModel) enterMux() tea.Cmd {
	return tea.Exec(m.muxMgr.Runner(), func(err error) tea.Msg {
		return muxDoneMsg{}
	})
}
