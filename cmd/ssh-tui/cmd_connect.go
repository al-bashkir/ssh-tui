package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/connect"
)

func runConnect(args []string, cfg config.Config, inv config.Inventory, noTmux bool) {
	if len(args) == 0 {
		fatal(fmt.Errorf("connect requires a subcommand: group|g or host|h\nUsage: ssh-tui connect group|host NAME"))
	}
	switch args[0] {
	case "group", "g":
		if len(args) < 2 {
			fatal(fmt.Errorf("connect group requires a name\nUsage: ssh-tui connect group NAME"))
		}
		connectGroup(args[1], cfg, inv, noTmux)
	case "host", "h":
		if len(args) < 2 {
			fatal(fmt.Errorf("connect host requires a name\nUsage: ssh-tui connect host NAME"))
		}
		connectHost(args[1], cfg, inv)
	default:
		fatal(fmt.Errorf("unknown connect subcommand %q: use group|g or host|h", args[0]))
	}
}

func connectGroup(name string, cfg config.Config, inv config.Inventory, noTmux bool) {
	var group config.Group
	found := false
	for _, g := range inv.Groups {
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
	if noTmux {
		group.Tmux = "never"
	}

	open(group.Hosts, cfg, inv, &group)
}

func connectHost(name string, cfg config.Config, inv config.Inventory) {
	open([]string{name}, cfg, inv, nil)
}

// open dispatches hosts the same way the TUI does, then either replaces this
// process with ssh/tmux or reports what was opened.
func open(hosts []string, cfg config.Config, inv config.Inventory, group *config.Group) {
	sshCmds := connect.BuildCommands(hosts, cfg.Defaults, inv, group, nil)
	mode, inTmux := connect.Mode(cfg.Defaults, group)

	execCmd, msg, err := connect.Open(hosts, sshCmds, cfg.Defaults, group, mode, inTmux)
	if err != nil {
		fatal(err)
	}
	if len(execCmd) != 0 {
		if err := execReplace(execCmd); err != nil {
			fatal(err)
		}
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, msg)
}
