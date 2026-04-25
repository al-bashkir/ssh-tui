package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// handleConfirmQuit processes a key when the quit-confirm dialog is shown.
// It returns (true, cmd) if the key was consumed, (false, nil) otherwise.
// quitting may be nil for models that don't track a quitting state (e.g. pickers).
// clearToast controls whether the toast is cleared when the user cancels.
func handleConfirmQuit(
	msg tea.KeyMsg,
	confirmQuit *bool,
	t *toast,
	quitting *bool,
	clearToast bool,
) (handled bool, cmd tea.Cmd) {
	if !*confirmQuit {
		return false, nil
	}
	switch msg.String() {
	case "y", "Y", "enter":
		if quitting != nil {
			*quitting = true
		}
		return true, tea.Quit
	case "n", "N", "esc":
		*confirmQuit = false
		if clearToast {
			*t = toast{}
		}
		return true, nil
	default:
		return true, nil
	}
}

// handleConfirmConnect processes a key when the connect-confirm dialog is shown.
// Returns (true, cmd) if the key was consumed, (false, nil) otherwise.
func handleConfirmConnect(
	msg tea.KeyMsg,
	confirmConnect *bool,
	pendingFn *func() tea.Cmd,
	t *toast,
) (handled bool, cmd tea.Cmd) {
	if !*confirmConnect {
		return false, nil
	}
	switch msg.String() {
	case "y", "Y", "enter":
		*confirmConnect = false
		fn := *pendingFn
		*pendingFn = nil
		if fn != nil {
			return true, fn()
		}
		return true, nil
	case "n", "N", "esc":
		*confirmConnect = false
		*pendingFn = nil
		*t = toast{}
		return true, nil
	default:
		return true, nil
	}
}

func handleVimListNav(msg tea.KeyMsg, l *list.Model, pendingG *bool) bool {
	switch msg.String() {
	case "g":
		if *pendingG {
			start, end := l.Paginator.GetSliceBounds(len(l.VisibleItems()))
			if start < end {
				l.Select(start)
			}
			*pendingG = false
			return true
		}
		*pendingG = true
		return true
	case "G":
		start, end := l.Paginator.GetSliceBounds(len(l.VisibleItems()))
		if start < end {
			l.Select(end - 1)
		}
		*pendingG = false
		return true
	default:
		*pendingG = false
		return false
	}
}

// updateSearchOrList delegates a tea.Msg to either the search input or the
// list, depending on the current focus. When the search value changes,
// applyFn is called with the new query. Returns the resulting tea.Cmd.
func updateSearchOrList(
	focus focusState,
	search *textinput.Model,
	l *list.Model,
	prevSearch *string,
	msg tea.Msg,
	applyFn func(string),
) tea.Cmd {
	var cmd tea.Cmd
	if focus == focusSearch {
		*search, cmd = search.Update(msg)
		cur := search.Value()
		if cur != *prevSearch {
			applyFn(cur)
			*prevSearch = cur
		}
		return cmd
	}
	*l, cmd = l.Update(msg)
	return cmd
}
