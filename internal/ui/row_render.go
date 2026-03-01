package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func truncateTail(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}

	r := []rune(s)
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max-1]) + "…"
}

// truncateFade truncates with a soft fade: the last visible character and
// the ellipsis are rendered in dim gray instead of a hard cut.
func truncateFade(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	if max <= 2 {
		return dim.Render("…") + strings.Repeat(" ", max-1)
	}

	r := []rune(s)
	cutoff := max - 2
	if cutoff > len(r) {
		cutoff = len(r)
	}
	normal := string(r[:cutoff])
	dimChar := ""
	if cutoff < len(r) {
		dimChar = string(r[cutoff : cutoff+1])
	}
	return normal + dim.Render(dimChar+"…")
}

func badgePlain(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return " " + text + " "
}

func renderHostLikeRow(width int, active bool, selected bool, host string, hasCfg bool, hidden bool) string {
	cur := " "
	if active {
		// Plain cursor — no inner ANSI so rowActiveStyle background fills uniformly.
		cur = "▸"
	}

	checked := "◻"
	if selected {
		checked = "◼"
	}
	if !active {
		// Style only on inactive rows; active rows get uniform rowActiveStyle.
		if selected {
			checked = checkedStyle.Render(checked)
		} else {
			checked = uncheckedStyle.Render(checked)
		}
	}

	prefix := cur + " " + checked + " "

	// Always reserve the same width for the cfg badge regardless of active
	// state so the host name column does not shift when the cursor moves.
	suffix := ""
	suffixW := 0
	if hasCfg {
		styledCfg := " " + badgeCfgStyle.Render("⚙")
		suffixW = lipgloss.Width(styledCfg)
		if active {
			// Same visual width as the styled badge (padding 0,1 = 1 space each side).
			suffix = "  ⚙ "
		} else {
			suffix = styledCfg
		}
	}

	// Compute host width budget.
	hostAvail := 0
	if width > 0 {
		hostAvail = width - lipgloss.Width(prefix) - suffixW
		if hostAvail < 0 {
			hostAvail = 0
			suffix = ""
			suffixW = 0
		}
	}

	// For hidden hosts, prepend ⊘ prefix to the display string.
	displayHost := host
	if hidden {
		displayHost = "⊘ " + host
	}

	hostStr := displayHost
	if width > 0 {
		if active {
			hostStr = truncateTail(displayHost, hostAvail)
		} else {
			hostStr = truncateFade(displayHost, hostAvail)
		}
	}

	if !active && hidden {
		hostStr = dim.Render(hostStr)
	}

	line := prefix + hostStr + suffix
	if width > 0 && active {
		// Fill to width for a full-row highlight.
		need := width - lipgloss.Width(line)
		if need > 0 {
			line = line + strings.Repeat(" ", need)
		}
	}

	if active {
		line = rowActiveStyle.Render(line)
	}
	return line
}

func renderSimpleRow(width int, active bool, text string) string {
	cur := " "
	if active {
		cur = "▸"
	}
	prefix := cur + " "
	if width > 0 {
		avail := width - lipgloss.Width(prefix)
		if avail < 0 {
			avail = 0
		}
		if active {
			text = truncateTail(text, avail)
		} else {
			text = truncateFade(text, avail)
		}
		line := prefix + text
		if active {
			need := width - lipgloss.Width(line)
			if need > 0 {
				line += strings.Repeat(" ", need)
			}
			return rowActiveStyle.Render(line)
		}
		return line
	}

	line := prefix + text
	if active {
		return rowActiveStyle.Render(line)
	}
	return line
}

func renderGroupRow(width int, active bool, name string, hostCount int, _ bool) string {
	cur := " "
	if active {
		cur = "▸"
	}
	prefix := cur + " "

	// Right-side badge: host count.
	// Always compute layout width from the styled (inactive) version so the
	// name column does not shift when the cursor moves onto or off the row.
	countStr := fmt.Sprintf("%d", hostCount)
	styledCountBadge := " " + badgeCountStyle.Render(countStr)
	countBadgeW := lipgloss.Width(styledCountBadge)
	var countBadge string
	if active {
		// Plain text occupying the same width as the styled badge.
		// badgeCountStyle has Padding(0,1) = 1 space on each side.
		countBadge = "  " + countStr + " "
	} else {
		countBadge = styledCountBadge
	}

	suffix := countBadge
	suffixW := countBadgeW

	if width > 0 {
		availName := width - lipgloss.Width(prefix) - suffixW
		if availName < 0 {
			availName = width - lipgloss.Width(prefix)
			suffix = ""
		}
		if availName < 0 {
			availName = 0
		}
		if active {
			name = truncateTail(name, availName)
		} else {
			name = truncateFade(name, availName)
		}
		line := prefix + name + suffix
		if active {
			pad := width - lipgloss.Width(line)
			if pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			return rowActiveStyle.Render(line)
		}
		return line
	}

	line := prefix + name + suffix
	if active {
		return rowActiveStyle.Render(line)
	}
	return line
}
