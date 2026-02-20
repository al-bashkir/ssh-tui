# Build

Local development:

```bash
go test ./...
go run ./cmd/ssh-tui
go build -o build/ssh-tui ./cmd/ssh-tui
```

Always build to the `build/` directory; never place binaries in the repo root.

Useful flags:

- `--config <path>`
- `--known-hosts <path>` (repeatable)
- `--no-tmux`
- `--debug`
