package ui

import (
	"fmt"
	"strings"
)

func renderQuitConfirm(width, height int) string {
	return placeCentered(width, height, renderConfirmBox(confirmDialogWidth(width, 52, 22),
		confirmTitleStyle.Render("Quit?"),
		[]string{"  Exit ssh-tui?"},
		styledFooter("y/⏎ quit  n/Esc cancel")))
}

// renderConfirmBox builds a confirm dialog using manual box primitives.
// title is embedded in the top border; body lines are used as given and the
// footer is indented 2 spaces.
func renderConfirmBox(totalW int, title string, body []string, footer string) string {
	parts := []string{focusedBoxTitleTop(totalW, title), focusedBoxLine(totalW, "")}
	for _, l := range body {
		parts = append(parts, focusedBoxLine(totalW, l))
	}
	parts = append(parts,
		focusedBoxLine(totalW, ""),
		focusedBoxLine(totalW, "  "+footer),
		focusedBoxLine(totalW, ""),
		focusedBoxBottom(totalW),
	)
	return strings.Join(parts, "\n")
}

func deleteGroupConfirmBox(maxWidth int, name string, hostCount int) string {
	return renderConfirmBox(confirmDialogWidth(maxWidth, 60, 24),
		confirmTitleStyle.Render("Delete group?"),
		[]string{fmt.Sprintf("  Delete %q (%d)?", name, hostCount)},
		styledFooter("y/⏎ delete  n/Esc cancel"))
}

// confirmDialogWidth clamps the dialog content to maxW (minW at least) within
// the terminal width and adds the border/padding overhead, returning the
// total box width.
func confirmDialogWidth(maxWidth, maxW, minW int) int {
	if maxWidth <= 0 {
		maxWidth = maxW
	}
	return min(maxW, max(minW, maxWidth-4)) + 6
}

// renderHostListConfirmBox is a confirm dialog listing up to 4 hosts, with a
// "+N more" trailer beyond that and an optional note line below the list.
func renderHostListConfirmBox(maxWidth int, title, footer string, hosts []string, note string) string {
	shown, extra := hosts, 0
	if len(hosts) > 4 {
		shown, extra = hosts[:4], len(hosts)-4
	}
	var body []string
	for i, h := range shown {
		line := "  " + h
		if i == len(shown)-1 && extra > 0 {
			line += fmt.Sprintf("    +%d more", extra)
		}
		body = append(body, line)
	}
	if note != "" {
		body = append(body, "", note)
	}
	return renderConfirmBox(confirmDialogWidth(maxWidth, 60, 24), title, body, footer)
}

func connectConfirmBox(maxWidth int, hostNames []string) string {
	return renderHostListConfirmBox(maxWidth,
		confirmTitleStyle.Render(fmt.Sprintf("Connect %d hosts?", len(hostNames))),
		styledFooter("y/⏎ connect  n/Esc cancel"),
		hostNames, "")
}

func removeHostsConfirmBox(maxWidth int, hosts []string, groupName string) string {
	note := ""
	if groupName != "" {
		note = dim.Render("  from ") + groupName
	}
	return renderHostListConfirmBox(maxWidth,
		confirmTitleStyle.Render("Remove hosts?"),
		styledFooter("y/⏎ remove  n/Esc cancel"),
		hosts, note)
}
