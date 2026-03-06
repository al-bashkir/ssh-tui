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
// Schema
// ---------------------------------------------------------------------------

func hostSchema(defs config.Defaults) formSchema {
	portPlaceholder := "22"
	if defs.Port != 0 {
		portPlaceholder = strconv.Itoa(defs.Port)
	}
	identPlaceholder := strings.TrimSpace(defs.IdentityFile)
	if identPlaceholder == "" {
		identPlaceholder = "~/.ssh/id_ed25519"
	}
	extraPlaceholder := strings.Join(defs.ExtraArgs, " ")
	if extraPlaceholder == "" {
		extraPlaceholder = "-o Option=value ..."
	}

	return formSchema{
		Sections: []sectionDef{
			{Key: "conn", Label: "Connection", Fields: []fieldDef{
				{Key: "host", Label: "Host", Kind: fieldText,
					Placeholder: "example.com or [10.0.0.1]:2222", CharLimit: 512},
				{Key: "user", Label: "User", Kind: fieldText,
					Placeholder: strings.TrimSpace(defs.User), CharLimit: 128},
				{Key: "port", Label: "Port", Kind: fieldNumber,
					Placeholder: portPlaceholder, CharLimit: 16, Narrow: true,
					Validate: validatePort},
			}},
			{Key: "auth", Label: "Authentication", Fields: []fieldDef{
				{Key: "identity", Label: "Identity file", Kind: fieldText,
					Placeholder: identPlaceholder, CharLimit: 512},
				{Key: "extra_args", Label: "Extra args", Kind: fieldText,
					Placeholder: extraPlaceholder, CharLimit: 1024},
			}},
		},
	}
}

// ---------------------------------------------------------------------------
// Value conversion
// ---------------------------------------------------------------------------

func hostToValues(h config.Host) map[string]string {
	portStr := ""
	if h.Port != 0 {
		portStr = strconv.Itoa(h.Port)
	}
	return map[string]string{
		"host":       strings.TrimSpace(h.Host),
		"user":       strings.TrimSpace(h.User),
		"port":       portStr,
		"identity":   strings.TrimSpace(h.IdentityFile),
		"extra_args": strings.Join(h.ExtraArgs, " "),
	}
}

func applyHostValues(values map[string]string, base config.Host) (config.Host, error) {
	h := base
	h.Host = strings.TrimSpace(values["host"])
	h.User = strings.TrimSpace(values["user"])
	h.IdentityFile = strings.TrimSpace(values["identity"])

	portStr := strings.TrimSpace(values["port"])
	if portStr == "" {
		h.Port = 0
	} else {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 {
			return h, fmt.Errorf("invalid port")
		}
		h.Port = p
	}

	extra := strings.TrimSpace(values["extra_args"])
	if extra == "" {
		h.ExtraArgs = nil
	} else {
		h.ExtraArgs = strings.Fields(extra)
	}

	if strings.TrimSpace(h.Host) == "" {
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
	defs        config.Defaults
	parentCrumb string

	width  int
	height int

	toast toast

	keymap keyMap

	confirmQuitEnabled bool
	confirmQuit        bool
}

func (m *hostFormModel) refreshAccentStyles() {
	m.form.refreshAccentStyles()
}

func newHostFormModel(index int, h config.Host, defs config.Defaults, confirmQuitEnabled bool) *hostFormModel {
	values := hostToValues(h)
	fm := newFormModel(hostSchema(defs), values, modalFormLabelWidth)

	return &hostFormModel{
		form:               fm,
		index:              index,
		host:               h,
		defs:               defs,
		keymap:             defaultKeyMap(),
		confirmQuitEnabled: confirmQuitEnabled,
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
		if m.confirmQuit {
			switch msg.String() {
			case "y", "Y", "enter":
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmQuit = false
				m.toast = toast{}
				return m, nil
			default:
				return m, nil
			}
		}

		// Insert mode — delegate to form first.
		if m.form.editing {
			handled, cmd := m.form.handleKey(msg)
			if handled {
				return m, cmd
			}
		}

		// Save.
		if key.Matches(msg, m.keymap.Settings) {
			m.form.exitEdit()
			m.toast = toast{}
			h, err := applyHostValues(m.form.values, m.host)
			if err != nil {
				m.toast = toast{text: err.Error(), level: toastErr}
				return m, nil
			}
			return m, func() tea.Msg { return hostFormSaveMsg{index: m.index, host: h} }
		}

		if key.Matches(msg, m.keymap.Quit) {
			if !m.confirmQuitEnabled {
				return m, tea.Quit
			}
			m.confirmQuit = true
			m.toast = toast{text: "quit? (y/n)", level: toastWarn}
			return m, nil
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
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	innerW := max(0, m.width-2)

	// Build content lines from the form.
	lines, focusLine := m.form.renderFormContent(innerW)

	// Footer.
	footer := m.form.renderFooter()

	// Scroll.
	innerH := max(0, m.height-2)
	reserved := 2 // sep + footer
	if !m.toast.empty() {
		reserved++
	}
	visibleH := innerH - reserved
	if visibleH < 1 {
		visibleH = 1
	}
	start, end := formScrollWindow(len(lines), visibleH, focusLine)
	visible := lines[start:end]

	// Title / breadcrumb.
	title := "Create Host"
	if m.index >= 0 {
		name := strings.TrimSpace(m.host.Host)
		if name == "" {
			name = strings.TrimSpace(m.form.value("host"))
		}
		if name != "" {
			title = breadcrumbTitle(m.parentCrumb, name)
		} else {
			title = breadcrumbTitle(m.parentCrumb, "Edit Host")
		}
	} else {
		title = breadcrumbTitle(m.parentCrumb, "Create Host")
	}

	toastStr := ""
	if !m.toast.empty() {
		toastStr = renderToast(m.toast)
	}
	return renderFormBox(m.width, title, visible, visibleH, toastStr, footer)
}
