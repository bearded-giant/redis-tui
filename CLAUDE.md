# CLAUDE.md

## Project Overview

Redis TUI Manager — terminal UI for managing Redis databases, built with Go and Bubble Tea. Alternative to redis-cli with interactive browsing, editing, monitoring, and advanced operations.

Fork of `davidbudnick/redis-tui` at `github.com/bearded-giant/redis-tui`. Adds:
- SSH tunnel support (connect through bastion) — `internal/redis/ssh.go`
- Auto-decoding of binary/structured Redis values (base64, JSON, msgpack, LangGraph JsonPlus) — `internal/decoder/`

## Build & Test Commands

```bash
make build        # Build binary to bin/redis-tui
make test         # Run tests: go test -v -race ./...
make test-cover   # Tests with coverage HTML report
make lint         # Run go vet
make fmt          # Format code with go fmt
make run          # Run the application
make dev-deps     # Install goreleaser
make decode-blob  # Build cmd/decode-blob CLI for one-off blob inspection
```

CI runs `go test -v -race ./...` on push/PR to `main` and a single linux/amd64 build verify. Goreleaser cross-compiles on tag push. New code should include tests; no enforced coverage threshold.

## Architecture

```
main.go                    # Entry point, CLI flag parsing, config init
cmd/
  decode-blob/             # Standalone CLI for inspecting Redis blobs offline
internal/
  cmd/                     # Bubble Tea command factories (return tea.Cmd)
  ui/                      # Bubble Tea UI (Model/Update/View pattern)
  redis/                   # Redis client wrapper (standalone + cluster + SSH tunnel)
  decoder/                 # Auto-detect + decode Redis values (base64/JSON/msgpack/JsonPlus)
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

### Test rules

- **Never suppress errors** — no `_, _ :=` or bare calls that return errors. Every error must be checked, even in test setup code. Use `t.Fatalf("... failed: %v", err)` for setup errors.
- **Config persistence round-trip** — any new config feature must have a test that writes data, reloads from disk via `reloadConfig(t, cfg)`, and asserts every field survived the JSON round-trip. This catches broken JSON tags.
- **Mocks simulate, never re-implement** — mock implementations should return configured values/errors. Never duplicate real business logic in mocks.
- **Test files are exempt from file length limits** — keep all tests for a given source file in a single `*_test.go` file alongside the source (e.g., `operations.go` ↔ `operations_test.go`). Do not split tests across `_extra_test.go` or similarly suffixed files just to stay under a length threshold.

### Test helpers (`internal/db/config_test.go`)

- `newTestConfig(t)` — creates a temp-dir-backed config
- `reloadConfig(t, cfg)` — creates a fresh `NewConfig` from the same file path, forcing a full JSON save/load cycle

## Key Dependencies

- `charmbracelet/bubbletea` — TUI framework
- `charmbracelet/bubbles` — TUI components
- `charmbracelet/lipgloss` — Terminal styling
- `redis/go-redis/v9` — Redis client
- `golang.org/x/crypto/ssh` — SSH tunnel client
- `vmihailenco/msgpack/v5` — msgpack decoder for blob viewer (custom walker handles ext types)
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
- **JSON tags on `types.Connection`** — changing JSON tags (especially to `json:"-"`) will break persistence. Round-trip tests in `config_test.go` enforce this.
- **Password stripping** — `save()` in `config.go` intentionally strips `Connection.Password`, `SSHConfig.Password`, and `SSHConfig.Passphrase` before writing to disk. This is a security invariant tested by `TestConfig_Persistence_PasswordStripping` and `TestConfig_Persistence_SSHPasswordStripping`. Never bypass this.
- **`UpdateConnection` field handling** — `UpdateConnection()` preserves `Group`, `Color`, `Created` from the existing record (those have no form UI). `UseSSH`/`SSHConfig`/`UseTLS`/`TLSConfig`/`UseCluster` come from the incoming caller — caller is responsible for sending the full desired state. Tested by `TestConfig_UpdateConnection_AcceptsSSHFromCaller`.
- **SSH tunneling** — implemented in `internal/redis/ssh.go` (local-listener tunnel via `golang.org/x/crypto/ssh`). Auth precedence: private key + passphrase → password → SSH agent. Strict `~/.ssh/known_hosts` host-key check (no skip flag). UI form on `ScreenSSHTunnel`, reachable via Ctrl+S from Add/Edit Connection. Tunnel + SSH client lifecycle owned by `redis.Client`; `disconnectLocked()` closes them after redis client drains.
- **Blob decoder** — `internal/decoder/` auto-detects format (base64/JSON/msgpack/JsonPlus envelope) and renders inline in the value viewer. `b` keypress on key detail cycles the decode override. Custom msgpack walker (`msgpack_walker.go`) handles ext types as `{_ext_id, _payload}` so LangGraph python-typed checkpoints decode without errors. Pickle types render as `[unsupported: pickle.X blob, N bytes]`. Standalone CLI: `bin/decode-blob <file>`.

## Release

- GoReleaser v2.13.1 builds for Linux/macOS/Windows (amd64/arm64) on tag push (`v*`)
- Homebrew tap: `bearded-giant/homebrew-tap`
- Version injected via ldflags (`-X main.version=...`)
- CGO disabled (`CGO_ENABLED=0`)

## CI / GitHub

Trimmed setup, intentionally minimal:
- `.github/workflows/ci.yml` — test + single-arch build verify on push/PR to `main`
- `.github/workflows/release.yml` — goreleaser on tag `v*`
- No security workflow, no dependabot, no PR/issue templates (deliberately removed)
