package tui

import (
	"fmt"
	"strings"
	"time"

	"Karazhan/internal/config"
	"Karazhan/internal/server"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type serverListModel struct {
	cfg         *config.Config
	nodes       []server.ESMNode
	viewMode    server.ViewMode
	groups      []server.GroupView
	cursor      int
	expanded    map[int]bool // expanded group index
	flatItems   []flatItem
	loading     bool
	err         string
	filter      string
	searching   bool
	searchInput textinput.Model
	cacheInfo   string
	spinner     spinner.Model
	toast       string
	toastTime   time.Time
	tabCount    int                 // open mux tabs, for help text
	openUsers   map[string][]string // hostname → users with an open session
}

type flatItem struct {
	groupIndex  int
	serverIndex int // -1 for group header
}

type serversLoadedMsg struct {
	nodes     []server.ESMNode
	cacheInfo string
}
type serversErrorMsg struct{ err error }

// connectMode says how a selected server should be opened.
type connectMode int

const (
	connectDefault connectMode = iota // focus the server's tab, or open one as the default user
	connectSu                         // focus the server's tab and switch it to irteamsu
	connectNewTab                     // always open a new tab as the default user
)

type serverSelectedMsg struct {
	server server.CIServer
	mode   connectMode
}
type toastClearMsg struct{}

func newServerListModel() serverListModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = selectedStyle

	ti := textinput.New()
	ti.Placeholder = "hostname, ip, or #tag"
	ti.CharLimit = 64
	ti.Width = 30
	ti.Prompt = "🔍 "

	return serverListModel{
		expanded:    make(map[int]bool),
		spinner:     s,
		searchInput: ti,
		viewMode:    server.ViewByServerGroup, // default
	}
}

func (m *serverListModel) setNodes(nodes []server.ESMNode, cacheInfo string) {
	m.nodes = nodes
	m.cacheInfo = cacheInfo
	m.rebuildGroups()
}

func (m *serverListModel) rebuildGroups() {
	m.groups = server.BuildGroups(m.nodes, m.viewMode, m.bookmarks())
	m.expanded = make(map[int]bool)
	m.cursor = 0
	m.rebuildFlat()
}

func (m *serverListModel) bookmarks() []string {
	if m.cfg == nil {
		return nil
	}
	return m.cfg.Bookmarks
}

func (m *serverListModel) rebuildFlat() {
	m.flatItems = nil
	f := strings.ToLower(m.filter)

	for i, group := range m.groups {
		if len(group.Servers) == 0 {
			continue
		}

		if f != "" {
			groupMatch := strings.Contains(strings.ToLower(group.GroupName), f)
			hasServerMatch := false
			for _, srv := range group.Servers {
				if matchServer(srv, f) {
					hasServerMatch = true
					break
				}
			}
			if !groupMatch && !hasServerMatch {
				continue
			}
			// Auto-expand groups with matching servers
			if hasServerMatch {
				m.expanded[i] = true
			}
		}

		m.flatItems = append(m.flatItems, flatItem{groupIndex: i, serverIndex: -1})
		if m.expanded[i] {
			for j, srv := range group.Servers {
				if f != "" && !matchServer(srv, f) {
					continue
				}
				m.flatItems = append(m.flatItems, flatItem{groupIndex: i, serverIndex: j})
			}
		}
	}
}

func matchServer(srv server.CIServer, f string) bool {
	// [3] Tag search: # prefix
	if strings.HasPrefix(f, "#") {
		tagQuery := strings.TrimPrefix(f, "#")
		if tagQuery == "" {
			return len(srv.AllTags()) > 0
		}
		for _, t := range srv.AllTags() {
			if strings.Contains(strings.ToLower(t), tagQuery) {
				return true
			}
		}
		return false
	}

	// [1] Hostname + [2] IP: unified substring search
	return strings.Contains(strings.ToLower(srv.HostName), f) ||
		strings.Contains(strings.ToLower(srv.IP), f)
}

func (m *serverListModel) setToast(msg string) tea.Cmd {
	m.toast = msg
	m.toastTime = time.Now()
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return toastClearMsg{}
	})
}

func (m *serverListModel) switchViewMode() tea.Cmd {
	m.viewMode = m.viewMode.Next()
	m.rebuildGroups()
	return m.setToast(fmt.Sprintf("View: %s", m.viewMode.Label()))
}

func (m serverListModel) Update(msg tea.Msg) (serverListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Search mode: forward input to textinput
		if m.searching {
			switch msg.String() {
			case "enter":
				m.searching = false
				m.searchInput.Blur()
				return m, nil
			case "esc":
				m.searching = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.filter = ""
				m.expanded = make(map[int]bool)
				m.rebuildFlat()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				m.filter = m.searchInput.Value()
				m.cursor = 0
				m.rebuildFlat()
				return m, cmd
			}
		}

		// Normal mode
		switch msg.String() {
		case "/":
			m.searching = true
			m.searchInput.Focus()
			return m, m.searchInput.Cursor.BlinkCmd()
		case "esc":
			if m.filter != "" {
				m.searchInput.SetValue("")
				m.filter = ""
				m.expanded = make(map[int]bool)
				m.rebuildFlat()
				return m, nil
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.flatItems)-1 {
				m.cursor++
			}
		case "enter", "right", "l":
			if m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				if item.serverIndex == -1 {
					m.expanded[item.groupIndex] = !m.expanded[item.groupIndex]
					m.rebuildFlat()
				} else {
					return m, m.selectServer(item, connectDefault)
				}
			}
		case "s", "o":
			if m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				if item.serverIndex >= 0 {
					mode := connectSu
					if msg.String() == "o" {
						mode = connectNewTab
					}
					return m, m.selectServer(item, mode)
				}
			}
		case "left", "h":
			if m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				if item.serverIndex == -1 {
					m.expanded[item.groupIndex] = false
					m.rebuildFlat()
				}
			}
		case "B":
			if m.cfg != nil && len(m.cfg.Bookmarks) > 0 {
				m.cfg.ClearBookmarks()
				_ = m.cfg.Save()
				if m.viewMode == server.ViewByBookmark {
					m.rebuildGroups()
				}
				toastCmd := m.setToast("☆ All bookmarks cleared")
				return m, toastCmd
			}
		case "b":
			if m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				if item.serverIndex >= 0 {
					srv := m.groups[item.groupIndex].Servers[item.serverIndex]
					added := m.cfg.ToggleBookmark(srv.HostName)
					_ = m.cfg.Save()
					if m.viewMode == server.ViewByBookmark {
						m.rebuildGroups()
					}
					label := "★ Bookmarked: "
					if !added {
						label = "☆ Removed: "
					}
					toastCmd := m.setToast(label + srv.HostName)
					return m, toastCmd
				}
			}
		case "tab":
			cmd := m.switchViewMode()
			return m, cmd
		}

	case serversLoadedMsg:
		m.loading = false
		m.err = ""
		m.setNodes(msg.nodes, msg.cacheInfo)
		totalServers := 0
		for _, g := range m.groups {
			totalServers += len(g.Servers)
		}
		toastCmd := m.setToast(fmt.Sprintf("Loaded %d groups, %d servers", len(m.groups), totalServers))
		return m, toastCmd

	case serversErrorMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil

	case toastClearMsg:
		m.toast = ""
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m serverListModel) selectServer(item flatItem, mode connectMode) tea.Cmd {
	srv := m.groups[item.groupIndex].Servers[item.serverIndex]
	return func() tea.Msg {
		return serverSelectedMsg{server: srv, mode: mode}
	}
}

// View renders the server list panel — view mode tabs, search line and the
// list — as exactly height lines of exactly width columns.
func (m serverListModel) View(width, height int) string {
	lines := []string{m.renderViewModeTabs()}
	switch {
	case m.searching:
		lines = append(lines, " "+m.searchInput.View())
	case m.filter != "":
		lines = append(lines, dimStyle.Render(fmt.Sprintf(" filter: \"%s\"  (esc: clear)", m.filter)))
	default:
		lines = append(lines, dimStyle.Render(" Search: /"))
	}

	switch {
	case m.err != "":
		lines = append(lines, " "+errorStyle.Render("✗ "+m.err), helpStyle.Render(" r: retry"))
	case m.loading && len(m.flatItems) == 0:
		lines = append(lines, fmt.Sprintf(" %s Loading servers...", m.spinner.View()))
	case len(m.flatItems) == 0:
		lines = append(lines, " No servers found. Press 'r' to refresh.")
	default:
		if m.loading {
			lines = append(lines, fmt.Sprintf(" %s Refreshing servers...", m.spinner.View()))
		}
		lines = append(lines, m.listLines(height-len(lines))...)
	}
	return fitBlock(lines, width, height)
}

// helpText is the key help for the status line while the list has focus.
func (m serverListModel) helpText() string {
	if m.searching {
		return "type to search • enter: confirm • esc: cancel"
	}
	help := "enter: connect • s: as irteamsu • o: new tab • /: search • tab: view mode • b/B: bookmark • r: refresh"
	if m.filter != "" {
		help += " • esc: clear filter"
	}
	if m.tabCount > 0 {
		help += fmt.Sprintf(" • esc/m: terminal (%d) • ^A e: hide list", m.tabCount)
	}
	return help + " • q: quit"
}

func (m serverListModel) renderViewModeTabs() string {
	modes := []server.ViewMode{server.ViewByESM, server.ViewByServerGroup, server.ViewByPrefix, server.ViewByBookmark}
	var parts []string
	for _, mode := range modes {
		label := mode.Label()
		if mode == m.viewMode {
			parts = append(parts, activeTabStyle.Render(label))
		} else {
			parts = append(parts, inactiveTabStyle.Render(label))
		}
	}
	return strings.Join(parts, "")
}

// listLines renders the rows of the list that fit in visible lines,
// scrolled so the cursor stays on screen.
func (m serverListModel) listLines(visible int) []string {
	if visible < 1 {
		visible = 1
	}
	startIdx := 0
	if m.cursor >= visible {
		startIdx = m.cursor - visible + 1
	}
	endIdx := startIdx + visible
	if endIdx > len(m.flatItems) {
		endIdx = len(m.flatItems)
	}

	lines := make([]string, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		item := m.flatItems[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "▸ "
		}
		var b strings.Builder
		if item.serverIndex == -1 {
			m.renderGroupHeader(&b, i, item, prefix)
		} else {
			m.renderServerRow(&b, i, item, prefix)
		}
		lines = append(lines, b.String())
	}
	return lines
}

// fitBlock truncates or pads lines to exactly width columns and height rows.
func fitBlock(lines []string, width, height int) string {
	if len(lines) > height {
		lines = lines[:height]
	}
	var b strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(fitLine(lineAt(lines, i), width))
	}
	return b.String()
}

// fitLine truncates or pads one line to exactly width columns.
func fitLine(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if ansi.StringWidth(line) > width {
		line = ansi.Truncate(line, width, "…")
	}
	if w := ansi.StringWidth(line); w < width {
		line += "\x1b[0m" + strings.Repeat(" ", width-w)
	}
	return line
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

func (m serverListModel) renderGroupHeader(b *strings.Builder, i int, item flatItem, prefix string) {
	group := m.groups[item.groupIndex]
	arrow := "▶"
	if m.expanded[item.groupIndex] {
		arrow = "▼"
	}

	count := len(group.Servers)
	label := fmt.Sprintf("%s%s %s", prefix, arrow, group.GroupName)

	badge := ""
	if group.Badge != "" {
		badge = fmt.Sprintf(" [%s]", group.Badge)
	}
	countStr := fmt.Sprintf(" (%d)", count)

	remark := ""
	if group.Remark != "" {
		r := strings.ReplaceAll(group.Remark, "\n", " ")
		if len(r) > 40 {
			r = r[:40] + "…"
		}
		remark = " - " + r
	}

	if i == m.cursor {
		b.WriteString(selectedStyle.Render(label))
	} else {
		b.WriteString(label)
	}
	b.WriteString(tagStyle.Render(badge))
	b.WriteString(dimStyle.Render(countStr))
	b.WriteString(dimStyle.Render(remark))
}

func (m serverListModel) renderServerRow(b *strings.Builder, i int, item flatItem, prefix string) {
	srv := m.groups[item.groupIndex].Servers[item.serverIndex]

	bookmark := ""
	if m.cfg != nil && m.cfg.IsBookmarked(srv.HostName) {
		bookmark = "★ "
	}
	host := fmt.Sprintf("%s    %s%s", prefix, bookmark, srv.HostName)
	ip := fmt.Sprintf(" %s", srv.PrimaryIP())
	idc := ""
	if srv.IDCName != "" {
		idc = fmt.Sprintf(" %s", srv.IDCName)
	}
	state := ""
	if srv.ServerState != "" && srv.ServerState != "가동중" {
		state = fmt.Sprintf(" [%s]", srv.ServerState)
	}

	tags := srv.AllTags()
	tagStr := ""
	if len(tags) > 0 {
		tagStr = " " + renderTags(tags)
	}

	remark := ""
	for _, sg := range srv.ServerGroupList {
		if sg.ServerRemark != "" {
			r := strings.ReplaceAll(sg.ServerRemark, "\n", " ")
			if len(r) > 30 {
				r = r[:30] + "…"
			}
			remark = " " + dimStyle.Render("// "+r)
			break
		}
	}

	if i == m.cursor {
		b.WriteString(selectedStyle.Render(host))
	} else {
		b.WriteString(dimStyle.Render(host))
	}
	for _, u := range m.openUsers[srv.HostName] {
		if u == "irteamsu" {
			b.WriteString(" " + suChipStyle.Render(u))
		} else {
			b.WriteString(" " + userChipStyle.Render(u))
		}
	}
	b.WriteString(serverInfoStyle.Render(ip))
	b.WriteString(dimStyle.Render(idc))
	if state != "" {
		b.WriteString(errorStyle.Render(state))
	}
	b.WriteString(tagStr)
	b.WriteString(remark)
}

func renderTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var parts []string
	for _, t := range tags {
		parts = append(parts, tagStyle.Render("#"+t))
	}
	return strings.Join(parts, " ")
}
