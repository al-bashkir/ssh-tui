package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var mainTabs = []string{"Hosts", "Groups", "Settings"}

func boxTop(w int) string {
	if w <= 1 {
		return ""
	}
	if w == 2 {
		return "┌┐"
	}
	return "┌" + strings.Repeat("─", w-2) + "┐"
}

func boxTitleTop(w int, title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return boxTop(w)
	}
	if w <= 1 {
		return ""
	}
	if w == 2 {
		return "┌┐"
	}
	innerW := w - 2
	seg := " " + title + " "
	segW := lipgloss.Width(seg)
	if segW > innerW {
		seg = " " + truncateTail(title, max(0, innerW-2)) + " "
		segW = lipgloss.Width(seg)
	}
	fill := innerW - segW
	if fill < 0 {
		fill = 0
	}
	return "┌" + seg + strings.Repeat("─", fill) + "┐"
}

func boxBottom(w int) string {
	if w <= 1 {
		return ""
	}
	if w == 2 {
		return "└┘"
	}
	return "└" + strings.Repeat("─", w-2) + "┘"
}

func boxSep(w int) string {
	if w <= 1 {
		return ""
	}
	if w == 2 {
		return "├┤"
	}
	return "├" + strings.Repeat("─", w-2) + "┤"
}

func boxLine(w int, content string) string {
	if w <= 1 {
		return ""
	}
	if w == 2 {
		return "││"
	}
	innerW := w - 2
	content = strings.TrimRight(content, "\n")
	cw := lipgloss.Width(content)
	if cw > innerW {
		content = lipgloss.NewStyle().MaxWidth(innerW).Render(content)
		cw = lipgloss.Width(content)
	}
	pad := innerW - cw
	if pad < 0 {
		pad = 0
	}
	return "│" + content + strings.Repeat(" ", pad) + "│"
}

func padVisible(s string, width int) string {
	if width <= 0 {
		return s
	}
	s = strings.TrimRight(s, "\n")
	cw := lipgloss.Width(s)
	if cw > width {
		s = lipgloss.NewStyle().MaxWidth(width).Render(s)
		cw = lipgloss.Width(s)
	}
	pad := width - cw
	if pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func renderTabsLine(active int, tabs []string) string {
	parts := make([]string, 0, len(tabs))
	for i, t := range tabs {
		if i == active {
			parts = append(parts, tabActiveStyle.Render(t))
		} else {
			parts = append(parts, tabInactiveStyle.Render(t))
		}
	}
	return strings.Join(parts, "  ")
}

func renderMainTabBox(width, height int, activeTab int, headerLeft string, headerRight string, listView string) string {
	return renderMainTabBoxWithFooter(width, height, activeTab, headerLeft, headerRight, listView, "")
}

func renderBreadcrumbTabBox(width, height int, breadcrumb string, headerLeft string, headerRight string, listView string, footer string) string {
	if width <= 0 || height <= 0 {
		return strings.TrimRight(breadcrumb+"\n"+headerLeft+"\n"+listView, "\n")
	}
	if height < 3 {
		return boxTop(width)
	}

	innerW := width - 2
	innerH := height - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	fixed := 4 // breadcrumb + sep + header + sep
	hasFooter := strings.TrimSpace(footer) != ""
	footerLines := []string{}
	if hasFooter {
		footerLines = strings.Split(strings.TrimRight(footer, "\n"), "\n")
		fixed += 1 + len(footerLines)
	}
	contentH := innerH - fixed
	if contentH < 0 {
		contentH = 0
	}

	bc := padVisible(breadcrumb, innerW)
	header := padVisible(joinHeader(innerW, headerLeft, headerRight), innerW)

	content := strings.TrimRight(listView, "\n")
	contentLines := []string{}
	if strings.TrimSpace(content) != "" {
		contentLines = strings.Split(content, "\n")
	}

	out := make([]string, 0, height)
	out = append(out, boxTop(width))
	out = append(out, boxLine(width, bc))
	out = append(out, boxSep(width))
	out = append(out, boxLine(width, header))
	out = append(out, boxSep(width))

	for i := 0; i < contentH; i++ {
		line := ""
		if i < len(contentLines) {
			line = padVisible(contentLines[i], innerW)
		} else {
			line = strings.Repeat(" ", innerW)
		}
		out = append(out, boxLine(width, line))
	}
	if hasFooter {
		out = append(out, boxSep(width))
		for _, fl := range footerLines {
			out = append(out, boxLine(width, padVisible(fl, innerW)))
		}
	}
	out = append(out, boxBottom(width))
	return strings.Join(out, "\n")
}

func renderMainTabBoxWithFooter(width, height int, activeTab int, headerLeft string, headerRight string, listView string, footer string) string {
	if width <= 0 || height <= 0 {
		// Fallback.
		return strings.TrimRight(renderTabsLine(activeTab, mainTabs)+"\n"+headerLeft+"\n"+listView, "\n")
	}
	if height < 3 {
		return boxTop(width)
	}

	innerW := width - 2
	innerH := height - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}

	// tabs + sep + header + sep
	fixed := 4
	hasFooter := strings.TrimSpace(footer) != ""
	footerLines := []string{}
	if hasFooter {
		footerLines = strings.Split(strings.TrimRight(footer, "\n"), "\n")
		fixed += 1 + len(footerLines) // sep + N footer lines
	}
	contentH := innerH - fixed
	if contentH < 0 {
		contentH = 0
	}

	tabs := padVisible(renderTabsLine(activeTab, mainTabs), innerW)
	header := padVisible(joinHeader(innerW, headerLeft, headerRight), innerW)

	// Content lines.
	content := strings.TrimRight(listView, "\n")
	contentLines := []string{}
	if strings.TrimSpace(content) != "" {
		contentLines = strings.Split(content, "\n")
	}

	out := make([]string, 0, height)
	out = append(out, boxTop(width))
	out = append(out, boxLine(width, tabs))
	out = append(out, boxSep(width))
	out = append(out, boxLine(width, header))
	out = append(out, boxSep(width))

	for i := 0; i < contentH; i++ {
		line := ""
		if i < len(contentLines) {
			line = padVisible(contentLines[i], innerW)
		} else {
			line = strings.Repeat(" ", innerW)
		}
		out = append(out, boxLine(width, line))
	}
	if hasFooter {
		out = append(out, boxSep(width))
		for _, fl := range footerLines {
			out = append(out, boxLine(width, padVisible(fl, innerW)))
		}
	}
	out = append(out, boxBottom(width))
	return strings.Join(out, "\n")
}
