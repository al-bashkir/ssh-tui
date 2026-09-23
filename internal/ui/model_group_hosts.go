package ui

import (
	"fmt"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type groupHostsModel struct {
	hostSelectList
	listScreen

	groupIndex int
	group      config.Group

	confirmRemove bool
	removeList    []string
}

func newGroupHostsModel(opts Options, groupIndex int) *groupHostsModel {
	g := config.Group{}
	if groupIndex >= 0 && groupIndex < len(opts.Inventory.Groups) {
		g = opts.Inventory.Groups[groupIndex]
	}

	m := &groupHostsModel{
		hostSelectList: newHostSelectList(g.Hosts),
		listScreen:     newListScreen(opts, hostDelegate),
		groupIndex:     groupIndex,
		group:          g,
	}
	m.bindList(&m.list)
	m.applyFilter("")
	return m
}

func (m *groupHostsModel) Init() tea.Cmd { return nil }

func (m *groupHostsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg, 24)
		return m, nil
	case tea.KeyMsg:
		if handled, cmd := m.handleOverlayKey(msg, m.handleConnectWithRemoteCommand); handled {
			return m, cmd
		}
		if m.confirmRemove {
			switch confirmAnswer(msg) {
			case answerYes:
				m.confirmRemove = false
				toRemove := m.removeList
				m.removeList = nil
				return m, func() tea.Msg { return removeHostsMsg{groupIndex: m.groupIndex, hosts: toRemove} }
			case answerNo:
				m.confirmRemove = false
				m.removeList = nil
				m.toast = toast{}
			}
			return m, nil
		}
		if m.handleVimNav(msg) {
			return m, nil
		}
		if cmd := m.tabCmd(msg, screenGroupHosts); cmd != nil {
			return m, cmd
		}

		if key.Matches(msg, m.keymap.Help) {
			m.openHelp("Group Hosts", m.helpKeys())
			return m, nil
		}
		if key.Matches(msg, m.keymap.DeleteGroup) && m.focus == focusList {
			toRemove := m.hostsToOpen()
			if len(toRemove) == 0 {
				m.toast = toast{text: "no host selected", level: toastWarn}
				return m, nil
			}
			m.confirmRemove = true
			m.removeList = toRemove
			m.toast = toast{text: fmt.Sprintf("remove %d? (y/n)", len(toRemove)), level: toastWarn}
			return m, nil
		}
		if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) || (key.Matches(msg, m.keymap.Quit) && m.focus != focusSearch) {
			// Esc/q priority: blur search → clear selection → clear search → back.
			if handleEscChain(&m.focus, &m.search, &m.prevSearch, len(m.selected), m.clearSelection, m.applyFilter) {
				return m, nil
			}
			return m, func() tea.Msg { return switchScreenMsg{to: screenGroups} }
		}
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				acceptSearch(&m.focus, &m.search, &m.prevSearch, len(m.list.Items()), m.applyFilter)
				return m, nil
			}
			m.toast = toast{}
			return m, m.connect(m.hostsToOpen(), m.opts, &m.group, false, "")
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
func (m *groupHostsModel) handleListKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.ToggleSel):
		m.toggleCurrentSelection()
	case key.Matches(msg, m.keymap.SelectAll):
		m.selectAllFiltered()
	case key.Matches(msg, m.keymap.ClearSel):
		m.clearSelection()
	case key.Matches(msg, m.keymap.ConnectCmd):
		if targets := m.hostsToOpen(); len(targets) > 0 {
			m.openCmdPrompt(cmdPromptCrumb("Groups > "+m.group.Name, targets))
		}
	case key.Matches(msg, m.keymap.OneWindow):
		m.toast = toast{}
		return true, m.connect(m.hostsToOpen(), m.opts, &m.group, true, "")
	case key.Matches(msg, m.keymap.ConnectSame):
		m.toast = toast{}
		return true, m.connectSameWindow(m.hostsToOpen(), m.opts, &m.group)
	case key.Matches(msg, m.keymap.AddHosts):
		return true, func() tea.Msg { return openHostPickerMsg{groupIndex: m.groupIndex, returnTo: screenGroupHosts} }
	case key.Matches(msg, m.keymap.CustomHost):
		return true, func() tea.Msg { return openCustomHostMsg{returnTo: screenGroupHosts, groupIndex: m.groupIndex} }
	case key.Matches(msg, m.keymap.HostConfig):
		return true, openHostConfigCmd(m.currentHost(), screenGroupHosts, &m.toast)
	case key.Matches(msg, m.keymap.Copy):
		return true, copyHostConfigCmd(m.opts.Inventory, m.currentHost(), screenGroupHosts, &m.toast)
	default:
		return false, nil
	}
	return true, nil
}

func (m *groupHostsModel) View() string {
	if s := m.overlayView("Group Hosts", m.helpKeys, "Connect and run a remote command (keeps session open)."); s != "" {
		return s
	}
	breadcrumb := dim.Render("Groups >") + " " + accentStyle.Render(m.group.Name)
	if m.confirmRemove {
		innerW := max(0, m.width-2)
		contentH := max(0, m.height-2-6)
		modal := removeHostsConfirmBox(innerW, m.removeList, m.group.Name)
		placed := lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, modal)
		return renderTabBox(m.width, m.height, breadcrumb, m.search.View(), statusDot(false), placed, "")
	}

	right := ""
	if !m.toast.empty() {
		right = renderToast(m.toast)
	} else {
		right = statusDot(false)
		total := len(m.allHosts)
		if strings.TrimSpace(m.search.Value()) != "" {
			right += dim.Render(fmt.Sprintf(" %d / %d hosts", len(m.list.Items()), total))
		} else {
			right += dim.Render(fmt.Sprintf(" %d hosts", total))
		}
		if selCount := len(m.selected); selCount > 0 {
			right += "   " + badgeSelStyle.Render(fmt.Sprintf("%d selected", selCount))
		}
	}
	var footer string
	if m.width < compactWidthThreshold {
		footer = styledFooter("↵ connect  ␣ select  esc back  ? help")
	} else {
		footer = styledFooter("↵ connect  O pane  ·  ␣ select  o panes  ·  Ctrl+o cmd  a add")
		if m.height >= twoLineFooterMinHeight {
			footer += "\n" + styledFooter("e config  c custom  d remove  y copy  ·  Alt+1 hosts  Alt+3 settings  esc back  ? help")
		}
	}

	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		listContent = renderListEmptyState(m.width, m.height, m.search.Value(),
			dim.Render("No hosts in this group.")+"\n"+dim.Render("a — add hosts"))
	}
	return renderTabBox(m.width, m.height, breadcrumb, m.search.View(), right, listContent, footer)
}

func (m *groupHostsModel) helpKeys() []helpSection {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	remove := key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "remove"),
	)

	return []helpSection{
		{title: "Navigation", keys: append(m.listNavKeys(),
			m.keymap.HostsTab,
			m.keymap.GroupsTab,
			m.keymap.Settings,
			m.keymap.FocusSearch,
			m.keymap.ToggleFocus,
			esc,
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
			m.keymap.AddHosts,
			m.keymap.CustomHost,
			m.keymap.HostConfig,
			m.keymap.Copy,
			remove,
		}},
		{title: "General", keys: []key.Binding{
			m.keymap.Help,
		}},
	}
}

func (m *groupHostsModel) applyFilter(query string) {
	m.filter(query, m.opts.Inventory, nil, nil)
}

func (m *groupHostsModel) handleConnectWithRemoteCommand(remoteCmd string) tea.Cmd {
	if strings.TrimSpace(remoteCmd) == "" {
		m.toast = toast{text: "command required", level: toastWarn}
		return nil
	}
	return m.connect(m.hostsToOpen(), m.opts, &m.group, false, remoteCmd)
}
