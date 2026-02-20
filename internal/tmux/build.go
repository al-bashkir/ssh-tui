package tmux

import "strings"

type OpenMode string

const (
	OpenCurrent OpenMode = "current"
	OpenWindow  OpenMode = "tmux-window"
	OpenPane    OpenMode = "tmux-pane"
)

func ResolveOpenMode(tmuxSetting string, openModeSetting string, inTmux bool) OpenMode {
	tmuxSetting = strings.ToLower(strings.TrimSpace(tmuxSetting))
	openModeSetting = strings.ToLower(strings.TrimSpace(openModeSetting))

	if tmuxSetting == "never" {
		return OpenCurrent
	}

	if tmuxSetting == "force" {
		if openModeSetting == string(OpenPane) {
			return OpenPane
		}
		return OpenWindow
	}

	switch openModeSetting {
	case string(OpenCurrent):
		return OpenCurrent
	case string(OpenWindow):
		return OpenWindow
	case string(OpenPane):
		return OpenPane
	case "", "auto":
		if inTmux {
			return OpenWindow
		}
		return OpenCurrent
	default:
		// Unknown value: keep MVP behavior predictable.
		if inTmux {
			return OpenWindow
		}
		return OpenCurrent
	}
}

func NewWindowCmd(name string, sshCmd []string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "ssh"
	}

	cmd := []string{"tmux", "new-window", "-n", name, "--"}
	cmd = append(cmd, sshCmd...)
	return cmd
}

func SplitPaneCmd(sshCmd []string) []string {
	return SplitPaneCmdFlag("-h", sshCmd)
}

func SplitPaneCmdFlag(splitFlag string, sshCmd []string) []string {
	splitFlag = strings.TrimSpace(splitFlag)
	if splitFlag != "-h" && splitFlag != "-v" {
		splitFlag = "-h"
	}

	cmd := []string{"tmux", "split-window", splitFlag, "--"}
	cmd = append(cmd, sshCmd...)
	return cmd
}

func NewSessionCmd(session string, sshCmd []string) []string {
	session = strings.TrimSpace(session)
	if session == "" {
		session = "ssh-tui"
	}

	cmd := []string{"tmux", "new-session", "-A", "-s", session, "--"}
	cmd = append(cmd, sshCmd...)
	return cmd
}
