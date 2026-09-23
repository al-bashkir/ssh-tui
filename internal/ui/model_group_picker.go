package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sahilm/fuzzy"
)

// groupPickerDelegate renders groupRows as plain names.
var groupPickerDelegate = rowDelegate(func(width int, active bool, item list.Item) string {
	row := item.(groupRow)
	return renderSimpleRow(width, active, row.name, row.matchedIndexes)
})

type groupPickerModel struct {
	pickerScreen

	// hosts being added — shown as a summary line in the picker.
	hosts []string

	all   []groupRow
	names []string // cached for fuzzy search, parallel to all
}

func newGroupPickerModel(opts Options) *groupPickerModel {
	all := groupsRows(opts.Inventory)
	m := &groupPickerModel{
		pickerScreen: newPickerScreen(opts, groupPickerDelegate),
		all:          all,
		names:        groupNames(all),
	}
	m.applyFilter("")
	return m
}

func (m *groupPickerModel) Init() tea.Cmd { return nil }

func (m *groupPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg, 1)
		return m, nil
	case tea.KeyMsg:
		if handled, cmd := m.handleKey(msg, m.applyFilter, groupPickerCancelMsg{}); handled {
			return m, cmd
		}
		if key.Matches(msg, m.keymap.Connect) {
			row, ok := m.list.SelectedItem().(groupRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return groupPickerDoneMsg{groupIndex: row.index} }
		}
	}

	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
}

func (m *groupPickerModel) View() string {
	innerW, _ := frameInnerSize(m.width, m.height)
	adding := dim.Render(truncateTail(fmt.Sprintf("adding %d: %s", len(m.hosts), strings.Join(m.hosts, ", ")), innerW))
	return m.view("Select group", m.helpKeys, adding, fmt.Sprintf("groups: %d/%d", len(m.list.Items()), len(m.all)))
}

func (m *groupPickerModel) helpKeys() []helpSection {
	return m.helpSections(key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")))
}

func (m *groupPickerModel) applyFilter(query string) {
	var items []list.Item
	if query = strings.TrimSpace(query); query == "" {
		for _, r := range m.all {
			items = append(items, r)
		}
	} else {
		for _, mt := range fuzzy.Find(query, m.names) {
			r := m.all[mt.Index]
			r.matchedIndexes = mt.MatchedIndexes
			items = append(items, r)
		}
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}
