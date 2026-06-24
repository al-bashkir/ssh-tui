package ui

import (
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/sshcmd"
)

func findHostConfig(inv config.Inventory, host string) (index int, hc config.Host) {
	if h, ok := sshcmd.FindHostConfig(inv.Hosts, host); ok {
		for i := range inv.Hosts {
			if strings.TrimSpace(inv.Hosts[i].Host) == strings.TrimSpace(h.Host) {
				return i, h
			}
		}
		return -1, h
	}
	return -1, config.Host{Host: strings.TrimSpace(host)}
}

func isHostHidden(inv config.Inventory, host string) bool {
	h := strings.TrimSpace(host)
	for _, hh := range inv.HiddenHosts {
		if strings.TrimSpace(hh) == h {
			return true
		}
	}
	hc, ok := sshcmd.FindHostConfig(inv.Hosts, host)
	return ok && hc.Hidden
}
