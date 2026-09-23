package ui

import (
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/connect"
	"github.com/al-bashkir/ssh-tui/internal/sshcmd"
	tmx "github.com/al-bashkir/ssh-tui/internal/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// connectActions is the connect behaviour shared by every list screen
// (Hosts, Group Hosts, Groups). Screens embed it, so its fields are promoted
// and referenced as m.toast, m.execCmd, m.confirmConnect, ...
type connectActions struct {
	toast    toast
	execCmd  []string
	quitting bool

	confirmConnect      bool
	confirmConnectHosts []string
	pendingConnectFn    func() tea.Cmd
}

type toastMsg toast

func connectThreshold(d config.Defaults) int {
	if d.ConnectConfirmThreshold < 0 {
		return 5
	}
	return d.ConnectConfirmThreshold
}

// connect dispatches hosts. oneWindow forces every host into a single tmux
// window of panes; remoteCmd, when set, runs on the remote and keeps the
// session open. Above the confirm threshold the dispatch is deferred until
// the user answers the confirm dialog.
func (c *connectActions) connect(hosts []string, opts Options, group *config.Group, oneWindow bool, remoteCmd string) tea.Cmd {
	if len(hosts) == 0 {
		c.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	if oneWindow && !tmx.InTmux() {
		c.toast = toast{text: "requires an active tmux session", level: toastWarn}
		return nil
	}

	defaults := opts.Config.Defaults
	doConnect := func() tea.Cmd {
		mode, inTmux := connect.Mode(defaults, group)
		if oneWindow {
			mode = tmx.OpenPane
		}
		sshCmds := connect.BuildCommands(hosts, defaults, opts.Inventory, group, remoteCmdModifier(remoteCmd))

		if connect.NeedsTmux(mode, inTmux) {
			// tmux work runs in a tea.Cmd so it never blocks the update loop.
			return func() tea.Msg {
				_, msg, err := connect.Open(hosts, sshCmds, defaults, group, mode, inTmux)
				if err != nil {
					return toastMsg{text: err.Error(), level: toastErr}
				}
				return toastMsg{text: msg, level: toastInfo}
			}
		}
		execCmd, _, err := connect.Open(hosts, sshCmds, defaults, group, mode, inTmux)
		if err != nil {
			c.toast = toast{text: err.Error(), level: toastWarn}
			return nil
		}
		c.execCmd = execCmd
		return tea.Quit
	}

	if len(hosts) > connectThreshold(defaults) {
		c.confirmConnect = true
		c.confirmConnectHosts = hosts
		c.pendingConnectFn = doConnect
		return nil
	}
	return doConnect()
}

// connectSameWindow replaces the TUI with a single ssh session in the current pane.
func (c *connectActions) connectSameWindow(hosts []string, opts Options, group *config.Group) tea.Cmd {
	if len(hosts) == 0 {
		c.toast = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	if len(hosts) > 1 {
		c.toast = toast{text: "select single host for same-window connect", level: toastWarn}
		return nil
	}
	c.execCmd = connect.BuildCommands(hosts, opts.Config.Defaults, opts.Inventory, group, nil)[0]
	return tea.Quit
}

// remoteCmdModifier returns the settings tweak that runs cmd on the remote
// host and drops into a shell afterwards, or nil when cmd is empty.
func remoteCmdModifier(cmd string) func(*sshcmd.Settings) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	// Avoid doubling when the caller already includes it.
	if !strings.Contains(cmd, "exec ${SHELL") && !strings.Contains(cmd, "exec $SHELL") {
		cmd += "; exec ${SHELL:-sh}"
	}
	return func(s *sshcmd.Settings) {
		s.ExtraArgs = ensureSSHForceTTY(s.ExtraArgs)
		s.RemoteCommand = cmd
	}
}

func ensureSSHForceTTY(extraArgs []string) []string {
	out := append([]string(nil), extraArgs...)
	for _, a := range out {
		if a == "-t" || a == "-tt" {
			return out
		}
	}
	return append(out, "-t")
}
