# ssh-tui

[![Release](https://img.shields.io/github/v/release/al-bashkir/ssh-tui)](https://github.com/al-bashkir/ssh-tui/releases/latest)
[![Go version](https://img.shields.io/badge/go-1.25.6-00ADD8?logo=go)](go.mod)

A terminal UI for managing SSH connections. Reads hosts from `known_hosts`, stores groups and per-host overrides in a TOML config, and delegates actual connections to the system `ssh` binary. Optionally integrates with tmux to open multiple connections as windows or panes.

## Install

Homebrew (macOS and Linux):

```bash
brew install al-bashkir/tools/ssh-tui
```

Or `brew tap al-bashkir/tools` and then `brew install ssh-tui`.

Go install:

```bash
go install github.com/al-bashkir/ssh-tui/cmd/ssh-tui@latest
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

### Build RPM

Requires `rpmbuild` and `go`. Version defaults to the value in `.copr/Makefile`
and can be overridden via the environment:

```bash
# SRPM using the default version
make -f .copr/Makefile srpm

# SRPM with an explicit version
VERSION=1.3.0 make -f .copr/Makefile srpm outdir=~/rpmbuild/SRPMS

# Build a local binary RPM from the SRPM
rpmbuild --rebuild ~/rpmbuild/SRPMS/ssh-tui-1.3.0-1.*.src.rpm
```

## Usage

### TUI (interactive)

```bash
ssh-tui
```

Launches the full terminal UI: host list with fuzzy search, group management, multi-select, host hiding, and tmux integration.

Key bindings:

| Key | Action |
|---|---|
| `Enter` | Connect current/selected host, or open selected group |
| `Space` | Toggle host selection |
| `Ctrl+A` | Select all |
| `Ctrl+D` | Clear selection |
| `o` | Open current/selected hosts in one tmux window (split panes) |
| `O` | Open a single host in the current pane |
| `C` | Connect all hosts in the selected group (Groups screen) |
| `Ctrl+O` | Connect with custom remote command |
| `c` | Connect a custom host |
| `a` | Add current/selected hosts to a group, or add hosts to the selected group |
| `Ctrl+H` | Hide / unhide the current host |
| `H` | Show / hide hidden hosts |
| `Ctrl+F`, `/` | Focus search bar |
| `Tab` | Toggle focus between search and list |
| `Esc` | Clear search / deselect / back |
| `e` | Edit host config |
| `r` | Reload known_hosts |
| `Alt+1` | Switch to Hosts tab |
| `Alt+2` | Switch to Groups tab |
| `Alt+3` | Switch to Settings tab |
| `Ctrl+S` | Save forms/settings |
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
ssh-tui list groups -json
ssh-tui l g

# List known hosts
ssh-tui list hosts
ssh-tui list hosts -json
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
colorscheme = ""         # default | dracula | nord | gruvbox | catppuccin | kanagawa
accent_color = ""        # default | blue | cyan | green | amber | red | magenta
load_known_hosts = true  # when false, Hosts list comes from hosts.toml only
user = ""
port = 22
identity_file = ""
extra_args = []

pane_split = "vertical"       # horizontal | vertical
pane_layout = "even-vertical" # auto | tiled | even-horizontal | even-vertical | main-horizontal | main-vertical
pane_sync = "on"              # on | off
pane_border_format = "..."    # selected tmux pane border format
pane_border_formats = []       # user-defined border formats managed by Settings
pane_border_status = "bottom" # off | top | bottom

tmux = "auto"            # auto | force | never
open_mode = "auto"       # auto | current | tmux-window | tmux-pane
tmux_session = "ssh-tui"
confirm_quit = false
connect_confirm_threshold = 5  # ask before connecting to more than N hosts; 0 disables
show_field_help = true
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
port = 22
identity_file = "~/.ssh/prod_ed25519"
extra_args = ["-o", "ServerAliveInterval=30"]
remote_command = ""      # executed as: sh -c '<remote_command>'

pane_split = ""          # optional override; empty inherits defaults
pane_layout = ""
pane_sync = ""
pane_border_format = ""
pane_border_status = ""

tmux = ""                # optional override; empty inherits defaults
open_mode = "tmux-pane"  # override open mode for this group
```

For single-host connections, SSH settings are `defaults` → matching `[[hosts]]` override. For group connections, they are `defaults` → matching `[[hosts]]` override → `[[groups]]` override.

## Limits

- No SSH protocol implementation — calls system `ssh`.
- Hashed `known_hosts` entries (`|1|...`) are ignored.
- No `~/.ssh/config` parsing — system `ssh` handles that normally.
- Multi-host connections require tmux.
- No secret management; config stores file paths and argv tokens only.
