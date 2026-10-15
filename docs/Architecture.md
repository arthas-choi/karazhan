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
│  ┌──────────┐    ┌────────────────────────────┐  │
│  │  Login   │ →  │       Servers screen       │  │
│  │  Screen  │    │ ┌──────────┐ ┌───────────┐ │  │
│  └──────────┘    │ │  Server  │ │ Terminal  │ │  │
│                  │ │   list   │ │   area    │ │  │
│                  │ └──────────┘ └─────┬─────┘ │  │
│                  └─────────────────────┼───────┘  │
└────────────────────────────────────────┼─────────┘
                                         │ View() / HandleInput()
                                         ▼
┌─────────────────────────────────────────────────┐
│                  internal/mux                    │
│    Terminal Multiplexer (renders a text frame)   │
│                                                  │
│  ┌──────────────────┐  ┌──────────────────┐     │
│  │  Tab (server A)  │  │  Tab (server B)  │ ... │
│  │ ┌──────┐┌──────┐ │  │ ┌──────┐         │     │
│  │ │irteam││irteam│ │  │ │irteam│         │     │
│  │ │      ││  su  │ │  │ │      │         │     │
│  │ └──────┘└──────┘ │  │ └──────┘         │     │
│  └──────────────────┘  └──────────────────┘     │
│     Session = PTY + vt10x virtual terminal       │
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
│   ├── app.go      # Main app model: screen router, list/terminal focus, layout
│   ├── login.go    # Kerberos login screen
│   ├── servers.go  # Server list panel, search, bookmarks, connect modes
│   ├── keys.go     # Bubble Tea key events → terminal byte sequences
│   ├── styles.go   # Lipgloss style definitions
│   └── mock.go     # Mock data for development
└── mux/            # Terminal multiplexer
    ├── manager.go  # Tabs, panes, Ctrl+A commands, input routing
    ├── tab.go      # Tab = one server, one session per SSH user
    ├── session.go  # SSH session: PTY + vt10x virtual terminal
    ├── widefilter.go # Pads double-width (CJK) characters for vt10x
    ├── layout.go   # Split presets, pane geometry, directional focus
    ├── render.go   # Cell-grid frame compositor (panes, headers, cursor)
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
Server selected in the list
  │
  ├─ Enter / l  → focus the server's tab, or open one as the default user
  ├─ s          → focus the server's tab and switch it to irteamsu
  └─ o          → always open a new tab as the default user
  │
  ├─ Manager.OpenTab / FocusServer / SwitchUser
  │   ├─ ssh -o GSSAPIAuthentication=yes
  │   │      -o GSSAPIDelegateCredentials=yes
  │   │      -o StrictHostKeyChecking=no
  │   │      -l <user> <hostname>
  │   ├─ PTY allocated (creack/pty, sized to the pane)
  │   └─ Session added to the server's Tab
  │
  └─ Keyboard focus moves to the terminal area (same Bubble Tea screen)
```

### Multiplexer

```
Bubble Tea event loop (internal/tui)
  │
  ├─ Key while the terminal area has focus
  │   ├─ keyToBytes() → terminal byte sequence
  │   └─ Manager.HandleInput(bytes)
  │       ├─ Ctrl+A prefix → dispatch command
  │       │   └─ returns an Action (focus list, toggle list) for the host UI
  │       └─ otherwise → write to the focused session's PTY
  │                      (or to every pane while broadcasting)
  │
  ├─ View() → Manager.View(focused)
  │   ├─ Paint the tab bar into a cell-grid frame
  │   ├─ Copy each pane's vt10x screen into the frame
  │   ├─ Draw pane headers, separators, cursor, help overlay
  │   └─ Serialize the frame to a string placed beside the server list
  │
  ├─ WindowSizeMsg → Manager.SetSize → resize PTYs to their panes
  │
  └─ termUpdateMsg (coalesced, ~16 ms)
      └─ Manager.Prune → drop finished sessions and empty tabs

Per-session goroutine (Manager.readLoop)
  ├─ Read from the PTY
  ├─ wideFilter pads double-width characters
  ├─ Write into the session's vt10x virtual terminal
  └─ Signal the Updates channel → Bubble Tea redraws
```

## Key Design Decisions

### Multiplexer rendered inside Bubble Tea

The multiplexer does no terminal I/O of its own. It keeps every session in a
vt10x virtual terminal and composites them into a text frame that the Bubble
Tea view places next to the server list. This lets the list and the terminal
area share one screen with a single focus model, and avoids switching the
real terminal in and out of raw mode. Key events are converted back into
terminal byte sequences (`internal/tui/keys.go`) before being forwarded to a
PTY, honouring application cursor mode for programs like vim.

### Tabs per server, sessions per user

A tab represents one server. Inside it, each SSH user (`irteam`, `irteamsu`)
has at most one session, and only the active one is shown. `Ctrl+A u` switches
between them without closing either shell. The server list shows chips for the
users that have an open session on each host.

### vt10x for every session

Every session writes to a virtual terminal, not just split panes. vt10x gives
each rune one cell, while the remote program counts CJK characters as two
columns, so a filter inserts a padding rune after each double-width character
before it reaches vt10x and the renderer skips it. The virtual terminal is
also given the PTY as its writer so it can answer status queries (cursor
position etc.) the way a real terminal would.

### Split presets instead of free-form splits

Layouts are fixed presets (single, left/right, top/bottom, main+stack, 2x2)
cycled with `Ctrl+A Space`. A preset that needs more tabs than exist shrinks
automatically. Zoom temporarily shows the focused pane full-size while keeping
the layout, and broadcast sends typed input to every pane.

### Concurrent server fetching

Multiple ESM codes are fetched in parallel using goroutines and `sync.WaitGroup`. Results are merged and cached locally as JSON. The cache serves as a fallback when the API is unreachable.

### Hostname prefix grouping

The `ViewByPrefix` mode uses a regex to strip trailing patterns like `-g0001`, `-c0001`, `-d0001` from hostnames, grouping servers that share the same role but differ in instance number.
