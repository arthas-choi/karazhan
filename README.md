# Karazhan

Kerberos-authenticated SSH server manager with a built-in terminal multiplexer.

Karazhan provides a TUI (Terminal User Interface) for browsing, searching, and connecting to servers fetched from the TIPS API. It supports Kerberos (GSSAPI) authentication and includes a tmux-like multiplexer for managing multiple SSH sessions with tabs and split panes.

## Features

- **Kerberos Authentication** - Integrated `kinit`/`klist` for seamless GSSAPI SSH
- **Server Discovery** - Fetch server lists from TIPS API with local caching
- **Smart Grouping** - View servers by ESM code, server group, or hostname prefix
- **Built-in Multiplexer** - Tabs, horizontal/vertical split panes, focus switching
- **Cross-platform** - Linux (amd64), macOS (amd64, arm64)

## Quick Start

```bash
# Initialize config
karazhan config init

# Edit your ESM codes
vi ~/.karazhan/config.yaml

# Run
karazhan
```

## Installation

### From GitHub Releases

Download the binary for your platform from the [Releases](../../releases) page.

```bash
# macOS (Apple Silicon)
curl -L -o karazhan https://github.com/arthas-choi/karazhan/releases/latest/download/karazhan-darwin-arm64
chmod +x karazhan
sudo mv karazhan /usr/local/bin/

# macOS (Intel)
curl -L -o karazhan https://github.com/arthas-choi/karazhan/releases/latest/download/karazhan-darwin-amd64
chmod +x karazhan
sudo mv karazhan /usr/local/bin/

# Linux
curl -L -o karazhan https://github.com/arthas-choi/karazhan/releases/latest/download/karazhan-linux-amd64
chmod +x karazhan
sudo mv karazhan /usr/local/bin/
```

### Build from Source

Requires Go 1.25+.

```bash
git clone https://github.com/arthas-choi/karazhan.git
cd karazhan
make build
```

## Configuration

Config file: `~/.karazhan/config.yaml`

```yaml
# Skip Kerberos auth for development
kerberos_mock: false

# Use built-in mock server data
servers_mock: false

# TIPS API settings
api:
  base_url: "https://karazhan.com/api/external"
  esm_codes:
    - "NE032765"

# SSH settings
ssh:
  default_user: "irteam"

# Cache directory (default: ~/.karazhan/cache)
# cache_dir: ""
```

### Config Commands

```bash
karazhan config init    # Create sample config
karazhan config show    # Display current config
```

## Usage

### Server List

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` / `l` | Expand group, or connect (focus the server's tab, or open one as the default user, `irteam` unless `default_user` is set) |
| `h` | Collapse group |
| `s` | Connect as `irteamsu` (switches the server's tab if already open) |
| `o` | Always open a new tab as the default user |
| `b` | Toggle bookmark |
| `B` | Clear all bookmarks |
| `Tab` | Switch view mode |
| `r` | Refresh server list |
| `m` / `Esc` | Focus the terminal area (when tabs are open) |
| `Ctrl+A l` / `d` | Focus the terminal area |
| `Ctrl+A e` | Hide the list and focus the terminal area |
| `/` | Search |
| `q` | Quit |

### Multiplexer

The server list and terminal area share the screen. Each tab is one server;
a tab keeps one session per SSH user (`irteam`, `irteamsu`).
All multiplexer commands use the `Ctrl+A` prefix (like tmux).

| Key | Action |
|-----|--------|
| `Ctrl+A d` | Focus the server list |
| `Ctrl+A e` | Show/hide the server list |
| `Ctrl+A c` | Duplicate tab (same server, same user) |
| `Ctrl+A u` | Switch the tab between `irteam` and `irteamsu` |
| `Ctrl+A x` | Close the current user session (tab closes with its last session) |
| `Ctrl+A n` / `]` | Next tab |
| `Ctrl+A p` / `[` | Previous tab |
| `Ctrl+A 1-9` | Switch to tab N |
| `Ctrl+A Space` | Cycle split presets |
| `Ctrl+A -` | Split horizontal |
| `Ctrl+A \|` | Split vertical |
| `Ctrl+A !` | Unsplit |
| `Ctrl+A Tab` | Next pane |
| `Ctrl+A Arrow` | Focus pane (directional) |
| `Ctrl+A z` | Zoom focused pane |
| `Ctrl+A b` | Broadcast input to all panes |
| `Ctrl+A Ctrl+A` | Send a literal `Ctrl+A` |
| `Ctrl+A ?` | Show help |

## Prerequisites

- **Kerberos** - A valid Kerberos setup (`kinit`, `klist` available in PATH)
- **SSH** - OpenSSH client with GSSAPI support
- **Terminal** - A modern terminal emulator with 256-color support

## License

MIT
