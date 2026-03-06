package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

// formLabel renders a padded label for a form field.
// When focused, the label is rendered in accent style.
func formLabel(s string, labelW int, focused bool) string {
	if len(s) < labelW {
		s += strings.Repeat(" ", labelW-len(s))
	}
	if focused {
		return headerStyle.Render(s)
	}
	return s
}

// formSegment renders a single option in a segmented picker.
// If the current value matches val, the option appears selected.
// When focused, the selected option uses the vivid segment style.
func formSegment(cur, val, text string, focused bool) string {
	cur = strings.TrimSpace(cur)
	val = strings.TrimSpace(val)
	if cur == val {
		box := "[" + text + "]"
		if focused {
			return segFocusedStyle.Render(box)
		}
		return checkedStyle.Render(box)
	}
	return tabInactiveStyle.Render(text)
}

// formInputLine renders a text input with underline fill.
// Convenience wrapper that documents intent in form context.
func formInputLine(in textinput.Model, focused bool, w int) string {
	return underlineInput(in, focused, w)
}

// formFooterLine renders the standard form footer with field position and
// hints. In edit mode, shows an INSERT indicator.
func formFooterLine(fieldPos string, editing bool, normalHints string) string {
	if editing {
		return footerStyle.Render(fieldPos) + "  " + headerStyle.Render("INSERT") + "  " + footerStyle.Render("Ctrl+S save   Esc done")
	}
	return footerStyle.Render(fieldPos + "  " + normalHints)
}

// renderFormBox renders a scrollable modal form box with a title, visible
// content lines, optional toast, and a footer.
//
// The footer is wrapped in footerStyle automatically (matching the modal
// form rendering convention where the footer string is plain or mixed-styled).
func renderFormBox(totalW int, title string, visibleLines []string, visibleH int, toastStr string, footer string) string {
	innerW := max(0, totalW-2)
	out := make([]string, 0, visibleH+5)
	out = append(out, boxTitleTop(totalW, title))
	for _, ln := range visibleLines {
		out = append(out, boxLine(totalW, padVisible(ln, innerW)))
	}
	for i := len(visibleLines); i < visibleH; i++ {
		out = append(out, boxLine(totalW, strings.Repeat(" ", innerW)))
	}
	if toastStr != "" {
		out = append(out, boxLine(totalW, padVisible(toastStr, innerW)))
	}
	out = append(out, boxSep(totalW))
	out = append(out, boxLine(totalW, padVisible(footerStyle.Render(footer), innerW)))
	out = append(out, boxBottom(totalW))
	return strings.Join(out, "\n")
}
