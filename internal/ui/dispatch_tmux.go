package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/sshcmd"
	tmx "github.com/al-bashkir/ssh-tui/internal/tmux"

	tea "github.com/charmbracelet/bubbletea"
)

// dispatchResult holds the outcome of dispatching SSH commands via tmux or direct exec.
type dispatchResult struct {
	// execCmd is set when the caller should exec this command after quitting the TUI.
	execCmd []string
	// quit signals that the TUI should exit (execCmd will be run after).
	quit bool
	// toast is set for in-app feedback (errors or success).
	toast toast
}

func buildSSHCommands(hosts []string, defaults config.Defaults, inv config.Inventory, group *config.Group, modifyFn func(*sshcmd.Settings)) [][]string {
	base := sshcmd.FromDefaults(defaults)
	sshCmds := make([][]string, 0, len(hosts))
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
		sshCmds = append(sshCmds, cmd)
	}
	return sshCmds
}

func resolveConnectMode(defaults config.Defaults, group *config.Group) (tmx.OpenMode, bool) {
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
			return dispatchResult{toast: toast{text: "multi-host requires tmux (window or pane mode)", level: toastWarn}}, nil
		}
		return dispatchResult{execCmd: sshCmds[0], quit: true}, nil
	}

	if !inTmux {
		if len(sshCmds) > 1 {
			return dispatchResult{toast: toast{text: "multi-host requires an active tmux session", level: toastWarn}}, nil
		}
		return dispatchResult{
			execCmd: tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0]),
			quit:    true,
		}, nil
	}

	return dispatchResult{}, func() tea.Msg {
		wName := tmx.GroupWindowName(hostsToOpen, group)

		if mode == tmx.OpenPane || (mode == tmx.OpenWindow && len(sshCmds) > 1) {
			ps := tmx.ResolvePaneSettings(defaults, group, len(sshCmds))
			err := tmx.OpenOneWindow(sshCmds, tmx.OneWindowOpts{
				WindowName:       wName,
				PaneTitles:       hostsToOpen,
				SplitFlag:        ps.SplitFlag,
				Layout:           ps.Layout,
				SyncPanes:        ps.SyncPanes,
				PaneBorderFormat: ps.BorderFormat,
				PaneBorderStatus: ps.BorderStatus,
			})
			if err != nil {
				return toastMsg{text: err.Error(), level: toastErr}
			}
			return toastMsg{text: fmt.Sprintf("opened %d in one window", len(sshCmds)), level: toastInfo}
		}

		for i, sshCmd := range sshCmds {
			name := tmx.GroupWindowName(hostsToOpen[i:i+1], group)
			tmuxCmd := tmx.NewWindowCmd(name, sshCmd)
			// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
			if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
				return toastMsg{text: "tmux error: " + err.Error(), level: toastErr}
			}
		}
		return toastMsg{text: fmt.Sprintf("opened %d", len(sshCmds)), level: toastInfo}
	}
}
