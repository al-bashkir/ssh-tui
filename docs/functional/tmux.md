# tmux

Detection:

- In tmux when `$TMUX` is set.

Mode settings:

- `defaults.tmux = auto|force|never`
- `defaults.open_mode = auto|current|tmux-window|tmux-pane`

Behaviors:

- Inside tmux: can open a new window or split panes.
- Outside tmux: can create/attach a session (when tmux is enabled by config).

Multi-select:

- `open_mode=current` cannot open multiple interactive sessions; multi-select requires tmux.

One window with panes:

- Opens a single tmux window and splits panes for each host.
- Applies layout/sync and pane border settings from defaults/group.
