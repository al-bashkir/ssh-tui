# ssh-tui

[![Release](https://img.shields.io/github/v/release/al-bashkir/ssh-tui)](https://github.com/al-bashkir/ssh-tui/releases/latest)
[![Go version](https://img.shields.io/badge/go-1.25.6-00ADD8?logo=go)](go.mod)

A terminal UI for managing SSH connections. Reads hosts from `known_hosts`, stores groups and per-host overrides in a TOML config, and delegates actual connections to the system `ssh` binary. Optionally integrates with tmux to open multiple connections as windows or panes.

## Install

```bash
go install github.com/bashkir/ssh-tui/cmd/ssh-tui@latest
```

Fedora (COPR): https://copr.fedorainfracloud.org/coprs/al-bashkir/ssh-tui/

```bash
sudo dnf copr enable al-bashkir/ssh-tui
sudo dnf install ssh-tui
```

Or build from source:

```bash
go build -o build/ssh-tui ./cmd/ssh-tui
```

## Usage

### TUI (interactive)

```bash
ssh-tui
```

Launches the full terminal UI: host list with fuzzy search, group management, multi-select, host hiding, and tmux integration.

Key bindings (hosts screen):

| Key | Action |
|---|---|
| `Enter` | Connect |
| `Space` | Toggle selection |
| `o` | Open all selected as tmux windows |
| `O` | Open in current pane |
| `Ctrl+H` | Hide / unhide the current host |
| `H` | Show / hide all hidden hosts |
| `Tab` | Focus search |
| `g` | Groups screen |
| `e` | Edit host config |
| `r` | Reload known_hosts |
| `?` | Help |

### CLI subcommands

```bash
# Connect to a specific host
ssh-tui connect host db01.example.com
ssh-tui c h db01.example.com

# Connect to all hosts in a group
ssh-tui connect group prod
ssh-tui c g prod

# List configured groups
ssh-tui list groups
ssh-tui l g

# List known hosts
ssh-tui list hosts
ssh-tui l h
```

CLI connections use the same settings and tmux logic as the TUI: host overrides, group overrides, `open_mode`, pane layout, etc. are all respected.

### Global flags

Flags must come before the subcommand:

```bash
ssh-tui [flags] [subcommand]

  --config <path>        Path to config.toml (default: XDG config dir)
  --known-hosts <path>   Extra known_hosts file (repeatable)
  --no-tmux              Disable tmux integration
  --debug                Enable debug logging
```

## Shell completion

### zsh

```bash
mkdir -p ~/.zfunc
ssh-tui completion zsh > ~/.zfunc/_ssh_tui
```

Add to `~/.zshrc` (before `compinit`):

```zsh
fpath=(~/.zfunc $fpath)
autoload -Uz compinit && compinit
```

### bash

Add to `~/.bashrc`:

```bash
eval "$(ssh-tui completion bash)"
```

Completion covers subcommands and dynamically loads group/host names from your config.

## Config

Default path: `~/.config/ssh-tui/config.toml` (respects `$XDG_CONFIG_HOME`).

```toml
version = 1

# Hosts hidden via Ctrl+H in the TUI (compact form; no per-host config needed).
hidden_hosts = []

[defaults]
load_known_hosts = true
user = ""
port = 22
identity_file = ""
extra_args = []

tmux = "auto"         # auto | force | never
open_mode = "auto"    # auto | current | tmux-window | tmux-pane
tmux_session = "ssh-tui"

pane_split = "vertical"       # horizontal | vertical
pane_layout = "even-vertical" # auto | tiled | even-horizontal | even-vertical | main-horizontal | main-vertical
pane_sync = "on"              # on | off
pane_border_status = "bottom" # off | top | bottom

[[hosts]]
host = "db01.example.com"
user = "admin"
port = 2222
identity_file = "~/.ssh/db01_ed25519"
extra_args = ["-o", "ServerAliveInterval=30"]
hidden = false  # set true to hide from the list (toggle with Ctrl+H)

[[groups]]
name = "prod"
hosts = ["web1.prod.example.com", "web2.prod.example.com", "[10.0.0.1]:2222"]
user = "deploy"
identity_file = "~/.ssh/prod_ed25519"
open_mode = "tmux-pane"  # override open mode for this group
```

Settings are merged in this order: `defaults` → `[[hosts]]` override → `[[groups]]` override.

## Limits

- No SSH protocol implementation — calls system `ssh`.
- Hashed `known_hosts` entries (`|1|...`) are ignored.
- No `~/.ssh/config` parsing — system `ssh` handles that normally.
- Multi-host connections require tmux.
- No secret management; config stores file paths and argv tokens only.
