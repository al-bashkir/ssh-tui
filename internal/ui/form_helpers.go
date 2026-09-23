package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"
)

// sshPlaceholders returns the placeholders for the shared SSH fields, showing
// the inherited defaults where they are set.
func sshPlaceholders(defs config.Defaults) (port, identity, extraArgs string) {
	port = "22"
	if defs.Port != 0 {
		port = strconv.Itoa(defs.Port)
	}
	identity = strings.TrimSpace(defs.IdentityFile)
	if identity == "" {
		identity = "~/.ssh/id_ed25519"
	}
	extraArgs = strings.Join(defs.ExtraArgs, " ")
	if extraArgs == "" {
		extraArgs = "-o Option=value ..."
	}
	return port, identity, extraArgs
}

// parsePort parses an optional port field; an empty field gives 0.
func parsePort(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	p, err := strconv.Atoi(s)
	if err != nil || p <= 0 || p > 65535 {
		return 0, fmt.Errorf("port must be a number 1-65535")
	}
	return p, nil
}

// parseNonNegative parses an optional non-negative integer field; an empty
// field gives def.
func parseNonNegative(s string, def int) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return def, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("must be a number >= 0")
	}
	return n, nil
}

// formatPort is the form value for port; 0 (unset) gives "".
func formatPort(p int) string {
	if p == 0 {
		return ""
	}
	return strconv.Itoa(p)
}

// parseArgs splits a space-separated args field; an empty field gives nil.
func parseArgs(s string) []string {
	if f := strings.Fields(s); len(f) > 0 {
		return f
	}
	return nil
}

// formTitle builds the modal title: the edited item's name under its parent
// crumb, or a create/edit label when there is no name yet.
func formTitle(parentCrumb string, index int, name, createTitle, editTitle string) string {
	if index < 0 {
		return breadcrumbTitle(parentCrumb, createTitle)
	}
	if n := strings.TrimSpace(name); n != "" {
		return breadcrumbTitle(parentCrumb, n)
	}
	return breadcrumbTitle(parentCrumb, editTitle)
}

// renderModalForm renders a form into its modal box: scrolled content with the
// focused field visible, any open picker overlaid, then toast and footer.
func renderModalForm(f *formModel, width, height int, title string, t toast) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	innerW := max(0, width-2)
	reserved := 2 // sep + footer
	toastStr := ""
	if !t.empty() {
		toastStr = renderToast(t)
		reserved++
	}
	visibleH := max(1, max(0, height-2)-reserved)
	return renderFormBox(width, title, f.visibleLines(innerW, visibleH), visibleH, toastStr, f.renderFooter())
}

// visibleLines renders the form content scrolled so the focused field shows
// within visibleH lines, with any open picker overlaid.
func (f *formModel) visibleLines(innerW, visibleH int) []string {
	lines, focusLine := f.renderFormContent(innerW)
	start, end := formScrollWindow(len(lines), visibleH, focusLine)
	visible := lines[start:end]
	if f.picker != nil {
		visible = f.overlayPickerOnVisible(visible, focusLine-start, innerW)
	}
	return visible
}

// formLabel renders a padded label for a form field.
// When focused, the label is rendered in accent style.
func formLabel(s string, labelW int, focused bool) string {
	if len(s) < labelW {
		s += strings.Repeat(" ", labelW-len(s))
	}
	if focused {
		return accentStyle.Render(s)
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

// validatePort is the field validator for parsePort.
func validatePort(s string) error {
	_, err := parsePort(s)
	return err
}

// validateNonNegativeInt is the field validator for parseNonNegative.
func validateNonNegativeInt(s string) error {
	_, err := parseNonNegative(s, 0)
	return err
}
