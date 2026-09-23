package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type hostPickerCancelMsg struct{}

type hostPickerDoneMsg struct {
	hosts []string
}

type hostPickerModel struct {
	hostSelectList
	pickerScreen
}

func newHostPickerModel(opts Options) *hostPickerModel {
	m := &hostPickerModel{
		hostSelectList: newHostSelectList(opts.Hosts),
		pickerScreen:   newPickerScreen(opts, hostDelegate),
	}
	m.bindList(&m.list)
	m.applyFilter("")
	return m
}

func (m *hostPickerModel) Init() tea.Cmd { return nil }

func (m *hostPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg, 0)
		return m, nil
	case tea.KeyMsg:
		if handled, cmd := m.handleKey(msg, m.applyFilter, hostPickerCancelMsg{}); handled {
			return m, cmd
		}
		if m.focus == focusList {
			switch {
			case key.Matches(msg, m.keymap.ToggleSel):
				m.toggleCurrentSelection()
				return m, nil
			case key.Matches(msg, m.keymap.SelectAll):
				m.selectAllFiltered()
				return m, nil
			case key.Matches(msg, m.keymap.ClearSel):
				m.clearSelection()
				return m, nil
			case key.Matches(msg, m.keymap.Connect):
				picked := m.hostsToOpen()
				return m, func() tea.Msg { return hostPickerDoneMsg{hosts: picked} }
			}
		}
	}

	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
}

func (m *hostPickerModel) View() string {
	counts := fmt.Sprintf("hosts: %d/%d  sel:%d", len(m.list.Items()), len(m.allHosts), len(m.selected))
	return m.view("Add hosts", m.helpKeys, "", counts)
}

func (m *hostPickerModel) helpKeys() []helpSection {
	add := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "add"))
	return m.helpSections(m.keymap.ToggleSel, m.keymap.SelectAll, m.keymap.ClearSel, add)
}

func (m *hostPickerModel) applyFilter(query string) {
	m.filter(query, m.opts.Inventory, nil, nil)
}
