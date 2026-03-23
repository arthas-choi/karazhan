package main

import (
	"fmt"
	"os"
	"path/filepath"

	"Karazhan/internal/config"
	"Karazhan/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	args := os.Args[1:]
	configPath := ""

	// Parse flags
	i := 0
	for i < len(args) {
		switch args[i] {
		case "-h", "--help":
			printHelp()
			return
		case "-v", "--version":
			fmt.Printf("karazhan %s\n", version)
			return
		case "--config":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --config requires a path argument")
				os.Exit(1)
			}
			i++
			configPath = args[i]
		default:
			// Subcommand
			if args[i] == "config" {
				handleConfigCmd(args[i+1:], configPath)
				return
			}
			fmt.Fprintf(os.Stderr, "Unknown argument: %s\n", args[i])
			fmt.Fprintln(os.Stderr, "Run 'karazhan --help' for usage.")
			os.Exit(1)
		}
		i++
	}

	// Load config
	var cfg *config.Config
	var err error
	if configPath != "" {
		cfg, err = config.LoadFrom(configPath)
	} else {
		cfg, err = config.Load()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Run TUI
	app := tui.NewApp(cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleConfigCmd(args []string, configPath string) {
	// Parse --config within subcommand args too
	var subcmd string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config":
			if i+1 < len(args) {
				i++
				configPath = args[i]
			}
		default:
			if subcmd == "" {
				subcmd = args[i]
			}
		}
	}

	switch subcmd {
	case "init":
		configInit(configPath)
	case "show":
		configShow(configPath)
	case "":
		fmt.Fprintln(os.Stderr, "Usage: karazhan config [init|show]")
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "Unknown config subcommand: %s\n", subcmd)
		fmt.Fprintln(os.Stderr, "Usage: karazhan config [init|show]")
		os.Exit(1)
	}
}

func configInit(configPath string) {
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Fprintf(os.Stderr, "Config already exists: %s\n", configPath)
		fmt.Fprintln(os.Stderr, "Remove it first or use --config to specify a different path.")
		os.Exit(1)
	}

	// Create directory
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	// Write sample config
	if err := os.WriteFile(configPath, []byte(config.SampleConfig), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config created: %s\n", configPath)
	fmt.Println("Edit esm_codes to add your ESM codes, then run 'karazhan'.")
}

func configShow(configPath string) {
	var cfg *config.Config
	var err error
	var path string

	if configPath != "" {
		cfg, err = config.LoadFrom(configPath)
		path = configPath
	} else {
		cfg, err = config.Load()
		path = config.DefaultConfigPath()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("# Config: %s\n\n", path)
	dump, err := cfg.Dump()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to dump config: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(dump)
}

func printHelp() {
	fmt.Printf(`Karazhan - Kerberos SSH Server Manager TUI (v%s)

Usage:
  karazhan [flags]
  karazhan config <command>

Flags:
  --config PATH    Config file path (default: ~/.karazhan/config.yaml)
  -h, --help       Show this help
  -v, --version    Show version

Commands:
  config init      Create sample config at default path (or --config PATH)
  config show      Display current loaded config

Config Fields (~/.karazhan/config.yaml):
  kerberos_mock    Skip Kerberos auth for development       (default: false)
  servers_mock     Use mock server data for development     (default: false)
  api.base_url     TIPS API base URL
  api.esm_codes    ESM codes to fetch servers from          (list)
  ssh.default_user Default SSH user                         (default: irteam)
  cache_dir        Cache directory for server lists

TUI Keybindings:
  Login:     enter (login) / ctrl+c (quit)
  Servers:   j/k (navigate) / enter (expand/select) / tab (view mode)
             r (refresh) / m (attach sessions) / q (quit)
  User:      1 (irteam) / 2 (irteamsu) / esc (back)

Multiplexer (Ctrl+A prefix):
  Ctrl+A d     Detach (back to server list)
  Ctrl+A c     New tab (select another server)
  Ctrl+A x     Close current tab/pane
  Ctrl+A n/]   Next tab
  Ctrl+A p/[   Previous tab
  Ctrl+A 1-9   Switch to tab N
  Ctrl+A -     Split horizontal (top/bottom)
  Ctrl+A |     Split vertical (left/right)
  Ctrl+A Tab   Switch focus between panes
  Ctrl+A Arrow Switch focus (directional)
  Ctrl+A z     Zoom (unsplit)
  Ctrl+A ?     Help

Quick Start:
  karazhan config init          # Create sample config
  vi ~/.karazhan/config.yaml    # Edit ESM codes
  karazhan                      # Run
`, version)
}
