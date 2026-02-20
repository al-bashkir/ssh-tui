package config

import (
	"sort"
	"strings"
)

// ConfigHosts returns the set of hosts referenced by the config file.
// This is used when defaults.load_known_hosts=false.
func ConfigHosts(cfg Config) []string {
	set := make(map[string]struct{})
	for _, h := range cfg.Hosts {
		k := strings.TrimSpace(h.Host)
		if k == "" {
			continue
		}
		set[k] = struct{}{}
	}
	for _, g := range cfg.Groups {
		for _, h := range g.Hosts {
			k := strings.TrimSpace(h)
			if k == "" {
				continue
			}
			set[k] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for h := range set {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}
