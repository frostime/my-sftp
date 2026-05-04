# Project Context

**Name**: my-sftp
**Description**: Modern SFTP CLI tool with interactive shell, auto-completion, progress visualization, and concurrent file transfers.
**Repo**: github.com/frostime/my-sftp

## Tech Stack
- Go 1.24+, core deps: `pkg/sftp`, `x/crypto/ssh`, `chzyer/readline`, `schollz/progressbar`, `bmatcuk/doublestar`, `kevinburke/ssh_config`, `x/sync/singleflight`

## Key Paths

| Path | Purpose |
|------|---------|
| `main.go` | Entry point: SSH connection, auth, host key verification |
| `client/` | SFTP client wrapper: file ops, transfer engine, caching |
| `shell/` | Interactive REPL: command parsing, routing, CLI option handling |
| `config/` | SSH config parsing (`~/.ssh/config` + `user@host:port`) |
| `completer/` | TAB auto-completion for commands and paths |

## Conventions

- PascalCase exported, camelCase private
- Errors wrapped with `fmt.Errorf` + context
- Exported symbols require doc comments
- Unified task execution engine (`executeTasks`) — all concurrent transfers go through it
- `singleflight` for idempotent directory creation
- `sync.Pool` for 512KB transfer buffer reuse
- Directory cache: 30s TTL, invalidated on mutation

## Spec-Docs Index

- [Architecture](spec-docs/architecture.md) — Module relationships, data flow map, design decisions
- [SFTP Transfer](spec-docs/sftp-transfer.md) — Transfer pipeline, progress strategy, concurrency model, edge cases
- [CLI Usage](spec-docs/cli-usage.md) — Command grammar, preserve/flatten contract, expected behaviors

## Notes
