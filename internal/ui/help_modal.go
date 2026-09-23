package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpSection is a named group of key bindings shown in the help modal.
type helpSection struct {
	title string
	keys  []key.Binding
}

// helpContent generates the rendered help text body (without the box).
func helpContent(title string, sections []helpSection) string {
	header := accentStyle.Render(title + " keybindings")
	footer := styledFooter("Esc/? close  j/k scroll")
	return header + "\n\n" + renderHelpSections(sections) + "\n\n" + footer
}

// renderHelpSections renders keybindings grouped by named sections.
func renderHelpSections(sections []helpSection) string {
	sectionStyle := secondaryStyle.Bold(true)

	var sb strings.Builder
	for i, sec := range sections {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(sectionStyle.Render(sec.title))
		sb.WriteByte('\n')
		for _, k := range sec.keys {
			if !k.Enabled() {
				continue
			}
			help := k.Help()
			sb.WriteString("  " + accentStyle.Render(help.Key) + hintStyle.Render("  "+help.Desc) + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func helpBoxWidth(termW int) int {
	boxW := min(88, termW-4)
	if boxW < 30 {
		boxW = min(termW, 30)
	}
	return boxW
}

// initHelpViewport creates a viewport sized for the help modal and sets its content.
func initHelpViewport(width, height int, title string, sections []helpSection) viewport.Model {
	// Subtract border (2) + horizontal padding (2*2).
	innerW := helpBoxWidth(width) - 6
	if innerW < 20 {
		innerW = 0
	}
	content := helpContent(title, sections)

	// Size viewport to fit content, but cap at available terminal height.
	contentLines := strings.Count(content, "\n") + 1
	vp := viewport.New(innerW, min(contentLines, max(3, height-4)))
	vp.SetContent(content)
	return vp
}

// updateHelpViewport sends a key message to the viewport for scrolling.
func updateHelpViewport(vp *viewport.Model, msg tea.KeyMsg) {
	switch msg.String() {
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

// renderHelpModal renders the help overlay. vp may be nil for screens that
// do not scroll their help; a fresh viewport is used then.
func renderHelpModal(width, height int, title string, sections []helpSection, vp *viewport.Model) string {
	// Fallback for very early render before we have window size.
	if width <= 0 || height <= 0 {
		return accentStyle.Render(title) + "\n\n" + renderHelpSections(sections)
	}
	if vp == nil {
		v := initHelpViewport(width, height, title, sections)
		vp = &v
	}

	content := vp.View()
	// Scroll indicator when content overflows.
	if vp.TotalLineCount() > vp.VisibleLineCount() {
		arrows := ""
		if vp.ScrollPercent() > 0 {
			arrows += "▲ "
		}
		if vp.ScrollPercent() < 1 {
			arrows += "▼ "
		}
		content += "\n" + dim.Render(fmt.Sprintf("%s%d%%", arrows, int(vp.ScrollPercent()*100)))
	}

	box := helpBoxStyle.Width(helpBoxWidth(width)).Render(content)
	return withThemeBG(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box))
}
