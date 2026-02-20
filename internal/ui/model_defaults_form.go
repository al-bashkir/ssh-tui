package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bashkir/ssh-tui/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type defaultsFormCancelMsg struct{}

type defaultsFormSaveMsg struct {
	defaults config.Defaults
}

type defaultsField int

const (
	defaultsFieldUser defaultsField = iota
	defaultsFieldPort
	defaultsFieldIdentity
	defaultsFieldExtraArgs
	defaultsFieldAccentColor
	defaultsFieldLoadKnownHosts
	defaultsFieldTmux
	defaultsFieldOpenMode
	defaultsFieldTmuxSession
	defaultsFieldConfirmQuit
	defaultsFieldConnectThreshold
	defaultsFieldPaneSplit
	defaultsFieldPaneLayout
	defaultsFieldPaneSync
	defaultsFieldPaneBorderStatus
	defaultsFieldPaneBorderFormat
)

type defaultsFormModel struct {
	defaults config.Defaults

	width  int
	height int

	focus   defaultsField
	editing bool // true when editing a text field (insert mode)

	inUser      textinput.Model
	inPort      textinput.Model
	inIdentity  textinput.Model
	inExtra     textinput.Model
	inSession   textinput.Model
	inThreshold textinput.Model

	borderPicker *paneBorderFormatsModel

	toast string

	keymap keyMap

	confirmQuitEnabled bool
	confirmQuit        bool
}

func (m *defaultsFormModel) refreshAccentStyles() {
	setSearchFocused(&m.inUser, m.focus == defaultsFieldUser)
	setSearchFocused(&m.inPort, m.focus == defaultsFieldPort)
	setSearchFocused(&m.inIdentity, m.focus == defaultsFieldIdentity)
	setSearchFocused(&m.inExtra, m.focus == defaultsFieldExtraArgs)
	setSearchFocused(&m.inSession, m.focus == defaultsFieldTmuxSession)
	setSearchFocused(&m.inThreshold, m.focus == defaultsFieldConnectThreshold)
	if m.borderPicker != nil {
		m.borderPicker.refreshAccentStyles()
	}
}

func newDefaultsFormModel(d config.Defaults, confirmQuitEnabled bool) *defaultsFormModel {
	user := textinput.New()
	user.CharLimit = 128
	user.Prompt = ""
	user.SetValue(strings.TrimSpace(d.User))
	user.Placeholder = "login username"
	configureSearch(&user)

	port := textinput.New()
	port.CharLimit = 16
	port.Prompt = ""
	if d.Port != 0 {
		port.SetValue(strconv.Itoa(d.Port))
	}
	port.Placeholder = "22"
	configureSearch(&port)

	identity := textinput.New()
	identity.CharLimit = 512
	identity.Prompt = ""
	identity.SetValue(strings.TrimSpace(d.IdentityFile))
	identity.Placeholder = "~/.ssh/id_ed25519"
	configureSearch(&identity)

	extra := textinput.New()
	extra.CharLimit = 1024
	extra.Prompt = ""
	extra.SetValue(strings.Join(d.ExtraArgs, " "))
	extra.Placeholder = "-o Option=value ..."
	configureSearch(&extra)

	session := textinput.New()
	session.CharLimit = 128
	session.Prompt = ""
	session.SetValue(strings.TrimSpace(d.TmuxSession))
	session.Placeholder = "ssh-tui"
	configureSearch(&session)

	threshold := textinput.New()
	threshold.CharLimit = 6
	threshold.Prompt = ""
	if d.ConnectConfirmThreshold >= 0 {
		threshold.SetValue(strconv.Itoa(d.ConnectConfirmThreshold))
	}
	threshold.Placeholder = "5"
	configureSearch(&threshold)

	m := &defaultsFormModel{
		defaults:           d,
		focus:              defaultsFieldUser,
		inUser:             user,
		inPort:             port,
		inIdentity:         identity,
		inExtra:            extra,
		inSession:          session,
		inThreshold:        threshold,
		keymap:             defaultKeyMap(),
		confirmQuitEnabled: confirmQuitEnabled,
	}

	// Start in normal mode (no text input focused).
	setSearchFocused(&m.inUser, true)
	setSearchFocused(&m.inPort, false)
	setSearchFocused(&m.inIdentity, false)
	setSearchFocused(&m.inExtra, false)
	setSearchFocused(&m.inSession, false)
	setSearchFocused(&m.inThreshold, false)
	return m
}

func (m *defaultsFormModel) Init() tea.Cmd { return nil }

func (m *defaultsFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case paneBorderFormatsCancelMsg:
		m.borderPicker = nil
		return m, nil
	case paneBorderFormatsDoneMsg:
		m.defaults.PaneBorderFmt = strings.TrimSpace(msg.value)
		m.defaults.PaneBorderFmts = append([]string(nil), msg.customFmts...)
		m.borderPicker = nil
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		innerW := max(0, msg.Width-2)
		labelW := 16
		fieldW := max(10, innerW-labelW-1)
		m.inUser.Width = fieldW
		m.inPort.Width = min(12, fieldW)
		m.inIdentity.Width = fieldW
		m.inExtra.Width = fieldW
		m.inSession.Width = fieldW
		m.inThreshold.Width = min(12, fieldW)
		if m.borderPicker != nil {
			mw, mh := pickerModalSize(msg.Width, msg.Height)
			_, _ = m.borderPicker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		return m, nil
	case tea.KeyMsg:
		if m.borderPicker != nil {
			model, cmd := m.borderPicker.Update(msg)
			if pm, ok := model.(*paneBorderFormatsModel); ok {
				m.borderPicker = pm
			}
			return m, cmd
		}

		if m.confirmQuit {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmQuit = false
				m.toast = ""
				return m, nil
			default:
				return m, nil
			}
		}

		if key.Matches(msg, m.keymap.Settings) {
			if m.editing {
				m.exitEdit()
			}
			m.toast = ""
			if err := m.apply(); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, func() tea.Msg { return defaultsFormSaveMsg{defaults: m.defaults} }
		}

		if key.Matches(msg, m.keymap.Quit) {
			if !m.confirmQuitEnabled {
				return m, tea.Quit
			}
			m.confirmQuit = true
			m.toast = "quit? (y/n)"
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
		s := msg.String()
		if key.Matches(msg, m.keymap.Esc) {
			return m, func() tea.Msg { return defaultsFormCancelMsg{} }
		}

		if (s == "enter" || s == " " || s == "l") && m.focus == defaultsFieldPaneBorderFormat {
			mw, mh := pickerModalSize(m.width, m.height)
			m.borderPicker = newPaneBorderFormatsModel(m.defaults, m.defaults.PaneBorderFmt, false, true)
			if mw > 0 && mh > 0 {
				_, _ = m.borderPicker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
			}
			return m, nil
		}

		switch s {
		case "j", "down", "tab", "enter":
			return m, m.moveFocus(1)
		case "k", "up", "shift+tab":
			return m, m.moveFocus(-1)
		case "i":
			if m.isTextField() {
				m.enterEdit()
			}
			return m, nil
		case "h", "l", "left", "right", " ":
			delta := 1
			if s == "h" || s == "left" {
				delta = -1
			}
			switch m.focus {
			case defaultsFieldAccentColor:
				m.defaults.AccentColor = cycleChoice(m.defaults.AccentColor, []string{"", "blue", "cyan", "green", "amber", "red", "magenta"}, delta)
				return m, nil
			case defaultsFieldLoadKnownHosts:
				m.defaults.LoadKnownHosts = !m.defaults.LoadKnownHosts
				return m, nil
			case defaultsFieldTmux:
				m.defaults.Tmux = cycleChoice(m.defaults.Tmux, []string{"auto", "force", "never"}, delta)
				return m, nil
			case defaultsFieldOpenMode:
				m.defaults.OpenMode = cycleChoice(m.defaults.OpenMode, []string{"auto", "current", "tmux-window", "tmux-pane"}, delta)
				return m, nil
			case defaultsFieldConfirmQuit:
				m.defaults.ConfirmQuit = !m.defaults.ConfirmQuit
				return m, nil
			case defaultsFieldPaneSplit:
				m.defaults.PaneSplit = cycleChoice(m.defaults.PaneSplit, []string{"horizontal", "vertical"}, delta)
				return m, nil
			case defaultsFieldPaneLayout:
				m.defaults.PaneLayout = cycleChoice(m.defaults.PaneLayout, []string{"auto", "tiled", "even-horizontal", "even-vertical", "main-horizontal", "main-vertical"}, delta)
				return m, nil
			case defaultsFieldPaneSync:
				m.defaults.PaneSync = cycleChoice(m.defaults.PaneSync, []string{"on", "off"}, delta)
				return m, nil
			case defaultsFieldPaneBorderStatus:
				m.defaults.PaneBorderPos = cycleChoice(m.defaults.PaneBorderPos, []string{"bottom", "top", "off"}, delta)
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *defaultsFormModel) moveFocus(delta int) tea.Cmd {
	order := []defaultsField{
		defaultsFieldUser,
		defaultsFieldPort,
		defaultsFieldIdentity,
		defaultsFieldExtraArgs,
		defaultsFieldAccentColor,
		defaultsFieldLoadKnownHosts,
		defaultsFieldTmux,
		defaultsFieldOpenMode,
		defaultsFieldTmuxSession,
		defaultsFieldConfirmQuit,
		defaultsFieldConnectThreshold,
		defaultsFieldPaneSplit,
		defaultsFieldPaneLayout,
		defaultsFieldPaneSync,
		defaultsFieldPaneBorderStatus,
		defaultsFieldPaneBorderFormat,
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

func (m *defaultsFormModel) setFocus(f defaultsField) {
	m.focus = f
	m.editing = false

	// Blur all text inputs.
	m.inUser.Blur()
	m.inPort.Blur()
	m.inIdentity.Blur()
	m.inExtra.Blur()
	m.inSession.Blur()
	m.inThreshold.Blur()
	setSearchFocused(&m.inUser, false)
	setSearchFocused(&m.inPort, false)
	setSearchFocused(&m.inIdentity, false)
	setSearchFocused(&m.inExtra, false)
	setSearchFocused(&m.inSession, false)
	setSearchFocused(&m.inThreshold, false)

	// Highlight the focused field label (but don't activate text cursor).
	switch f {
	case defaultsFieldUser:
		setSearchFocused(&m.inUser, true)
	case defaultsFieldPort:
		setSearchFocused(&m.inPort, true)
	case defaultsFieldIdentity:
		setSearchFocused(&m.inIdentity, true)
	case defaultsFieldExtraArgs:
		setSearchFocused(&m.inExtra, true)
	case defaultsFieldTmuxSession:
		setSearchFocused(&m.inSession, true)
	case defaultsFieldConnectThreshold:
		setSearchFocused(&m.inThreshold, true)
	}
}

func (m *defaultsFormModel) isTextField() bool {
	switch m.focus {
	case defaultsFieldUser, defaultsFieldPort, defaultsFieldIdentity, defaultsFieldExtraArgs, defaultsFieldTmuxSession, defaultsFieldConnectThreshold:
		return true
	}
	return false
}

func (m *defaultsFormModel) enterEdit() {
	m.editing = true
	switch m.focus {
	case defaultsFieldUser:
		_ = m.inUser.Focus()
	case defaultsFieldPort:
		_ = m.inPort.Focus()
	case defaultsFieldIdentity:
		_ = m.inIdentity.Focus()
	case defaultsFieldExtraArgs:
		_ = m.inExtra.Focus()
	case defaultsFieldTmuxSession:
		_ = m.inSession.Focus()
	case defaultsFieldConnectThreshold:
		_ = m.inThreshold.Focus()
	}
}

func (m *defaultsFormModel) exitEdit() {
	m.editing = false
	m.inUser.Blur()
	m.inPort.Blur()
	m.inIdentity.Blur()
	m.inExtra.Blur()
	m.inSession.Blur()
	m.inThreshold.Blur()
}

func (m *defaultsFormModel) updateFocusedInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.focus {
	case defaultsFieldUser:
		m.inUser, cmd = m.inUser.Update(msg)
	case defaultsFieldPort:
		m.inPort, cmd = m.inPort.Update(msg)
	case defaultsFieldIdentity:
		m.inIdentity, cmd = m.inIdentity.Update(msg)
	case defaultsFieldExtraArgs:
		m.inExtra, cmd = m.inExtra.Update(msg)
	case defaultsFieldTmuxSession:
		m.inSession, cmd = m.inSession.Update(msg)
	case defaultsFieldConnectThreshold:
		m.inThreshold, cmd = m.inThreshold.Update(msg)
	}
	return cmd
}

func (m *defaultsFormModel) apply() error {
	m.defaults.User = strings.TrimSpace(m.inUser.Value())
	m.defaults.IdentityFile = strings.TrimSpace(m.inIdentity.Value())
	m.defaults.TmuxSession = strings.TrimSpace(m.inSession.Value())
	m.defaults.AccentColor = strings.ToLower(strings.TrimSpace(m.defaults.AccentColor))
	if m.defaults.AccentColor == "default" {
		m.defaults.AccentColor = ""
	}
	m.defaults.PaneBorderFmt = strings.TrimSpace(m.defaults.PaneBorderFmt)
	if m.defaults.PaneBorderFmt == "" {
		m.defaults.PaneBorderFmt = config.DefaultPaneBorderFormat
	}
	// Sanitize formats list.
	clean := make([]string, 0, len(m.defaults.PaneBorderFmts))
	seen := map[string]bool{strings.TrimSpace(config.DefaultPaneBorderFormat): true}
	for _, s := range m.defaults.PaneBorderFmts {
		v := strings.TrimSpace(s)
		if v == "" {
			continue
		}
		if seen[v] {
			continue
		}
		clean = append(clean, v)
		seen[v] = true
	}
	m.defaults.PaneBorderFmts = clean

	portStr := strings.TrimSpace(m.inPort.Value())
	if portStr == "" {
		m.defaults.Port = 22
	} else {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 {
			return fmt.Errorf("invalid port")
		}
		m.defaults.Port = p
	}

	extra := strings.TrimSpace(m.inExtra.Value())
	if extra == "" {
		m.defaults.ExtraArgs = nil
	} else {
		m.defaults.ExtraArgs = strings.Fields(extra)
	}

	if m.defaults.TmuxSession == "" {
		m.defaults.TmuxSession = "ssh-tui"
	}

	threshStr := strings.TrimSpace(m.inThreshold.Value())
	if threshStr == "" {
		m.defaults.ConnectConfirmThreshold = 5
	} else {
		t, err := strconv.Atoi(threshStr)
		if err != nil || t < 0 {
			return fmt.Errorf("confirm threshold must be a number >= 0")
		}
		m.defaults.ConnectConfirmThreshold = t
	}

	return nil
}

func (m *defaultsFormModel) View() string {
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.borderPicker != nil {
		return placeCentered(m.width, m.height, m.borderPicker.View())
	}

	innerW := max(0, m.width-2)
	innerH := max(0, m.height-2)
	contentH := max(0, innerH-4)
	labelW := 16
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

	seg := func(cur, val, text string, focused bool) string {
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

	lines := []string{}
	focusLine := 0

	lines = append(lines, formSection("SSH", innerW))
	if m.focus == defaultsFieldUser {
		focusLine = len(lines)
	}
	lines = append(lines, label("User:", m.focus == defaultsFieldUser)+" "+inputLine(m.inUser, m.focus == defaultsFieldUser, fieldW))
	if m.focus == defaultsFieldPort {
		focusLine = len(lines)
	}
	lines = append(lines, label("Port:", m.focus == defaultsFieldPort)+" "+inputLine(m.inPort, m.focus == defaultsFieldPort, min(12, fieldW)))
	if m.focus == defaultsFieldIdentity {
		focusLine = len(lines)
	}
	lines = append(lines, label("Identity file:", m.focus == defaultsFieldIdentity)+" "+inputLine(m.inIdentity, m.focus == defaultsFieldIdentity, fieldW))
	if m.focus == defaultsFieldExtraArgs {
		focusLine = len(lines)
	}
	lines = append(lines, label("Extra args:", m.focus == defaultsFieldExtraArgs)+" "+inputLine(m.inExtra, m.focus == defaultsFieldExtraArgs, fieldW))

	lines = append(lines, formSection("UI", innerW))
	if m.focus == defaultsFieldAccentColor {
		focusLine = len(lines)
	}

	accentCur := strings.TrimSpace(m.defaults.AccentColor)
	accentFocused := m.focus == defaultsFieldAccentColor
	accent1 := seg(accentCur, "", "default", accentFocused) + "  " + seg(accentCur, "blue", "blue", accentFocused) + "  " + seg(accentCur, "cyan", "cyan", accentFocused) + "  " + seg(accentCur, "green", "green", accentFocused)
	accent2 := seg(accentCur, "amber", "amber", accentFocused) + "  " + seg(accentCur, "red", "red", accentFocused) + "  " + seg(accentCur, "magenta", "magenta", accentFocused)
	lines = append(lines, label("Accent:", accentFocused)+" "+accent1)
	lines = append(lines, "  "+accent2)

	loadCur := "no"
	if m.defaults.LoadKnownHosts {
		loadCur = "yes"
	}
	loadFocused := m.focus == defaultsFieldLoadKnownHosts
	loadLine := seg(loadCur, "yes", "yes", loadFocused) + "  " + seg(loadCur, "no", "no", loadFocused)
	if loadFocused {
		focusLine = len(lines)
	}
	lines = append(lines, label("Load known_hosts:", loadFocused)+" "+loadLine)

	lines = append(lines, formSection("Tmux", innerW))

	tmuxCur := strings.TrimSpace(m.defaults.Tmux)
	tmuxFocused := m.focus == defaultsFieldTmux
	tmuxLine := seg(tmuxCur, "auto", "auto", tmuxFocused) + "  " + seg(tmuxCur, "force", "force", tmuxFocused) + "  " + seg(tmuxCur, "never", "never", tmuxFocused)
	if tmuxFocused {
		focusLine = len(lines)
	}
	lines = append(lines, label("Tmux:", tmuxFocused)+" "+tmuxLine)

	openCur := strings.TrimSpace(m.defaults.OpenMode)
	openFocused := m.focus == defaultsFieldOpenMode
	open1 := seg(openCur, "auto", "auto", openFocused) + "  " + seg(openCur, "current", "current", openFocused)
	open2 := seg(openCur, "tmux-window", "tmux-window", openFocused) + "  " + seg(openCur, "tmux-pane", "tmux-pane", openFocused)
	if openFocused {
		focusLine = len(lines)
	}
	lines = append(lines, label("Open mode:", openFocused)+" "+open1)
	lines = append(lines, "  "+open2)

	if m.focus == defaultsFieldTmuxSession {
		focusLine = len(lines)
	}
	lines = append(lines, label("Tmux session:", m.focus == defaultsFieldTmuxSession)+" "+inputLine(m.inSession, m.focus == defaultsFieldTmuxSession, fieldW))

	confirmCur := "no"
	if m.defaults.ConfirmQuit {
		confirmCur = "yes"
	}
	confirmFocused := m.focus == defaultsFieldConfirmQuit
	confirmLine := seg(confirmCur, "yes", "yes", confirmFocused) + "  " + seg(confirmCur, "no", "no", confirmFocused)
	if confirmFocused {
		focusLine = len(lines)
	}
	lines = append(lines, label("Confirm quit:", confirmFocused)+" "+confirmLine)

	if m.focus == defaultsFieldConnectThreshold {
		focusLine = len(lines)
	}
	lines = append(lines, label("Confirm at:", m.focus == defaultsFieldConnectThreshold)+" "+inputLine(m.inThreshold, m.focus == defaultsFieldConnectThreshold, min(12, fieldW)))

	lines = append(lines, formSection("Panes", innerW))

	splitCur := strings.TrimSpace(m.defaults.PaneSplit)
	splitFocused := m.focus == defaultsFieldPaneSplit
	splitLine := seg(splitCur, "horizontal", "horizontal", splitFocused) + "  " + seg(splitCur, "vertical", "vertical", splitFocused)
	if splitFocused {
		focusLine = len(lines)
	}
	lines = append(lines, label("Pane split:", splitFocused)+" "+splitLine)

	layoutCur := strings.TrimSpace(m.defaults.PaneLayout)
	layoutFocused := m.focus == defaultsFieldPaneLayout
	layout1 := seg(layoutCur, "auto", "auto", layoutFocused) + "  " + seg(layoutCur, "tiled", "tiled", layoutFocused) + "  " + seg(layoutCur, "even-horizontal", "even-horizontal", layoutFocused)
	layout2 := seg(layoutCur, "even-vertical", "even-vertical", layoutFocused) + "  " + seg(layoutCur, "main-horizontal", "main-horizontal", layoutFocused) + "  " + seg(layoutCur, "main-vertical", "main-vertical", layoutFocused)
	if layoutFocused {
		focusLine = len(lines)
	}
	lines = append(lines, label("Pane layout:", layoutFocused)+" "+layout1)
	lines = append(lines, "  "+layout2)

	syncCur := strings.TrimSpace(m.defaults.PaneSync)
	syncFocused := m.focus == defaultsFieldPaneSync
	syncLine := seg(syncCur, "on", "on", syncFocused) + "  " + seg(syncCur, "off", "off", syncFocused)
	if syncFocused {
		focusLine = len(lines)
	}
	lines = append(lines, label("Pane sync:", syncFocused)+" "+syncLine)

	borderCur := strings.TrimSpace(m.defaults.PaneBorderPos)
	borderFocused := m.focus == defaultsFieldPaneBorderStatus
	borderLine := seg(borderCur, "bottom", "bottom", borderFocused) + "  " + seg(borderCur, "top", "top", borderFocused) + "  " + seg(borderCur, "off", "off", borderFocused)
	if borderFocused {
		focusLine = len(lines)
	}
	lines = append(lines, label("Pane border:", borderFocused)+" "+borderLine)
	fmtVal := strings.TrimSpace(m.defaults.PaneBorderFmt)
	showFmt := fmtVal
	if fmtVal == "" || strings.TrimSpace(config.DefaultPaneBorderFormat) == fmtVal {
		showFmt = "default"
	}
	bf := padVisible(showFmt, fieldW)
	if m.focus == defaultsFieldPaneBorderFormat {
		bf = checkedStyle.Render(bf)
		focusLine = len(lines)
	} else {
		bf = dim.Render(bf)
	}
	lines = append(lines, label("Border format:", m.focus == defaultsFieldPaneBorderFormat)+" "+bf)

	fieldPos := fmt.Sprintf("%d/%d", int(m.focus)+1, int(defaultsFieldPaneBorderFormat)+1)
	footer := footerStyle.Render(fieldPos + "  Ctrl+S save   j/k move   h/l option   i edit   Esc back")
	if m.editing {
		footer = footerStyle.Render(fieldPos) + "  " + headerStyle.Render("INSERT") + "  " + footerStyle.Render("Ctrl+S save   Esc done")
	}
	toast := ""
	if strings.TrimSpace(m.toast) != "" {
		toast = statusWarn.Render(m.toast)
	}

	// Reserve lines for toast + footer.
	reservedBottom := 1 // footer
	if toast != "" {
		reservedBottom++
	}
	visibleH := contentH - reservedBottom
	if visibleH < 1 {
		visibleH = 1
	}

	// Scroll so focused field is visible.
	start, end := formScrollWindow(len(lines), visibleH, focusLine)
	visible := lines[start:end]

	contentLines := make([]string, 0, contentH)
	contentLines = append(contentLines, visible...)
	// Fill remaining space.
	fill := visibleH - len(visible)
	for i := 0; i < fill; i++ {
		contentLines = append(contentLines, "")
	}
	if toast != "" {
		contentLines = append(contentLines, toast)
	}
	contentLines = append(contentLines, footer)

	headLeft := headerStyle.Render("Settings")
	headRight := statusDot(true, false)
	return renderMainTabBox(m.width, m.height, 2, headLeft, headRight, strings.Join(contentLines, "\n"))
}
