package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/al-bashkir/ssh-tui/internal/config"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sahilm/fuzzy"
)

type groupRow struct {
	index          int
	name           string
	hostCount      int
	matchedIndexes []int
}

func (i groupRow) FilterValue() string { return i.name }

var groupsDelegate = rowDelegate(func(width int, active bool, item list.Item) string {
	row := item.(groupRow)
	return renderGroupRow(width, active, row.name, row.hostCount, row.matchedIndexes)
})

type groupsModel struct {
	listScreen

	allRows []groupRow

	confirmDelete bool
	deleteIndex   int
}

func newGroupsModel(opts Options) *groupsModel {
	m := &groupsModel{listScreen: newListScreen(opts, groupsDelegate)}
	m.Refresh(opts.Inventory)
	return m
}

func groupsRows(inv config.Inventory) []groupRow {
	rows := make([]groupRow, 0, len(inv.Groups))
	for i, g := range inv.Groups {
		rows = append(rows, groupRow{index: i, name: g.Name, hostCount: len(g.Hosts)})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].name) < strings.ToLower(rows[j].name)
	})
	return rows
}

func (m *groupsModel) Refresh(inv config.Inventory) {
	m.opts.Inventory = inv
	m.allRows = groupsRows(inv)
	m.applyFilter(m.search.Value())
}

func (m *groupsModel) Init() tea.Cmd { return nil }

func (m *groupsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg, 18)
		return m, nil
	case tea.KeyMsg:
		run := func(cmd string) tea.Cmd { return m.connectAllCmd(false, cmd) }
		if handled, cmd := m.handleOverlayKey(msg, run); handled {
			return m, cmd
		}
		if m.confirmDelete {
			switch confirmAnswer(msg) {
			case answerYes:
				m.confirmDelete = false
				idx := m.deleteIndex
				return m, func() tea.Msg { return deleteGroupMsg{index: idx} }
			case answerNo:
				m.confirmDelete = false
				m.toast = toast{}
			}
			return m, nil
		}
		if m.handleVimNav(msg) {
			return m, nil
		}
		if cmd := m.tabCmd(msg, screenGroups); cmd != nil {
			return m, cmd
		}

		if key.Matches(msg, m.keymap.Quit) && m.focus != focusSearch {
			return m, m.quit()
		}
		if key.Matches(msg, m.keymap.Help) {
			m.openHelp("Groups", m.helpKeys())
			return m, nil
		}
		if handleFocusKeys(msg, m.keymap, &m.focus, &m.search) {
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) {
			handleEscChain(&m.focus, &m.search, &m.prevSearch, 0, nil, m.applyFilter)
			return m, nil
		}
		if key.Matches(msg, m.keymap.Connect) && m.focus == focusSearch {
			acceptSearch(&m.focus, &m.search, &m.prevSearch, len(m.list.Items()), m.applyFilter)
			return m, nil
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
func (m *groupsModel) handleListKey(msg tea.KeyMsg) (bool, tea.Cmd) {
	row, ok := m.list.SelectedItem().(groupRow)
	switch {
	case key.Matches(msg, m.keymap.NewGroup):
		return true, func() tea.Msg { return openGroupFormMsg{index: -1} }
	case key.Matches(msg, m.keymap.ConnectAll):
		m.toast = toast{}
		return true, m.connectAllCmd(false, "")
	case key.Matches(msg, m.keymap.OneWindow):
		m.toast = toast{}
		return true, m.connectAllCmd(true, "")
	case key.Matches(msg, m.keymap.Copy):
		if !ok || row.index >= len(m.opts.Inventory.Groups) {
			m.toast = toast{text: "no group selected", level: toastWarn}
			return true, nil
		}
		g := m.opts.Inventory.Groups[row.index]
		g.Name = suggestCopyGroupName(m.opts.Inventory, g.Name)
		return true, func() tea.Msg { return openGroupFormMsg{index: -1, group: g} }
	case key.Matches(msg, m.keymap.EditGroup, m.keymap.AddHosts, m.keymap.CustomHost,
		m.keymap.DeleteGroup, m.keymap.Connect, m.keymap.ConnectCmd):
		if !ok {
			return true, nil
		}
		return true, m.rowAction(msg, row)
	}
	return false, nil
}

// rowAction runs a key that acts on the selected group.
func (m *groupsModel) rowAction(msg tea.KeyMsg, row groupRow) tea.Cmd {
	switch {
	case key.Matches(msg, m.keymap.EditGroup):
		return func() tea.Msg { return openGroupFormMsg{index: row.index} }
	case key.Matches(msg, m.keymap.AddHosts):
		return func() tea.Msg { return openHostPickerMsg{groupIndex: row.index, returnTo: screenGroups} }
	case key.Matches(msg, m.keymap.CustomHost):
		return func() tea.Msg { return openCustomHostMsg{returnTo: screenGroups, groupIndex: row.index} }
	case key.Matches(msg, m.keymap.DeleteGroup):
		m.confirmDelete = true
		m.deleteIndex = row.index
		m.toast = toast{text: "delete? (y/n)", level: toastWarn}
	case key.Matches(msg, m.keymap.Connect):
		return func() tea.Msg { return openGroupHostsMsg{index: row.index} }
	case key.Matches(msg, m.keymap.ConnectCmd):
		m.openCmdPrompt("Groups > " + row.name)
	}
	return nil
}

func (m *groupsModel) View() string {
	if s := m.overlayView("Groups", m.helpKeys, "Connect and run a remote command for all hosts (keeps sessions open)."); s != "" {
		return s
	}
	status := statusDot(false) + "   " + dim.Render(fmt.Sprintf("%d groups", len(m.allRows)))
	if m.confirmDelete {
		name := ""
		hostCount := 0
		if m.deleteIndex >= 0 && m.deleteIndex < len(m.opts.Inventory.Groups) {
			name = m.opts.Inventory.Groups[m.deleteIndex].Name
			hostCount = len(m.opts.Inventory.Groups[m.deleteIndex].Hosts)
		}
		innerW := max(0, m.width-2)
		contentH := max(0, m.height-2-4)
		modal := deleteGroupConfirmBox(innerW, name, hostCount)
		placed := lipgloss.Place(innerW, contentH, lipgloss.Center, lipgloss.Center, modal)
		return renderMainTabBox(m.width, m.height, 1, m.search.View(), status, placed, "")
	}

	right := status
	if !m.toast.empty() {
		right = renderToast(m.toast)
	}
	var footer string
	if m.width < compactWidthThreshold {
		footer = styledFooter("↵ open  C connect  ? help")
	} else {
		footer = styledFooter("↵ open  C connect  ·  o panes  Ctrl+o cmd  ·  n new")
		if m.height >= twoLineFooterMinHeight {
			footer += "\n" + styledFooter("e edit  d delete  y copy  a add hosts  c custom  ·  Alt+1 hosts  Alt+3 settings  tab search  ? help")
		}
	}

	listContent := m.list.View()
	if len(m.list.Items()) == 0 {
		listContent = renderListEmptyState(m.width, m.height, m.search.Value(),
			dim.Render("No groups yet.")+"\n"+dim.Render("n — create a new group"))
	}
	return renderMainTabBox(m.width, m.height, 1, m.search.View(), right, listContent, footer)
}

func (m *groupsModel) helpKeys() []helpSection {
	openGroup := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open group"),
	)

	return []helpSection{
		{title: "Navigation", keys: append(m.listNavKeys(),
			m.keymap.FocusSearch,
			m.keymap.ToggleFocus,
			m.keymap.HostsTab,
			m.keymap.Settings,
			m.keymap.Esc,
		)},
		{title: "Connection", keys: []key.Binding{
			openGroup,
			m.keymap.ConnectAll,
			m.keymap.ConnectCmd,
			m.keymap.OneWindow,
			m.keymap.CustomHost,
		}},
		{title: "Editing", keys: []key.Binding{
			m.keymap.NewGroup,
			m.keymap.EditGroup,
			m.keymap.DeleteGroup,
			m.keymap.AddHosts,
			m.keymap.Copy,
		}},
		{title: "General", keys: []key.Binding{
			m.keymap.Settings,
			m.keymap.Help,
			m.keymap.Quit,
		}},
	}
}

func (m *groupsModel) applyFilter(query string) {
	var prevName string
	prevIndex := -1
	if row, ok := m.list.SelectedItem().(groupRow); ok {
		prevName = row.name
		prevIndex = row.index
	}

	query = strings.TrimSpace(query)
	var rows []groupRow
	if query == "" {
		rows = m.allRows
	} else {
		for _, mt := range fuzzy.Find(query, groupNames(m.allRows)) {
			r := m.allRows[mt.Index]
			r.matchedIndexes = mt.MatchedIndexes
			rows = append(rows, r)
		}
	}
	items := make([]list.Item, len(rows))
	for i, r := range rows {
		items[i] = r
	}
	m.list.SetItems(items)

	if len(rows) == 0 {
		return
	}
	// Prefer index-based restore: stable across renames.
	if prevIndex >= 0 {
		for i, r := range rows {
			if r.index == prevIndex {
				m.list.Select(i)
				return
			}
		}
	}
	restoreCursor(&m.list, groupNames(m.allRows), groupNames(rows), prevName)
}

func groupNames(rows []groupRow) []string {
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.name)
	}
	return names
}

func (m *groupsModel) connectAllCmd(oneWindow bool, remoteCmd string) tea.Cmd {
	row, ok := m.list.SelectedItem().(groupRow)
	if !ok {
		m.toast = toast{text: "no group selected", level: toastWarn}
		return nil
	}
	if row.index < 0 || row.index >= len(m.opts.Inventory.Groups) {
		m.toast = toast{text: "invalid group", level: toastErr}
		return nil
	}
	g := m.opts.Inventory.Groups[row.index]
	if len(g.Hosts) == 0 {
		m.toast = toast{text: "group has no hosts", level: toastWarn}
		return nil
	}
	return m.connect(append([]string(nil), g.Hosts...), m.opts, &g, oneWindow, remoteCmd)
}
