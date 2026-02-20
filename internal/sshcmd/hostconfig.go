package sshcmd

import (
	"strconv"
	"strings"

	"github.com/bashkir/ssh-tui/internal/config"
)

// FindHostConfig returns the per-host config for the given host address.
// It first tries an exact match, then falls back to bracket-host notation
// (e.g. "[host]:port" entries from known_hosts).
func FindHostConfig(cfg config.Config, host string) (config.Host, bool) {
	h := strings.TrimSpace(host)
	if h == "" {
		return config.Host{}, false
	}

	// Exact match first.
	for i := range cfg.Hosts {
		if strings.TrimSpace(cfg.Hosts[i].Host) == h {
			return cfg.Hosts[i], true
		}
	}

	// Fallback: known_hosts uses "[host]:port".
	if base, ok := parseBracketHostConfig(h); ok {
		for i := range cfg.Hosts {
			if strings.TrimSpace(cfg.Hosts[i].Host) == base {
				return cfg.Hosts[i], true
			}
		}
	}

	return config.Host{}, false
}

func parseBracketHostConfig(s string) (host string, ok bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") {
		return "", false
	}
	idx := strings.LastIndex(s, "]:")
	if idx < 0 {
		return "", false
	}
	h := strings.TrimSpace(s[1:idx])
	if h == "" {
		return "", false
	}
	ps := strings.TrimSpace(s[idx+2:])
	if ps == "" {
		return "", false
	}
	if p, err := strconv.Atoi(ps); err != nil || p <= 0 {
		return "", false
	}
	return h, true
}
