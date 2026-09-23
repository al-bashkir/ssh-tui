package ui

import (
	"fmt"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

func hostSchema(defs config.Defaults) formSchema {
	portPlaceholder, identPlaceholder, extraPlaceholder := sshPlaceholders(defs)

	return formSchema{
		{Label: "Connection", Fields: []fieldDef{
			{Key: "host", Label: "Host", Kind: fieldText,
				Placeholder: "example.com or [10.0.0.1]:2222", CharLimit: 512},
			{Key: "user", Label: "User", Kind: fieldText,
				Placeholder: strings.TrimSpace(defs.User), CharLimit: 128},
			{Key: "port", Label: "Port", Kind: fieldNumber,
				Placeholder: portPlaceholder, CharLimit: 16, Narrow: true,
				Validate: validatePort},
		}},
		{Label: "Authentication", Fields: []fieldDef{
			{Key: "identity", Label: "Identity file", Kind: fieldText,
				Placeholder: identPlaceholder, CharLimit: 512},
			{Key: "extra_args", Label: "Extra args", Kind: fieldText,
				Placeholder: extraPlaceholder, CharLimit: 1024},
		}},
	}
}

// ---------------------------------------------------------------------------
// Value conversion
// ---------------------------------------------------------------------------

func hostToValues(h config.Host) map[string]string {
	return map[string]string{
		"host":       strings.TrimSpace(h.Host),
		"user":       strings.TrimSpace(h.User),
		"port":       formatPort(h.Port),
		"identity":   strings.TrimSpace(h.IdentityFile),
		"extra_args": strings.Join(h.ExtraArgs, " "),
	}
}

func applyHostValues(values map[string]string, base config.Host) (config.Host, error) {
	h := base
	h.Host = strings.TrimSpace(values["host"])
	h.User = strings.TrimSpace(values["user"])
	h.IdentityFile = strings.TrimSpace(values["identity"])
	h.ExtraArgs = parseArgs(values["extra_args"])

	var err error
	if h.Port, err = parsePort(values["port"]); err != nil {
		return h, err
	}
	if h.Host == "" {
		return h, fmt.Errorf("host required")
	}
	return h, nil
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type hostFormModel struct {
	form        formModel
	index       int
	host        config.Host
	parentCrumb string

	width  int
	height int

	toast toast

	keymap keyMap
}

func (m *hostFormModel) refreshAccentStyles() {
	m.form.applyFocusStyles()
}

func newHostFormModel(index int, h config.Host, defs config.Defaults) *hostFormModel {
	values := hostToValues(h)
	fm := newFormModel(hostSchema(defs), values, modalFormLabelWidth, defs.ShowFieldHelp)

	return &hostFormModel{
		form:   fm,
		index:  index,
		host:   h,
		keymap: defaultKeyMap(),
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m *hostFormModel) Init() tea.Cmd { return nil }

func (m *hostFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.form.handleResize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// Insert mode — delegate to form first.
		if m.form.editing {
			handled, cmd := m.form.handleKey(msg)
			if handled {
				return m, cmd
			}
		}

		// Save.
		if key.Matches(msg, m.keymap.Save) {
			m.form.exitEdit()
			m.toast = toast{}
			h, err := applyHostValues(m.form.values, m.host)
			if err != nil {
				m.toast = toast{text: err.Error(), level: toastErr}
				return m, nil
			}
			return m, func() tea.Msg { return hostFormSaveMsg{index: m.index, host: h} }
		}

		if key.Matches(msg, m.keymap.Esc) {
			return m, func() tea.Msg { return hostFormCancelMsg{} }
		}

		// Generic form key handling.
		handled, cmd := m.form.handleKey(msg)
		if handled {
			return m, cmd
		}
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m *hostFormModel) View() string {
	name := m.host.Host
	if strings.TrimSpace(name) == "" {
		name = m.form.values["host"]
	}
	title := formTitle(m.parentCrumb, m.index, name, "Create Host", "Edit Host")
	return renderModalForm(&m.form, m.width, m.height, title, m.toast)
}
