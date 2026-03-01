# AGENTS.md — ssh-tui

This file is intentionally short. Use the documents below.

## Quick Start

```bash
go test ./...
go run ./cmd/ssh-tui
go build -o build/ssh-tui ./cmd/ssh-tui
```

Always build to the `build/` directory. Never place binaries in the repo root.

## Documentation

- Docs index: [docs/README.md](docs/README.md)
- Functional spec:
  - [Product](docs/functional/product.md)
  - [UI](docs/functional/ui.md)
  - [Config](docs/functional/config.md)
  - [known_hosts](docs/functional/known_hosts.md)
  - [SSH](docs/functional/ssh.md)
  - [tmux](docs/functional/tmux.md)
- Engineering:
  - [Structure](docs/engineering/structure.md)
  - [UI Structure](docs/engineering/ui.md)
  - [Build](docs/engineering/build.md)
  - [Limits](docs/engineering/limits.md)

## Repo Structure (high level)

- `cmd/ssh-tui/main.go` — CLI entrypoint + flag parsing
- `cmd/ssh-tui/cmd_connect.go` — `connect host|group` subcommand
- `cmd/ssh-tui/cmd_list.go` — `list hosts|groups` subcommand
- `cmd/ssh-tui/cmd_completion.go` — `completion bash|zsh` + internal `__complete`
- `internal/config` — config schema + load/save (atomic, 0600)
- `internal/hosts` — known_hosts parsing/loading
- `internal/sshcmd` — build ssh argv from merged settings
- `internal/tmux` — tmux detection, argv builders, pane/window helpers
- `internal/ui` — Bubble Tea models/views, styles, keybindings

## Git Workflow

After making file changes: **stop, report what was done, and wait for the next instruction.**
Committing is a separate explicit step — never part of making changes.

| Rule | Requirement |
|------|-------------|
| Trigger | Only run `git commit` when the user's message explicitly contains the word **"commit"** |
| Frequency | One commit per user request maximum — stop after the first commit and wait |
| Branch | Never commit directly to `main` — always work on a feature branch |
| Push | Never run `git push` — leave pushing to the user |

## Limits (short)

- Uses system `ssh` (no SSH protocol implementation).
- Multi-select interactive sessions require tmux.
- Hashed known_hosts entries are ignored.
- No secret management; config stores paths and argv tokens only.
