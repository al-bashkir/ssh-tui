package ui

import tmx "github.com/al-bashkir/ssh-tui/internal/tmux"

// tmuxOneWindowOpts is an alias so existing callers in this package compile unchanged.
type tmuxOneWindowOpts = tmx.OneWindowOpts

func tmuxOpenOneWindow(sshCmds [][]string, opts tmuxOneWindowOpts) error {
	return tmx.OpenOneWindow(sshCmds, opts)
}
