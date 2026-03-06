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

// renderFormBox renders a scrollable modal form box with a title, visible
// content lines, optional toast, and a pre-styled footer.
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
	out = append(out, boxLine(totalW, padVisible(footer, innerW)))
	out = append(out, boxBottom(totalW))
	return strings.Join(out, "\n")
}
