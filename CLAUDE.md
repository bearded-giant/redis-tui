# CLAUDE.md

## Project Overview

Redis TUI Manager — a terminal user interface for managing Redis databases, built with Go and Bubble Tea. Alternative to redis-cli with interactive browsing, editing, monitoring, and advanced operations.

## Build & Test Commands

```bash
make build        # Build binary to bin/redis-tui
make test         # Run tests: go test -v ./...
make test-cover   # Tests with coverage report
make lint         # Run go vet
make fmt          # Format code with go fmt
make run          # Run the application
make dev-deps     # Install goreleaser
```

CI runs `go test -v -race -coverprofile=coverage.out ./...` — always ensure tests pass with `-race`.

## Architecture

```
main.go                    # Entry point, CLI flag parsing, config init
internal/
  cmd/                     # Bubble Tea command factories (return tea.Cmd)
  ui/                      # Bubble Tea UI (Model/Update/View pattern)
  redis/                   # Redis client wrapper (standalone + cluster)
  types/                   # Shared type definitions and messages
  db/                      # Config persistence (~/.config/redis-tui/config.json)
  service/                 # Interfaces (ConfigService, RedisService) and DI container
  testutil/                # Test helpers and mock implementations
```

**Bubble Tea message flow**: KeyMsg → handleKeyPress() → Command (tea.Cmd) → async execution → Message (tea.Msg) → Update() → View()

## Code Conventions

- **Go version**: 1.26 (set in go.mod)
- **Package names**: lowercase, single word (`cmd`, `redis`, `types`, `db`, `ui`)
- **Receivers**: short names (`c *Client`, `m Model`, `cfg *Config`)
- **Message types**: PascalCase with `Msg` suffix (`ConnectedMsg`, `KeysLoadedMsg`)
- **Command methods**: return `tea.Cmd` — pattern: `func (c *Commands) Op() tea.Cmd { return func() tea.Msg { ... } }`
- **Error handling**: return `(result, error)` tuples, wrap with `fmt.Errorf("context: %w", err)`
- **Logging**: `log/slog` structured logging (`slog.Error()`, `slog.Warn()`, `slog.Info()`)
- **Dependency injection**: services injected via `Commands{config, redis}` struct; use interfaces from `service/interfaces.go`
- **Thread safety**: `sync.RWMutex` for shared state (see `db/config.go`)

## Testing

- Tests use Go standard `testing` package
- Redis tests use `alicebob/miniredis` for in-memory Redis
- Mock implementations in `internal/testutil/` (`mock_redis.go`, `mock_redis_full.go`)
- Generic assertion helpers: `AssertEqual[T]`, `AssertNoError`, `AssertError`
- Test files live alongside source files (`*_test.go`)

## Key Dependencies

- `charmbracelet/bubbletea` — TUI framework
- `charmbracelet/bubbles` — TUI components
- `charmbracelet/lipgloss` — Terminal styling
- `redis/go-redis/v9` — Redis client
- `kujtimiihoxha/vimtea` — VIM editor component
- `alicebob/miniredis/v2` — In-memory Redis for tests
- `alecthomas/chroma/v2` — Syntax highlighting

## Git Conventions

- **Conventional commits**: `feat:`, `fix:`, `docs:`, `test:`, `perf:`, `refactor:`, `chore:`
- Keep subject line under 50 characters, imperative mood

## Guardrails

- All Redis operations must go through `internal/redis/` — never use `go-redis` directly elsewhere
- Config schema changes in `internal/db/` must be backward-compatible with existing `~/.config/redis-tui/config.json` files
- All new command methods must go through the `Commands` struct with injected services — no global state
- New message types must follow the `Msg` suffix convention and be defined in the appropriate `messages_*.go` file

## Release

- GoReleaser v2.13.1 builds for Linux/macOS/Windows (amd64/arm64)
- Homebrew tap: `davidbudnick/homebrew-tap`
- Version injected via ldflags (`-X main.version=...`)
- CGO disabled (`CGO_ENABLED=0`)
