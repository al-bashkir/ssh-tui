package ui

import (
	"fmt"
	"strings"
)

func (m *hostsModel) View() string {
	if s := m.overlayView("Hosts", m.helpKeys, "Connect and run a remote command (keeps session open)."); s != "" {
		return s
	}

	hasWarn := m.opts.SkippedLines > 0 || len(m.opts.LoadErrors) > 0
	right := ""
	if !m.toast.empty() {
		right = renderToast(m.toast)
	} else {
		right = statusDot(hasWarn)
		if hasWarn {
			right += dim.Render(" load warnings")
		}
		if selCount := len(m.selected); selCount > 0 {
			right += "  " + badgeSelStyle.Render(fmt.Sprintf("%d selected", selCount))
		}
		hc := m.hiddenCount()
		total := len(m.allHosts)
		if !m.showHidden {
			total -= hc
		}
		shown := len(m.list.Items())
		q := strings.TrimSpace(m.search.Value())
		if q != "" {
			right += dim.Render(fmt.Sprintf(" %d / %d hosts", shown, total))
		} else {
			right += dim.Render(fmt.Sprintf(" %d hosts", total))
		}
		if hc > 0 {
			if m.showHidden {
				right += "  " + accentStyle.Render(fmt.Sprintf("%d showed", hc))
			} else {
				right += "  " + dim.Render(fmt.Sprintf("%d hidden", hc))
			}
		}
	}
	var footer string
	hasSel := len(m.selected) > 0
	if m.width < compactWidthThreshold {
		if hasSel {
			footer = styledFooter("\u21b5 connect  o panes  \u2423 clear  ? help")
		} else {
			footer = styledFooter("\u21b5 connect  \u2423 select  ? help")
		}
	} else {
		if hasSel {
			footer = styledFooter("\u21b5 connect  o panes  ·  Ctrl+o cmd  a add-to-group  ·  \u2423 clear")
			if m.height >= twoLineFooterMinHeight {
				footer += "\n" + styledFooter("e config  r reload  ·  Alt+2 groups  Alt+3 settings  ? help")
			}
		} else {
			footer = styledFooter("\u21b5 connect  O pane  ·  \u2423 select  o panes  ·  c custom  Alt+2 groups  Ctrl+H hide")
			if m.height >= twoLineFooterMinHeight {
				footer += "\n" + styledFooter("e config  Ctrl+o cmd  a add  r reload  ·  tab search  H show hidden  ? help")
			}
		}
	}

	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		listContent = m.emptyStateView()
	}
	return renderMainTabBox(m.width, m.height, 0, m.search.View(), right, listContent, footer)
}

func (m *hostsModel) emptyStateView() string {
	var defaultMsg string
	if len(m.allHosts) == 0 {
		divider := formSection("", 26)
		hint1 := accentStyle.Render("c") + hintStyle.Render("          custom host")
		hint3 := accentStyle.Render("Alt+3") + hintStyle.Render("      open settings")
		if m.opts.Config.Defaults.LoadKnownHosts {
			hint2 := accentStyle.Render("r") + hintStyle.Render("          reload known_hosts")
			defaultMsg = dim.Render("No hosts found.") + "\n" + divider + "\n" + hint1 + "\n" + hint2 + "\n" + hint3
		} else {
			note := dim.Render("(known_hosts loading is off)")
			defaultMsg = dim.Render("No hosts found.") + "\n" + divider + "\n" + hint1 + "\n" + hint3 + "\n" + note
		}
	} else {
		defaultMsg = dim.Render("No hosts.")
	}
	return renderListEmptyState(m.width, m.height, m.search.Value(), defaultMsg)
}
