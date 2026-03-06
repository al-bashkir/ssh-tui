package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Content rendering
// ---------------------------------------------------------------------------

// renderFormContent builds all visible lines for the form content area.
// Returns the lines and the line index of the focused item (for scrolling).
func (m *formModel) renderFormContent(innerW int) ([]string, int) {
	lines := make([]string, 0, 40)
	focusLine := 0
	prevSectionIdx := -1
	fieldW := max(10, innerW-m.fieldIndent-m.labelWidth-1)

	for i, item := range m.items {
		focused := i == m.focusIdx
		sec := &m.schema.Sections[item.sectionIdx]

		// Render a section separator before the first field of each section.
		if item.sectionIdx != prevSectionIdx {
			lines = append(lines, formSection(sec.Label, innerW))
			prevSectionIdx = item.sectionIdx
		}

		fd := &sec.Fields[item.fieldIdx]
		if focused {
			focusLine = len(lines)
		}
		lines = append(lines, m.renderFieldLines(fd, focused, innerW, fieldW)...)
	}

	return lines, focusLine
}

// ---------------------------------------------------------------------------
// Field rendering
// ---------------------------------------------------------------------------

// renderFieldLines renders a single field as one or more lines.
// Multi-option pickers may wrap to additional continuation lines.
// The focused field gets a helper text line appended.
func (m *formModel) renderFieldLines(fd *fieldDef, focused bool, innerW, fieldW int) []string {
	var lines []string

	indent := strings.Repeat(" ", m.fieldIndent)
	prefix := indent
	if focused {
		// Replace indent with focus indicator.
		if m.fieldIndent >= 2 {
			prefix = "> "
			if m.fieldIndent > 2 {
				prefix = strings.Repeat(" ", m.fieldIndent-2) + "> "
			}
		} else {
			prefix = ">"
		}
	}

	labelStr := formLabel(fd.Label+":", m.labelWidth, focused)

	switch fd.Kind {
	case fieldText, fieldNumber:
		in := m.input(fd.Key)
		w := fieldW
		if fd.Narrow {
			w = min(12, fieldW)
		}
		if in != nil {
			lines = append(lines, prefix+labelStr+" "+underlineInput(*in, focused, w))
		} else {
			lines = append(lines, prefix+labelStr+" "+underlineText(m.values[fd.Key], focused, w))
		}

	case fieldPicker, fieldToggle:
		cur := strings.TrimSpace(m.values[fd.Key])
		segLines := renderPickerLines(fd.Options, cur, focused, fieldW)
		if len(segLines) > 0 {
			lines = append(lines, prefix+labelStr+" "+segLines[0])
			// Continuation lines aligned to the value column.
			contIndent := strings.Repeat(" ", m.fieldIndent+m.labelWidth+1)
			for _, sl := range segLines[1:] {
				lines = append(lines, contIndent+sl)
			}
		}

	case fieldSubModal:
		val := strings.TrimSpace(m.values[fd.Key])
		displayVal := val
		if displayVal == "" {
			displayVal = "default"
		}
		rendered := padVisible(displayVal, fieldW)
		if focused {
			rendered = checkedStyle.Render(rendered)
		} else {
			rendered = dim.Render(rendered)
		}
		lines = append(lines, prefix+labelStr+" "+rendered)
	}

	// Helper text below focused field.
	if focused && fd.Helper != "" {
		helperPrefix := strings.Repeat(" ", m.fieldIndent)
		lines = append(lines, dim.Render(helperPrefix+"╰ "+fd.Helper))
	}

	return lines
}

// ---------------------------------------------------------------------------
// Picker options rendering
// ---------------------------------------------------------------------------

// renderPickerLines renders picker/toggle options as segments, wrapping to
// multiple lines when they exceed the available width.
func renderPickerLines(options []fieldOption, cur string, focused bool, maxW int) []string {
	if len(options) == 0 {
		return nil
	}

	// Pre-render all segments and measure their visible widths.
	segments := make([]string, len(options))
	widths := make([]int, len(options))
	for i, opt := range options {
		segments[i] = formSegment(cur, opt.Value, opt.Display, focused)
		widths[i] = lipgloss.Width(segments[i])
	}

	const sepW = 2 // "  " between options
	sep := "  "

	// Group segments into lines that fit within maxW.
	var result []string
	var line string
	lineW := 0

	for i, seg := range segments {
		segWidth := widths[i]
		if i > 0 && lineW+sepW+segWidth > maxW {
			// Start a new line.
			result = append(result, line)
			line = seg
			lineW = segWidth
		} else if line == "" {
			line = seg
			lineW = segWidth
		} else {
			line += sep + seg
			lineW += sepW + segWidth
		}
	}
	if line != "" {
		result = append(result, line)
	}

	return result
}
