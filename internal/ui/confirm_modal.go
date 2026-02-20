package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	confirmTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(cErr)
)

func renderQuitConfirm(width, height int) string {
	box := quitConfirmBox(width)
	if width <= 0 || height <= 0 {
		return strings.TrimSpace(box)
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func renderDeleteGroupConfirm(width, height int, name string, hostCount int) string {
	box := deleteGroupConfirmBox(width, name, hostCount)
	if width <= 0 || height <= 0 {
		return strings.TrimSpace(box)
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// renderConfirmBox builds a confirm dialog using manual box primitives.
// title is embedded in the top border; body and footer are indented 2 spaces.
func renderConfirmBox(totalW int, title, body, footer string) string {
	return strings.Join([]string{
		boxTitleTop(totalW, title),
		boxLine(totalW, ""),
		boxLine(totalW, "  "+body),
		boxLine(totalW, ""),
		boxLine(totalW, "  "+footer),
		boxLine(totalW, ""),
		boxBottom(totalW),
	}, "\n")
}

func quitConfirmBox(maxWidth int) string {
	boxW := maxWidth
	if boxW <= 0 {
		boxW = 52
	}
	boxW = min(52, max(22, boxW-4))
	title := confirmTitleStyle.Render("Quit?")
	body := "Exit ssh-tui?"
	footer := footerKeyStyle.Render("[y/\u21b5]") + dim.Render(" quit") +
		"     " + footerKeyStyle.Render("[n/Esc]") + dim.Render(" cancel")
	return renderConfirmBox(boxW+6, title, body, footer)
}

func deleteGroupConfirmBox(maxWidth int, name string, hostCount int) string {
	name = strings.TrimSpace(name)
	boxW := maxWidth
	if boxW <= 0 {
		boxW = 60
	}
	boxW = min(60, max(24, boxW-4))
	title := confirmTitleStyle.Render("Delete group?")
	body := "This will remove the group"
	if name != "" {
		body = fmt.Sprintf("Delete %q (%d)?", name, hostCount)
	}
	footer := footerKeyStyle.Render("[y/\u21b5]") + dim.Render(" delete") +
		"     " + footerKeyStyle.Render("[n/Esc]") + dim.Render(" cancel")
	return renderConfirmBox(boxW+6, title, body, footer)
}

func connectConfirmBox(maxWidth int, count int, hostNames []string) string {
	boxW := maxWidth
	if boxW <= 0 {
		boxW = 60
	}
	boxW = min(60, max(24, boxW-4))
	totalW := boxW + 6
	title := confirmTitleStyle.Render(fmt.Sprintf("Connect %d hosts?", count))
	footer := footerKeyStyle.Render("[y/\u21b5]") + dim.Render(" connect") +
		"     " + footerKeyStyle.Render("[n/Esc]") + dim.Render(" cancel")

	parts := []string{boxTitleTop(totalW, title), boxLine(totalW, "")}
	shown := hostNames
	extra := 0
	if len(hostNames) > 4 {
		shown = hostNames[:4]
		extra = len(hostNames) - 4
	}
	for i, h := range shown {
		line := "  " + h
		if i == len(shown)-1 && extra > 0 {
			line += fmt.Sprintf("    +%d more", extra)
		}
		parts = append(parts, boxLine(totalW, line))
	}
	parts = append(parts, boxLine(totalW, ""))
	parts = append(parts, boxLine(totalW, "  "+footer))
	parts = append(parts, boxLine(totalW, ""))
	parts = append(parts, boxBottom(totalW))
	return strings.Join(parts, "\n")
}

func removeHostsConfirmBox(maxWidth int, hosts []string, groupName string) string {
	boxW := maxWidth
	if boxW <= 0 {
		boxW = 60
	}
	boxW = min(60, max(24, boxW-4))
	totalW := boxW + 6
	count := len(hosts)
	title := confirmTitleStyle.Render("Remove hosts?")
	footer := footerKeyStyle.Render("[y/\u21b5]") + dim.Render(" remove") +
		"     " + footerKeyStyle.Render("[n/Esc]") + dim.Render(" cancel")

	parts := []string{boxTitleTop(totalW, title), boxLine(totalW, "")}
	shown := hosts
	extra := 0
	if count > 4 {
		shown = hosts[:4]
		extra = count - 4
	}
	for i, h := range shown {
		line := "  " + h
		if i == len(shown)-1 && extra > 0 {
			line += fmt.Sprintf("    +%d more", extra)
		}
		parts = append(parts, boxLine(totalW, line))
	}
	if groupName != "" {
		parts = append(parts, boxLine(totalW, ""))
		parts = append(parts, boxLine(totalW, dim.Render("  from ")+groupName))
	}
	parts = append(parts, boxLine(totalW, ""))
	parts = append(parts, boxLine(totalW, "  "+footer))
	parts = append(parts, boxLine(totalW, ""))
	parts = append(parts, boxBottom(totalW))
	return strings.Join(parts, "\n")
}
