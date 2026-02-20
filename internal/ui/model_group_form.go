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

type groupFormCancelMsg struct{}

type groupFormSaveMsg struct {
	index int
	group config.Group
}

type groupField int

const (
	groupFieldName groupField = iota
	groupFieldUser
	groupFieldPort
	groupFieldIdentity
	groupFieldExtraArgs
	groupFieldRemoteCommand
	groupFieldOpenMode
	groupFieldTmux
	groupFieldPaneSplit
	groupFieldPaneLayout
	groupFieldPaneSync
	groupFieldPaneBorderStatus
	groupFieldPaneBorderFormat
)

type groupFormModel struct {
	index int
	group config.Group
	defs  config.Defaults

	width  int
	height int

	focus   groupField
	editing bool // true when editing a text field (insert mode)

	inName     textinput.Model
	inUser     textinput.Model
	inPort     textinput.Model
	inIdentity textinput.Model
	inExtra    textinput.Model
	inRemote   textinput.Model

	borderPicker *paneBorderFormatsModel

	toast string

	keymap keyMap

	confirmQuitEnabled bool
	confirmQuit        bool
}

func (m *groupFormModel) refreshAccentStyles() {
	setSearchFocused(&m.inName, m.focus == groupFieldName)
	setSearchFocused(&m.inUser, m.focus == groupFieldUser)
	setSearchFocused(&m.inPort, m.focus == groupFieldPort)
	setSearchFocused(&m.inIdentity, m.focus == groupFieldIdentity)
	setSearchFocused(&m.inExtra, m.focus == groupFieldExtraArgs)
	setSearchFocused(&m.inRemote, m.focus == groupFieldRemoteCommand)
	if m.borderPicker != nil {
		m.borderPicker.refreshAccentStyles()
	}
}

func newGroupFormModel(index int, g config.Group, defs config.Defaults, confirmQuitEnabled bool) *groupFormModel {
	name := textinput.New()
	name.CharLimit = 128
	name.Prompt = ""
	name.SetValue(strings.TrimSpace(g.Name))
	name.Placeholder = "prod"
	configureSearch(&name)

	user := textinput.New()
	user.CharLimit = 128
	user.Prompt = ""
	user.SetValue(strings.TrimSpace(g.User))
	user.Placeholder = strings.TrimSpace(defs.User)
	configureSearch(&user)

	port := textinput.New()
	port.CharLimit = 16
	port.Prompt = ""
	if g.Port != 0 {
		port.SetValue(strconv.Itoa(g.Port))
	}
	if defs.Port != 0 {
		port.Placeholder = strconv.Itoa(defs.Port)
	} else {
		port.Placeholder = "22"
	}
	configureSearch(&port)

	identity := textinput.New()
	identity.CharLimit = 512
	identity.Prompt = ""
	identity.SetValue(strings.TrimSpace(g.IdentityFile))
	identity.Placeholder = strings.TrimSpace(defs.IdentityFile)
	if identity.Placeholder == "" {
		identity.Placeholder = "~/.ssh/id_ed25519"
	}
	configureSearch(&identity)

	extra := textinput.New()
	extra.CharLimit = 1024
	extra.Prompt = ""
	extra.SetValue(strings.Join(g.ExtraArgs, " "))
	extra.Placeholder = strings.Join(defs.ExtraArgs, " ")
	if extra.Placeholder == "" {
		extra.Placeholder = "-o Option=value ..."
	}
	configureSearch(&extra)

	remote := textinput.New()
	remote.CharLimit = 1024
	remote.Prompt = ""
	remote.SetValue(strings.TrimSpace(g.RemoteCommand))
	remote.Placeholder = "command to run on connect"
	configureSearch(&remote)

	m := &groupFormModel{
		index:              index,
		group:              g,
		defs:               defs,
		focus:              groupFieldName,
		inName:             name,
		inUser:             user,
		inPort:             port,
		inIdentity:         identity,
		inExtra:            extra,
		inRemote:           remote,
		keymap:             defaultKeyMap(),
		confirmQuitEnabled: confirmQuitEnabled,
	}

	// Reasonable default for new groups.
	if index < 0 && strings.TrimSpace(m.group.OpenMode) == "" {
		m.group.OpenMode = "tmux-window"
	}

	// Start in normal mode (no text input focused).
	setSearchFocused(&m.inName, true)
	setSearchFocused(&m.inUser, false)
	setSearchFocused(&m.inPort, false)
	setSearchFocused(&m.inIdentity, false)
	setSearchFocused(&m.inExtra, false)
	setSearchFocused(&m.inRemote, false)
	return m
}

func (m *groupFormModel) Init() tea.Cmd { return nil }

func (m *groupFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case paneBorderFormatsCancelMsg:
		m.borderPicker = nil
		return m, nil
	case paneBorderFormatsDoneMsg:
		m.group.PaneBorderFmt = strings.TrimSpace(msg.value)
		m.borderPicker = nil
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		innerW := max(0, msg.Width-2)
		labelW := 14
		fieldW := max(10, innerW-labelW-1)
		m.inName.Width = fieldW
		m.inUser.Width = fieldW
		m.inPort.Width = min(12, fieldW)
		m.inIdentity.Width = fieldW
		m.inExtra.Width = fieldW
		m.inRemote.Width = fieldW
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

		// Save.
		if key.Matches(msg, m.keymap.Settings) {
			if m.editing {
				m.exitEdit()
			}
			m.toast = ""
			if err := m.apply(); err != nil {
				m.toast = err.Error()
				return m, nil
			}
			return m, func() tea.Msg { return groupFormSaveMsg{index: m.index, group: m.group} }
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
			return m, func() tea.Msg { return groupFormCancelMsg{} }
		}

		if (s == "enter" || s == " " || s == "l") && m.focus == groupFieldPaneBorderFormat {
			mw, mh := pickerModalSize(m.width, m.height)
			m.borderPicker = newPaneBorderFormatsModel(m.defs, m.group.PaneBorderFmt, true, false)
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
			case groupFieldOpenMode:
				m.cycleOpenMode(delta)
				return m, nil
			case groupFieldTmux:
				m.cycleTmux(delta)
				return m, nil
			case groupFieldPaneSplit:
				m.cyclePaneSplit(delta)
				return m, nil
			case groupFieldPaneLayout:
				m.cyclePaneLayout(delta)
				return m, nil
			case groupFieldPaneSync:
				m.cyclePaneSync(delta)
				return m, nil
			case groupFieldPaneBorderStatus:
				m.cyclePaneBorderStatus(delta)
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *groupFormModel) updateFocusedInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.focus {
	case groupFieldName:
		m.inName, cmd = m.inName.Update(msg)
	case groupFieldUser:
		m.inUser, cmd = m.inUser.Update(msg)
	case groupFieldPort:
		m.inPort, cmd = m.inPort.Update(msg)
	case groupFieldIdentity:
		m.inIdentity, cmd = m.inIdentity.Update(msg)
	case groupFieldExtraArgs:
		m.inExtra, cmd = m.inExtra.Update(msg)
	case groupFieldRemoteCommand:
		m.inRemote, cmd = m.inRemote.Update(msg)
	default:
		// no-op
	}
	return cmd
}

func (m *groupFormModel) moveFocus(delta int) tea.Cmd {
	order := []groupField{
		groupFieldName,
		groupFieldUser,
		groupFieldPort,
		groupFieldIdentity,
		groupFieldExtraArgs,
		groupFieldRemoteCommand,
		groupFieldOpenMode,
		groupFieldTmux,
		groupFieldPaneSplit,
		groupFieldPaneLayout,
		groupFieldPaneSync,
		groupFieldPaneBorderStatus,
		groupFieldPaneBorderFormat,
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

func (m *groupFormModel) setFocus(f groupField) {
	m.focus = f
	m.editing = false

	// Blur all text inputs.
	m.inName.Blur()
	m.inUser.Blur()
	m.inPort.Blur()
	m.inIdentity.Blur()
	m.inExtra.Blur()
	m.inRemote.Blur()
	setSearchFocused(&m.inName, false)
	setSearchFocused(&m.inUser, false)
	setSearchFocused(&m.inPort, false)
	setSearchFocused(&m.inIdentity, false)
	setSearchFocused(&m.inExtra, false)
	setSearchFocused(&m.inRemote, false)

	// Highlight the focused field label (but don't activate text cursor).
	switch f {
	case groupFieldName:
		setSearchFocused(&m.inName, true)
	case groupFieldUser:
		setSearchFocused(&m.inUser, true)
	case groupFieldPort:
		setSearchFocused(&m.inPort, true)
	case groupFieldIdentity:
		setSearchFocused(&m.inIdentity, true)
	case groupFieldExtraArgs:
		setSearchFocused(&m.inExtra, true)
	case groupFieldRemoteCommand:
		setSearchFocused(&m.inRemote, true)
	}
}

func (m *groupFormModel) isTextField() bool {
	switch m.focus {
	case groupFieldName, groupFieldUser, groupFieldPort, groupFieldIdentity, groupFieldExtraArgs, groupFieldRemoteCommand:
		return true
	}
	return false
}

func (m *groupFormModel) enterEdit() {
	m.editing = true
	switch m.focus {
	case groupFieldName:
		_ = m.inName.Focus()
	case groupFieldUser:
		_ = m.inUser.Focus()
	case groupFieldPort:
		_ = m.inPort.Focus()
	case groupFieldIdentity:
		_ = m.inIdentity.Focus()
	case groupFieldExtraArgs:
		_ = m.inExtra.Focus()
	case groupFieldRemoteCommand:
		_ = m.inRemote.Focus()
	}
}

func (m *groupFormModel) exitEdit() {
	m.editing = false
	m.inName.Blur()
	m.inUser.Blur()
	m.inPort.Blur()
	m.inIdentity.Blur()
	m.inExtra.Blur()
	m.inRemote.Blur()
}

func (m *groupFormModel) cycleOpenMode(delta int) {
	vals := []string{"", "auto", "current", "tmux-window", "tmux-pane"}
	m.group.OpenMode = cycleChoice(m.group.OpenMode, vals, delta)
}

func (m *groupFormModel) cycleTmux(delta int) {
	vals := []string{"", "auto", "force", "never"}
	m.group.Tmux = cycleChoice(m.group.Tmux, vals, delta)
}

func (m *groupFormModel) cyclePaneSplit(delta int) {
	vals := []string{"", "horizontal", "vertical"}
	m.group.PaneSplit = cycleChoice(m.group.PaneSplit, vals, delta)
}

func (m *groupFormModel) cyclePaneLayout(delta int) {
	vals := []string{"", "auto", "tiled", "even-horizontal", "even-vertical", "main-horizontal", "main-vertical"}
	m.group.PaneLayout = cycleChoice(m.group.PaneLayout, vals, delta)
}

func (m *groupFormModel) cyclePaneSync(delta int) {
	vals := []string{"", "on", "off"}
	m.group.PaneSync = cycleChoice(m.group.PaneSync, vals, delta)
}

func (m *groupFormModel) cyclePaneBorderStatus(delta int) {
	vals := []string{"", "bottom", "top", "off"}
	m.group.PaneBorderPos = cycleChoice(m.group.PaneBorderPos, vals, delta)
}

func cycleChoice(cur string, vals []string, delta int) string {
	cur = strings.TrimSpace(cur)
	idx := 0
	for i := range vals {
		if strings.TrimSpace(vals[i]) == cur {
			idx = i
			break
		}
	}
	idx += delta
	if idx < 0 {
		idx = len(vals) - 1
	}
	if idx >= len(vals) {
		idx = 0
	}
	return vals[idx]
}

func (m *groupFormModel) apply() error {
	m.group.Name = strings.TrimSpace(m.inName.Value())
	m.group.User = strings.TrimSpace(m.inUser.Value())
	m.group.IdentityFile = strings.TrimSpace(m.inIdentity.Value())
	m.group.RemoteCommand = strings.TrimSpace(m.inRemote.Value())
	m.group.PaneBorderFmt = strings.TrimSpace(m.group.PaneBorderFmt)

	portStr := strings.TrimSpace(m.inPort.Value())
	if portStr == "" {
		m.group.Port = 0
	} else {
		p, err := strconv.Atoi(portStr)
		if err != nil || p <= 0 {
			return fmt.Errorf("invalid port")
		}
		m.group.Port = p
	}

	extra := strings.TrimSpace(m.inExtra.Value())
	if extra == "" {
		m.group.ExtraArgs = nil
	} else {
		m.group.ExtraArgs = strings.Fields(extra)
	}

	return config.ValidateGroupName(m.group.Name)
}

func (m *groupFormModel) View() string {
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	if m.borderPicker != nil {
		return placeCentered(m.width, m.height, m.borderPicker.View())
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

	seg := func(cur, val, label string, focused bool) string {
		cur = strings.TrimSpace(cur)
		val = strings.TrimSpace(val)
		if cur == val {
			box := "[" + label + "]"
			if focused {
				return segFocusedStyle.Render(box)
			}
			return checkedStyle.Render(box)
		}
		return tabInactiveStyle.Render(label)
	}

	openCur := strings.TrimSpace(m.group.OpenMode)
	openFocused := m.focus == groupFieldOpenMode
	open1 := seg(openCur, "", "inherit", openFocused) + "  " + seg(openCur, "auto", "auto", openFocused) + "  " + seg(openCur, "current", "current", openFocused)
	open2 := seg(openCur, "tmux-window", "tmux-window", openFocused) + "  " + seg(openCur, "tmux-pane", "tmux-pane", openFocused)

	tmuxCur := strings.TrimSpace(m.group.Tmux)
	tmuxFocused := m.focus == groupFieldTmux
	tmuxLine := seg(tmuxCur, "", "inherit", tmuxFocused) + "  " + seg(tmuxCur, "auto", "auto", tmuxFocused) + "  " + seg(tmuxCur, "force", "force", tmuxFocused) + "  " + seg(tmuxCur, "never", "never", tmuxFocused)

	splitCur := strings.TrimSpace(m.group.PaneSplit)
	splitFocused := m.focus == groupFieldPaneSplit
	splitLine := seg(splitCur, "", "inherit", splitFocused) + "  " + seg(splitCur, "horizontal", "horizontal", splitFocused) + "  " + seg(splitCur, "vertical", "vertical", splitFocused)

	layoutCur := strings.TrimSpace(m.group.PaneLayout)
	layoutFocused := m.focus == groupFieldPaneLayout
	layout1 := seg(layoutCur, "", "inherit", layoutFocused) + "  " + seg(layoutCur, "auto", "auto", layoutFocused) + "  " + seg(layoutCur, "tiled", "tiled", layoutFocused) + "  " + seg(layoutCur, "even-horizontal", "even-horizontal", layoutFocused)
	layout2 := seg(layoutCur, "even-vertical", "even-vertical", layoutFocused) + "  " + seg(layoutCur, "main-horizontal", "main-horizontal", layoutFocused) + "  " + seg(layoutCur, "main-vertical", "main-vertical", layoutFocused)

	syncCur := strings.TrimSpace(m.group.PaneSync)
	syncFocused := m.focus == groupFieldPaneSync
	syncLine := seg(syncCur, "", "inherit", syncFocused) + "  " + seg(syncCur, "on", "on", syncFocused) + "  " + seg(syncCur, "off", "off", syncFocused)

	borderPosCur := strings.TrimSpace(m.group.PaneBorderPos)
	borderPosFocused := m.focus == groupFieldPaneBorderStatus
	borderPosLine := seg(borderPosCur, "", "inherit", borderPosFocused) + "  " + seg(borderPosCur, "bottom", "bottom", borderPosFocused) + "  " + seg(borderPosCur, "top", "top", borderPosFocused) + "  " + seg(borderPosCur, "off", "off", borderPosFocused)

	lines := []string{}
	focusLine := 0

	if m.focus == groupFieldName { focusLine = len(lines) }
	lines = append(lines, label("Name:", m.focus == groupFieldName)+" "+inputLine(m.inName, m.focus == groupFieldName, fieldW))
	lines = append(lines, formSection("SSH", innerW))
	if m.focus == groupFieldUser { focusLine = len(lines) }
	lines = append(lines, label("User:", m.focus == groupFieldUser)+" "+inputLine(m.inUser, m.focus == groupFieldUser, fieldW))
	if m.focus == groupFieldPort { focusLine = len(lines) }
	lines = append(lines, label("Port:", m.focus == groupFieldPort)+" "+inputLine(m.inPort, m.focus == groupFieldPort, min(12, fieldW)))
	if m.focus == groupFieldIdentity { focusLine = len(lines) }
	lines = append(lines, label("Identity file:", m.focus == groupFieldIdentity)+" "+inputLine(m.inIdentity, m.focus == groupFieldIdentity, fieldW))
	if m.focus == groupFieldExtraArgs { focusLine = len(lines) }
	lines = append(lines, label("Extra args:", m.focus == groupFieldExtraArgs)+" "+inputLine(m.inExtra, m.focus == groupFieldExtraArgs, fieldW))
	if m.focus == groupFieldRemoteCommand { focusLine = len(lines) }
	lines = append(lines, label("Remote cmd:", m.focus == groupFieldRemoteCommand)+" "+inputLine(m.inRemote, m.focus == groupFieldRemoteCommand, fieldW))
	lines = append(lines, formSection("Tmux", innerW))
	if openFocused { focusLine = len(lines) }
	lines = append(lines, label("Open mode:", openFocused)+" "+open1)
	lines = append(lines, "  "+open2)
	if tmuxFocused { focusLine = len(lines) }
	lines = append(lines, label("Tmux:", tmuxFocused)+" "+tmuxLine)
	lines = append(lines, formSection("Panes", innerW))
	if splitFocused { focusLine = len(lines) }
	lines = append(lines, label("Pane split:", splitFocused)+" "+splitLine)
	if layoutFocused { focusLine = len(lines) }
	lines = append(lines, label("Pane layout:", layoutFocused)+" "+layout1)
	lines = append(lines, "  "+layout2)
	if syncFocused { focusLine = len(lines) }
	lines = append(lines, label("Pane sync:", syncFocused)+" "+syncLine)
	if borderPosFocused { focusLine = len(lines) }
	lines = append(lines, label("Pane border:", borderPosFocused)+" "+borderPosLine)
	fmtVal := strings.TrimSpace(m.group.PaneBorderFmt)
	showFmt := fmtVal
	if fmtVal == "" {
		showFmt = "inherit"
	} else if strings.TrimSpace(config.DefaultPaneBorderFormat) == fmtVal {
		showFmt = "default"
	}
	bf := padVisible(showFmt, fieldW)
	if m.focus == groupFieldPaneBorderFormat {
		bf = checkedStyle.Render(bf)
		focusLine = len(lines)
	} else {
		bf = dim.Render(bf)
	}
	lines = append(lines, label("Border format:", m.focus == groupFieldPaneBorderFormat)+" "+bf)

	fieldPos := fmt.Sprintf("%d/%d", int(m.focus)+1, int(groupFieldPaneBorderFormat)+1)
	footer := fieldPos + "  Ctrl+S save   j/k move   h/l option   i edit   Esc cancel"
	if m.editing {
		footer = footerStyle.Render(fieldPos) + "  " + headerStyle.Render("INSERT") + "  " + footerStyle.Render("Ctrl+S save   Esc done")
	}

	// Build full-height box with scroll.
	innerH := max(0, m.height-2)
	reserved := 2 // sep + footer
	if strings.TrimSpace(m.toast) != "" {
		reserved++
	}
	visibleH := innerH - reserved
	if visibleH < 1 {
		visibleH = 1
	}

	start, end := formScrollWindow(len(lines), visibleH, focusLine)
	visible := lines[start:end]

	out := make([]string, 0, m.height)
	title := "Create Group"
	if m.index >= 0 {
		name := strings.TrimSpace(m.group.Name)
		if name == "" {
			name = strings.TrimSpace(m.inName.Value())
		}
		if name != "" {
			title = "Groups > " + name
		} else {
			title = "Edit Group"
		}
	}
	out = append(out, boxTitleTop(m.width, title))
	for _, ln := range visible {
		out = append(out, boxLine(m.width, padVisible(ln, innerW)))
	}
	fill := visibleH - len(visible)
	for i := 0; i < fill; i++ {
		out = append(out, boxLine(m.width, strings.Repeat(" ", innerW)))
	}
	if strings.TrimSpace(m.toast) != "" {
		out = append(out, boxLine(m.width, padVisible(statusWarn.Render(m.toast), innerW)))
	}
	out = append(out, boxSep(m.width))
	out = append(out, boxLine(m.width, padVisible(footerStyle.Render(footer), innerW)))
	out = append(out, boxBottom(m.width))
	return strings.Join(out, "\n")
}
