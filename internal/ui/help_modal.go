package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(cFrameBorder).
			Padding(1, 2)

	helpTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
)

// helpContent generates the rendered help text body (without the box).
func helpContent(title string, h help.Model, keys helpMap, innerW int) string {
	hh := h
	hh.ShowAll = true
	hh.Width = innerW
	keyStyle := lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	hh.Styles.ShortKey = keyStyle
	hh.Styles.FullKey = keyStyle

	header := helpTitleStyle.Render(title + " keybindings")
	body := strings.TrimSpace(hh.View(keys))
	footer := dim.Render("Esc or ? to close  j/k scroll")
	return header + "\n\n" + body + "\n\n" + footer
}

func helpBoxWidth(termW int) int {
	boxW := min(88, termW-4)
	if boxW < 30 {
		boxW = min(termW, 30)
	}
	return boxW
}

func helpInnerWidth(boxW int) int {
	innerW := boxW - 6
	if innerW < 20 {
		innerW = 0
	}
	return innerW
}

// initHelpViewport creates a viewport sized for the help modal and sets its content.
func initHelpViewport(width, height int, title string, h help.Model, keys helpMap) viewport.Model {
	boxW := helpBoxWidth(width)
	innerW := helpInnerWidth(boxW)

	content := helpContent(title, h, keys, innerW)

	// Size viewport to fit content, but cap at available terminal height.
	contentLines := strings.Count(content, "\n") + 1
	// borders (2) + padding (2) = 4 lines overhead
	maxVPH := height - 4
	if maxVPH < 3 {
		maxVPH = 3
	}
	vpH := min(contentLines, maxVPH)

	vp := viewport.New(innerW, vpH)
	vp.SetContent(content)
	return vp
}

// updateHelpViewport sends a key message to the viewport for scrolling.
func updateHelpViewport(vp *viewport.Model, msg tea.KeyMsg) {
	s := msg.String()
	switch s {
	case "j", "down":
		vp.ScrollDown(1)
	case "k", "up":
		vp.ScrollUp(1)
	case "pgdown", "ctrl+d":
		vp.HalfPageDown()
	case "pgup", "ctrl+u":
		vp.HalfPageUp()
	}
}

func renderHelpModal(width, height int, title string, h help.Model, keys helpMap) string {
	return renderHelpModalWithVP(width, height, title, h, keys, nil)
}

func renderHelpModalWithVP(width, height int, title string, h help.Model, keys helpMap, vp *viewport.Model) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Help"
	}

	// Fallback for very early render before we have window size.
	if width <= 0 || height <= 0 {
		hh := h
		hh.ShowAll = true
		hh.Width = 0
		return helpTitleStyle.Render(title) + "\n\n" + hh.View(keys)
	}

	boxW := helpBoxWidth(width)
	innerW := helpInnerWidth(boxW)

	if vp != nil {
		content := vp.View()

		// Scroll indicator when content overflows.
		if vp.TotalLineCount() > vp.VisibleLineCount() {
			pct := int(vp.ScrollPercent() * 100)
			arrows := ""
			if vp.ScrollPercent() > 0 {
				arrows += "▲ "
			}
			if vp.ScrollPercent() < 1 {
				arrows += "▼ "
			}
			content += "\n" + dim.Render(fmt.Sprintf("%s%d%%", arrows, pct))
		}

		box := helpBoxStyle.Width(boxW).Render(content)
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
	}

	// Non-viewport fallback (legacy).
	content := helpContent(title, h, keys, innerW)
	box := helpBoxStyle.Width(boxW).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
