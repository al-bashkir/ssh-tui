package ui

import (
	"strings"
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

// renderFormBox renders a scrollable modal form box with a title, visible
// content lines, optional toast, and a pre-styled footer.
func renderFormBox(totalW int, title string, visibleLines []string, visibleH int, toastStr string, footer string) string {
	innerW := max(0, totalW-2)
	out := make([]string, 0, visibleH+5)
	out = append(out, focusedBoxTitleTop(totalW, title))
	for _, ln := range visibleLines {
		out = append(out, focusedBoxLine(totalW, padVisible(ln, innerW)))
	}
	for i := len(visibleLines); i < visibleH; i++ {
		out = append(out, focusedBoxLine(totalW, strings.Repeat(" ", innerW)))
	}
	if toastStr != "" {
		out = append(out, focusedBoxLine(totalW, padVisible(toastStr, innerW)))
	}
	out = append(out, focusedBoxSep(totalW))
	out = append(out, focusedBoxLine(totalW, padVisible(footer, innerW)))
	out = append(out, focusedBoxBottom(totalW))
	return strings.Join(out, "\n")
}
