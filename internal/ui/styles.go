package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

var (
	// theme is the active color scheme. Set by applyScheme (via
	// ApplyColorScheme at init and on every settings change).
	theme colorScheme

	// bgANSICode is the precomputed ANSI escape sequence for theme.Background,
	// e.g. "\x1b[48;2;40;42;54m". Empty when no theme background is active.
	bgANSICode string
)

func init() { ApplyColorScheme("", "") }

// colorScheme holds the full set of color variables for a named theme.
type colorScheme struct {
	Accent      lipgloss.AdaptiveColor
	Muted       lipgloss.AdaptiveColor
	OK          lipgloss.AdaptiveColor
	Warn        lipgloss.AdaptiveColor
	Err         lipgloss.AdaptiveColor
	FrameBorder lipgloss.AdaptiveColor
	SearchDim   lipgloss.AdaptiveColor
	RowActiveBG lipgloss.AdaptiveColor
	RowActiveFG lipgloss.AdaptiveColor
	Background  string // hex color for frame/modal background; "" = terminal default

	FocusedBorder lipgloss.AdaptiveColor // border of focused panel/modal
	Disabled      lipgloss.AdaptiveColor // disabled/unavailable elements
	Hint          lipgloss.AdaptiveColor // helper/placeholder text
	Secondary     lipgloss.AdaptiveColor // secondary accent (section headers, breadcrumbs)
}

// themeColor creates an AdaptiveColor from a single truecolor hex value.
// Named themes are dark-themed so the same hex is used for both light and dark modes.
func themeColor(hex string) lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: hex, Dark: hex}
}

// builtinColorSchemes maps scheme names to their full color definitions.
// Named theme colors are sourced from official palette specifications.
// The "default" entry uses adaptive ANSI codes for proper light/dark terminal support.
var builtinColorSchemes = map[string]colorScheme{
	"default": {
		// Adaptive ANSI 256 codes — no background override.
		Accent:        lipgloss.AdaptiveColor{Light: "25", Dark: "39"},
		Muted:         lipgloss.AdaptiveColor{Light: "242", Dark: "242"},
		OK:            lipgloss.AdaptiveColor{Light: "28", Dark: "35"},
		Warn:          lipgloss.AdaptiveColor{Light: "166", Dark: "214"},
		Err:           lipgloss.AdaptiveColor{Light: "160", Dark: "203"},
		FrameBorder:   lipgloss.AdaptiveColor{Light: "250", Dark: "238"},
		SearchDim:     lipgloss.AdaptiveColor{Light: "247", Dark: "246"},
		RowActiveBG:   lipgloss.AdaptiveColor{Light: "253", Dark: "238"},
		RowActiveFG:   lipgloss.AdaptiveColor{Light: "0", Dark: "255"},
		FocusedBorder: lipgloss.AdaptiveColor{Light: "25", Dark: "39"},   // = accent
		Disabled:      lipgloss.AdaptiveColor{Light: "248", Dark: "240"}, // lighter gray
		Hint:          lipgloss.AdaptiveColor{Light: "245", Dark: "245"}, // mid gray
		Secondary:     lipgloss.AdaptiveColor{Light: "67", Dark: "67"},   // steel blue
	},
	// Dracula — https://draculatheme.com/contribute
	"dracula": {
		Accent:        themeColor("#bd93f9"), // purple
		Muted:         themeColor("#6272a4"), // comment
		OK:            themeColor("#50fa7b"), // green
		Warn:          themeColor("#ffb86c"), // orange
		Err:           themeColor("#ff5555"), // red
		FrameBorder:   themeColor("#44475a"), // current line
		SearchDim:     themeColor("#6272a4"), // comment
		RowActiveBG:   themeColor("#44475a"), // selection
		RowActiveFG:   themeColor("#f8f8f2"), // foreground
		Background:    "#282a36",             // background
		FocusedBorder: themeColor("#bd93f9"), // purple (= accent)
		Disabled:      themeColor("#383a59"), // darker than current line
		Hint:          themeColor("#6272a4"), // comment
		Secondary:     themeColor("#ff79c6"), // pink
	},
	// Nord — https://www.nordtheme.com/docs/colors-and-palettes
	"nord": {
		Accent:        themeColor("#88c0d0"), // nord8 — frost teal
		Muted:         themeColor("#4c566a"), // nord3 — polar night (dim)
		OK:            themeColor("#a3be8c"), // nord14 — aurora green
		Warn:          themeColor("#ebcb8b"), // nord13 — aurora yellow
		Err:           themeColor("#bf616a"), // nord11 — aurora red
		FrameBorder:   themeColor("#3b4252"), // nord1 — polar night
		SearchDim:     themeColor("#4c566a"), // nord3
		RowActiveBG:   themeColor("#434c5e"), // nord2 — selection
		RowActiveFG:   themeColor("#eceff4"), // nord6
		Background:    "#2e3440",             // nord0 — polar night base
		FocusedBorder: themeColor("#88c0d0"), // frost teal (= accent)
		Disabled:      themeColor("#434c5e"), // nord2
		Hint:          themeColor("#4c566a"), // nord3
		Secondary:     themeColor("#81a1c1"), // nord9 — frost light
	},
	// Gruvbox dark — https://github.com/morhetz/gruvbox
	"gruvbox": {
		Accent:        themeColor("#fe8019"), // bright orange
		Muted:         themeColor("#928374"), // gray
		OK:            themeColor("#b8bb26"), // bright green
		Warn:          themeColor("#fabd2f"), // bright yellow
		Err:           themeColor("#fb4934"), // bright red
		FrameBorder:   themeColor("#504945"), // bg2 — warm mid-dark
		SearchDim:     themeColor("#928374"), // gray
		RowActiveBG:   themeColor("#665c54"), // bg3 — selection
		RowActiveFG:   themeColor("#ebdbb2"), // fg
		Background:    "#282828",             // bg
		FocusedBorder: themeColor("#fe8019"), // bright orange (= accent)
		Disabled:      themeColor("#504945"), // bg2
		Hint:          themeColor("#665c54"), // bg3
		Secondary:     themeColor("#fabd2f"), // bright yellow
	},
	// Catppuccin Mocha — https://github.com/catppuccin/catppuccin
	"catppuccin": {
		Accent:        themeColor("#cba6f7"), // mauve
		Muted:         themeColor("#6c7086"), // overlay0
		OK:            themeColor("#a6e3a1"), // green
		Warn:          themeColor("#f9e2af"), // yellow
		Err:           themeColor("#f38ba8"), // red
		FrameBorder:   themeColor("#45475a"), // surface1 — border
		SearchDim:     themeColor("#585b70"), // overlay2
		RowActiveBG:   themeColor("#585b70"), // overlay2 — selection
		RowActiveFG:   themeColor("#cdd6f4"), // text
		Background:    "#1e1e2e",             // base
		FocusedBorder: themeColor("#cba6f7"), // mauve (= accent)
		Disabled:      themeColor("#313244"), // surface0
		Hint:          themeColor("#6c7086"), // overlay0
		Secondary:     themeColor("#b4befe"), // lavender
	},
	// Kanagawa Wave — https://github.com/rebelot/kanagawa.nvim
	"kanagawa": {
		Accent:        themeColor("#7e9cd8"), // crystalBlue
		Muted:         themeColor("#727169"), // fujiGray
		OK:            themeColor("#98bb6c"), // springGreen
		Warn:          themeColor("#ffa066"), // surimiOrange
		Err:           themeColor("#e46876"), // peachRed
		FrameBorder:   themeColor("#2a2a37"), // bg_p1
		SearchDim:     themeColor("#727169"), // fujiGray
		RowActiveBG:   themeColor("#223249"), // waveBlue1 — selection
		RowActiveFG:   themeColor("#dcd7ba"), // fujiWhite
		Background:    "#1f1f28",             // sumiInk2 / base bg
		FocusedBorder: themeColor("#7e9cd8"), // crystalBlue (= accent)
		Disabled:      themeColor("#363646"), // sumiInk3
		Hint:          themeColor("#54546d"), // sumiInk4
		Secondary:     themeColor("#938aa9"), // springViolet1
	},
}

var accentPresets = map[string]lipgloss.AdaptiveColor{
	"default": {Light: "25", Dark: "39"},
	"blue":    {Light: "25", Dark: "39"},
	"cyan":    {Light: "30", Dark: "45"},
	"green":   {Light: "28", Dark: "35"},
	"amber":   {Light: "166", Dark: "214"},
	"red":     {Light: "160", Dark: "203"},
	"magenta": {Light: "127", Dark: "213"},
}

// Every style below is (re)built by rebuildAllStyles. Declarations only.
var (
	statusOK   lipgloss.Style
	statusWarn lipgloss.Style
	statusErr  lipgloss.Style
	dim        lipgloss.Style

	// accentStyle is bold accent text: headers, keys, focused values.
	accentStyle lipgloss.Style

	hintStyle      lipgloss.Style
	secondaryStyle lipgloss.Style
	disabledStyle  lipgloss.Style

	focusedFrameStyle lipgloss.Style

	// Active list row: solid background + foreground + bold — no inner styles allowed.
	rowActiveStyle lipgloss.Style

	badgeCfgStyle   lipgloss.Style
	badgeCountStyle lipgloss.Style
	// Selection count pill badge — inverted accent.
	badgeSelStyle lipgloss.Style

	searchUnfocused lipgloss.Style

	confirmTitleStyle lipgloss.Style
	helpBoxStyle      lipgloss.Style

	// Box-drawing chrome: tab box borders, focused modal borders, and
	// interior padding filled with the theme background.
	tabBoxBorderStyle     lipgloss.Style
	focusedBoxBorderStyle lipgloss.Style
	tabBoxPadStyle        lipgloss.Style

	// underlineFill styles the trailing rule of form inputs.
	underlineFill lipgloss.Style
)

// ApplyColorScheme applies a named colorscheme if scheme is non-empty and
// known; otherwise falls back to the default scheme with accentColor, for
// configs that only set accent_color.
func ApplyColorScheme(scheme, accentColor string) {
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	if cs, ok := builtinColorSchemes[scheme]; ok && scheme != "default" {
		applyScheme(cs)
		return
	}

	accent := strings.ToLower(strings.TrimSpace(accentColor))
	if accent == "" {
		accent = "default"
	}
	cs := builtinColorSchemes["default"]
	if v, ok := accentPresets[accent]; ok {
		cs.Accent = v
	} else {
		// Allow arbitrary lipgloss color values ("#RRGGBB", "34", "colour196", ...).
		cs.Accent = lipgloss.AdaptiveColor{Light: accent, Dark: accent}
	}
	cs.FocusedBorder = cs.Accent
	applyScheme(cs)
}

// applyScheme makes cs the active theme and rebuilds every style.
func applyScheme(cs colorScheme) {
	theme = cs
	bgANSICode = hexToANSIBG(cs.Background)
	rebuildAllStyles()
}

// rebuildAllStyles reconstructs every style variable from theme. This is the
// single source of truth for all style construction.
func rebuildAllStyles() {
	// withBG adds the theme background, if any.
	withBG := func(s lipgloss.Style) lipgloss.Style {
		if theme.Background != "" {
			return s.Background(lipgloss.Color(theme.Background))
		}
		return s
	}

	statusOK = lipgloss.NewStyle().Foreground(theme.OK)
	statusWarn = lipgloss.NewStyle().Foreground(theme.Warn)
	statusErr = lipgloss.NewStyle().Foreground(theme.Err)
	dim = lipgloss.NewStyle().Foreground(theme.Muted)
	accentStyle = lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)

	hintStyle = lipgloss.NewStyle().Foreground(theme.Hint).Italic(true)
	secondaryStyle = lipgloss.NewStyle().Foreground(theme.Secondary)
	disabledStyle = lipgloss.NewStyle().Foreground(theme.Disabled).Italic(true)
	underlineFill = lipgloss.NewStyle().Foreground(theme.FrameBorder)

	focusedFrameStyle = withBG(lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.FocusedBorder).
		Padding(0, 1))
	helpBoxStyle = withBG(lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.FocusedBorder).
		Padding(1, 2))

	rowActiveStyle = lipgloss.NewStyle().Background(theme.RowActiveBG).Foreground(theme.RowActiveFG).Bold(true)

	badgeCfgStyle = lipgloss.NewStyle().
		Foreground(theme.Accent).
		Background(lipgloss.AdaptiveColor{Light: "254", Dark: "235"}).
		Padding(0, 1).Bold(true)
	badgeCountStyle = lipgloss.NewStyle().
		Foreground(theme.Muted).
		Background(lipgloss.AdaptiveColor{Light: "254", Dark: "236"}).
		Padding(0, 1)
	badgeSelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "255", Dark: "16"}).
		Background(theme.Accent).
		Padding(0, 1).
		Bold(true)

	searchUnfocused = lipgloss.NewStyle().Foreground(theme.SearchDim)

	tabBoxBorderStyle = withBG(lipgloss.NewStyle().Foreground(theme.FrameBorder))
	focusedBoxBorderStyle = withBG(lipgloss.NewStyle().Foreground(theme.FocusedBorder))
	tabBoxPadStyle = withBG(lipgloss.NewStyle())

	confirmTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Err)
}

// hexToANSIBG converts a "#RRGGBB" hex string to an ANSI truecolor background
// escape sequence "\x1b[48;2;R;G;Bm". Returns "" on parse failure.
func hexToANSIBG(hex string) string {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return ""
	}
	r, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

// withThemeBG prepends bgANSICode to s and re-injects it after every ANSI
// reset ("\x1b[0m" or "\x1b[m") so the theme background persists through all
// styled sub-elements. Does nothing when bgANSICode is empty.
func withThemeBG(s string) string {
	if bgANSICode == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgANSICode)
	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+bgANSICode)
	return bgANSICode + s
}

// frameInnerSize returns the content size of a focused frame: 1-char border
// on each side plus horizontal padding of 1.
func frameInnerSize(w, h int) (innerW, innerH int) {
	return max(0, w-4), max(0, h-2)
}

func joinHeader(width int, left, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if right == "" {
		return left
	}
	if width <= 0 {
		return strings.TrimSpace(left + " " + right)
	}

	rw := lipgloss.Width(right)
	if rw >= width {
		return lipgloss.NewStyle().MaxWidth(width).Render(right)
	}
	leftAvail := width - rw - 1
	if left == "" || leftAvail <= 0 {
		return strings.Repeat(" ", width-rw) + right
	}
	left = lipgloss.NewStyle().MaxWidth(leftAvail).Render(left)
	gap := max(1, width-lipgloss.Width(left)-rw)
	return left + strings.Repeat(" ", gap) + right
}

// renderFocusedFrame renders a bordered modal frame using focusedFrameStyle
// (theme.FocusedBorder) so overlays stand out from the background tab box.
func renderFocusedFrame(w, h int, title string, headerRight string, body string, footer string) string {
	if strings.TrimSpace(footer) == "" {
		footer = ""
	}
	if w <= 0 || h <= 0 {
		out := ""
		if header := joinHeader(0, title, headerRight); header != "" {
			out = accentStyle.Render(header) + "\n"
		}
		out += strings.TrimSpace(body)
		if footer != "" {
			out += "\n" + footer
		}
		return strings.TrimSpace(out)
	}

	innerW, _ := frameInnerSize(w, h)
	content := strings.TrimRight(accentStyle.Render(joinHeader(innerW, title, headerRight))+"\n"+body, "\n")
	if footer != "" {
		content += "\n" + footer
	}
	return withThemeBG(focusedFrameStyle.Width(w).Height(h).Render(content))
}

func configureSearch(m *textinput.Model) {
	m.PromptStyle = accentStyle
	m.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "255"})
	m.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Accent)
	m.PlaceholderStyle = hintStyle
}

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
	for i, line := range lines {
		var styled []string
		for _, p := range strings.Split(line, "  ") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if p == "·" {
				styled = append(styled, hintStyle.Render("·"))
				continue
			}
			k, a, hasAction := strings.Cut(p, " ")
			if !hasAction {
				styled = append(styled, accentStyle.Render(p))
				continue
			}
			styled = append(styled, accentStyle.Render(k)+hintStyle.Render(" "+a))
		}
		lines[i] = strings.Join(styled, "  ")
	}
	return strings.Join(lines, "\n")
}

// formScrollWindow returns the (start, end) line indexes to display, scrolled
// so that focusLine is centered within visibleH lines where possible.
func formScrollWindow(totalLines, visibleH, focusLine int) (int, int) {
	if totalLines <= visibleH {
		return 0, totalLines
	}
	start := min(max(0, focusLine-visibleH/2), totalLines-visibleH)
	return start, start + visibleH
}

// formSection renders a lightweight section divider: "── Label ──────"
func formSection(label string, width int) string {
	seg := "── " + strings.TrimSpace(label) + " "
	return secondaryStyle.Render(seg + strings.Repeat("─", max(0, width-lipgloss.Width(seg))))
}

// statusDot returns a colored dot for status display.
func statusDot(hasWarnings bool) string {
	if hasWarnings {
		return statusWarn.Render("●")
	}
	return statusOK.Render("●")
}
