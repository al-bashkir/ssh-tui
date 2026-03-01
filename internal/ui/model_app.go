package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/hosts"

	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenHosts screen = iota
	screenGroups
	screenGroupForm
	screenGroupHosts
	screenHostPicker
	screenGroupPicker
	screenDefaultsForm
	screenCustomHost
	screenHostForm
)

type switchScreenMsg struct {
	to screen
}

type openGroupFormMsg struct {
	index int
}

type openGroupFormPrefillMsg struct {
	group config.Group
}

type deleteGroupMsg struct {
	index int
}

type openGroupHostsMsg struct {
	index int
}

type openHostPickerMsg struct {
	groupIndex int
	returnTo   screen
}

type openGroupPickerMsg struct {
	hosts []string
}

type openDefaultsFormMsg struct {
	returnTo screen
}

type openCustomHostMsg struct {
	returnTo   screen
	groupIndex int // -1 means no group context; Ctrl+a will pick a group
}

type openHostFormMsg struct {
	host     string
	returnTo screen
}

type openHostFormPrefillMsg struct {
	host     config.Host
	returnTo screen
}

type customHostCancelMsg struct{}

type customHostConnectMsg struct {
	returnTo   screen
	groupIndex int
	hosts      []string
}

type customHostDoneMsg struct {
	returnTo   screen
	groupIndex int
	hosts      []string
}

type customHostPickGroupMsg struct {
	returnTo screen
	hosts    []string
}

type groupPickerCancelMsg struct{}

type groupPickerDoneMsg struct {
	groupIndex int
}

type removeHostsMsg struct {
	groupIndex int
	hosts      []string
}

type hostFormCancelMsg struct{}

type hostFormSaveMsg struct {
	index int
	host  config.Host
}

type toggleHiddenHostMsg struct {
	host string
	hide bool
}

type defaultsToastExpireMsg struct {
	token int
}

type toastDismissMsg struct {
	token int
}

type appModel struct {
	opts Options

	width  int
	height int

	screen             screen
	hosts              *hostsModel
	groups             *groupsModel
	form               *groupFormModel
	gh                 *groupHostsModel
	picker             *hostPickerModel
	gp                 *groupPickerModel
	defaultsForm       *defaultsFormModel
	customHost         *customHostModel
	hostForm           *hostFormModel
	gpHosts            []string
	gpReturnTo         screen
	gpConnectAfterAdd  bool
	returnTo           screen
	returnGroupIndex   int
	defaultsReturnTo   screen
	hostFormReturnTo   screen
	defaultsToastToken int
	toastToken         int

	quitting bool
	execCmd  []string
}

func newAppModel(opts Options) *appModel {
	m := &appModel{
		opts:   opts,
		screen: screenHosts,
		hosts:  newHostsModel(opts),
		groups: newGroupsModel(opts),
	}
	return m
}

func (m *appModel) Init() tea.Cmd {
	return m.hosts.Init()
}

func (m *appModel) applyWindowSize(ws tea.WindowSizeMsg) tea.Cmd {
	m.width = ws.Width
	m.height = ws.Height

	var cmds []tea.Cmd
	if m.hosts != nil {
		model, cmd := m.hosts.Update(ws)
		if hm, ok := model.(*hostsModel); ok {
			m.hosts = hm
		}
		cmds = append(cmds, cmd)
	}
	if m.groups != nil {
		model, cmd := m.groups.Update(ws)
		if gm, ok := model.(*groupsModel); ok {
			m.groups = gm
		}
		cmds = append(cmds, cmd)
	}
	if m.form != nil {
		mw, mh := groupFormModalSize(ws.Width, ws.Height)
		model, cmd := m.form.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if fm, ok := model.(*groupFormModel); ok {
			m.form = fm
		}
		cmds = append(cmds, cmd)
	}
	if m.gh != nil {
		model, cmd := m.gh.Update(ws)
		if gh, ok := model.(*groupHostsModel); ok {
			m.gh = gh
		}
		cmds = append(cmds, cmd)
	}
	if m.picker != nil {
		mw, mh := pickerModalSize(ws.Width, ws.Height)
		model, cmd := m.picker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if pm, ok := model.(*hostPickerModel); ok {
			m.picker = pm
		}
		cmds = append(cmds, cmd)
	}
	if m.gp != nil {
		mw, mh := pickerModalSize(ws.Width, ws.Height)
		model, cmd := m.gp.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if gm, ok := model.(*groupPickerModel); ok {
			m.gp = gm
		}
		cmds = append(cmds, cmd)
	}
	if m.defaultsForm != nil {
		model, cmd := m.defaultsForm.Update(ws)
		if dm, ok := model.(*defaultsFormModel); ok {
			m.defaultsForm = dm
		}
		cmds = append(cmds, cmd)
	}
	if m.customHost != nil {
		mw, mh := customHostModalSize(ws.Width, ws.Height)
		model, cmd := m.customHost.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if cm, ok := model.(*customHostModel); ok {
			m.customHost = cm
		}
		cmds = append(cmds, cmd)
	}
	if m.hostForm != nil {
		mw, mh := hostFormModalSize(ws.Width, ws.Height)
		model, cmd := m.hostForm.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		if hm, ok := model.(*hostFormModel); ok {
			m.hostForm = hm
		}
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m *appModel) collectToastKey() string {
	var b strings.Builder
	if m.hosts != nil && !m.hosts.toast.empty() {
		b.WriteString(m.hosts.toast.text)
	}
	if m.groups != nil && !m.groups.toast.empty() {
		b.WriteByte('|')
		b.WriteString(m.groups.toast.text)
	}
	if m.gh != nil && !m.gh.toast.empty() {
		b.WriteByte('|')
		b.WriteString(m.gh.toast.text)
	}
	return b.String()
}

func (m *appModel) clearToasts() {
	if m.hosts != nil {
		m.hosts.toast = toast{}
	}
	if m.groups != nil {
		m.groups.toast = toast{}
	}
	if m.gh != nil {
		m.gh.toast = toast{}
	}
}

func (m *appModel) maxToastLevel() toastLevel {
	var lvl toastLevel
	if m.hosts != nil && !m.hosts.toast.empty() && m.hosts.toast.level > lvl {
		lvl = m.hosts.toast.level
	}
	if m.groups != nil && !m.groups.toast.empty() && m.groups.toast.level > lvl {
		lvl = m.groups.toast.level
	}
	if m.gh != nil && !m.gh.toast.empty() && m.gh.toast.level > lvl {
		lvl = m.gh.toast.level
	}
	return lvl
}

func (m *appModel) breadcrumb() string {
	switch m.screen {
	case screenHosts:
		return "Hosts"
	case screenGroups:
		return "Groups"
	case screenGroupHosts:
		if m.gh != nil {
			return "Groups > " + m.gh.group.Name
		}
		return "Groups"
	case screenDefaultsForm:
		return "Settings"
	default:
		return ""
	}
}

func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Spinner tick.
	if _, ok := msg.(spinnerTickMsg); ok {
		if spinnerActive {
			spinnerIndex++
			// Check if minimum duration has elapsed and work is done.
			if !spinnerMinEnd.IsZero() && time.Now().After(spinnerMinEnd) && !m.hosts.reloading {
				spinnerActive = false
				return m, nil
			}
			return m, tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg { return spinnerTickMsg{} })
		}
		return m, nil
	}

	// Auto-dismiss toasts.
	if tdm, ok := msg.(toastDismissMsg); ok {
		if tdm.token == m.toastToken {
			m.clearToasts()
		}
		return m, nil
	}

	prev := m.collectToastKey()
	result, cmd := m.doUpdate(msg)
	cur := m.collectToastKey()

	if cur != "" && cur != prev {
		m.toastToken++
		token := m.toastToken
		level := m.maxToastLevel()
		dismiss := tea.Tick(toastDuration(level), func(time.Time) tea.Msg {
			return toastDismissMsg{token: token}
		})
		return result, tea.Batch(cmd, dismiss)
	}
	return result, cmd
}

func (m *appModel) doUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.applyWindowSize(msg)
	case switchScreenMsg:
		m.screen = msg.to
		return m, nil
	case openGroupFormMsg:
		var g config.Group
		if msg.index >= 0 && msg.index < len(m.opts.Inventory.Groups) {
			g = m.opts.Inventory.Groups[msg.index]
		}
		m.form = newGroupFormModel(msg.index, g, m.opts.Config.Defaults, m.opts.Config.Defaults.ConfirmQuit)
		m.form.parentCrumb = "Groups"
		if m.width > 0 && m.height > 0 {
			mw, mh := groupFormModalSize(m.width, m.height)
			_, _ = m.form.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		m.screen = screenGroupForm
		return m, nil
	case openGroupFormPrefillMsg:
		m.form = newGroupFormModel(-1, msg.group, m.opts.Config.Defaults, m.opts.Config.Defaults.ConfirmQuit)
		m.form.parentCrumb = "Groups"
		if m.width > 0 && m.height > 0 {
			mw, mh := groupFormModalSize(m.width, m.height)
			_, _ = m.form.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		m.screen = screenGroupForm
		return m, nil
	case groupFormCancelMsg:
		m.form = nil
		m.screen = screenGroups
		return m, nil
	case groupFormSaveMsg:
		if err := m.saveGroup(msg.index, msg.group); err != nil {
			// Keep form open on error.
			m.form.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.groups.toast = toast{text: "saved", level: toastOK}
		m.form = nil
		m.screen = screenGroups
		return m, nil
	case deleteGroupMsg:
		if err := m.deleteGroup(msg.index); err != nil {
			m.groups.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.groups.toast = toast{text: "deleted", level: toastOK}
		return m, nil
	case openGroupHostsMsg:
		if msg.index < 0 || msg.index >= len(m.opts.Inventory.Groups) {
			m.groups.toast = toast{text: "invalid group", level: toastErr}
			return m, nil
		}
		m.gh = newGroupHostsModel(m.opts, msg.index)
		if m.width > 0 && m.height > 0 {
			_, _ = m.gh.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		m.screen = screenGroupHosts
		return m, nil
	case openHostPickerMsg:
		m.picker = newHostPickerModel(m.opts, msg.groupIndex)
		m.picker.parentCrumb = m.breadcrumb()
		if m.width > 0 && m.height > 0 {
			mw, mh := pickerModalSize(m.width, m.height)
			_, _ = m.picker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		m.returnTo = msg.returnTo
		m.returnGroupIndex = msg.groupIndex
		m.screen = screenHostPicker
		return m, nil
	case openGroupPickerMsg:
		m.gpHosts = append([]string(nil), msg.hosts...)
		m.gp = newGroupPickerModel(m.opts)
		m.gp.hosts = append([]string(nil), m.gpHosts...)
		m.gp.parentCrumb = m.breadcrumb()
		m.gpReturnTo = screenHosts
		m.gpConnectAfterAdd = false
		if m.width > 0 && m.height > 0 {
			mw, mh := pickerModalSize(m.width, m.height)
			_, _ = m.gp.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		m.screen = screenGroupPicker
		return m, nil
	case openDefaultsFormMsg:
		m.defaultsForm = newDefaultsFormModel(m.opts.Config.Defaults, m.opts.Config.Defaults.ConfirmQuit)
		m.defaultsReturnTo = msg.returnTo
		if m.width > 0 && m.height > 0 {
			_, _ = m.defaultsForm.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		m.screen = screenDefaultsForm
		return m, nil
	case defaultsFormCancelMsg:
		m.defaultsForm = nil
		m.screen = m.defaultsReturnTo
		return m, nil
	case defaultsFormSaveMsg:
		if err := m.saveDefaults(msg.defaults); err != nil {
			m.defaultsForm.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.screen = screenDefaultsForm
		if m.defaultsForm != nil {
			m.defaultsForm.defaults = m.opts.Config.Defaults
			m.defaultsForm.toast = toast{text: "saved", level: toastOK}
			m.defaultsToastToken++
			token := m.defaultsToastToken
			return m, tea.Tick(5*time.Second, func(time.Time) tea.Msg { return defaultsToastExpireMsg{token: token} })
		}
		return m, nil
	case defaultsToastExpireMsg:
		if msg.token != m.defaultsToastToken {
			return m, nil
		}
		// Match on text literal; if the "saved" message ever changes this
		// timer will stop clearing the toast. A sentinel level or token on
		// defaultsFormModel would be cleaner.
		if m.defaultsForm != nil && m.defaultsForm.toast.text == "saved" {
			m.defaultsForm.toast = toast{}
		}
		return m, nil
	case openCustomHostMsg:
		m.customHost = newCustomHostModel(m.opts, msg.groupIndex, msg.returnTo)
		// Build breadcrumb: for screenGroups we need to include the specific
		// group name since m.breadcrumb() only returns "Groups" at that level.
		// For screenGroupHosts, m.breadcrumb() already returns "Groups > name".
		// Other screens don't carry a meaningful groupIndex, so fall through.
		crumb := m.breadcrumb()
		if msg.groupIndex >= 0 && msg.groupIndex < len(m.opts.Inventory.Groups) {
			if m.screen == screenGroups {
				crumb = "Groups > " + m.opts.Inventory.Groups[msg.groupIndex].Name
			}
		}
		m.customHost.parentCrumb = crumb
		if m.width > 0 && m.height > 0 {
			mw, mh := customHostModalSize(m.width, m.height)
			_, _ = m.customHost.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		m.screen = screenCustomHost
		return m, nil
	case openHostFormMsg:
		idx, hc := findHostConfig(m.opts.Inventory, msg.host)
		m.hostForm = newHostFormModel(idx, hc, m.opts.Config.Defaults, m.opts.Config.Defaults.ConfirmQuit)
		m.hostForm.parentCrumb = m.breadcrumb()
		m.hostFormReturnTo = msg.returnTo
		if m.width > 0 && m.height > 0 {
			mw, mh := hostFormModalSize(m.width, m.height)
			_, _ = m.hostForm.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		m.screen = screenHostForm
		return m, nil
	case openHostFormPrefillMsg:
		m.hostForm = newHostFormModel(-1, msg.host, m.opts.Config.Defaults, m.opts.Config.Defaults.ConfirmQuit)
		m.hostForm.parentCrumb = m.breadcrumb()
		m.hostFormReturnTo = msg.returnTo
		if m.width > 0 && m.height > 0 {
			mw, mh := hostFormModalSize(m.width, m.height)
			_, _ = m.hostForm.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		m.screen = screenHostForm
		return m, nil
	case customHostCancelMsg:
		ret := screenHosts
		if m.customHost != nil {
			ret = m.customHost.returnTo
		}
		m.customHost = nil
		m.screen = ret
		return m, nil
	case customHostPickGroupMsg:
		m.gpHosts = append([]string(nil), msg.hosts...)
		m.gp = newGroupPickerModel(m.opts)
		m.gp.hosts = append([]string(nil), m.gpHosts...)
		m.gp.parentCrumb = m.breadcrumb()
		m.gpReturnTo = msg.returnTo
		m.gpConnectAfterAdd = false
		if m.width > 0 && m.height > 0 {
			mw, mh := pickerModalSize(m.width, m.height)
			_, _ = m.gp.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
		}
		m.customHost = nil
		m.screen = screenGroupPicker
		return m, nil
	case customHostConnectMsg:
		var execCmd []string
		var toastResult toast
		var err error
		if msg.groupIndex >= 0 {
			execCmd, toastResult, err = m.connectHostsForGroup(msg.groupIndex, msg.hosts, "")
		} else {
			execCmd, toastResult, err = m.connectHostsWithDefaults(msg.hosts)
		}
		if err != nil {
			toastResult = toast{text: err.Error(), level: toastErr}
		}
		if !toastResult.empty() {
			switch msg.returnTo {
			case screenGroups:
				m.groups.toast = toastResult
			case screenGroupHosts:
				if m.gh != nil {
					m.gh.toast = toastResult
				}
			default:
				m.hosts.toast = toastResult
			}
		}
		m.customHost = nil
		m.screen = msg.returnTo
		if len(execCmd) != 0 {
			m.execCmd = execCmd
			return m, tea.Quit
		}
		if m.opts.Popup && !toastResult.empty() && toastResult.level != toastErr {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case customHostDoneMsg:
		if err := m.addHostsToGroup(msg.groupIndex, msg.hosts); err != nil {
			if m.customHost != nil {
				m.customHost.toast = toast{text: err.Error(), level: toastErr}
			}
			return m, nil
		}
		if msg.returnTo == screenGroupHosts {
			m.gh = newGroupHostsModel(m.opts, msg.groupIndex)
			if m.width > 0 && m.height > 0 {
				_, _ = m.gh.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			}
		}
		addedToast := toast{text: fmt.Sprintf("added %d", len(msg.hosts)), level: toastOK}
		switch msg.returnTo {
		case screenGroups:
			m.groups.toast = addedToast
		case screenGroupHosts:
			if m.gh != nil {
				m.gh.toast = addedToast
			}
		default:
			m.hosts.toast = addedToast
		}
		m.customHost = nil
		m.screen = msg.returnTo
		return m, nil
	case groupPickerCancelMsg:
		m.gp = nil
		m.gpHosts = nil
		m.gpConnectAfterAdd = false
		m.screen = m.gpReturnTo
		return m, nil
	case groupPickerDoneMsg:
		if err := m.addHostsToGroup(msg.groupIndex, m.gpHosts); err != nil {
			m.gp.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		if m.gpConnectAfterAdd {
			execCmd, toastResult, err := m.connectHostsForGroup(msg.groupIndex, m.gpHosts, "")
			if err != nil {
				toastResult = toast{text: err.Error(), level: toastErr}
			}
			if !toastResult.empty() {
				switch m.gpReturnTo {
				case screenGroups:
					m.groups.toast = toastResult
				default:
					m.hosts.toast = toastResult
				}
			}
			m.gp = nil
			m.gpHosts = nil
			m.gpConnectAfterAdd = false
			m.screen = m.gpReturnTo
			if len(execCmd) != 0 {
				m.execCmd = execCmd
				return m, tea.Quit
			}
			if m.opts.Popup && !toastResult.empty() && toastResult.level != toastErr {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
		addedToast := toast{text: fmt.Sprintf("added %d", len(m.gpHosts)), level: toastOK}
		switch m.gpReturnTo {
		case screenGroups:
			m.groups.toast = addedToast
		case screenGroupHosts:
			if m.gh != nil {
				m.gh.toast = addedToast
			}
		default:
			m.hosts.toast = addedToast
		}
		m.gp = nil
		m.gpHosts = nil
		m.screen = m.gpReturnTo
		return m, nil
	case hostPickerCancelMsg:
		m.picker = nil
		m.screen = m.returnTo
		return m, nil
	case hostPickerDoneMsg:
		if err := m.addHostsToGroup(m.returnGroupIndex, msg.hosts); err != nil {
			m.picker.toast = toast{text: err.Error(), level: toastErr}
			return m, nil
		}
		m.groups.toast = toast{text: fmt.Sprintf("added %d", len(msg.hosts)), level: toastOK}
		m.picker = nil
		m.screen = m.returnTo
		if m.screen == screenGroupHosts {
			m.gh = newGroupHostsModel(m.opts, m.returnGroupIndex)
			if m.width > 0 && m.height > 0 {
				_, _ = m.gh.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			}
		}
		return m, nil
	case removeHostsMsg:
		if err := m.removeHostsFromGroup(msg.groupIndex, msg.hosts); err != nil {
			if m.screen == screenGroupHosts && m.gh != nil {
				m.gh.toast = toast{text: err.Error(), level: toastErr}
			}
			return m, nil
		}
		if m.screen == screenGroupHosts && m.gh != nil {
			m.gh.toast = toast{text: fmt.Sprintf("removed %d", len(msg.hosts)), level: toastOK}
		}
		if m.screen == screenGroupHosts {
			m.gh = newGroupHostsModel(m.opts, msg.groupIndex)
			if m.width > 0 && m.height > 0 {
				_, _ = m.gh.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			}
		}
		return m, nil
	case hostFormCancelMsg:
		m.hostForm = nil
		m.screen = m.hostFormReturnTo
		return m, nil
	case hostFormSaveMsg:
		hadForm := m.hostForm != nil
		if err := m.saveHostConfig(msg.index, msg.host); err != nil {
			if m.hostForm != nil {
				m.hostForm.toast = toast{text: err.Error(), level: toastErr}
			}
			return m, nil
		}
		if m.hosts != nil {
			m.hosts.refreshVisibleBadges()
			m.hosts.reapplyFilter()
		}
		if m.gh != nil {
			m.gh.refreshVisibleBadges()
		}
		if m.picker != nil {
			m.picker.refreshVisibleBadges()
		}
		savedToast := toast{text: "saved", level: toastOK}
		if hadForm {
			switch m.hostFormReturnTo {
			case screenGroups:
				m.groups.toast = savedToast
			case screenGroupHosts:
				if m.gh != nil {
					m.gh.toast = savedToast
				}
			default:
				m.hosts.toast = savedToast
			}
			m.hostForm = nil
			m.screen = m.hostFormReturnTo
		} else {
			m.hosts.toast = savedToast
		}
		return m, nil
	case toggleHiddenHostMsg:
		if err := m.saveToggleHidden(msg.host, msg.hide); err != nil {
			if m.hosts != nil {
				m.hosts.toast = toast{text: err.Error(), level: toastErr}
			}
			return m, nil
		}
		if m.hosts != nil {
			m.hosts.reapplyFilter()
			m.hosts.toast = toast{text: "saved", level: toastOK}
		}
		return m, nil
	}

	switch m.screen {
	case screenHosts:
		model, cmd := m.hosts.Update(msg)
		if hm, ok := model.(*hostsModel); ok {
			m.hosts = hm
			if len(hm.execCmd) != 0 {
				m.execCmd = hm.execCmd
				return m, tea.Quit
			}
			if hm.quitting {
				m.quitting = true
				return m, tea.Quit
			}
		}
		return m, cmd
	case screenGroups:
		model, cmd := m.groups.Update(msg)
		if gm, ok := model.(*groupsModel); ok {
			m.groups = gm
			if len(gm.execCmd) != 0 {
				m.execCmd = gm.execCmd
				return m, tea.Quit
			}
			if gm.quitting {
				m.quitting = true
				return m, tea.Quit
			}
		}
		return m, cmd
	case screenGroupForm:
		model, cmd := m.form.Update(msg)
		if fm, ok := model.(*groupFormModel); ok {
			m.form = fm
		}
		return m, cmd
	case screenGroupHosts:
		model, cmd := m.gh.Update(msg)
		if gh, ok := model.(*groupHostsModel); ok {
			m.gh = gh
			if len(gh.execCmd) != 0 {
				m.execCmd = gh.execCmd
				return m, tea.Quit
			}
			if gh.quitting {
				m.quitting = true
				return m, tea.Quit
			}
		}
		return m, cmd
	case screenHostPicker:
		model, cmd := m.picker.Update(msg)
		if pm, ok := model.(*hostPickerModel); ok {
			m.picker = pm
		}
		return m, cmd
	case screenGroupPicker:
		model, cmd := m.gp.Update(msg)
		if gm, ok := model.(*groupPickerModel); ok {
			m.gp = gm
		}
		return m, cmd
	case screenDefaultsForm:
		model, cmd := m.defaultsForm.Update(msg)
		if dm, ok := model.(*defaultsFormModel); ok {
			m.defaultsForm = dm
		}
		return m, cmd
	case screenCustomHost:
		model, cmd := m.customHost.Update(msg)
		if cm, ok := model.(*customHostModel); ok {
			m.customHost = cm
		}
		return m, cmd
	case screenHostForm:
		model, cmd := m.hostForm.Update(msg)
		if hm, ok := model.(*hostFormModel); ok {
			m.hostForm = hm
		}
		return m, cmd
	default:
		return m, nil
	}
}

func (m *appModel) View() string {
	switch m.screen {
	case screenGroups:
		return m.groups.View()
	case screenGroupForm:
		return placeCentered(m.width, m.height, m.form.View())
	case screenGroupHosts:
		return m.gh.View()
	case screenHostPicker:
		return placeCentered(m.width, m.height, m.picker.View())
	case screenGroupPicker:
		return placeCentered(m.width, m.height, m.gp.View())
	case screenDefaultsForm:
		return m.defaultsForm.View()
	case screenCustomHost:
		return placeCentered(m.width, m.height, m.customHost.View())
	case screenHostForm:
		return placeCentered(m.width, m.height, m.hostForm.View())
	default:
		return m.hosts.View()
	}
}

func (m *appModel) saveGroup(index int, g config.Group) error {
	if strings.TrimSpace(g.Name) == "" {
		return fmt.Errorf("group name required")
	}
	g.Name = strings.TrimSpace(g.Name)

	// Unique name check.
	for i := range m.opts.Inventory.Groups {
		if i == index {
			continue
		}
		if strings.TrimSpace(m.opts.Inventory.Groups[i].Name) == g.Name {
			return fmt.Errorf("group name already exists")
		}
	}

	newInv := m.opts.Inventory
	if index < 0 {
		newInv.Groups = append(newInv.Groups, g)
	} else {
		if index >= len(newInv.Groups) {
			return fmt.Errorf("invalid group index")
		}
		newInv.Groups[index] = g
	}

	if _, err := config.SaveInventory(m.opts.InventoryPath, newInv); err != nil {
		return err
	}

	m.opts.Inventory = newInv
	// Propagate to screens.
	m.hosts.opts.Inventory = newInv
	m.groups.Refresh(newInv)
	return nil
}

func (m *appModel) saveDefaults(d config.Defaults) error {
	oldLoadKnownHosts := m.opts.Config.Defaults.LoadKnownHosts
	oldKnownPaths := append([]string(nil), m.opts.KnownHosts...)

	newCfg := m.opts.Config
	newCfg.Defaults = d
	if _, err := config.Save(m.opts.ConfigPath, newCfg); err != nil {
		return err
	}

	m.opts.Config = newCfg
	SetAccentColor(newCfg.Defaults.AccentColor)
	m.refreshAccentStyles()

	// Update hosts source if load_known_hosts toggled.
	if oldLoadKnownHosts != newCfg.Defaults.LoadKnownHosts {
		if newCfg.Defaults.LoadKnownHosts {
			if len(oldKnownPaths) != 0 {
				m.opts.KnownHosts = oldKnownPaths
			}
			if len(m.opts.KnownHosts) == 0 {
				m.opts.KnownHosts = hosts.DefaultKnownHostsPaths()
			}
			res, errs := hosts.LoadKnownHosts(m.opts.KnownHosts)
			m.opts.Hosts = res.Hosts
			m.opts.SkippedLines = res.SkippedLines
			m.opts.LoadErrors = errs
		} else {
			m.opts.KnownHosts = nil
			m.opts.Hosts = config.ConfigHosts(m.opts.Inventory)
			m.opts.SkippedLines = 0
			m.opts.LoadErrors = nil
		}

		if m.hosts != nil {
			m.hosts.keymap.Reload.SetEnabled(newCfg.Defaults.LoadKnownHosts)
			m.hosts.opts = m.opts
			_, _ = m.hosts.Update(knownHostsReloadMsg{res: hosts.LoadResult{Hosts: m.opts.Hosts, SkippedLines: m.opts.SkippedLines}, errs: m.opts.LoadErrors})
		}
		if m.picker != nil {
			// Recreate to refresh list source.
			m.picker = newHostPickerModel(m.opts, m.returnGroupIndex)
			if m.width > 0 && m.height > 0 {
				mw, mh := pickerModalSize(m.width, m.height)
				_, _ = m.picker.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
			}
		}
	}
	if m.hosts != nil {
		m.hosts.opts = m.opts
		m.hosts.refreshVisibleBadges()
	}
	if m.groups != nil {
		m.groups.opts = m.opts
		m.groups.Refresh(m.opts.Inventory)
	}
	if m.gh != nil {
		m.gh.opts = m.opts
		m.gh.refreshVisibleBadges()
	}
	if m.picker != nil {
		m.picker.opts = m.opts
		m.picker.refreshVisibleBadges()
	}
	if m.gp != nil {
		m.gp.opts = m.opts
	}
	return nil
}

func (m *appModel) refreshAccentStyles() {
	if m.hosts != nil {
		setSearchBarFocused(&m.hosts.search, m.hosts.focus == focusSearch)
		if m.hosts.cmdPrompt {
			setSearchFocused(&m.hosts.cmdInput, true)
		}
	}
	if m.groups != nil {
		setSearchBarFocused(&m.groups.search, m.groups.focus == focusSearch)
		if m.groups.cmdPrompt {
			setSearchFocused(&m.groups.cmdInput, true)
		}
	}
	if m.gh != nil {
		setSearchBarFocused(&m.gh.search, m.gh.focus == focusSearch)
		if m.gh.cmdPrompt {
			setSearchFocused(&m.gh.cmdInput, true)
		}
	}
	if m.picker != nil {
		setSearchBarFocused(&m.picker.search, m.picker.focus == focusSearch)
	}
	if m.gp != nil {
		setSearchBarFocused(&m.gp.search, m.gp.focus == focusSearch)
	}
	if m.customHost != nil {
		setSearchFocused(&m.customHost.input, true)
	}
	if m.defaultsForm != nil {
		m.defaultsForm.refreshAccentStyles()
	}
	if m.form != nil {
		m.form.refreshAccentStyles()
	}
	if m.hostForm != nil {
		m.hostForm.refreshAccentStyles()
	}
}

func (m *appModel) saveHostConfig(index int, h config.Host) error {
	if strings.TrimSpace(h.Host) == "" {
		return fmt.Errorf("host required")
	}
	h.Host = strings.TrimSpace(h.Host)

	// Unique host check.
	for i := range m.opts.Inventory.Hosts {
		if i == index {
			continue
		}
		if strings.TrimSpace(m.opts.Inventory.Hosts[i].Host) == h.Host {
			return fmt.Errorf("host config already exists")
		}
	}

	newInv := m.opts.Inventory
	newInv.Hosts = append([]config.Host(nil), newInv.Hosts...)
	if index < 0 {
		newInv.Hosts = append(newInv.Hosts, h)
	} else {
		if index >= len(newInv.Hosts) {
			return fmt.Errorf("invalid host index")
		}
		newInv.Hosts[index] = h
	}

	if _, err := config.SaveInventory(m.opts.InventoryPath, newInv); err != nil {
		return err
	}

	m.opts.Inventory = newInv
	if m.hosts != nil {
		m.hosts.opts.Inventory = newInv
	}
	if m.groups != nil {
		m.groups.Refresh(newInv)
	}
	if m.gh != nil {
		m.gh.opts.Inventory = newInv
	}
	if m.picker != nil {
		m.picker.opts.Inventory = newInv
	}
	if m.gp != nil {
		m.gp.opts.Inventory = newInv
	}
	if m.customHost != nil {
		m.customHost.opts.Inventory = newInv
	}
	return nil
}

func (m *appModel) deleteGroup(index int) error {
	if index < 0 || index >= len(m.opts.Inventory.Groups) {
		return fmt.Errorf("invalid group index")
	}

	newInv := m.opts.Inventory
	newInv.Groups = append([]config.Group(nil), newInv.Groups...)
	newInv.Groups = append(newInv.Groups[:index], newInv.Groups[index+1:]...)

	if _, err := config.SaveInventory(m.opts.InventoryPath, newInv); err != nil {
		return err
	}

	m.opts.Inventory = newInv
	m.hosts.opts.Inventory = newInv
	m.groups.Refresh(newInv)
	return nil
}

func (m *appModel) addHostsToGroup(groupIndex int, hostsToAdd []string) error {
	if groupIndex < 0 || groupIndex >= len(m.opts.Inventory.Groups) {
		return fmt.Errorf("invalid group index")
	}
	if len(hostsToAdd) == 0 {
		return nil
	}

	newInv := m.opts.Inventory
	newInv.Groups = append([]config.Group(nil), newInv.Groups...)
	g := newInv.Groups[groupIndex]

	set := make(map[string]bool, len(g.Hosts))
	for _, h := range g.Hosts {
		set[h] = true
	}
	for _, h := range hostsToAdd {
		if h == "" {
			continue
		}
		if set[h] {
			continue
		}
		g.Hosts = append(g.Hosts, h)
		set[h] = true
	}

	newInv.Groups[groupIndex] = g
	if _, err := config.SaveInventory(m.opts.InventoryPath, newInv); err != nil {
		return err
	}

	m.opts.Inventory = newInv
	m.hosts.opts.Inventory = newInv
	m.groups.Refresh(newInv)
	return nil
}

func (m *appModel) removeHostsFromGroup(groupIndex int, hostsToRemove []string) error {
	if groupIndex < 0 || groupIndex >= len(m.opts.Inventory.Groups) {
		return fmt.Errorf("invalid group index")
	}
	if len(hostsToRemove) == 0 {
		return nil
	}

	removeSet := make(map[string]bool, len(hostsToRemove))
	for _, h := range hostsToRemove {
		removeSet[h] = true
	}

	newInv := m.opts.Inventory
	newInv.Groups = append([]config.Group(nil), newInv.Groups...)
	g := newInv.Groups[groupIndex]
	kept := make([]string, 0, len(g.Hosts))
	for _, h := range g.Hosts {
		if removeSet[h] {
			continue
		}
		kept = append(kept, h)
	}
	g.Hosts = kept
	newInv.Groups[groupIndex] = g

	if _, err := config.SaveInventory(m.opts.InventoryPath, newInv); err != nil {
		return err
	}

	m.opts.Inventory = newInv
	m.hosts.opts.Inventory = newInv
	m.groups.Refresh(newInv)
	return nil
}

func (m *appModel) saveToggleHidden(host string, hide bool) error {
	newInv := m.opts.Inventory
	h := strings.TrimSpace(host)

	// Work on independent copies of mutable slices.
	newInv.HiddenHosts = append([]string(nil), newInv.HiddenHosts...)
	newInv.Hosts = append([]config.Host(nil), newInv.Hosts...)

	idx, hc := findHostConfig(newInv, host)

	if hide {
		if idx >= 0 {
			// Host has existing [[hosts]] entry — set Hidden there.
			hc.Hidden = true
			newInv.Hosts[idx] = hc
		} else {
			// No per-host config — use compact list.
			present := false
			for _, hh := range newInv.HiddenHosts {
				if strings.TrimSpace(hh) == h {
					present = true
					break
				}
			}
			if !present {
				newInv.HiddenHosts = append(newInv.HiddenHosts, h)
			}
		}
	} else {
		// Remove from compact list (safe even if not present).
		out := newInv.HiddenHosts[:0]
		for _, hh := range newInv.HiddenHosts {
			if strings.TrimSpace(hh) != h {
				out = append(out, hh)
			}
		}
		newInv.HiddenHosts = out
		// Also clear Hidden flag in [[hosts]] entry if set (handles old-format configs).
		if idx >= 0 && hc.Hidden {
			hc.Hidden = false
			newInv.Hosts[idx] = hc
		}
	}

	if _, err := config.SaveInventory(m.opts.InventoryPath, newInv); err != nil {
		return err
	}

	m.opts.Inventory = newInv
	if m.hosts != nil {
		m.hosts.opts.Inventory = newInv
	}
	if m.groups != nil {
		m.groups.Refresh(newInv)
	}
	if m.gh != nil {
		m.gh.opts.Inventory = newInv
	}
	if m.picker != nil {
		m.picker.opts.Inventory = newInv
	}
	if m.gp != nil {
		m.gp.opts.Inventory = newInv
	}
	if m.customHost != nil {
		m.customHost.opts.Inventory = newInv
	}
	return nil
}

func (m *appModel) IsQuitting() bool { return m.quitting }
func (m *appModel) ExecCmd() []string {
	return m.execCmd
}
