package ui

import (
	"strings"

	"github.com/bashkir/ssh-tui/internal/config"
	"github.com/bashkir/ssh-tui/internal/sshcmd"
)

func hostConfigFor(cfg config.Config, host string) (config.Host, bool) {
	return sshcmd.FindHostConfig(cfg, host)
}

func findHostConfig(cfg config.Config, host string) (index int, hc config.Host) {
	if h, ok := hostConfigFor(cfg, host); ok {
		for i := range cfg.Hosts {
			if strings.TrimSpace(cfg.Hosts[i].Host) == strings.TrimSpace(h.Host) {
				return i, h
			}
		}
		return -1, h
	}
	return -1, config.Host{Host: strings.TrimSpace(host)}
}

func isHostHidden(cfg config.Config, host string) bool {
	h := strings.TrimSpace(host)
	for _, hh := range cfg.HiddenHosts {
		if strings.TrimSpace(hh) == h {
			return true
		}
	}
	hc, ok := hostConfigFor(cfg, host)
	return ok && hc.Hidden
}
