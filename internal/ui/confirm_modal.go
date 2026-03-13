package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderQuitConfirm(width, height int) string {
	box := quitConfirmBox(width)
	if width <= 0 || height <= 0 {
		return strings.TrimSpace(box)
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// renderConfirmBox builds a confirm dialog using manual box primitives.
// title is embedded in the top border; body and footer are indented 2 spaces.
func renderConfirmBox(totalW int, title, body, footer string) string {
	return strings.Join([]string{
		focusedBoxTitleTop(totalW, title),
		focusedBoxLine(totalW, ""),
		focusedBoxLine(totalW, "  "+body),
		focusedBoxLine(totalW, ""),
		focusedBoxLine(totalW, "  "+footer),
		focusedBoxLine(totalW, ""),
		focusedBoxBottom(totalW),
	}, "\n")
}

func quitConfirmBox(maxWidth int) string {
	boxW := maxWidth
	if boxW <= 0 {
		boxW = quitConfirmMaxW
	}
	boxW = min(quitConfirmMaxW, max(22, boxW-confirmDialogMargin))
	title := confirmTitleStyle.Render("Quit?")
	body := "Exit ssh-tui?"
	footer := styledFooter("y/⏎ quit  n/Esc cancel")
	return renderConfirmBox(boxW+confirmDialogPadding, title, body, footer)
}

func deleteGroupConfirmBox(maxWidth int, name string, hostCount int) string {
	name = strings.TrimSpace(name)
	boxW := maxWidth
	if boxW <= 0 {
		boxW = confirmDialogMaxW
	}
	boxW = min(confirmDialogMaxW, max(24, boxW-confirmDialogMargin))
	title := confirmTitleStyle.Render("Delete group?")
	body := "This will remove the group"
	if name != "" {
		body = fmt.Sprintf("Delete %q (%d)?", name, hostCount)
	}
	footer := styledFooter("y/⏎ delete  n/Esc cancel")
	return renderConfirmBox(boxW+confirmDialogPadding, title, body, footer)
}

func connectConfirmBox(maxWidth int, count int, hostNames []string) string {
	boxW := maxWidth
	if boxW <= 0 {
		boxW = confirmDialogMaxW
	}
	boxW = min(confirmDialogMaxW, max(24, boxW-confirmDialogMargin))
	totalW := boxW + confirmDialogPadding
	title := confirmTitleStyle.Render(fmt.Sprintf("Connect %d hosts?", count))
	footer := styledFooter("y/⏎ connect  n/Esc cancel")

	parts := []string{focusedBoxTitleTop(totalW, title), focusedBoxLine(totalW, "")}
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
		parts = append(parts, focusedBoxLine(totalW, line))
	}
	parts = append(parts, focusedBoxLine(totalW, ""))
	parts = append(parts, focusedBoxLine(totalW, "  "+footer))
	parts = append(parts, focusedBoxLine(totalW, ""))
	parts = append(parts, focusedBoxBottom(totalW))
	return strings.Join(parts, "\n")
}

func removeHostsConfirmBox(maxWidth int, hosts []string, groupName string) string {
	boxW := maxWidth
	if boxW <= 0 {
		boxW = confirmDialogMaxW
	}
	boxW = min(confirmDialogMaxW, max(24, boxW-confirmDialogMargin))
	totalW := boxW + confirmDialogPadding
	count := len(hosts)
	title := confirmTitleStyle.Render("Remove hosts?")
	footer := styledFooter("y/⏎ remove  n/Esc cancel")

	parts := []string{focusedBoxTitleTop(totalW, title), focusedBoxLine(totalW, "")}
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
		parts = append(parts, focusedBoxLine(totalW, line))
	}
	if groupName != "" {
		parts = append(parts, focusedBoxLine(totalW, ""))
		parts = append(parts, focusedBoxLine(totalW, dim.Render("  from ")+groupName))
	}
	parts = append(parts, focusedBoxLine(totalW, ""))
	parts = append(parts, focusedBoxLine(totalW, "  "+footer))
	parts = append(parts, focusedBoxLine(totalW, ""))
	parts = append(parts, focusedBoxBottom(totalW))
	return strings.Join(parts, "\n")
}
