package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/sshcmd"
	tmx "github.com/al-bashkir/ssh-tui/internal/tmux"
)

func runConnect(args []string, cfg config.Config, noTmux bool) {
	if len(args) == 0 {
		fatal(fmt.Errorf("connect requires a subcommand: group|g or host|h\nUsage: ssh-tui connect group|host NAME"))
	}
	switch args[0] {
	case "group", "g":
		if len(args) < 2 {
			fatal(fmt.Errorf("connect group requires a name\nUsage: ssh-tui connect group NAME"))
		}
		connectGroup(args[1], cfg, noTmux)
	case "host", "h":
		if len(args) < 2 {
			fatal(fmt.Errorf("connect host requires a name\nUsage: ssh-tui connect host NAME"))
		}
		connectHost(args[1], cfg)
	default:
		fatal(fmt.Errorf("unknown connect subcommand %q: use group|g or host|h", args[0]))
	}
}

func connectGroup(name string, cfg config.Config, noTmux bool) {
	var group config.Group
	found := false
	for _, g := range cfg.Groups {
		if strings.EqualFold(g.Name, name) {
			group = g
			found = true
			break
		}
	}
	if !found {
		fatal(fmt.Errorf("group %q not found", name))
	}
	if len(group.Hosts) == 0 {
		fatal(fmt.Errorf("group %q has no hosts", name))
	}

	// Build SSH commands with the same precedence as the TUI:
	// Defaults → per-host override → Group settings.
	base := sshcmd.FromDefaults(cfg.Defaults)
	sshCmds := make([][]string, 0, len(group.Hosts))
	for _, h := range group.Hosts {
		s := base
		if hc, ok := sshcmd.FindHostConfig(cfg, h); ok {
			s = sshcmd.ApplyHost(s, hc)
		}
		s = sshcmd.ApplyGroup(s, group)
		cmd, err := sshcmd.BuildCommand(h, s)
		if err != nil {
			fatal(fmt.Errorf("build ssh command for %s: %w", h, err))
		}
		sshCmds = append(sshCmds, cmd)
	}

	// Resolve effective tmux / open-mode settings (group overrides defaults).
	tmuxSetting := cfg.Defaults.Tmux
	if strings.TrimSpace(group.Tmux) != "" {
		tmuxSetting = group.Tmux
	}
	openModeSetting := cfg.Defaults.OpenMode
	if strings.TrimSpace(group.OpenMode) != "" {
		openModeSetting = group.OpenMode
	}
	if noTmux {
		tmuxSetting = "never"
	}

	inTmux := tmx.InTmux()
	mode := tmx.ResolveOpenMode(tmuxSetting, openModeSetting, inTmux)
	execConnect(group.Hosts, sshCmds, cfg.Defaults, &group, mode, inTmux)
}

func connectHost(name string, cfg config.Config) {
	// Build SSH command with the same precedence as the TUI:
	// Defaults → per-host override (no group).
	base := sshcmd.FromDefaults(cfg.Defaults)
	s := base
	if hc, ok := sshcmd.FindHostConfig(cfg, name); ok {
		s = sshcmd.ApplyHost(s, hc)
	}
	cmd, err := sshcmd.BuildCommand(name, s)
	if err != nil {
		fatal(fmt.Errorf("build ssh command for %s: %w", name, err))
	}

	inTmux := tmx.InTmux()
	mode := tmx.ResolveOpenMode(cfg.Defaults.Tmux, cfg.Defaults.OpenMode, inTmux)
	execConnect([]string{name}, [][]string{cmd}, cfg.Defaults, nil, mode, inTmux)
}

// execConnect dispatches SSH commands using the same logic as the TUI's dispatchConnect.
func execConnect(
	hosts []string,
	sshCmds [][]string,
	defaults config.Defaults,
	group *config.Group,
	mode tmx.OpenMode,
	inTmux bool,
) {
	wName := tmx.GroupWindowName(hosts, group)

	switch {
	case mode == tmx.OpenCurrent:
		if len(sshCmds) > 1 {
			fatal(fmt.Errorf("multi-host requires tmux (set open_mode to tmux-window or tmux-pane)"))
		}
		if err := execReplace(sshCmds[0]); err != nil {
			fatal(err)
		}

	case !inTmux:
		if len(sshCmds) > 1 {
			fatal(fmt.Errorf("multi-host requires an active tmux session"))
		}
		// NewSessionCmd uses -A so it attaches to an existing session instead of failing.
		if err := execReplace(tmx.NewSessionCmd(defaults.TmuxSession, sshCmds[0])); err != nil {
			fatal(err)
		}

	case mode == tmx.OpenPane || (mode == tmx.OpenWindow && len(sshCmds) > 1):
		// Open all hosts as panes in a single new tmux window.
		ps := tmx.ResolvePaneSettings(defaults, group, len(sshCmds))
		if err := tmx.OpenOneWindow(sshCmds, tmx.OneWindowOpts{
			WindowName:       wName,
			PaneTitles:       hosts,
			SplitFlag:        ps.SplitFlag,
			Layout:           ps.Layout,
			SyncPanes:        ps.SyncPanes,
			PaneBorderFormat: ps.BorderFormat,
			PaneBorderStatus: ps.BorderStatus,
		}); err != nil {
			fatal(err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "opened %d in one window\n", len(sshCmds))

	default:
		// OpenWindow: one tmux window per host.
		for i, sshCmd := range sshCmds {
			name := tmx.GroupWindowName(hosts[i:i+1], group)
			tmuxCmd := tmx.NewWindowCmd(name, sshCmd)
			// #nosec G204 -- tmux argv is constructed (no shell) from known host/group settings.
			if err := exec.Command(tmuxCmd[0], tmuxCmd[1:]...).Run(); err != nil {
				fatal(fmt.Errorf("tmux new-window: %w", err))
			}
		}
		_, _ = fmt.Fprintf(os.Stderr, "opened %d\n", len(sshCmds))
	}
}
