package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *hostsModel) View() string {
	if m.showHelp {
		return renderHelpModalWithVP(m.width, m.height, "Hosts", m.help, m.helpKeys(), &m.helpVP)
	}
	if m.cmdPrompt {
		mw, mh := modalSize(m.width, m.height, 88, 9, 6, 10)
		var b strings.Builder
		b.WriteString("Connect and run a remote command (keeps session open).\n\n")
		b.WriteString(m.cmdInput.View())
		b.WriteString("\n")
		b.WriteString(footerStyle.Render("Enter connect  Esc cancel"))
		box := renderFrame(mw, mh, breadcrumbTitle("Hosts", "Command"), "", strings.TrimRight(b.String(), "\n"), "")
		return placeCentered(m.width, m.height, box)
	}
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.confirmConnect {
		modal := connectConfirmBox(max(0, m.width-4), m.confirmConnectCount, m.confirmConnectHosts)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
	}

	hasWarn := m.opts.SkippedLines > 0 || len(m.opts.LoadErrors) > 0
	right := ""
	if !m.toast.empty() {
		right = renderToastWithSpinner(m.toast, spinnerActive)
	} else if spinnerActive {
		right = statusWarn.Render(spinnerFrame())
	} else {
		right = statusDot(true, hasWarn)
		if hasWarn {
			right += dim.Render(" load warnings")
		}
		if selCount := len(m.selected); selCount > 0 {
			right += "  " + badgeSelStyle.Render(fmt.Sprintf("%d selected", selCount))
		}
		shown := len(m.list.Items())
		total := len(m.allHosts)
		q := strings.TrimSpace(m.search.Value())
		if q != "" {
			right += dim.Render(fmt.Sprintf(" %d / %d hosts", shown, total))
		} else {
			right += dim.Render(fmt.Sprintf(" %d hosts", total))
		}
		if hc := m.hiddenCount(); hc > 0 {
			if m.showHidden {
				right += "  " + headerStyle.Render(fmt.Sprintf("%d showed", hc))
			} else {
				right += "  " + dim.Render(fmt.Sprintf("%d hidden", hc))
			}
		}
	}
	var footer string
	hasSel := len(m.selected) > 0
	if m.width < 60 {
		if hasSel {
			footer = styledFooter("\u21b5 connect  o panes  \u2423 clear  ? help")
		} else {
			footer = styledFooter("\u21b5 connect  \u2423 select  ? help")
		}
	} else {
		if hasSel {
			footer = styledFooter("\u21b5 connect  o panes  ·  Ctrl+o cmd  a add-to-group  ·  \u2423 clear")
			if m.height >= 20 {
				footer += "\n" + styledFooter("e config  r reload  ·  g groups  ? help")
			}
		} else {
			footer = styledFooter("\u21b5 connect  O pane  ·  \u2423 select  o panes  ·  c custom  g groups  Ctrl+H hide")
			if m.height >= 20 {
				footer += "\n" + styledFooter("e config  Ctrl+o cmd  a add  r reload  ·  tab search  H show hidden  ? help")
			}
		}
	}

	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		listContent = m.emptyStateView()
	}
	return renderMainTabBoxWithFooter(m.width, m.height, 0, m.search.View(), right, listContent, footer)
}

func (m *hostsModel) emptyStateView() string {
	innerW := max(0, m.width-2)
	innerH := max(0, m.height-2)
	contentH := max(0, innerH-6) // tabs+sep+header+sep+footsep+footer

	q := strings.TrimSpace(m.search.Value())
	dots := dim.Render("·  ·  ·")
	var msg string
	if q != "" {
		msg = dots + "\n\n" + dim.Render(fmt.Sprintf("No matches for %q", q)) + "\n" + dim.Render("Esc to clear search")
	} else if len(m.allHosts) == 0 {
		divider := formSection("", 26)
		hint1 := footerKeyStyle.Render("c") + dim.Render("          custom host")
		hint3 := footerKeyStyle.Render("Ctrl+s") + dim.Render("     open settings")
		if m.opts.Config.Defaults.LoadKnownHosts {
			hint2 := footerKeyStyle.Render("r") + dim.Render("          reload known_hosts")
			msg = dots + "\n\n" + dim.Render("No hosts found.") + "\n" + divider + "\n" + hint1 + "\n" + hint2 + "\n" + hint3
		} else {
			note := dim.Render("(known_hosts loading is off)")
			msg = dots + "\n\n" + dim.Render("No hosts found.") + "\n" + divider + "\n" + hint1 + "\n" + hint3 + "\n" + note
		}
	} else {
		msg = dots + "\n\n" + dim.Render("No hosts.")
	}

	return lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, msg)
}

func (m *hostsModel) statusLine() string {
	shown := len(m.list.Items())
	total := len(m.allHosts)
	sel := len(m.selected)
	pg := ""
	if m.list.Paginator.TotalPages > 1 {
		pg = fmt.Sprintf("pg:%d/%d", m.list.Paginator.Page+1, m.list.Paginator.TotalPages)
	}

	q := strings.TrimSpace(m.search.Value())
	searchInfo := "search"
	if q != "" {
		if len(q) > 40 {
			q = q[:40] + "..."
		}
		searchInfo = "search: " + q
	}
	searchStatus := statusOK.Render(searchInfo)

	loadBits := []string{}
	if m.opts.SkippedLines > 0 {
		loadBits = append(loadBits, fmt.Sprintf("skipped:%d", m.opts.SkippedLines))
	}
	if len(m.opts.LoadErrors) > 0 {
		loadBits = append(loadBits, fmt.Sprintf("errors:%d", len(m.opts.LoadErrors)))
	}
	loadInfo := ""
	if len(loadBits) > 0 {
		loadInfo = statusWarn.Render("load: " + strings.Join(loadBits, " "))
	}

	pos := ""
	if shown > 0 {
		pos = fmt.Sprintf("  %d of %d", m.list.Index()+1, shown)
	}
	left := fmt.Sprintf("hosts: %d/%d  sel:%d", shown, total, sel) + dim.Render(pos)
	if pg != "" {
		left += "  " + dim.Render(pg)
	}
	if loadInfo != "" {
		left = left + "  " + loadInfo
	}
	if !m.toast.empty() {
		left = left + "  " + renderToast(m.toast)
	}

	return left + "  " + searchStatus
}
