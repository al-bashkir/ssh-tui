package ui

import (
	"github.com/bashkir/ssh-tui/internal/config"
	tmx "github.com/bashkir/ssh-tui/internal/tmux"
)

// paneSettings is an alias so existing callers in this package compile unchanged.
type paneSettings = tmx.PaneSettings

func resolvePaneSettings(defaults config.Defaults, group *config.Group, paneCount int) paneSettings {
	return tmx.ResolvePaneSettings(defaults, group, paneCount)
}
