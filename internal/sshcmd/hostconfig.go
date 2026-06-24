package sshcmd

import (
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
)

// FindHostConfig returns the per-host config for the given host address.
// It first tries an exact match, then falls back to bracket-host notation
// (e.g. "[host]:port" entries from known_hosts).
func FindHostConfig(hosts []config.Host, host string) (config.Host, bool) {
	h := strings.TrimSpace(host)
	if h == "" {
		return config.Host{}, false
	}

	// Exact match first.
	for i := range hosts {
		if strings.TrimSpace(hosts[i].Host) == h {
			return hosts[i], true
		}
	}

	// Fallback: known_hosts uses "[host]:port".
	if base, ok := parseBracketHostConfig(h); ok {
		for i := range hosts {
			if strings.TrimSpace(hosts[i].Host) == base {
				return hosts[i], true
			}
		}
	}

	return config.Host{}, false
}

func parseBracketHostConfig(s string) (host string, ok bool) {
	h, _, ok := parseBracketHost(strings.TrimSpace(s))
	return h, ok
}
