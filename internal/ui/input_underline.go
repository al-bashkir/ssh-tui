package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func underlineInput(in textinput.Model, focused bool, width int) string {
	s := strings.TrimRight(in.View(), "\n")
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) > width {
		s = lipgloss.NewStyle().MaxWidth(width).Render(s)
	}
	pad := width - lipgloss.Width(s)
	if pad <= 0 {
		return s
	}
	fill := strings.Repeat("_", pad)
	if focused {
		fill = checkedStyle.Render(fill)
	} else {
		fill = dim.Render(fill)
	}
	return s + fill
}

func underlineText(s string, focused bool, width int) string {
	s = strings.TrimRight(s, "\n")
	if width <= 0 {
		return s
	}
	if lipgloss.Width(s) > width {
		s = lipgloss.NewStyle().MaxWidth(width).Render(s)
	}
	pad := width - lipgloss.Width(s)
	if pad <= 0 {
		if focused {
			return checkedStyle.Render(s)
		}
		return dim.Render(s)
	}
	fill := strings.Repeat("_", pad)
	if focused {
		return checkedStyle.Render(s) + checkedStyle.Render(fill)
	}
	return dim.Render(s) + dim.Render(fill)
}
