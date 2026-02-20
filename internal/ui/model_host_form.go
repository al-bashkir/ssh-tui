package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type hostField int

const (
	hostFieldHost hostField = iota
	hostFieldUser
	hostFieldPort
	hostFieldIdentity
	hostFieldExtraArgs
)

type hostFormModel struct {
	index       int
	host        config.Host
	defs        config.Defaults
	parentCrumb string

	width  int
	height int

	focus   hostField
	editing bool // true when editing a text field (insert mode)

	inHost     textinput.Model
	inUser     textinput.Model
	inPort     textinput.Model
	inIdentity textinput.Model
	inExtra    textinput.Model

	toast toast

	keymap keyMap

	confirmQuitEnabled bool
	confirmQuit        bool
}

func (m *hostFormModel) refreshAccentStyles() {
	setSearchFocused(&m.inHost, m.focus == hostFieldHost)
	setSearchFocused(&m.inUser, m.focus == hostFieldUser)
	setSearchFocused(&m.inPort, m.focus == hostFieldPort)
	setSearchFocused(&m.inIdentity, m.focus == hostFieldIdentity)
	setSearchFocused(&m.inExtra, m.focus == hostFieldExtraArgs)
}

func newHostFormModel(index int, h config.Host, defs config.Defaults, confirmQuitEnabled bool) *hostFormModel {
	inHost := textinput.New()
	inHost.CharLimit = 512
	inHost.Prompt = ""
	inHost.SetValue(strings.TrimSpace(h.Host))
	inHost.Placeholder = "example.com or [10.0.0.1]:2222"
	configureSearch(&inHost)

	inUser := textinput.New()
	inUser.CharLimit = 128
	inUser.Prompt = ""
	inUser.SetValue(strings.TrimSpace(h.User))
	inUser.Placeholder = strings.TrimSpace(defs.User)
	configureSearch(&inUser)

	inPort := textinput.New()
	inPort.CharLimit = 16
	inPort.Prompt = ""
	if h.Port != 0 {
		inPort.SetValue(strconv.Itoa(h.Port))
	}
	if defs.Port != 0 {
		inPort.Placeholder = strconv.Itoa(defs.Port)
	} else {
		inPort.Placeholder = "22"
	}
	configureSearch(&inPort)

	inIdentity := textinput.New()
	inIdentity.CharLimit = 512
	inIdentity.Prompt = ""
	inIdentity.SetValue(strings.TrimSpace(h.IdentityFile))
	inIdentity.Placeholder = strings.TrimSpace(defs.IdentityFile)
	if inIdentity.Placeholder == "" {
		inIdentity.Placeholder = "~/.ssh/id_ed25519"
	}
	configureSearch(&inIdentity)

	inExtra := textinput.New()
	inExtra.CharLimit = 1024
	inExtra.Prompt = ""
	inExtra.SetValue(strings.Join(h.ExtraArgs, " "))
	inExtra.Placeholder = strings.Join(defs.ExtraArgs, " ")
	if inExtra.Placeholder == "" {
		inExtra.Placeholder = "-o Option=value ..."
	}
	configureSearch(&inExtra)

	m := &hostFormModel{
		index:              index,
		host:               h,
		defs:               defs,
		focus:              hostFieldHost,
		inHost:             inHost,
		inUser:             inUser,
		inPort:             inPort,
		inIdentity:         inIdentity,
		inExtra:            inExtra,
		keymap:             defaultKeyMap(),
		confirmQuitEnabled: confirmQuitEnabled,
	}

	// Start in normal mode (no text input focused).
	setSearchFocused(&m.inHost, true)
	setSearchFocused(&m.inUser, false)
	setSearchFocused(&m.inPort, false)
	setSearchFocused(&m.inIdentity, false)
	setSearchFocused(&m.inExtra, false)
	return m
}

func (m *hostFormModel) Init() tea.Cmd { return nil }

func (m *hostFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		innerW := max(0, msg.Width-2)
		labelW := 14
		fieldW := max(10, innerW-labelW-1)
		m.inHost.Width = fieldW
		m.inUser.Width = fieldW
		m.inPort.Width = min(12, fieldW)
		m.inIdentity.Width = fieldW
		m.inExtra.Width = fieldW
		return m, nil
	case tea.KeyMsg:
		if m.confirmQuit {
			s := msg.String()
			switch s {
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

		// Save.
		if key.Matches(msg, m.keymap.Settings) {
			if m.editing {
				m.exitEdit()
			}
			m.toast = toast{}
			if err := m.apply(); err != nil {
				m.toast = toast{text: err.Error(), level: toastErr}
				return m, nil
			}
			return m, func() tea.Msg { return hostFormSaveMsg{index: m.index, host: m.host} }
		}

		if key.Matches(msg, m.keymap.Quit) {
			if !m.confirmQuitEnabled {
				return m, tea.Quit
			}
			m.confirmQuit = true
			m.toast = toast{text: "quit? (y/n)", level: toastWarn}
			return m, nil
		}

		// Insert mode: route keys to the text input.
		if m.editing {
			s := msg.String()
			switch s {
			case "esc":
				m.exitEdit()
				return m, nil
			case "enter":
				m.exitEdit()
				return m, m.moveFocus(1)
			default:
				return m, m.updateFocusedInput(msg)
			}
		}

		// Normal mode: vim-like navigation.
		if key.Matches(msg, m.keymap.Esc) {
			return m, func() tea.Msg { return hostFormCancelMsg{} }
		}

		s := msg.String()
		switch s {
		case "j", "down", "tab", "enter":
			return m, m.moveFocus(1)
		case "k", "up", "shift+tab":
			return m, m.moveFocus(-1)
		case "i":
			m.enterEdit()
			return m, nil
		}
	}

	return m, nil
}

func (m *hostFormModel) updateFocusedInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.focus {
	case hostFieldHost:
		m.inHost, cmd = m.inHost.Update(msg)
	case hostFieldUser:
		m.inUser, cmd = m.inUser.Update(msg)
	case hostFieldPort:
		m.inPort, cmd = m.inPort.Update(msg)
	case hostFieldIdentity:
		m.inIdentity, cmd = m.inIdentity.Update(msg)
	case hostFieldExtraArgs:
		m.inExtra, cmd = m.inExtra.Update(msg)
	default:
		// no-op
	}
	return cmd
}

func (m *hostFormModel) moveFocus(delta int) tea.Cmd {
	order := []hostField{
		hostFieldHost,
		hostFieldUser,
		hostFieldPort,
		hostFieldIdentity,
		hostFieldExtraArgs,
	}
	pos := 0
	for i := range order {
		if order[i] == m.focus {
			pos = i
			break
		}
	}
	pos += delta
	if pos < 0 {
		pos = len(order) - 1
	}
	if pos >= len(order) {
		pos = 0
	}
	m.setFocus(order[pos])
	return nil
}

func (m *hostFormModel) setFocus(f hostField) {
	// Validate port when leaving the field.
	if m.focus == hostFieldPort && f != hostFieldPort {
		v := strings.TrimSpace(m.inPort.Value())
		if v != "" {
			if _, err := strconv.Atoi(v); err != nil {
				m.toast = toast{text: "port must be a number", level: toastWarn}
			} else {
				m.toast = toast{}
			}
		}
	}
	m.focus = f
	m.editing = false

	// Blur all text inputs.
	m.inHost.Blur()
	m.inUser.Blur()
	m.inPort.Blur()
	m.inIdentity.Blur()
	m.inExtra.Blur()
	setSearchFocused(&m.inHost, false)
	setSearchFocused(&m.inUser, false)
	setSearchFocused(&m.inPort, false)
	setSearchFocused(&m.inIdentity, false)
	setSearchFocused(&m.inExtra, false)

	// Highlight the focused field label.
	switch f {
	case hostFieldHost:
		setSearchFocused(&m.inHost, true)
	case hostFieldUser:
		setSearchFocused(&m.inUser, true)
	case hostFieldPort:
		setSearchFocused(&m.inPort, true)
	case hostFieldIdentity:
		setSearchFocused(&m.inIdentity, true)
	case hostFieldExtraArgs:
		setSearchFocused(&m.inExtra, true)
	}
}

func (m *hostFormModel) enterEdit() {
	m.editing = true
	switch m.focus {
	case hostFieldHost:
		_ = m.inHost.Focus()
	case hostFieldUser:
		_ = m.inUser.Focus()
	case hostFieldPort:
		_ = m.inPort.Focus()
	case hostFieldIdentity:
		_ = m.inIdentity.Focus()
	case hostFieldExtraArgs:
		_ = m.inExtra.Focus()
	}
}

func (m *hostFormModel) exitEdit() {
	m.editing = false
	m.inHost.Blur()
	m.inUser.Blur()
	m.inPort.Blur()
	m.inIdentity.Blur()
	m.inExtra.Blur()
}

func (m *hostFormModel) apply() error {
	m.host.Host = strings.TrimSpace(m.inHost.Value())
	m.host.User = strings.TrimSpace(m.inUser.Value())
	m.host.IdentityFile = strings.TrimSpace(m.inIdentity.Value())

	portStr := strings.TrimSpace(m.inPort.Value())
	if portStr == "" {
		m.host.Port = 0
	} else {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 {
			return fmt.Errorf("invalid port")
		}
		m.host.Port = p
	}

	extra := strings.TrimSpace(m.inExtra.Value())
	if extra == "" {
		m.host.ExtraArgs = nil
	} else {
		m.host.ExtraArgs = strings.Fields(extra)
	}

	if strings.TrimSpace(m.host.Host) == "" {
		return fmt.Errorf("host required")
	}
	return nil
}

func (m *hostFormModel) View() string {
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	innerW := max(0, m.width-2)
	labelW := 14
	fieldW := max(10, innerW-labelW-1)

	label := func(s string, focused bool) string {
		padded := s
		if len(padded) < labelW {
			padded = padded + strings.Repeat(" ", labelW-len(padded))
		}
		if focused {
			return headerStyle.Render(padded)
		}
		return padded
	}

	inputLine := func(in textinput.Model, focused bool, w int) string {
		return underlineInput(in, focused, w)
	}

	lines := []string{}
	focusLine := 0

	lines = append(lines, formSection("Connection", innerW))
	if m.focus == hostFieldHost {
		focusLine = len(lines)
	}
	lines = append(lines, label("Host:", m.focus == hostFieldHost)+" "+inputLine(m.inHost, m.focus == hostFieldHost, fieldW))
	if m.focus == hostFieldUser {
		focusLine = len(lines)
	}
	lines = append(lines, label("User:", m.focus == hostFieldUser)+" "+inputLine(m.inUser, m.focus == hostFieldUser, fieldW))
	if m.focus == hostFieldPort {
		focusLine = len(lines)
	}
	lines = append(lines, label("Port:", m.focus == hostFieldPort)+" "+inputLine(m.inPort, m.focus == hostFieldPort, min(12, fieldW)))
	lines = append(lines, formSection("Authentication", innerW))
	if m.focus == hostFieldIdentity {
		focusLine = len(lines)
	}
	lines = append(lines, label("Identity file:", m.focus == hostFieldIdentity)+" "+inputLine(m.inIdentity, m.focus == hostFieldIdentity, fieldW))
	if m.focus == hostFieldExtraArgs {
		focusLine = len(lines)
	}
	lines = append(lines, label("Extra args:", m.focus == hostFieldExtraArgs)+" "+inputLine(m.inExtra, m.focus == hostFieldExtraArgs, fieldW))

	fieldPos := fmt.Sprintf("%d/%d", int(m.focus)+1, int(hostFieldExtraArgs)+1)
	footer := fieldPos + "  Ctrl+S save   j/k move   i edit   Esc cancel"
	if m.editing {
		footer = footerStyle.Render(fieldPos) + "  " + headerStyle.Render("INSERT") + "  " + footerStyle.Render("Ctrl+S save   Esc done")
	}

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

	out := make([]string, 0, m.height)
	title := "Create Host"
	if m.index >= 0 {
		name := strings.TrimSpace(m.host.Host)
		if name == "" {
			name = strings.TrimSpace(m.inHost.Value())
		}
		if name != "" {
			title = breadcrumbTitle(m.parentCrumb, name)
		} else {
			title = breadcrumbTitle(m.parentCrumb, "Edit Host")
		}
	} else {
		title = breadcrumbTitle(m.parentCrumb, "Create Host")
	}
	out = append(out, boxTitleTop(m.width, title))
	for _, ln := range visible {
		out = append(out, boxLine(m.width, padVisible(ln, innerW)))
	}
	fill := visibleH - len(visible)
	for i := 0; i < fill; i++ {
		out = append(out, boxLine(m.width, strings.Repeat(" ", innerW)))
	}
	if !m.toast.empty() {
		out = append(out, boxLine(m.width, padVisible(renderToast(m.toast), innerW)))
	}
	out = append(out, boxSep(m.width))
	out = append(out, boxLine(m.width, padVisible(footerStyle.Render(footer), innerW)))
	out = append(out, boxBottom(m.width))
	return strings.Join(out, "\n")
}
