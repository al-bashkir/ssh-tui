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
| `Enter` | Connect (or cursor host if nothing selected) |
| `Space` | Toggle selection |
| `Ctrl+A` | Select all |
| `Ctrl+D` | Clear selection |
| `o` | Open selected in one tmux window (split panes) |
| `O` | Open in current pane |
| `C` | Connect all hosts in group (groups screen) |
| `Ctrl+O` | Connect with custom remote command |
| `c` | Connect a custom host |
| `Ctrl+H` | Hide / unhide the current host |
| `H` | Show / hide hidden hosts |
| `Ctrl+F` | Focus search bar |
| `Tab` | Toggle focus between search and list |
| `Esc` | Clear search / deselect / back |
| `e` | Edit host config |
| `r` | Reload known_hosts |
| `g` | Switch to groups tab |
| `Ctrl+S` | Settings |
| `?` | Help |
| `q` | Quit |

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

  -config <path>        Path to config.toml (default: XDG config dir)
  -hosts <path>         Path to hosts.toml (default: same dir as config.toml)
  -known-hosts <path>   Extra known_hosts file (repeatable)
  -no-tmux              Disable tmux integration
  -popup                Quit after connecting (for tmux popup use)
  -debug                Enable debug logging
```

#### tmux popup

`-popup` is designed for running ssh-tui inside a tmux popup window. After
hosts are opened in new tmux windows or panes the TUI quits automatically,
closing the popup. Opening in the current pane (`O`) is unaffected.

```bash
# bind a key in tmux.conf to open ssh-tui as a popup
bind-key f display-popup -E 'ssh-tui -popup'
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

Settings live in two files in the same directory (default: `~/.config/ssh-tui/`, respects `$XDG_CONFIG_HOME`):

- **`config.toml`** — application settings and SSH/tmux defaults.
- **`hosts.toml`** — host overrides, groups, and hidden-hosts list.

On first run after upgrading from an older single-file layout, hosts.toml is created automatically from the existing config.toml.

### config.toml

```toml
version = 1

[defaults]
load_known_hosts = true  # when false, host list comes from hosts.toml only
user = ""
port = 22
identity_file = ""
extra_args = []

tmux = "auto"            # auto | force | never
open_mode = "auto"       # auto | current | tmux-window | tmux-pane
tmux_session = "ssh-tui"

pane_split = "vertical"       # horizontal | vertical
pane_layout = "even-vertical" # auto | tiled | even-horizontal | even-vertical | main-horizontal | main-vertical
pane_sync = "on"              # on | off
pane_border_status = "bottom" # off | top | bottom
```

### hosts.toml

```toml
version = 1

# Hosts hidden via Ctrl+H in the TUI (no [[hosts]] entry needed).
hidden_hosts = []

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

Settings are merged in this order: `defaults` (config.toml) → `[[groups]]` override → `[[hosts]]` override.

## Limits

- No SSH protocol implementation — calls system `ssh`.
- Hashed `known_hosts` entries (`|1|...`) are ignored.
- No `~/.ssh/config` parsing — system `ssh` handles that normally.
- Multi-host connections require tmux.
- No secret management; config stores file paths and argv tokens only.
