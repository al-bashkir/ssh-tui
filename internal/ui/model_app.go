package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/connect"
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

// openGroupFormMsg opens the group form. A valid index edits that group;
// otherwise a new group is created, prefilled from group.
type openGroupFormMsg struct {
	index int
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
	hosts    []string
	returnTo screen
}

type openDefaultsFormMsg struct {
	returnTo screen
}

type openCustomHostMsg struct {
	returnTo   screen
	groupIndex int // -1 means no group context; Ctrl+a will pick a group
}

// openHostFormMsg opens the host form for host's config, or, when prefill
// is set, a new host config prefilled from it.
type openHostFormMsg struct {
	host     string
	prefill  *config.Host
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

type toastDismissMsg struct {
	token int
}

type appModel struct {
	opts Options

	width  int
	height int

	screen           screen
	hosts            *hostsModel
	groups           *groupsModel
	form             *groupFormModel
	gh               *groupHostsModel
	picker           *hostPickerModel
	gp               *groupPickerModel
	defaultsForm     *defaultsFormModel
	customHost       *customHostModel
	hostForm         *hostFormModel
	gpReturnTo       screen
	returnTo         screen
	returnGroupIndex int
	defaultsReturnTo screen
	hostFormReturnTo screen
	toastToken       int

	quitting bool
	execCmd  []string
}

func newAppModel(opts Options) *appModel {
	return &appModel{
		opts:   opts,
		screen: screenHosts,
		hosts:  newHostsModel(opts),
		groups: newGroupsModel(opts),
	}
}

func (m *appModel) Init() tea.Cmd {
	return m.hosts.Init()
}

// sizeTo sends the current window size to model, shrunk by sizeFn for
// modals. It does nothing before the first WindowSizeMsg.
func (m *appModel) sizeTo(model tea.Model, sizeFn func(w, h int) (int, int)) {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	w, h := m.width, m.height
	if sizeFn != nil {
		w, h = sizeFn(w, h)
	}
	_, _ = model.Update(tea.WindowSizeMsg{Width: w, Height: h})
}

func (m *appModel) applyWindowSize(ws tea.WindowSizeMsg) {
	m.width = ws.Width
	m.height = ws.Height

	m.sizeTo(m.hosts, nil)
	m.sizeTo(m.groups, nil)
	if m.form != nil {
		m.sizeTo(m.form, groupFormModalSize)
	}
	if m.gh != nil {
		m.sizeTo(m.gh, nil)
	}
	if m.picker != nil {
		m.sizeTo(m.picker, pickerModalSize)
	}
	if m.gp != nil {
		m.sizeTo(m.gp, pickerModalSize)
	}
	if m.defaultsForm != nil {
		m.sizeTo(m.defaultsForm, nil)
	}
	if m.customHost != nil {
		m.sizeTo(m.customHost, customHostModalSize)
	}
	if m.hostForm != nil {
		m.sizeTo(m.hostForm, hostFormModalSize)
	}
}

// toasts returns the auto-dismissed toasts of the open screens.
func (m *appModel) toasts() []*toast {
	ts := []*toast{&m.hosts.toast, &m.groups.toast}
	if m.gh != nil {
		ts = append(ts, &m.gh.toast)
	}
	// The discard prompt lives only in the settings toast; keep it up.
	if m.defaultsForm != nil && !m.defaultsForm.confirmDiscard {
		ts = append(ts, &m.defaultsForm.toast)
	}
	return ts
}

func (m *appModel) collectToastKey() string {
	var parts []string
	for _, t := range m.toasts() {
		if !t.empty() {
			parts = append(parts, t.text)
		}
	}
	return strings.Join(parts, "|")
}

func (m *appModel) maxToastLevel() toastLevel {
	var lvl toastLevel
	for _, t := range m.toasts() {
		if !t.empty() {
			lvl = max(lvl, t.level)
		}
	}
	return lvl
}

// toastOn shows t on the list screen s (Hosts for any other screen).
func (m *appModel) toastOn(s screen, t toast) {
	switch s {
	case screenGroups:
		m.groups.toast = t
	case screenGroupHosts:
		if m.gh != nil {
			m.gh.toast = t
		}
	default:
		m.hosts.toast = t
	}
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
			for _, t := range m.toasts() {
				*t = toast{}
			}
		}
		return m, nil
	}

	prev := m.collectToastKey()
	cmd := m.doUpdate(msg)
	cur := m.collectToastKey()

	if cur != "" && cur != prev {
		m.toastToken++
		token := m.toastToken
		dismiss := tea.Tick(toastDuration(m.maxToastLevel()), func(time.Time) tea.Msg {
			return toastDismissMsg{token: token}
		})
		return m, tea.Batch(cmd, dismiss)
	}
	return m, cmd
}

func (m *appModel) doUpdate(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.applyWindowSize(msg)
		return nil
	case switchScreenMsg:
		m.screen = msg.to
		return nil
	case openGroupFormMsg:
		g := msg.group
		if msg.index >= 0 && msg.index < len(m.opts.Inventory.Groups) {
			g = m.opts.Inventory.Groups[msg.index]
		}
		m.form = newGroupFormModel(msg.index, g, m.opts.Config.Defaults)
		m.form.parentCrumb = "Groups"
		m.sizeTo(m.form, groupFormModalSize)
		m.screen = screenGroupForm
		return nil
	case groupFormCancelMsg:
		m.form = nil
		m.screen = screenGroups
		return nil
	case groupFormSaveMsg:
		if err := m.saveGroup(msg.index, msg.group); err != nil {
			// Keep form open on error.
			m.form.toast = toast{text: err.Error(), level: toastErr}
			return nil
		}
		m.groups.toast = toast{text: "saved", level: toastOK}
		m.form = nil
		m.screen = screenGroups
		return nil
	case deleteGroupMsg:
		if err := m.deleteGroup(msg.index); err != nil {
			m.groups.toast = toast{text: err.Error(), level: toastErr}
			return nil
		}
		m.groups.toast = toast{text: "deleted", level: toastOK}
		return nil
	case openGroupHostsMsg:
		if msg.index < 0 || msg.index >= len(m.opts.Inventory.Groups) {
			m.groups.toast = toast{text: "invalid group", level: toastErr}
			return nil
		}
		m.reloadGroupHosts(msg.index)
		m.screen = screenGroupHosts
		return nil
	case openHostPickerMsg:
		m.picker = newHostPickerModel(m.opts)
		m.picker.parentCrumb = m.breadcrumb()
		m.sizeTo(m.picker, pickerModalSize)
		m.returnTo = msg.returnTo
		m.returnGroupIndex = msg.groupIndex
		m.screen = screenHostPicker
		return nil
	case openGroupPickerMsg:
		m.gp = newGroupPickerModel(m.opts)
		m.gp.hosts = append([]string(nil), msg.hosts...)
		m.gp.parentCrumb = m.breadcrumb()
		m.gpReturnTo = msg.returnTo
		m.sizeTo(m.gp, pickerModalSize)
		m.customHost = nil
		m.screen = screenGroupPicker
		return nil
	case openDefaultsFormMsg:
		m.defaultsForm = newDefaultsFormModel(m.opts.Config.Defaults)
		m.defaultsReturnTo = msg.returnTo
		m.sizeTo(m.defaultsForm, nil)
		m.screen = screenDefaultsForm
		return nil
	case defaultsFormCancelMsg:
		m.defaultsForm = nil
		m.screen = m.defaultsReturnTo
		return nil
	case defaultsFormSaveMsg:
		if err := m.saveDefaults(msg.defaults); err != nil {
			m.defaultsForm.toast = toast{text: err.Error(), level: toastErr}
			return nil
		}
		m.screen = screenDefaultsForm
		if m.defaultsForm != nil {
			m.defaultsForm.defaults = m.opts.Config.Defaults
			m.defaultsForm.form.resetOriginals()
			m.defaultsForm.toast = toast{text: "saved", level: toastOK}
		}
		return nil
	case openCustomHostMsg:
		m.customHost = newCustomHostModel(m.opts, msg.groupIndex, msg.returnTo)
		// m.breadcrumb() only returns "Groups" on the Groups screen, so name
		// the target group there. On Group Hosts it already includes it.
		crumb := m.breadcrumb()
		if m.screen == screenGroups && msg.groupIndex >= 0 && msg.groupIndex < len(m.opts.Inventory.Groups) {
			crumb = "Groups > " + m.opts.Inventory.Groups[msg.groupIndex].Name
		}
		m.customHost.parentCrumb = crumb
		m.sizeTo(m.customHost, customHostModalSize)
		m.screen = screenCustomHost
		return nil
	case openHostFormMsg:
		var idx int
		var hc config.Host
		if msg.prefill != nil {
			idx, hc = -1, *msg.prefill
		} else {
			idx, hc = findHostConfig(m.opts.Inventory, msg.host)
		}
		m.hostForm = newHostFormModel(idx, hc, m.opts.Config.Defaults)
		m.hostForm.parentCrumb = m.breadcrumb()
		m.hostFormReturnTo = msg.returnTo
		m.sizeTo(m.hostForm, hostFormModalSize)
		m.screen = screenHostForm
		return nil
	case customHostCancelMsg:
		ret := screenHosts
		if m.customHost != nil {
			ret = m.customHost.returnTo
		}
		m.customHost = nil
		m.screen = ret
		return nil
	case customHostConnectMsg:
		var group *config.Group
		var execCmd []string
		var toastResult toast
		var err error
		if msg.groupIndex >= len(m.opts.Inventory.Groups) {
			err = fmt.Errorf("invalid group")
		} else {
			if msg.groupIndex >= 0 {
				g := m.opts.Inventory.Groups[msg.groupIndex]
				group = &g
			}
			execCmd, toastResult, err = m.connectHosts(msg.hosts, group)
		}
		if err != nil {
			toastResult = toast{text: err.Error(), level: toastErr}
		}
		if !toastResult.empty() {
			m.toastOn(msg.returnTo, toastResult)
		}
		m.customHost = nil
		m.screen = msg.returnTo
		if len(execCmd) != 0 {
			m.execCmd = execCmd
			return tea.Quit
		}
		if m.opts.Popup && !toastResult.empty() && toastResult.level != toastErr {
			m.quitting = true
			return tea.Quit
		}
		return nil
	case customHostDoneMsg:
		if err := m.addHostsToGroup(msg.groupIndex, msg.hosts); err != nil {
			if m.customHost != nil {
				m.customHost.toast = toast{text: err.Error(), level: toastErr}
			}
			return nil
		}
		if msg.returnTo == screenGroupHosts {
			m.reloadGroupHosts(msg.groupIndex)
		}
		m.toastOn(msg.returnTo, toast{text: fmt.Sprintf("added %d", len(msg.hosts)), level: toastOK})
		m.customHost = nil
		m.screen = msg.returnTo
		return nil
	case groupPickerCancelMsg:
		m.gp = nil
		m.screen = m.gpReturnTo
		return nil
	case groupPickerDoneMsg:
		if err := m.addHostsToGroup(msg.groupIndex, m.gp.hosts); err != nil {
			m.gp.toast = toast{text: err.Error(), level: toastErr}
			return nil
		}
		m.toastOn(m.gpReturnTo, toast{text: fmt.Sprintf("added %d", len(m.gp.hosts)), level: toastOK})
		m.gp = nil
		m.screen = m.gpReturnTo
		return nil
	case hostPickerCancelMsg:
		m.picker = nil
		m.screen = m.returnTo
		return nil
	case hostPickerDoneMsg:
		if err := m.addHostsToGroup(m.returnGroupIndex, msg.hosts); err != nil {
			m.picker.toast = toast{text: err.Error(), level: toastErr}
			return nil
		}
		m.groups.toast = toast{text: fmt.Sprintf("added %d", len(msg.hosts)), level: toastOK}
		m.picker = nil
		m.screen = m.returnTo
		if m.screen == screenGroupHosts {
			m.reloadGroupHosts(m.returnGroupIndex)
		}
		return nil
	case removeHostsMsg:
		err := m.removeHostsFromGroup(msg.groupIndex, msg.hosts)
		if m.screen != screenGroupHosts {
			return nil
		}
		if err != nil {
			if m.gh != nil {
				m.gh.toast = toast{text: err.Error(), level: toastErr}
			}
			return nil
		}
		m.reloadGroupHosts(msg.groupIndex)
		m.gh.toast = toast{text: fmt.Sprintf("removed %d", len(msg.hosts)), level: toastOK}
		return nil
	case hostFormCancelMsg:
		m.hostForm = nil
		m.screen = m.hostFormReturnTo
		return nil
	case hostFormSaveMsg:
		if err := m.saveHostConfig(msg.index, msg.host); err != nil {
			if m.hostForm != nil {
				m.hostForm.toast = toast{text: err.Error(), level: toastErr}
			}
			return nil
		}
		m.hosts.refreshVisibleBadges(m.opts.Inventory)
		m.hosts.reapplyFilter()
		if m.gh != nil {
			m.gh.refreshVisibleBadges(m.opts.Inventory)
		}
		if m.picker != nil {
			m.picker.refreshVisibleBadges(m.opts.Inventory)
		}
		savedToast := toast{text: "saved", level: toastOK}
		if m.hostForm == nil {
			m.hosts.toast = savedToast
			return nil
		}
		m.toastOn(m.hostFormReturnTo, savedToast)
		m.hostForm = nil
		m.screen = m.hostFormReturnTo
		return nil
	case toggleHiddenHostMsg:
		if err := m.saveToggleHidden(msg.host, msg.hide); err != nil {
			m.hosts.toast = toast{text: err.Error(), level: toastErr}
			return nil
		}
		m.hosts.reapplyFilter()
		m.hosts.toast = toast{text: "saved", level: toastOK}
		return nil
	}

	model, _ := m.activeModel()
	_, cmd := model.Update(msg)
	switch m.screen {
	case screenHosts:
		return m.exitFrom(&m.hosts.connectActions, cmd)
	case screenGroups:
		return m.exitFrom(&m.groups.connectActions, cmd)
	case screenGroupHosts:
		return m.exitFrom(&m.gh.connectActions, cmd)
	}
	return cmd
}

// exitFrom propagates a list screen's exec or quit request to the app.
func (m *appModel) exitFrom(c *connectActions, cmd tea.Cmd) tea.Cmd {
	if len(c.execCmd) != 0 {
		m.execCmd = c.execCmd
		return tea.Quit
	}
	if c.quitting {
		m.quitting = true
		return tea.Quit
	}
	return cmd
}

// activeModel returns the current screen's model and whether it is a modal
// drawn centered over the terminal.
func (m *appModel) activeModel() (model tea.Model, modal bool) {
	switch m.screen {
	case screenGroups:
		return m.groups, false
	case screenGroupForm:
		return m.form, true
	case screenGroupHosts:
		return m.gh, false
	case screenHostPicker:
		return m.picker, true
	case screenGroupPicker:
		return m.gp, true
	case screenDefaultsForm:
		return m.defaultsForm, false
	case screenCustomHost:
		return m.customHost, true
	case screenHostForm:
		return m.hostForm, true
	}
	return m.hosts, false
}

func (m *appModel) View() string {
	model, modal := m.activeModel()
	s := model.View()
	if modal {
		s = placeCentered(m.width, m.height, s)
	}
	// Apply theme background to the entire screen output so every pixel —
	// including lipgloss.Place padding around modal dialogs — uses the
	// colorscheme background. withThemeBG is a no-op when no theme is active.
	return withThemeBG(s)
}

func (m *appModel) reloadGroupHosts(groupIndex int) {
	m.gh = newGroupHostsModel(m.opts, groupIndex)
	m.sizeTo(m.gh, nil)
}

// connectHosts builds and dispatches SSH commands synchronously, for flows
// driven by appModel rather than by a screen model.
func (m *appModel) connectHosts(hostsToOpen []string, group *config.Group) (execCmd []string, toastResult toast, err error) {
	if len(hostsToOpen) == 0 {
		return nil, toast{}, fmt.Errorf("no host selected")
	}

	defaults := m.opts.Config.Defaults
	sshCmds := connect.BuildCommands(hostsToOpen, defaults, m.opts.Inventory, group, nil)
	mode, inTmux := connect.Mode(defaults, group)

	execCmd, msg, err := connect.Open(hostsToOpen, sshCmds, defaults, group, mode, inTmux)
	if err != nil {
		return nil, toast{}, err
	}
	if msg != "" {
		return nil, toast{text: msg, level: toastInfo}, nil
	}
	return execCmd, toast{}, nil
}

// commitInventory saves inv and hands it to every open screen.
func (m *appModel) commitInventory(inv config.Inventory) error {
	if _, err := config.SaveInventory(m.opts.InventoryPath, inv); err != nil {
		return err
	}
	m.opts.Inventory = inv
	m.hosts.opts.Inventory = inv
	m.groups.Refresh(inv)
	if m.gh != nil {
		m.gh.opts.Inventory = inv
	}
	if m.picker != nil {
		m.picker.opts.Inventory = inv
	}
	if m.gp != nil {
		m.gp.opts.Inventory = inv
	}
	if m.customHost != nil {
		m.customHost.opts.Inventory = inv
	}
	return nil
}

// inventoryCopy returns the inventory with its slices copied, safe to edit
// before commitInventory.
func (m *appModel) inventoryCopy() config.Inventory {
	inv := m.opts.Inventory
	inv.HiddenHosts = slices.Clone(inv.HiddenHosts)
	inv.Hosts = slices.Clone(inv.Hosts)
	inv.Groups = slices.Clone(inv.Groups)
	return inv
}

func (m *appModel) saveGroup(index int, g config.Group) error {
	g.Name = strings.TrimSpace(g.Name)
	if g.Name == "" {
		return fmt.Errorf("group name required")
	}

	// Unique name check.
	for i := range m.opts.Inventory.Groups {
		if i != index && strings.TrimSpace(m.opts.Inventory.Groups[i].Name) == g.Name {
			return fmt.Errorf("group name already exists")
		}
	}

	inv := m.inventoryCopy()
	switch {
	case index < 0:
		inv.Groups = append(inv.Groups, g)
	case index < len(inv.Groups):
		inv.Groups[index] = g
	default:
		return fmt.Errorf("invalid group index")
	}
	return m.commitInventory(inv)
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
	ApplyColorScheme(newCfg.Defaults.Colorscheme, newCfg.Defaults.AccentColor)
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

		m.hosts.keymap.Reload.SetEnabled(newCfg.Defaults.LoadKnownHosts)
		m.hosts.opts = m.opts
		_, _ = m.hosts.Update(knownHostsReloadMsg{res: hosts.LoadResult{Hosts: m.opts.Hosts, SkippedLines: m.opts.SkippedLines}, errs: m.opts.LoadErrors})
		if m.picker != nil {
			// Recreate to refresh list source.
			m.picker = newHostPickerModel(m.opts)
			m.sizeTo(m.picker, pickerModalSize)
		}
	}
	m.hosts.opts = m.opts
	m.hosts.refreshVisibleBadges(m.opts.Inventory)
	m.groups.opts = m.opts
	m.groups.Refresh(m.opts.Inventory)
	if m.gh != nil {
		m.gh.opts = m.opts
		m.gh.refreshVisibleBadges(m.opts.Inventory)
	}
	if m.picker != nil {
		m.picker.opts = m.opts
		m.picker.refreshVisibleBadges(m.opts.Inventory)
	}
	if m.gp != nil {
		m.gp.opts = m.opts
	}
	return nil
}

func (m *appModel) refreshAccentStyles() {
	m.hosts.refreshAccentStyles()
	m.groups.refreshAccentStyles()
	if m.gh != nil {
		m.gh.refreshAccentStyles()
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
	h.Host = strings.TrimSpace(h.Host)
	if h.Host == "" {
		return fmt.Errorf("host required")
	}

	// Unique host check.
	for i := range m.opts.Inventory.Hosts {
		if i != index && strings.TrimSpace(m.opts.Inventory.Hosts[i].Host) == h.Host {
			return fmt.Errorf("host config already exists")
		}
	}

	inv := m.inventoryCopy()
	switch {
	case index < 0:
		inv.Hosts = append(inv.Hosts, h)
	case index < len(inv.Hosts):
		inv.Hosts[index] = h
	default:
		return fmt.Errorf("invalid host index")
	}
	return m.commitInventory(inv)
}

func (m *appModel) deleteGroup(index int) error {
	if index < 0 || index >= len(m.opts.Inventory.Groups) {
		return fmt.Errorf("invalid group index")
	}
	inv := m.inventoryCopy()
	inv.Groups = slices.Delete(inv.Groups, index, index+1)
	return m.commitInventory(inv)
}

func (m *appModel) addHostsToGroup(groupIndex int, hostsToAdd []string) error {
	if groupIndex < 0 || groupIndex >= len(m.opts.Inventory.Groups) {
		return fmt.Errorf("invalid group index")
	}
	if len(hostsToAdd) == 0 {
		return nil
	}

	inv := m.inventoryCopy()
	g := &inv.Groups[groupIndex]
	g.Hosts = slices.Clone(g.Hosts)
	for _, h := range hostsToAdd {
		if h != "" && !slices.Contains(g.Hosts, h) {
			g.Hosts = append(g.Hosts, h)
		}
	}
	return m.commitInventory(inv)
}

func (m *appModel) removeHostsFromGroup(groupIndex int, hostsToRemove []string) error {
	if groupIndex < 0 || groupIndex >= len(m.opts.Inventory.Groups) {
		return fmt.Errorf("invalid group index")
	}
	if len(hostsToRemove) == 0 {
		return nil
	}

	inv := m.inventoryCopy()
	g := &inv.Groups[groupIndex]
	g.Hosts = slices.DeleteFunc(slices.Clone(g.Hosts), func(h string) bool {
		return slices.Contains(hostsToRemove, h)
	})
	return m.commitInventory(inv)
}

func (m *appModel) saveToggleHidden(host string, hide bool) error {
	inv := m.inventoryCopy()
	h := strings.TrimSpace(host)
	idx, hc := findHostConfig(inv, host)
	isHost := func(hh string) bool { return strings.TrimSpace(hh) == h }

	switch {
	case hide && idx >= 0:
		// Host has an existing [[hosts]] entry — set Hidden there.
		hc.Hidden = true
		inv.Hosts[idx] = hc
	case hide:
		// No per-host config — use the compact list.
		if !slices.ContainsFunc(inv.HiddenHosts, isHost) {
			inv.HiddenHosts = append(inv.HiddenHosts, h)
		}
	default:
		inv.HiddenHosts = slices.DeleteFunc(inv.HiddenHosts, isHost)
		// Also clear Hidden flag in [[hosts]] entry if set (handles old-format configs).
		if idx >= 0 && hc.Hidden {
			hc.Hidden = false
			inv.Hosts[idx] = hc
		}
	}
	return m.commitInventory(inv)
}
