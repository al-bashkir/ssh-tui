package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/al-bashkir/ssh-tui/internal/config"
	"github.com/al-bashkir/ssh-tui/internal/hosts"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type hostRow struct {
	host           string
	selected       bool
	hasCfg         bool
	hidden         bool
	matchedIndexes []int
}

func (i hostRow) FilterValue() string { return i.host }

var hostDelegate = rowDelegate(func(width int, active bool, item list.Item) string {
	row := item.(hostRow)
	return renderHostLikeRow(width, active, row.selected, row.host, row.hasCfg, row.hidden, row.matchedIndexes)
})

type knownHostsReloadMsg struct {
	res  hosts.LoadResult
	errs []hosts.PathError
}

type hostsModel struct {
	hostSelectList
	listScreen

	reloading  bool
	showHidden bool
}

func newHostsModel(opts Options) *hostsModel {
	m := &hostsModel{
		hostSelectList: newHostSelectList(opts.Hosts),
		listScreen:     newListScreen(opts, hostDelegate),
	}
	m.bindList(&m.list)
	if !opts.Config.Defaults.LoadKnownHosts {
		m.keymap.Reload.SetEnabled(false)
	}
	m.applyFilter("")
	return m
}

func (m *hostsModel) Init() tea.Cmd {
	return nil
}

func (m *hostsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg, 24)
		return m, nil

	case knownHostsReloadMsg:
		spinnerStop()
		m.reloading = false
		m.opts.Hosts = msg.res.Hosts
		m.opts.SkippedLines = msg.res.SkippedLines
		m.opts.LoadErrors = msg.errs
		m.allHosts = append([]string(nil), msg.res.Hosts...)
		present := make(map[string]bool, len(m.allHosts))
		for _, h := range m.allHosts {
			present[h] = true
		}
		for h := range m.selected {
			if !present[h] {
				delete(m.selected, h)
			}
		}
		m.reapplyFilter()
		m.toast = toast{text: fmt.Sprintf("%d hosts loaded", len(m.allHosts)), level: toastInfo}
		return m, nil

	case tea.KeyMsg:
		if handled, cmd := m.handleOverlayKey(msg, m.handleConnectWithRemoteCommand); handled {
			return m, cmd
		}
		if m.handleVimNav(msg) {
			return m, nil
		}
		if cmd := m.tabCmd(msg, screenHosts); cmd != nil {
			return m, cmd
		}

		if key.Matches(msg, m.keymap.Quit) && m.focus != focusSearch {
			return m, m.quit()
		}
		if key.Matches(msg, m.keymap.Help) {
			m.openHelp("Hosts", m.helpKeys())
			return m, nil
		}
		if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) {
			// Esc priority: blur search → clear selection → clear search.
			handleEscChain(&m.focus, &m.search, &m.prevSearch, len(m.selected), m.clearSelection, m.applyFilter)
			return m, nil
		}
		if key.Matches(msg, m.keymap.Connect) {
			// In search focus, Enter should accept the query and go back to list.
			if m.focus == focusSearch {
				acceptSearch(&m.focus, &m.search, &m.prevSearch, len(m.list.Items()), m.applyFilter)
				return m, nil
			}
			m.toast = toast{}
			return m, m.connect(m.hostsToOpen(), m.opts, nil, false, "")
		}
		if m.focus == focusList {
			if handled, cmd := m.handleListKey(msg); handled {
				return m, cmd
			}
		}

	case toastMsg:
		return m, m.showToast(toast(msg))
	}

	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
}

// handleListKey handles the keys that act on the list while it has focus.
func (m *hostsModel) handleListKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Reload):
		if !m.opts.Config.Defaults.LoadKnownHosts {
			m.toast = toast{text: "known_hosts disabled", level: toastWarn}
			return true, nil
		}
		m.toast = toast{text: "reloading", level: toastInfo}
		m.reloading = true
		spinnerStart()
		return true, tea.Batch(
			reloadKnownHostsCmd(m.opts.KnownHosts),
			tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg { return spinnerTickMsg{} }),
		)
	case key.Matches(msg, m.keymap.ToggleSel):
		m.toggleCurrentSelection()
	case key.Matches(msg, m.keymap.SelectAll):
		m.selectAllFiltered()
	case key.Matches(msg, m.keymap.ClearSel):
		m.clearSelection()
	case key.Matches(msg, m.keymap.ConnectCmd):
		if targets := m.hostsToOpen(); len(targets) > 0 {
			m.openCmdPrompt(cmdPromptCrumb("Hosts", targets))
		}
	case key.Matches(msg, m.keymap.OneWindow):
		m.toast = toast{}
		return true, m.connect(m.hostsToOpen(), m.opts, nil, true, "")
	case key.Matches(msg, m.keymap.ConnectSame):
		m.toast = toast{}
		return true, m.connectSameWindow(m.hostsToOpen(), m.opts, nil)
	case key.Matches(msg, m.keymap.AddHosts):
		hostsToAdd := m.hostsToOpen()
		if len(hostsToAdd) == 0 {
			m.toast = toast{text: "no host selected", level: toastWarn}
			return true, nil
		}
		return true, func() tea.Msg { return openGroupPickerMsg{hosts: hostsToAdd, returnTo: screenHosts} }
	case key.Matches(msg, m.keymap.CustomHost):
		return true, func() tea.Msg { return openCustomHostMsg{returnTo: screenHosts, groupIndex: -1} }
	case key.Matches(msg, m.keymap.HostConfig):
		return true, openHostConfigCmd(m.currentHost(), screenHosts, &m.toast)
	case key.Matches(msg, m.keymap.Copy):
		return true, copyHostConfigCmd(m.opts.Inventory, m.currentHost(), screenHosts, &m.toast)
	case key.Matches(msg, m.keymap.HideHost):
		host := m.currentHost()
		if host == "" {
			return true, nil
		}
		hide := !isHostHidden(m.opts.Inventory, host)
		return true, func() tea.Msg { return toggleHiddenHostMsg{host: host, hide: hide} }
	case key.Matches(msg, m.keymap.ShowHidden):
		m.showHidden = !m.showHidden
		m.reapplyFilter()
	default:
		return false, nil
	}
	return true, nil
}

func (m *hostsModel) applyFilter(query string) {
	var excludeFn func(config.Inventory, string) bool
	if !m.showHidden {
		excludeFn = isHostHidden
	}
	m.filter(query, m.opts.Inventory, excludeFn, isHostHidden)
}

func (m *hostsModel) reapplyFilter() {
	m.applyFilter(m.search.Value())
}

func (m *hostsModel) hiddenCount() int {
	count := 0
	for _, h := range m.allHosts {
		if isHostHidden(m.opts.Inventory, h) {
			count++
		}
	}
	return count
}

func (m *hostsModel) handleConnectWithRemoteCommand(remoteCmd string) tea.Cmd {
	if strings.TrimSpace(remoteCmd) == "" {
		m.toast = toast{text: "command required", level: toastWarn}
		return nil
	}
	return m.connect(m.hostsToOpen(), m.opts, nil, false, remoteCmd)
}

func (m *hostsModel) helpKeys() []helpSection {
	return []helpSection{
		{title: "Navigation", keys: append(m.listNavKeys(),
			m.keymap.FocusSearch,
			m.keymap.ToggleFocus,
			m.keymap.GroupsTab,
			m.keymap.Settings,
			m.keymap.Esc,
		)},
		{title: "Selection", keys: []key.Binding{
			m.keymap.ToggleSel,
			m.keymap.SelectAll,
			m.keymap.ClearSel,
		}},
		{title: "Connection", keys: []key.Binding{
			m.keymap.Connect,
			m.keymap.ConnectSame,
			m.keymap.ConnectCmd,
			m.keymap.OneWindow,
		}},
		{title: "Editing", keys: []key.Binding{
			m.keymap.CustomHost,
			m.keymap.HostConfig,
			m.keymap.AddHosts,
			m.keymap.Copy,
			m.keymap.HideHost,
			m.keymap.ShowHidden,
		}},
		{title: "General", keys: []key.Binding{
			m.keymap.Settings,
			m.keymap.Reload,
			m.keymap.Help,
			m.keymap.Quit,
		}},
	}
}

func reloadKnownHostsCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		res, errs := hosts.LoadKnownHosts(paths)
		return knownHostsReloadMsg{res: res, errs: errs}
	}
}
