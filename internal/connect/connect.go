// Package connect builds SSH commands and dispatches them, either directly or
// through tmux. It is shared by the TUI and the CLI so both behave identically.
package connect

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/sshcmd"
	tmx "github.com/al-bashkir/ssh-tui/internal/tmux"
)

// BuildCommands returns one ssh argv per host, applying settings in
// precedence order: defaults → per-host override → group → modifyFn.
func BuildCommands(hosts []string, defaults config.Defaults, inv config.Inventory, group *config.Group, modifyFn func(*sshcmd.Settings)) [][]string {
	base := sshcmd.FromDefaults(defaults)
	out := make([][]string, 0, len(hosts))
	for _, h := range hosts {
		s := base
		if hc, ok := sshcmd.FindHostConfig(inv.Hosts, h); ok {
			s = sshcmd.ApplyHost(s, hc)
		}
		if group != nil {
			s = sshcmd.ApplyGroup(s, *group)
		}
		if modifyFn != nil {
			modifyFn(&s)
		}
		cmd, _ := sshcmd.BuildCommand(h, s)
		out = append(out, cmd)
	}
	return out
}

// Mode resolves the effective open mode; a group may override the defaults.
// The second return value reports whether we are inside a tmux session.
func Mode(defaults config.Defaults, group *config.Group) (tmx.OpenMode, bool) {
	tmuxSetting := defaults.Tmux
	openModeSetting := defaults.OpenMode
	if group != nil {
		if strings.TrimSpace(group.Tmux) != "" {
			tmuxSetting = group.Tmux
		}
		if strings.TrimSpace(group.OpenMode) != "" {
			openModeSetting = group.OpenMode
		}
	}
	inTmux := tmx.InTmux()
	return tmx.ResolveOpenMode(tmuxSetting, openModeSetting, inTmux), inTmux
}

// NeedsTmux reports whether Open will talk to tmux (and therefore may block).
// Callers that must not stall an event loop use it to move Open off-thread.
func NeedsTmux(mode tmx.OpenMode, inTmux bool) bool {
	return mode != tmx.OpenCurrent && inTmux
}

// Open dispatches sshCmds according to mode. Exactly one of the results is
// meaningful: execCmd is an argv the caller must exec in place of the current
// process, msg describes what was opened in tmux.
func Open(hosts []string, sshCmds [][]string, defaults config.Defaults, group *config.Group, mode tmx.OpenMode, inTmux bool) (execCmd []string, msg string, err error) {
	switch {
	case mode == tmx.OpenCurrent:
		if len(sshCmds) > 1 {
			return nil, "", fmt.Errorf("multi-host requires tmux (set open_mode to tmux-window or tmux-pane)")
		}
		return sshCmds[0], "", nil

	case !inTmux:
		if len(sshCmds) > 1 {
			return nil, "", fmt.Errorf("multi-host requires an active tmux session")
		}
		// NewSessionCmd uses -A so it attaches to an existing session instead of failing.
		return tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0]), "", nil

	case mode == tmx.OpenPane || (mode == tmx.OpenWindow && len(sshCmds) > 1):
		// All hosts as panes in a single new tmux window.
		ps := tmx.ResolvePaneSettings(defaults, group, len(sshCmds))
		if err := tmx.OpenOneWindow(sshCmds, tmx.OneWindowOpts{
			WindowName:       tmx.GroupWindowName(hosts, group),
			PaneTitles:       hosts,
			SplitFlag:        ps.SplitFlag,
			Layout:           ps.Layout,
			SyncPanes:        ps.SyncPanes,
			PaneBorderFormat: ps.BorderFormat,
			PaneBorderStatus: ps.BorderStatus,
		}); err != nil {
			return nil, "", err
		}
		return nil, fmt.Sprintf("opened %d in one window", len(sshCmds)), nil

	default:
		// OpenWindow: one tmux window per host.
		for i, sshCmd := range sshCmds {
			tmuxCmd := tmx.NewWindowCmd(tmx.GroupWindowName(hosts[i:i+1], group), sshCmd)
			// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
			if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
				return nil, "", fmt.Errorf("tmux new-window: %w", err)
			}
		}
		return nil, fmt.Sprintf("opened %d", len(sshCmds)), nil
	}
}
