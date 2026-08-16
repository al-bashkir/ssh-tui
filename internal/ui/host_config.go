package ui

import (
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/sshcmd"

	tea "github.com/charmbracelet/bubbletea"
)

// openHostConfigCmd opens the per-host config form for host.
func openHostConfigCmd(host string, returnTo screen, t *toast) tea.Cmd {
	if strings.TrimSpace(host) == "" {
		*t = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	return func() tea.Msg { return openHostFormMsg{host: host, returnTo: returnTo} }
}

// copyHostConfigCmd prefills a new host form from host's existing config.
func copyHostConfigCmd(inv config.Inventory, host string, returnTo screen, t *toast) tea.Cmd {
	if strings.TrimSpace(host) == "" {
		*t = toast{text: "no host selected", level: toastWarn}
		return nil
	}
	hc, ok := sshcmd.FindHostConfig(inv.Hosts, host)
	if !ok {
		*t = toast{text: "no host config", level: toastWarn}
		return nil
	}
	hc.Host = suggestCopyHostKey(inv, hc.Host)
	return func() tea.Msg { return openHostFormPrefillMsg{host: hc, returnTo: returnTo} }
}

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
