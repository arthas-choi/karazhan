# Architecture

## Overview

Karazhan is a Go TUI application built on the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework. It follows the Elm architecture (Model-Update-View) for UI state management and uses goroutines for concurrent SSH session handling.

```
┌─────────────────────────────────────────────────┐
│                    main.go                       │
│              CLI flags & config                  │
└──────────────────────┬──────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────┐
│                  internal/tui                    │
│         Bubble Tea TUI (Elm Architecture)        │
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │  Login   │→ │ Servers  │→ │  UserSelect   │  │
│  │  Screen  │  │  Screen  │  │    Screen     │  │
│  └──────────┘  └──────────┘  └───────┬───────┘  │
└──────────────────────────────────────┼──────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────┐
│                  internal/mux                    │
│            Terminal Multiplexer                   │
│                                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐         │
│  │ Session │  │ Session │  │ Session │  ...     │
│  │  (PTY)  │  │  (PTY)  │  │  (PTY)  │         │
│  └─────────┘  └─────────┘  └─────────┘         │
└─────────────────────────────────────────────────┘

┌──────────────────┐  ┌──────────────────┐
│  internal/auth   │  │ internal/config  │
│    Kerberos      │  │   YAML config    │
└──────────────────┘  └──────────────────┘

┌──────────────────────────────────────────┐
│           internal/server                 │
│  API fetch, caching, grouping models     │
└──────────────────────────────────────────┘
```

## Package Structure

```
internal/
├── auth/           # Kerberos authentication
│   └── kerberos.go # kinit/klist wrappers
├── config/         # Configuration management
│   └── config.go   # YAML load/save, defaults
├── server/         # Server data layer
│   ├── model.go    # Data models, grouping logic
│   └── cache.go    # API fetching, JSON caching
├── tui/            # Terminal UI screens
│   ├── app.go      # Main app model (screen router)
│   ├── login.go    # Kerberos login screen
│   ├── servers.go  # Server list & selection
│   ├── styles.go   # Lipgloss style definitions
│   └── mock.go     # Mock data for development
└── mux/            # Terminal multiplexer
    ├── manager.go  # Session manager (input/output routing)
    ├── session.go  # Individual SSH session (PTY wrapper)
    ├── render.go   # Split-pane compositor
    ├── layout.go   # Pane layout calculations
    └── tabbar.go   # Tab bar & help overlay
```

## Data Flow

### Authentication

```
App Start
  │
  ├─ klist -s (check existing ticket)
  │   ├─ Valid   → extract principal → Server List
  │   └─ Invalid → Login Screen
  │                   │
  │                   ├─ User enters password
  │                   ├─ kinit (acquire ticket)
  │                   ├─ klist (verify & extract principal)
  │                   └─ Success → Server List
  │
  └─ Periodic recheck (every 30 minutes)
```

### Server Loading

```
Server List Screen
  │
  ├─ FetchAllServers()
  │   ├─ Concurrent HTTP GET per ESM code (goroutine + WaitGroup)
  │   ├─ Parse API responses
  │   └─ Merge results
  │
  ├─ SaveCache() → ~/.karazhan/cache/servers.json
  │
  └─ BuildGroups(viewMode)
      ├─ ViewByESM         → group by ESM code
      ├─ ViewByServerGroup → group by server group name
      └─ ViewByPrefix      → group by hostname prefix
```

### SSH Connection

```
Server Selected → User Selected (irteam/irteamsu)
  │
  ├─ ssh -o GSSAPIAuthentication=yes
  │      -o GSSAPIDelegateCredentials=yes
  │      -o StrictHostKeyChecking=no
  │      -l <user> <ip>
  │
  ├─ PTY allocated (creack/pty)
  ├─ Session added to Manager
  └─ Bubble Tea hands control to Multiplexer (tea.Exec)
```

### Multiplexer

```
Manager.Run()
  │
  ├─ Input Loop (main goroutine)
  │   ├─ Read stdin
  │   ├─ Ctrl+A prefix → dispatch command
  │   └─ Forward to focused session PTY
  │
  ├─ Output Relay (per-session goroutine)
  │   ├─ Read from session PTY
  │   ├─ Tab mode: write directly to stdout
  │   └─ Split mode: write to vt10x virtual terminal
  │
  ├─ Render Ticker (split mode, ~10fps)
  │   ├─ Read all virtual terminals
  │   ├─ Composite panes with separators
  │   └─ Write to stdout
  │
  └─ SIGWINCH handler → resize all PTYs
```

## Key Design Decisions

### Bubble Tea + raw exec for multiplexer

The TUI screens (login, server list) run inside Bubble Tea's event loop. When entering the multiplexer, `tea.Exec` is used to hand over raw terminal control to the Manager, bypassing Bubble Tea's abstraction. This allows direct PTY I/O with minimal latency.

### vt10x for split-pane rendering

In split mode, each session writes to a virtual terminal (vt10x) instead of stdout. A render ticker composites all virtual terminals into a single frame, drawing separators between panes. This avoids interleaved output from concurrent sessions.

### Concurrent server fetching

Multiple ESM codes are fetched in parallel using goroutines and `sync.WaitGroup`. Results are merged and cached locally as JSON. The cache serves as a fallback when the API is unreachable.

### Hostname prefix grouping

The `ViewByPrefix` mode uses a regex to strip trailing patterns like `-g0001`, `-c0001`, `-d0001` from hostnames, grouping servers that share the same role but differ in instance number.
