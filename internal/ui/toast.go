package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// toastLevel represents the severity of a toast notification.
type toastLevel int

const (
	toastInfo toastLevel = iota // neutral information
	toastOK                     // success
	toastWarn                   // warning / prompt
	toastErr                    // error
)

// toast holds a notification message with an associated severity level.
type toast struct {
	text  string
	level toastLevel
}

func (t toast) empty() bool {
	return strings.TrimSpace(t.text) == ""
}

// toastDuration returns the auto-dismiss delay for a given severity.
func toastDuration(l toastLevel) time.Duration {
	switch l {
	case toastOK, toastInfo:
		return 3 * time.Second
	case toastWarn:
		return 5 * time.Second
	case toastErr:
		return 8 * time.Second
	}
	return 4 * time.Second
}

// Toast rendering styles (initialised once, refreshed on accent change).
var (
	toastOKStyle   = lipgloss.NewStyle().Foreground(cOK)
	toastInfoStyle = lipgloss.NewStyle().Foreground(cMuted)
	toastErrStyle  = lipgloss.NewStyle().Foreground(cErr)
	// toastWarn reuses the existing statusWarn (orange).
)

// renderToast returns the styled string for a toast.
func renderToast(t toast) string {
	if t.empty() {
		return ""
	}
	switch t.level {
	case toastOK:
		return toastOKStyle.Render(t.text)
	case toastInfo:
		return toastInfoStyle.Render(t.text)
	case toastWarn:
		return statusWarn.Render(t.text)
	case toastErr:
		return toastErrStyle.Render(t.text)
	}
	return statusWarn.Render(t.text)
}

// renderToastWithSpinner returns the styled string for a toast, optionally prepended with a spinner frame.
func renderToastWithSpinner(t toast, spinner bool) string {
	if t.empty() {
		if spinner {
			return statusWarn.Render(spinnerFrame())
		}
		return ""
	}
	prefix := ""
	if spinner {
		prefix = spinnerFrame() + " "
	}
	switch t.level {
	case toastOK:
		return toastOKStyle.Render(prefix + t.text)
	case toastInfo:
		return toastInfoStyle.Render(prefix + t.text)
	case toastWarn:
		return statusWarn.Render(prefix + t.text)
	case toastErr:
		return toastErrStyle.Render(prefix + t.text)
	}
	return statusWarn.Render(prefix + t.text)
}

// breadcrumbTitle builds a breadcrumb-style title:
// "parentCrumb > leafTitle" where parent segments are dim and leaf is accented.
func breadcrumbTitle(parentCrumb, leafTitle string) string {
	parentCrumb = strings.TrimSpace(parentCrumb)
	leafTitle = strings.TrimSpace(leafTitle)
	if parentCrumb == "" {
		return leafTitle
	}
	return dim.Render(parentCrumb+" >") + " " + headerStyle.Render(leafTitle)
}
