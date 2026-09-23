package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
)

func suggestCopyName(name string, exists func(string) bool) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "copy"
	}

	c1 := base + "-copy"
	if !exists(c1) {
		return c1
	}
	for i := 2; i < 1000; i++ {
		c := fmt.Sprintf("%s-copy%d", base, i)
		if !exists(c) {
			return c
		}
	}
	return c1
}

func suggestCopyGroupName(inv config.Inventory, from string) string {
	return suggestCopyName(from, func(n string) bool {
		return slices.ContainsFunc(inv.Groups, func(g config.Group) bool { return strings.TrimSpace(g.Name) == n })
	})
}

func suggestCopyHostKey(inv config.Inventory, from string) string {
	return suggestCopyName(from, func(n string) bool {
		return slices.ContainsFunc(inv.Hosts, func(h config.Host) bool { return strings.TrimSpace(h.Host) == n })
	})
}
