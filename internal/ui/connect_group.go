package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/sshcmd"
	tmx "github.com/al-bashkir/ssh-tui/internal/tmux"
)

func (m *appModel) connectHosts(hostsToOpen []string, group *config.Group, remoteCommandOverride string) (execCmd []string, toastResult toast, err error) {
	if len(hostsToOpen) == 0 {
		return nil, toast{}, fmt.Errorf("no host selected")
	}

	defaults := m.opts.Config.Defaults
	rc := strings.TrimSpace(remoteCommandOverride)
	var modifyFn func(*sshcmd.Settings)
	if rc != "" {
		modifyFn = func(s *sshcmd.Settings) {
			s.RemoteCommand = rc
		}
	}
	sshCmds := buildSSHCommands(hostsToOpen, defaults, m.opts.Inventory, group, modifyFn)

	mode, inTmux := resolveConnectMode(defaults, group)

	if mode == tmx.OpenCurrent {
		if len(sshCmds) > 1 {
			return nil, toast{}, fmt.Errorf("multi-host requires tmux (window or pane mode)")
		}
		return sshCmds[0], toast{}, nil
	}

	if !inTmux {
		if len(sshCmds) > 1 {
			return nil, toast{}, fmt.Errorf("multi-host requires an active tmux session")
		}
		return tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0]), toast{}, nil
	}

	window := tmx.GroupWindowName(hostsToOpen, group)
	ps := tmx.ResolvePaneSettings(defaults, group, len(sshCmds))
	if mode == tmx.OpenPane || (mode == tmx.OpenWindow && len(sshCmds) > 1) {
		if err := tmx.OpenOneWindow(sshCmds, tmx.OneWindowOpts{
			WindowName:       window,
			PaneTitles:       hostsToOpen,
			SplitFlag:        ps.SplitFlag,
			Layout:           ps.Layout,
			SyncPanes:        ps.SyncPanes,
			PaneBorderFormat: ps.BorderFormat,
			PaneBorderStatus: ps.BorderStatus,
		}); err != nil {
			return nil, toast{}, err
		}
		return nil, toast{text: fmt.Sprintf("opened %d in one window", len(sshCmds)), level: toastInfo}, nil
	}

	for i, sshCmd := range sshCmds {
		name := tmx.GroupWindowName(hostsToOpen[i:i+1], group)
		tmuxCmd := tmx.NewWindowCmd(name, sshCmd)
		// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
		if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
			return nil, toast{}, fmt.Errorf("tmux error: %s", err.Error())
		}
	}
	return nil, toast{text: fmt.Sprintf("opened %d", len(sshCmds)), level: toastInfo}, nil
}
