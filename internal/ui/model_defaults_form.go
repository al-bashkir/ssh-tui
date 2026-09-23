package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"

	"github.com/charmbracelet/bubbles/key"

	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type defaultsFormCancelMsg struct{}

type defaultsFormSaveMsg struct {
	defaults config.Defaults
}

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// settingsSchema returns the declarative form definition for the Settings
// screen, organized into five sections following Variant A layout.
func settingsSchema() formSchema {
	return formSchema{
		{Label: "SSH", Fields: []fieldDef{
			{Key: "user", Label: "User", Kind: fieldText,
				Placeholder: "login username",
				Helper:      "Default SSH username for all connections",
				CharLimit:   128},
			{Key: "port", Label: "Port", Kind: fieldNumber,
				Placeholder: "22",
				Helper:      "Default SSH port (1\u201365535)",
				CharLimit:   6, Narrow: true,
				Validate: validatePort},
			{Key: "identity", Label: "Identity file", Kind: fieldText,
				Placeholder: "~/.ssh/id_ed25519",
				Helper:      "Path to SSH private key for authentication",
				CharLimit:   512},
			{Key: "extra_args", Label: "Extra arguments", Kind: fieldText,
				Placeholder: "-o Option=value ...",
				Helper:      "Additional ssh flags, space-separated",
				CharLimit:   1024},
			{Key: "load_known_hosts", Label: "Load known_hosts", Kind: fieldPicker,
				Options: opts("yes", "no"),
				Helper:  "Auto-import hosts from ~/.ssh/known_hosts"},
		}},
		{Label: "Appearance", Fields: []fieldDef{
			{Key: "colorscheme", Label: "Colorscheme", Kind: fieldPicker,
				Options: opts("default", "dracula", "nord", "gruvbox", "catppuccin", "kanagawa"),
				Helper:  "Application color theme"},
			{Key: "accent", Label: "Accent color", Kind: fieldPicker,
				Options: opts("default", "blue", "cyan", "green", "amber", "red", "magenta"),
				Helper:  "UI accent color (overridden when a colorscheme is active)",
				DisabledWhen: func(values map[string]string) string {
					if cs := values["colorscheme"]; cs != "" && cs != "default" {
						return "overridden by " + cs + " theme"
					}
					return ""
				}},
			{Key: "show_field_help", Label: "Field help", Kind: fieldPicker,
				Options: opts("yes", "no"),
				Helper:  "Show helper text below focused form fields"},
		}},
		{Label: "Tmux", Fields: []fieldDef{
			{Key: "tmux", Label: "Tmux mode", Kind: fieldPicker,
				Options: opts("auto", "force", "never"),
				Helper:  "auto: use tmux when available, force: require, never: disable"},
			{Key: "open_mode", Label: "Open mode", Kind: fieldPicker,
				Options: opts("auto", "current", "tmux-window", "tmux-pane"),
				Helper:  "How to open new SSH sessions"},
			{Key: "session", Label: "Session name", Kind: fieldText,
				Placeholder: "ssh-tui",
				Helper:      "Name of the tmux session to use",
				CharLimit:   128},
		}},
		{Label: "Behavior", Fields: []fieldDef{
			{Key: "confirm_quit", Label: "Confirm quit", Kind: fieldPicker,
				Options: opts("yes", "no"),
				Helper:  "Show confirmation dialog before quitting"},
			{Key: "threshold", Label: "Confirm threshold", Kind: fieldNumber,
				Placeholder: "5",
				Helper:      "Ask before connecting to more than N hosts (0 = never)",
				CharLimit:   6, Narrow: true,
				Validate: validateNonNegativeInt},
		}},
		{Label: "Panes", Fields: []fieldDef{
			{Key: "pane_split", Label: "Split direction", Kind: fieldPicker,
				Options: opts("horizontal", "vertical"),
				Helper:  "Direction for splitting tmux panes"},
			{Key: "pane_layout", Label: "Layout", Kind: fieldPicker,
				Options: opts("auto", "tiled", "even-horizontal", "even-vertical", "main-horizontal", "main-vertical"),
				Helper:  "Tmux pane arrangement algorithm"},
			{Key: "pane_sync", Label: "Synchronize input", Kind: fieldPicker,
				Options: opts("on", "off"),
				Helper:  "Send keystrokes to all panes simultaneously"},
			{Key: "border_pos", Label: "Border position", Kind: fieldPicker,
				Options: opts("bottom", "top", "off"),
				Helper:  "Position of pane border status line"},
			{Key: "border_fmt", Label: "Border format", Kind: fieldSubModal,
				Helper: "Tmux format string for pane borders"},
		}},
	}
}

// ---------------------------------------------------------------------------
// Value conversion: config.Defaults <-> map[string]string
// ---------------------------------------------------------------------------

// defaultsToValues maps a config.Defaults to the flat string map used by
// the form model.
func defaultsToValues(d config.Defaults) map[string]string {
	yesNo := func(b bool) string {
		if b {
			return "yes"
		}
		return "no"
	}
	threshStr := ""
	if d.ConnectConfirmThreshold >= 0 {
		threshStr = strconv.Itoa(d.ConnectConfirmThreshold)
	}
	return map[string]string{
		"user":             strings.TrimSpace(d.User),
		"port":             formatPort(d.Port),
		"identity":         strings.TrimSpace(d.IdentityFile),
		"extra_args":       strings.Join(d.ExtraArgs, " "),
		"colorscheme":      d.Colorscheme,
		"accent":           d.AccentColor,
		"load_known_hosts": yesNo(d.LoadKnownHosts),
		"tmux":             d.Tmux,
		"open_mode":        d.OpenMode,
		"session":          strings.TrimSpace(d.TmuxSession),
		"confirm_quit":     yesNo(d.ConfirmQuit),
		"show_field_help":  yesNo(d.ShowFieldHelp),
		"threshold":        threshStr,
		"pane_split":       d.PaneSplit,
		"pane_layout":      d.PaneLayout,
		"pane_sync":        d.PaneSync,
		"border_pos":       d.PaneBorderPos,
		"border_fmt":       d.PaneBorderFmt,
	}
}

// applySettingsValues converts form values back to a config.Defaults.
func applySettingsValues(values map[string]string, base config.Defaults) (config.Defaults, error) {
	d := base

	d.User = strings.TrimSpace(values["user"])
	d.IdentityFile = strings.TrimSpace(values["identity"])

	// Colorscheme / accent.
	d.Colorscheme = strings.ToLower(strings.TrimSpace(values["colorscheme"]))
	if d.Colorscheme == "default" {
		d.Colorscheme = ""
	}
	d.AccentColor = strings.ToLower(strings.TrimSpace(values["accent"]))
	if d.AccentColor == "default" {
		d.AccentColor = ""
	}

	d.LoadKnownHosts = values["load_known_hosts"] != "no"
	d.ConfirmQuit = values["confirm_quit"] == "yes"
	d.ShowFieldHelp = values["show_field_help"] != "no"

	var err error
	if d.Port, err = parsePort(values["port"]); err != nil {
		return d, err
	}
	if d.Port == 0 {
		d.Port = 22
	}
	d.ExtraArgs = parseArgs(values["extra_args"])

	// Tmux fields.
	d.Tmux = strings.TrimSpace(values["tmux"])
	d.OpenMode = strings.TrimSpace(values["open_mode"])
	d.TmuxSession = strings.TrimSpace(values["session"])
	if d.TmuxSession == "" {
		d.TmuxSession = "ssh-tui"
	}

	if d.ConnectConfirmThreshold, err = parseNonNegative(values["threshold"], 5); err != nil {
		return d, fmt.Errorf("confirm threshold %w", err)
	}

	// Pane fields.
	d.PaneSplit = strings.TrimSpace(values["pane_split"])
	d.PaneLayout = strings.TrimSpace(values["pane_layout"])
	d.PaneSync = strings.TrimSpace(values["pane_sync"])
	d.PaneBorderPos = strings.TrimSpace(values["border_pos"])

	// Border format (may have been updated via sub-modal).
	d.PaneBorderFmt = strings.TrimSpace(values["border_fmt"])
	if d.PaneBorderFmt == "" {
		d.PaneBorderFmt = config.DefaultPaneBorderFormat
	}

	// Sanitize custom formats: the choices minus the built-in default.
	d.PaneBorderFmts = paneBorderFormatChoices(config.Defaults{PaneBorderFmts: d.PaneBorderFmts})[1:]

	return d, nil
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type defaultsFormModel struct {
	form     formModel
	defaults config.Defaults // preserved for border-format sub-modal

	width  int
	height int

	borderPicker *paneBorderFormatsModel

	toast toast

	keymap keyMap

	confirmQuit    bool
	confirmDiscard bool // discard unsaved changes? The toast is its only prompt.
}

func (m *defaultsFormModel) refreshAccentStyles() {
	m.form.applyFocusStyles()
	if m.borderPicker != nil {
		m.borderPicker.refreshAccentStyles()
	}
}

func newDefaultsFormModel(d config.Defaults) *defaultsFormModel {
	return &defaultsFormModel{
		form:     newFormModel(settingsSchema(), defaultsToValues(d), 16, d.ShowFieldHelp),
		defaults: d,
		keymap:   defaultKeyMap(),
	}
}

func (m *defaultsFormModel) Init() tea.Cmd { return nil }

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m *defaultsFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// Sub-modal messages.
	case paneBorderFormatsCancelMsg:
		m.borderPicker = nil
		return m, nil
	case paneBorderFormatsDoneMsg:
		m.defaults.PaneBorderFmt = strings.TrimSpace(msg.value)
		m.defaults.PaneBorderFmts = append([]string(nil), msg.customFmts...)
		m.form.values["border_fmt"] = m.defaults.PaneBorderFmt
		m.borderPicker = nil
		return m, nil

	// Resize.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.form.handleResize(msg.Width, msg.Height)
		if m.borderPicker != nil {
			mw, mh := pickerModal.size(msg.Width, msg.Height)
			_, _ = m.borderPicker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		return m, nil

	// Key handling.
	case tea.KeyMsg:
		// Sub-modal intercepts all keys.
		if m.borderPicker != nil {
			_, cmd := m.borderPicker.Update(msg)
			return m, cmd
		}

		// Confirm-quit dialog.
		if m.confirmQuit {
			switch confirmAnswer(msg) {
			case answerYes:
				return m, tea.Quit
			case answerNo:
				m.confirmQuit = false
				m.toast = toast{}
			}
			return m, nil
		}

		// Confirm-discard dialog. Enter does not discard.
		if m.confirmDiscard {
			switch a := confirmAnswer(msg); {
			case a == answerYes && msg.String() != "enter":
				return m, func() tea.Msg { return defaultsFormCancelMsg{} }
			case a == answerNo:
				m.confirmDiscard = false
				m.toast = toast{}
			}
			return m, nil
		}

		// In insert mode or picker popup, delegate everything to the form.
		if m.form.editing || m.form.picker != nil {
			handled, cmd := m.form.handleKey(msg)
			if handled {
				// Live-preview: sync after picker/edit confirms.
				m.form.showFieldHelp = m.form.values["show_field_help"] != "no"
				return m, cmd
			}
		}

		if key.Matches(msg, m.keymap.HostsTab) || key.Matches(msg, m.keymap.GroupsTab) {
			if m.form.isDirty() {
				m.toast = toast{text: "unsaved changes; save or Esc", level: toastWarn}
				return m, nil
			}
			to := screenHosts
			if key.Matches(msg, m.keymap.GroupsTab) {
				to = screenGroups
			}
			return m, func() tea.Msg { return switchScreenMsg{to: to} }
		}

		// Save (Ctrl+S).
		if key.Matches(msg, m.keymap.Save) {
			m.form.exitEdit()
			m.toast = toast{}
			d, err := applySettingsValues(m.form.values, m.defaults)
			if err != nil {
				m.toast = toast{text: err.Error(), level: toastErr}
				return m, nil
			}
			return m, func() tea.Msg { return defaultsFormSaveMsg{defaults: d} }
		}

		// Quit.
		if key.Matches(msg, m.keymap.Quit) {
			if !m.defaults.ConfirmQuit {
				return m, tea.Quit
			}
			m.confirmQuit = true
			m.toast = toast{text: "quit? (y/n)", level: toastWarn}
			return m, nil
		}

		// Cancel (Esc) — with unsaved-changes guard.
		if key.Matches(msg, m.keymap.Esc) {
			if m.form.isDirty() {
				m.confirmDiscard = true
				m.toast = toast{text: "unsaved changes \u2014 discard? (y/n)", level: toastWarn}
				return m, nil
			}
			return m, func() tea.Msg { return defaultsFormCancelMsg{} }
		}

		// Sub-modal activation for border format field.
		fk := m.form.focusedFieldKey()
		s := msg.String()
		if fk == "border_fmt" && (s == "enter" || s == " " || s == "l") {
			mw, mh := pickerModal.size(m.width, m.height)
			m.borderPicker = newPaneBorderFormatsModel(m.defaults, m.defaults.PaneBorderFmt, false, true)
			m.borderPicker.parentCrumb = "Settings"
			if mw > 0 && mh > 0 {
				_, _ = m.borderPicker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
			}
			return m, nil
		}

		// Generic form key handling.
		handled, cmd := m.form.handleKey(msg)
		if handled {
			// Live-preview: sync showFieldHelp toggle.
			m.form.showFieldHelp = m.form.values["show_field_help"] != "no"
			return m, cmd
		}
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m *defaultsFormModel) View() string {
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.borderPicker != nil {
		return placeCentered(m.width, m.height, m.borderPicker.View())
	}

	// Content rows below the tab header, less the footer and optional toast.
	bottom := []string{m.form.renderFooter()}
	if !m.toast.empty() {
		bottom = []string{renderToast(m.toast), bottom[0]}
	}
	visibleH := max(1, max(0, m.height-2)-tabBoxHeaderLines-len(bottom))

	lines := m.form.visibleLines(max(0, m.width-2), visibleH)
	// Pad so the toast and footer stay pinned to the bottom.
	for len(lines) < visibleH {
		lines = append(lines, "")
	}
	lines = append(lines, bottom...)

	headRight := statusDot(false)
	if m.form.isDirty() {
		headRight = statusWarn.Render("[mod]") + " " + headRight
	}
	return renderMainTabBox(m.width, m.height, 2, accentStyle.Render("Settings"), headRight, strings.Join(lines, "\n"), "")
}
