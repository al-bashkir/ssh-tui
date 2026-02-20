package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/bashkir/ssh-tui/internal/sshcmd"
	tmx "github.com/bashkir/ssh-tui/internal/tmux"
)

func (m *appModel) connectHostsWithDefaults(hostsToOpen []string) (execCmd []string, toast string, err error) {
	if len(hostsToOpen) == 0 {
		return nil, "", fmt.Errorf("no host selected")
	}

	defaults := m.opts.Config.Defaults
	base := sshcmd.FromDefaults(defaults)
	sshCmds := make([][]string, 0, len(hostsToOpen))
	for _, h := range hostsToOpen {
		s := base
		if hc, ok := hostConfigFor(m.opts.Config, h); ok {
			s = sshcmd.ApplyHost(s, hc)
		}
		cmd, _ := sshcmd.BuildCommand(h, s)
		sshCmds = append(sshCmds, cmd)
	}

	inTmux := tmx.InTmux()
	mode := tmx.ResolveOpenMode(defaults.Tmux, defaults.OpenMode, inTmux)

	if mode == tmx.OpenCurrent {
		if len(sshCmds) > 1 {
			return nil, "", fmt.Errorf("multi-host requires tmux (window or pane mode)")
		}
		return sshCmds[0], "", nil
	}

	if !inTmux {
		if len(sshCmds) > 1 {
			return nil, "", fmt.Errorf("multi-host requires an active tmux session")
		}
		return tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0]), "", nil
	}

	window := windowName(hostsToOpen[0])
	ps := resolvePaneSettings(defaults, nil, len(sshCmds))
	if mode == tmx.OpenPane || (mode == tmx.OpenWindow && len(sshCmds) > 1) {
		if err := tmuxOpenOneWindow(sshCmds, tmuxOneWindowOpts{
			WindowName:       window,
			PaneTitles:       hostsToOpen,
			SplitFlag:        ps.SplitFlag,
			Layout:           ps.Layout,
			SyncPanes:        ps.SyncPanes,
			PaneBorderFormat: ps.BorderFormat,
			PaneBorderStatus: ps.BorderStatus,
		}); err != nil {
			return nil, "", err
		}
		return nil, fmt.Sprintf("opened %d in one window", len(sshCmds)), nil
	}

	for i, sshCmd := range sshCmds {
		tmuxCmd := tmx.NewWindowCmd(windowName(hostsToOpen[i]), sshCmd)
		// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
		if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
			return nil, "", fmt.Errorf("tmux error: %s", err.Error())
		}
	}
	return nil, fmt.Sprintf("opened %d", len(sshCmds)), nil
}

func (m *appModel) connectHostsForGroup(groupIndex int, hostsToOpen []string, remoteCommandOverride string) (execCmd []string, toast string, err error) {
	if groupIndex < 0 || groupIndex >= len(m.opts.Config.Groups) {
		return nil, "", fmt.Errorf("invalid group")
	}
	if len(hostsToOpen) == 0 {
		return nil, "", fmt.Errorf("no host selected")
	}

	g := m.opts.Config.Groups[groupIndex]
	defaults := m.opts.Config.Defaults
	base := sshcmd.FromDefaults(defaults)
	rc := strings.TrimSpace(remoteCommandOverride)

	sshCmds := make([][]string, 0, len(hostsToOpen))
	for _, h := range hostsToOpen {
		s := base
		if hc, ok := hostConfigFor(m.opts.Config, h); ok {
			s = sshcmd.ApplyHost(s, hc)
		}
		s = sshcmd.ApplyGroup(s, g)
		if rc != "" {
			s.RemoteCommand = rc
		}
		cmd, _ := sshcmd.BuildCommand(h, s)
		sshCmds = append(sshCmds, cmd)
	}

	tmuxSetting := defaults.Tmux
	if strings.TrimSpace(g.Tmux) != "" {
		tmuxSetting = g.Tmux
	}
	openModeSetting := defaults.OpenMode
	if strings.TrimSpace(g.OpenMode) != "" {
		openModeSetting = g.OpenMode
	}

	inTmux := tmx.InTmux()
	mode := tmx.ResolveOpenMode(tmuxSetting, openModeSetting, inTmux)

	if mode == tmx.OpenCurrent {
		if len(sshCmds) > 1 {
			return nil, "", fmt.Errorf("multi-host requires tmux (window or pane mode)")
		}
		return sshCmds[0], "", nil
	}

	if !inTmux {
		if len(sshCmds) > 1 {
			return nil, "", fmt.Errorf("multi-host requires an active tmux session")
		}
		return tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0]), "", nil
	}

	window := strings.TrimSpace(g.Name)
	if window == "" {
		window = windowName(hostsToOpen[0])
	}

	ps := resolvePaneSettings(defaults, &g, len(sshCmds))
	if mode == tmx.OpenPane || (mode == tmx.OpenWindow && len(sshCmds) > 1) {
		if err := tmuxOpenOneWindow(sshCmds, tmuxOneWindowOpts{
			WindowName:       window,
			PaneTitles:       hostsToOpen,
			SplitFlag:        ps.SplitFlag,
			Layout:           ps.Layout,
			SyncPanes:        ps.SyncPanes,
			PaneBorderFormat: ps.BorderFormat,
			PaneBorderStatus: ps.BorderStatus,
		}); err != nil {
			return nil, "", err
		}
		return nil, fmt.Sprintf("opened %d in one window", len(sshCmds)), nil
	}

	for _, sshCmd := range sshCmds {
		tmuxCmd := tmx.NewWindowCmd(window, sshCmd)
		// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
		if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
			return nil, "", fmt.Errorf("tmux error: %s", err.Error())
		}
	}
	return nil, fmt.Sprintf("opened %d", len(sshCmds)), nil
}
