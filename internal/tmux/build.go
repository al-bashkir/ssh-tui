package tmux

import (
	"os"
	"strings"
)

type OpenMode string

const (
	OpenCurrent OpenMode = "current"
	OpenWindow  OpenMode = "tmux-window"
	OpenPane    OpenMode = "tmux-pane"
)

func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

func ResolveOpenMode(tmuxSetting string, openModeSetting string, inTmux bool) OpenMode {
	tmuxSetting = strings.ToLower(strings.TrimSpace(tmuxSetting))
	openMode := OpenMode(strings.ToLower(strings.TrimSpace(openModeSetting)))

	switch {
	case tmuxSetting == "never":
		return OpenCurrent
	case tmuxSetting == "force" && openMode == OpenPane:
		return OpenPane
	case tmuxSetting == "force":
		return OpenWindow
	}

	switch openMode {
	case OpenCurrent, OpenWindow, OpenPane:
		return openMode
	}
	// "", "auto" and unknown values.
	if inTmux {
		return OpenWindow
	}
	return OpenCurrent
}

func NewWindowCmd(name string, sshCmd []string) []string {
	return append([]string{"tmux", "new-window", "-n", name, "--"}, sshCmd...)
}

func NewSessionCmd(session string, sshCmd []string) []string {
	session = strings.TrimSpace(session)
	if session == "" {
		session = "ssh-tui"
	}
	return append([]string{"tmux", "new-session", "-A", "-s", session, "--"}, sshCmd...)
}
