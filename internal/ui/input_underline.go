package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// underlineText pads s to width with a "─" rule, accenting s when focused.
func underlineText(s string, focused bool, width int) string {
	s = strings.TrimRight(s, "\n")
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) > width {
		s = lipgloss.NewStyle().MaxWidth(width).Render(s)
	}
	if focused {
		s = accentStyle.Render(s)
	}
	if pad := width - lipgloss.Width(s); pad > 0 {
		s += underlineFill.Render(strings.Repeat("─", pad))
	}
	return s
}
