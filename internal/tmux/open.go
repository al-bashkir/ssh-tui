package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// OneWindowOpts controls how OpenOneWindow creates panes in a single tmux window.
type OneWindowOpts struct {
	WindowName string
	PaneTitles []string
	PaneSettings
}

// OpenOneWindow creates a new tmux window and splits it into panes, one per SSH command.
func OpenOneWindow(sshCmds [][]string, opts OneWindowOpts) error {
	if len(sshCmds) == 0 {
		return fmt.Errorf("no hosts selected")
	}

	// Create window and capture both window_id and pane_id.
	args := append([]string{"new-window", "-P", "-F", "#{window_id} #{pane_id}", "-n", opts.WindowName, "--"}, sshCmds[0]...)
	// #nosec G204 -- running tmux with argv (no shell); args are constructed by the app.
	out, err := exec.Command("tmux", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux error: %s", tmuxErrMsg(out, err))
	}

	fields := strings.Fields(string(out))
	if len(fields) < 2 {
		return fmt.Errorf("tmux error: missing window/pane id")
	}
	winID := fields[0]

	// Best-effort window settings (do not fail if unsupported).
	_ = tmuxRun("set-window-option", "-t", winID, "automatic-rename", "off")
	_ = tmuxRun("set-window-option", "-t", winID, "allow-rename", "off")
	if strings.EqualFold(opts.BorderStatus, "off") {
		_ = tmuxRun("set-window-option", "-t", winID, "pane-border-status", "off")
	} else {
		_ = tmuxRun("set-window-option", "-t", winID, "pane-border-status", opts.BorderStatus)
		_ = tmuxRun("set-window-option", "-t", winID, "pane-border-format", opts.BorderFormat)
	}
	if opts.SyncPanes && len(sshCmds) > 1 {
		_ = tmuxRun("set-window-option", "-t", winID, "synchronize-panes", "on")
	}
	setPaneTitle(fields[1], opts.PaneTitles, 0)

	for i := 1; i < len(sshCmds); i++ {
		splitArgs := append([]string{"split-window", "-t", winID, opts.SplitFlag, "-P", "-F", "#{pane_id}", "--"}, sshCmds[i]...)
		// #nosec G204 -- running tmux with argv (no shell); args are constructed by the app.
		out, err := exec.Command("tmux", splitArgs...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("tmux error: %s", tmuxErrMsg(out, err))
		}
		if paneFields := strings.Fields(string(out)); len(paneFields) > 0 {
			setPaneTitle(paneFields[0], opts.PaneTitles, i)
		}
	}

	// Best-effort layout.
	_ = tmuxRun("select-layout", "-t", winID, opts.Layout)
	return nil
}

func setPaneTitle(paneID string, titles []string, idx int) {
	if idx < len(titles) {
		if title := strings.TrimSpace(titles[idx]); title != "" {
			_ = tmuxRun("select-pane", "-t", paneID, "-T", title)
		}
	}
}

func tmuxRun(args ...string) error {
	// #nosec G204 -- running tmux with argv (no shell); args are internal.
	return exec.Command("tmux", args...).Run()
}

func tmuxErrMsg(out []byte, err error) string {
	msg := strings.TrimSpace(string(out))
	if msg == "" {
		return err.Error()
	}
	return msg
}
