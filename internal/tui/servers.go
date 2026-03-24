package tui

import (
	"fmt"
	"strings"
	"time"

	"Karazhan/internal/server"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type serverListModel struct {
	nodes     []server.ESMNode
	viewMode  server.ViewMode
	groups    []server.GroupView
	cursor    int
	expanded  map[int]bool // expanded group index
	flatItems []flatItem
	loading   bool
	err       string
	filter    string
	searching bool
	searchInput textinput.Model
	cacheInfo string
	spinner   spinner.Model
	toast     string
	toastTime time.Time
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
type serverSelectedMsg struct {
	server server.CIServer
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
	m.groups = server.BuildGroups(m.nodes, m.viewMode)
	m.expanded = make(map[int]bool)
	m.cursor = 0
	m.rebuildFlat()
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
					srv := m.groups[item.groupIndex].Servers[item.serverIndex]
					return m, func() tea.Msg {
						return serverSelectedMsg{server: srv}
					}
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

func (m serverListModel) View(width, height int) string {
	var b strings.Builder

	// Title bar with view mode indicator
	b.WriteString(titleStyle.Render("📡 Server List"))
	b.WriteString("  ")
	b.WriteString(m.renderViewModeTabs())
	if m.cacheInfo != "" {
		b.WriteString("  " + dimStyle.Render(m.cacheInfo))
	}
	if m.toast != "" {
		b.WriteString("  " + successStyle.Render("✓ "+m.toast))
	}
	b.WriteString("\n")

	// Search bar
	if m.searching {
		b.WriteString("  " + m.searchInput.View() + "\n")
	} else if m.filter != "" {
		b.WriteString("  " + dimStyle.Render(fmt.Sprintf("filter: \"%s\"  (esc: clear)", m.filter)) + "\n")
	} else {
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString(fmt.Sprintf("  %s Refreshing servers...\n", m.spinner.View()))
		if len(m.flatItems) > 0 {
			b.WriteString("\n")
			m.renderList(&b, width, height-3)
		}
		return b.String()
	}

	if m.err != "" {
		b.WriteString("  " + errorStyle.Render("✗ "+m.err) + "\n")
		b.WriteString(helpStyle.Render("  r: retry • q: quit"))
		return b.String()
	}

	if len(m.flatItems) == 0 {
		b.WriteString("  No servers found. Press 'r' to refresh from API.\n")
		return b.String()
	}

	m.renderList(&b, width, height)

	return b.String()
}

func (m serverListModel) renderViewModeTabs() string {
	modes := []server.ViewMode{server.ViewByESM, server.ViewByServerGroup, server.ViewByPrefix}
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

func (m serverListModel) renderList(b *strings.Builder, width, height int) {
	visibleHeight := height - 6
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	startIdx := 0
	if m.cursor >= visibleHeight {
		startIdx = m.cursor - visibleHeight + 1
	}
	endIdx := startIdx + visibleHeight
	if endIdx > len(m.flatItems) {
		endIdx = len(m.flatItems)
	}

	for i := startIdx; i < endIdx; i++ {
		item := m.flatItems[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "▸ "
		}

		if item.serverIndex == -1 {
			m.renderGroupHeader(b, i, item, prefix)
		} else {
			m.renderServerRow(b, i, item, prefix)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if m.searching {
		b.WriteString(helpStyle.Render("  type to search • enter: confirm • esc: cancel"))
	} else if m.filter != "" {
		b.WriteString(helpStyle.Render("  ↑↓/jk: navigate • enter: expand/select • /: search • esc: clear filter • tab: view mode • r: refresh • q: quit"))
	} else {
		b.WriteString(helpStyle.Render("  ↑↓/jk: navigate • enter: expand/select • /: search • tab: view mode • r: refresh • q: quit"))
	}
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

	host := fmt.Sprintf("%s    %s", prefix, srv.HostName)
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
