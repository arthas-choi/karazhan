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
| `Enter` | Expand group or select server |
| `Tab` | Switch view mode (ESM / Group / Prefix) |
| `r` | Refresh server list |
| `m` | Attach to existing sessions |
| `/` | Search |
| `q` | Quit |

### User Selection

After selecting a server, choose the SSH user:

| Key | Action |
|-----|--------|
| `1` | Connect as `irteam` |
| `2` | Connect as `irteamsu` |
| `Esc` | Back to server list |

### Multiplexer

All multiplexer commands use the `Ctrl+A` prefix (like tmux).

| Key | Action |
|-----|--------|
| `Ctrl+A d` | Detach (back to server list) |
| `Ctrl+A c` | New tab |
| `Ctrl+A x` | Close current tab/pane |
| `Ctrl+A n` / `]` | Next tab |
| `Ctrl+A p` / `[` | Previous tab |
| `Ctrl+A 1-9` | Switch to tab N |
| `Ctrl+A -` | Split horizontal |
| `Ctrl+A \|` | Split vertical |
| `Ctrl+A Tab` | Switch focus between panes |
| `Ctrl+A Arrow` | Switch focus (directional) |
| `Ctrl+A z` | Zoom (unsplit) |
| `Ctrl+A ?` | Show help |

## Prerequisites

- **Kerberos** - A valid Kerberos setup (`kinit`, `klist` available in PATH)
- **SSH** - OpenSSH client with GSSAPI support
- **Terminal** - A modern terminal emulator with 256-color support

## License

MIT
