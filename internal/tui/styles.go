package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	serverInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0AF68"))

	// Open-session markers in the server list; irteamsu stands out as privileged.
	userChipStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C4B5FD"))

	suChipStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1B1B1E")).
			Background(lipgloss.Color("#E0AF68")).
			Bold(true)

	krbOkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	krbExpiredStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)
)
