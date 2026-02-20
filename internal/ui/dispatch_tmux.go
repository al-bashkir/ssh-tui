package ui

import (
	"fmt"
	"os/exec"

	"github.com/bashkir/ssh-tui/internal/config"
	tmx "github.com/bashkir/ssh-tui/internal/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// dispatchResult holds the outcome of dispatching SSH commands via tmux or direct exec.
type dispatchResult struct {
	// execCmd is set when the caller should exec this command after quitting the TUI.
	execCmd []string
	// quit signals that the TUI should exit (execCmd will be run after).
	quit bool
	// toast is set for in-app feedback (errors or success).
	toast string
}

// dispatchConnect dispatches SSH commands based on the resolved open mode.
// It handles OpenCurrent (single host, direct exec), new-session (not in tmux),
// and in-tmux modes (pane, window, per-window).
//
// For async tmux operations it returns a tea.Cmd; otherwise result fields are set directly.
func dispatchConnect(
	hostsToOpen []string,
	sshCmds [][]string,
	defaults config.Defaults,
	group *config.Group,
	mode tmx.OpenMode,
	inTmux bool,
) (result dispatchResult, cmd tea.Cmd) {
	if mode == tmx.OpenCurrent {
		if len(sshCmds) > 1 {
			return dispatchResult{toast: "multi-host requires tmux (window or pane mode)"}, nil
		}
		return dispatchResult{execCmd: sshCmds[0], quit: true}, nil
	}

	if !inTmux {
		if len(sshCmds) > 1 {
			return dispatchResult{toast: "multi-host requires an active tmux session"}, nil
		}
		return dispatchResult{
			execCmd: tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0]),
			quit:    true,
		}, nil
	}

	return dispatchResult{}, func() tea.Msg {
		wName := tmuxWindowName(hostsToOpen, group)

		if mode == tmx.OpenPane || (mode == tmx.OpenWindow && len(sshCmds) > 1) {
			ps := resolvePaneSettings(defaults, group, len(sshCmds))
			err := tmuxOpenOneWindow(sshCmds, tmuxOneWindowOpts{
				WindowName:       wName,
				PaneTitles:       hostsToOpen,
				SplitFlag:        ps.SplitFlag,
				Layout:           ps.Layout,
				SyncPanes:        ps.SyncPanes,
				PaneBorderFormat: ps.BorderFormat,
				PaneBorderStatus: ps.BorderStatus,
			})
			if err != nil {
				return toastMsg(err.Error())
			}
			return toastMsg(fmt.Sprintf("opened %d in one window", len(sshCmds)))
		}

		for i, sshCmd := range sshCmds {
			name := tmuxWindowName(hostsToOpen[i:i+1], group)
			tmuxCmd := tmx.NewWindowCmd(name, sshCmd)
			// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
			if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
				return toastMsg("tmux error: " + err.Error())
			}
		}
		return toastMsg(fmt.Sprintf("opened %d", len(sshCmds)))
	}
}

// tmuxWindowName returns a name for the tmux window, preferring the group name if set.
func tmuxWindowName(hosts []string, group *config.Group) string {
	return tmx.GroupWindowName(hosts, group)
}
