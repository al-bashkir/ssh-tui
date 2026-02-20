package config

import (
	"fmt"
	"regexp"
)

const DefaultPaneBorderFormat = "#[bg=green,fg=black] #T#{?pane_synchronized, #[fg=colour196]#[bold][SYNC]#[default],} #[default]"

var validGroupName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateGroupName returns an error if name is empty or contains characters
// other than letters, digits, hyphens and underscores.
func ValidateGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("name required")
	}
	if !validGroupName.MatchString(name) {
		return fmt.Errorf("group name %q is invalid: only letters, digits, - and _ are allowed", name)
	}
	return nil
}

type Config struct {
	Version     int      `toml:"version"`
	HiddenHosts []string `toml:"hidden_hosts,omitempty"`
	Defaults    Defaults `toml:"defaults"`
	Hosts       []Host   `toml:"hosts"`
	Groups      []Group  `toml:"groups"`
}

// Host is a per-host override (inherits from [defaults]).
// It is intentionally minimal (SSH args only) and applies by exact host match.
// Example TOML:
//
//	[[hosts]]
//	host = "db01.example.com"
//	user = "admin"
//	port = 22
//	identity_file = "~/.ssh/id_ed25519"
//	extra_args = ["-o", "ServerAliveInterval=30"]
type Host struct {
	Host         string   `toml:"host"`
	User         string   `toml:"user"`
	Port         int      `toml:"port"`
	IdentityFile string   `toml:"identity_file"`
	ExtraArgs    []string `toml:"extra_args"`
	Hidden       bool     `toml:"hidden,omitempty"`
}

type Defaults struct {
	AccentColor    string   `toml:"accent_color"` // default UI accent color (preset name or color code)
	LoadKnownHosts bool     `toml:"load_known_hosts"`
	User           string   `toml:"user"`
	Port           int      `toml:"port"`
	IdentityFile   string   `toml:"identity_file"`
	ExtraArgs      []string `toml:"extra_args"`
	PaneSplit      string   `toml:"pane_split"`          // horizontal|vertical
	PaneLayout     string   `toml:"pane_layout"`         // auto|tiled|even-horizontal|even-vertical|main-horizontal|main-vertical
	PaneSync       string   `toml:"pane_sync"`           // on|off
	PaneBorderFmt  string   `toml:"pane_border_format"`  // tmux format string
	PaneBorderFmts []string `toml:"pane_border_formats"` // user-defined formats (built-in default is always available)
	PaneBorderPos  string   `toml:"pane_border_status"`  // off|top|bottom
	Tmux           string   `toml:"tmux"`                // auto|force|never
	OpenMode       string   `toml:"open_mode"`           // auto|current|tmux-window|tmux-pane
	TmuxSession    string   `toml:"tmux_session"`        // session name
	ConfirmQuit            bool     `toml:"confirm_quit"`
	ConnectConfirmThreshold int     `toml:"connect_confirm_threshold"`
}

type Group struct {
	Name          string   `toml:"name"`
	User          string   `toml:"user"`
	Port          int      `toml:"port"`
	IdentityFile  string   `toml:"identity_file"`
	ExtraArgs     []string `toml:"extra_args"`
	RemoteCommand string   `toml:"remote_command"`
	PaneSplit     string   `toml:"pane_split"`
	PaneLayout    string   `toml:"pane_layout"`
	PaneSync      string   `toml:"pane_sync"`
	PaneBorderFmt string   `toml:"pane_border_format"`
	PaneBorderPos string   `toml:"pane_border_status"`
	Tmux          string   `toml:"tmux"`      // optional override
	OpenMode      string   `toml:"open_mode"` // optional override
	Hosts         []string `toml:"hosts"`
}

func DefaultConfig() Config {
	return Config{
		Version: 1,
		Defaults: Defaults{
			AccentColor:    "",
			LoadKnownHosts: true,
			User:           "",
			Port:           22,
			IdentityFile:   "",
			ExtraArgs:      nil,
			PaneSplit:      "vertical",
			PaneLayout:     "even-vertical",
			PaneSync:       "on",
			PaneBorderFmt:  DefaultPaneBorderFormat,
			PaneBorderFmts: nil,
			PaneBorderPos:  "bottom",
			Tmux:           "auto",
			OpenMode:       "auto",
			TmuxSession:    "ssh-tui",
			ConfirmQuit:            false,
			ConnectConfirmThreshold: 5,
		},
		Hosts:  nil,
		Groups: nil,
	}
}
