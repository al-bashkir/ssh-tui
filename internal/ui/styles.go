package ui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	defaultAccent = lipgloss.AdaptiveColor{Light: "25", Dark: "39"} // blue/cyan
	cAccent       = defaultAccent
	cMuted        = lipgloss.AdaptiveColor{Light: "242", Dark: "242"} // gray
	cOK           = lipgloss.AdaptiveColor{Light: "28", Dark: "35"}
	cWarn         = lipgloss.AdaptiveColor{Light: "166", Dark: "214"}
	cErr          = lipgloss.AdaptiveColor{Light: "160", Dark: "203"}

	cSearchDim   = lipgloss.AdaptiveColor{Light: "247", Dark: "246"}
	cFrameBorder = lipgloss.AdaptiveColor{Light: "250", Dark: "238"}

	// Active list row: solid background bar with contrasting text.
	cRowActiveBG = lipgloss.AdaptiveColor{Light: "253", Dark: "238"}
	cRowActiveFG = lipgloss.AdaptiveColor{Light: "0", Dark: "255"}

	// Vivid per-accent bg/fg for form segment focus (option pickers).
	defaultSegFocusedBG = lipgloss.AdaptiveColor{Light: "153", Dark: "24"}
	cSegFocusedBG       = defaultSegFocusedBG
	cSegFocusedFG       = lipgloss.AdaptiveColor{Light: "17", Dark: "231"}

)

var accentPresets = map[string]lipgloss.AdaptiveColor{
	"default": {Light: "25", Dark: "39"},
	"blue":    {Light: "25", Dark: "39"},
	"cyan":    {Light: "30", Dark: "45"},
	"green":   {Light: "28", Dark: "35"},
	"amber":   {Light: "166", Dark: "214"},
	"red":     {Light: "160", Dark: "203"},
	"magenta": {Light: "127", Dark: "213"},
}

// segFocusedBGPreset maps accent names to vivid backgrounds for form option pickers.
var segFocusedBGPreset = map[string]lipgloss.AdaptiveColor{
	"default": {Light: "153", Dark: "24"},
	"blue":    {Light: "153", Dark: "24"},
	"cyan":    {Light: "159", Dark: "30"},
	"green":   {Light: "157", Dark: "22"},
	"amber":   {Light: "229", Dark: "94"},
	"red":     {Light: "224", Dark: "88"},
	"magenta": {Light: "225", Dark: "90"},
}

var (
	statusOK   = lipgloss.NewStyle().Foreground(cOK)
	statusWarn = lipgloss.NewStyle().Foreground(cWarn)
	statusErr  = lipgloss.NewStyle().Foreground(cErr)
	dim        = lipgloss.NewStyle().Foreground(cMuted)

	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(cFrameBorder).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	footerStyle = lipgloss.NewStyle().Foreground(cMuted)

	checkedStyle   = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	uncheckedStyle = lipgloss.NewStyle().Foreground(cMuted)

	// Active list row: solid background + foreground + bold — no inner styles allowed.
	rowActiveStyle = lipgloss.NewStyle().Background(cRowActiveBG).Foreground(cRowActiveFG).Bold(true)

	// Form option picker focus: vivid accent background (unchanged from original behavior).
	segFocusedStyle = lipgloss.NewStyle().Background(cSegFocusedBG).Foreground(cSegFocusedFG).Bold(true)

	badgeCfgStyle   = lipgloss.NewStyle().Foreground(cAccent).Background(lipgloss.AdaptiveColor{Light: "254", Dark: "235"}).Padding(0, 1).Bold(true)
	badgeCountStyle = lipgloss.NewStyle().Foreground(cMuted).Background(lipgloss.AdaptiveColor{Light: "254", Dark: "236"}).Padding(0, 1)

	// Selection count pill badge — inverted accent.
	badgeSelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "255", Dark: "16"}).
			Background(cAccent).
			Padding(0, 1).
			Bold(true)

	footerKeyStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)

	tabActiveStyle   = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	tabInactiveStyle = lipgloss.NewStyle().Foreground(cMuted)
)


func SetAccentColor(name string) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || name == "default" {
		cAccent = defaultAccent
	} else if v, ok := accentPresets[name]; ok {
		cAccent = v
	} else {
		// Allow arbitrary lipgloss color values ("#RRGGBB", "34", "colour196", ...).
		cAccent = lipgloss.AdaptiveColor{Light: name, Dark: name}
	}

	bgKey := name
	if bgKey == "" {
		bgKey = "default"
	}
	if v, ok := segFocusedBGPreset[bgKey]; ok {
		cSegFocusedBG = v
	} else {
		cSegFocusedBG = cAccent
	}

	// Rebuild styles that capture color vars at creation time.
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	checkedStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	badgeCfgStyle = lipgloss.NewStyle().Foreground(cAccent).Background(lipgloss.AdaptiveColor{Light: "254", Dark: "235"}).Padding(0, 1).Bold(true)
	badgeSelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "255", Dark: "16"}).
		Background(cAccent).
		Padding(0, 1).
		Bold(true)
	tabActiveStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	footerKeyStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)

	// rowActiveStyle stays fixed (subtle gray bg, no accent dependency).
	segFocusedStyle = lipgloss.NewStyle().Background(cSegFocusedBG).Foreground(cSegFocusedFG).Bold(true)

	// Help modal title style lives in help_modal.go.
	helpTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
}

func frameInnerSize(w, h int) (innerW, innerH int) {
	// frameStyle has 1-char border on each side + horizontal padding=1.
	innerW = w - 2 - 2
	innerH = h - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}
	return innerW, innerH
}

func joinHeader(width int, left, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if width <= 0 {
		if right == "" {
			return left
		}
		if left == "" {
			return right
		}
		return left + " " + right
	}
	if right == "" {
		return left
	}

	rw := lipgloss.Width(right)
	if rw >= width {
		return lipgloss.NewStyle().MaxWidth(width).Render(right)
	}

	if left == "" {
		return strings.Repeat(" ", width-rw) + right
	}

	leftAvail := width - rw - 1
	if leftAvail <= 0 {
		return strings.Repeat(" ", width-rw) + right
	}
	left = lipgloss.NewStyle().MaxWidth(leftAvail).Render(left)
	lw := lipgloss.Width(left)
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func renderFrame(w, h int, title string, headerRight string, body string, footer string) string {
	if w <= 0 || h <= 0 {
		// Early render fallback.
		out := strings.TrimSpace(title)
		if out != "" {
			header := out
			if strings.TrimSpace(headerRight) != "" {
				header = header + " " + strings.TrimSpace(headerRight)
			}
			out = headerStyle.Render(header) + "\n"
		}
		out += strings.TrimSpace(body)
		if strings.TrimSpace(footer) != "" {
			out += "\n" + footer
		}
		return strings.TrimSpace(out)
	}

	innerW, _ := frameInnerSize(w, h)
	head := headerStyle.Render(joinHeader(innerW, title, headerRight))
	foot := ""
	if strings.TrimSpace(footer) != "" {
		foot = footer
	}

	content := strings.TrimRight(head+"\n"+body, "\n")
	if foot != "" {
		content = strings.TrimRight(content, "\n") + "\n" + foot
	}
	box := frameStyle.Width(w).Height(h).Render(content)
	return box
}

func configureSearch(m *textinput.Model) {
	m.PromptStyle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	m.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "255"})
	m.Cursor.Style = lipgloss.NewStyle().Foreground(cAccent)
}

var searchUnfocused = lipgloss.NewStyle().Foreground(cSearchDim)

func setSearchFocused(m *textinput.Model, focused bool) {
	if focused {
		configureSearch(m)
		return
	}
	m.PromptStyle = searchUnfocused
	m.TextStyle = searchUnfocused
	m.Cursor.Style = searchUnfocused
}

func setSearchBarFocused(m *textinput.Model, focused bool) {
	setSearchFocused(m, focused)
	if focused {
		m.Placeholder = "search"
	} else {
		m.Placeholder = "type to search..."
	}
}

// styledFooter renders a footer string with keys in accent and actions dimmed.
// Input format: "⏎ connect  ␣ select  o panes" (double-space separated hints).
func styledFooter(raw string) string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "  ")
		styled := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if p == "·" {
				styled = append(styled, dim.Render("·"))
				continue
			}
			idx := strings.IndexByte(p, ' ')
			if idx < 0 {
				styled = append(styled, footerKeyStyle.Render(p))
				continue
			}
			k := p[:idx]
			a := p[idx:] // includes leading space
			styled = append(styled, footerKeyStyle.Render(k)+dim.Render(a))
		}
		out = append(out, strings.Join(styled, "  "))
	}
	return strings.Join(out, "\n")
}


// Spinner for loading states.
var (
	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIndex  int
	spinnerActive bool
	// spinnerMinEnd ensures the spinner shows for at least spinnerMinDuration.
	spinnerMinEnd time.Time
)

const (
	spinnerTickInterval = 80 * time.Millisecond
	spinnerMinDuration  = 600 * time.Millisecond
)

type spinnerTickMsg struct{}

// spinnerStart activates the spinner with a minimum visible duration.
func spinnerStart() {
	spinnerActive = true
	spinnerIndex = 0
	spinnerMinEnd = time.Now().Add(spinnerMinDuration)
}

// spinnerStop deactivates the spinner (but it may keep running until min duration).
func spinnerStop() {
	// The tick handler checks time and will deactivate.
	if time.Now().After(spinnerMinEnd) {
		spinnerActive = false
	}
	// Otherwise the tick handler will stop it when minEnd is reached.
}

func spinnerFrame() string {
	return spinnerFrames[spinnerIndex%len(spinnerFrames)]
}

// formScrollWindow returns the slice of lines to display, scrolled so that
// focusLine is visible within visibleH lines. Returns (start, end) indices.
func formScrollWindow(totalLines, visibleH, focusLine int) (int, int) {
	if totalLines <= visibleH {
		return 0, totalLines
	}
	// Center focused line in the window.
	start := focusLine - visibleH/2
	if start < 0 {
		start = 0
	}
	end := start + visibleH
	if end > totalLines {
		end = totalLines
		start = end - visibleH
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

// formSection renders a lightweight section divider: "── Label ──────"
func formSection(label string, width int) string {
	label = strings.TrimSpace(label)
	seg := "── " + label + " "
	segW := lipgloss.Width(seg)
	fill := width - segW
	if fill < 0 {
		fill = 0
	}
	return dim.Render(seg + strings.Repeat("─", fill))
}

// statusDot returns a colored dot for status display.
func statusDot(ok bool, hasWarnings bool) string {
	if !ok || hasWarnings {
		return statusWarn.Render("●")
	}
	return statusOK.Render("●")
}
