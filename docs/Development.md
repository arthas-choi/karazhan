# Development Guide

## Prerequisites

- Go 1.25+
- Kerberos client (`kinit`, `klist`) - optional if using mock mode
- Make

## Getting Started

```bash
git clone https://github.com/<owner>/karazhan.git
cd karazhan

# Build and run
make run

# Or build only
make build
./karazhan
```

## Project Structure

```
.
├── main.go              # Entry point, CLI parsing
├── go.mod               # Go module definition
├── Makefile             # Build automation
├── internal/
│   ├── auth/            # Kerberos authentication
│   ├── config/          # YAML configuration
│   ├── server/          # Server models, API, caching
│   ├── tui/             # Bubble Tea UI screens
│   └── mux/             # Terminal multiplexer
├── docs/                # Documentation
└── .github/workflows/   # CI/CD
```

## Build Commands

| Command | Description |
|---------|-------------|
| `make build` | Build for current platform |
| `make build-linux` | Build for Linux amd64 |
| `make build-mac` | Build for macOS (arm64 + amd64) |
| `make build-all` | Build for all platforms |
| `make clean` | Remove build artifacts |
| `make run` | Build and run |
| `make vet` | Static analysis |
| `make test` | Run tests |

## Development Mode

Karazhan supports mock modes for developing without Kerberos or API access.

### Mock Configuration

Edit `~/.karazhan/config.yaml`:

```yaml
# Skip Kerberos authentication
kerberos_mock: true

# Use built-in mock server data instead of TIPS API
servers_mock: true
```

- `kerberos_mock: true` - Bypasses `kinit`/`klist`, uses a fake principal
- `servers_mock: true` - Returns hardcoded server data from `internal/tui/mock.go`

### Custom Config Path

```bash
karazhan --config ./dev-config.yaml
```

## Version Injection

Version is injected at build time from git tags:

```bash
git tag v1.0.0
make build
./karazhan --version
# karazhan v1.0.0
```

The `Makefile` uses:
```
-ldflags "-s -w -X main.version=$(VERSION)"
```

Where `VERSION` comes from `git describe --tags --always --dirty`.

## Key Dependencies

| Package | Purpose |
|---------|---------|
| [bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework (Elm architecture) |
| [bubbles](https://github.com/charmbracelet/bubbles) | TUI components (text input, spinner) |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| [pty](https://github.com/creack/pty) | Pseudo-terminal allocation |
| [vt10x](https://github.com/hinshun/vt10x) | VT100 terminal emulation |
| [yaml.v3](https://gopkg.in/yaml.v3) | YAML config parsing |

## Adding a New Screen

1. Create a new file in `internal/tui/` (e.g., `settings.go`)
2. Add a new screen constant in `app.go`
3. Implement the screen's `Update` and `View` logic
4. Add routing in `app.go`'s `Update` and `View` methods

## Adding a New Multiplexer Command

1. Open `internal/mux/manager.go`
2. Add a case in the `handlePrefix` method
3. Implement the handler function
4. Update the help text in `internal/mux/tabbar.go`

## Modifying Server Grouping

Server grouping logic is in `internal/server/model.go`:

- `ViewMode` enum defines available modes
- `BuildGroups()` dispatches to the appropriate grouping function
- Add a new `ViewMode` constant and a corresponding `groupBy*()` function
- Update `internal/tui/servers.go` to include the new mode in tab cycling
