package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type hostPickerCancelMsg struct{}

type hostPickerDoneMsg struct {
	hosts []string
}

type hostPickerModel struct {
	hostSelectList
	opts        Options
	parentCrumb string

	width  int
	height int

	list   list.Model
	search textinput.Model
	focus  focusState

	keymap   keyMap
	help     help.Model
	showHelp bool
	toast    toast

	prevSearch string
}

func newHostPickerModel(opts Options) *hostPickerModel {
	all := append([]string(nil), opts.Hosts...)
	items := make([]list.Item, 0, len(all))
	for _, h := range all {
		_, ok := hostConfigFor(opts.Inventory, h)
		items = append(items, hostRow{host: h, hasCfg: ok})
	}

	l := list.New(items, hostDelegate{}, 0, 0)
	l.Title = "Add hosts"
	configureList(&l)

	search := newSearchInput()

	m := &hostPickerModel{
		hostSelectList: hostSelectList{
			allHosts: all,
			filtered: append([]string(nil), all...),
			selected: make(map[string]bool),
		},
		opts:     opts,
		list:     l,
		search:   search,
		focus:    focusList,
		keymap:   defaultKeyMap(),
		help:     help.New(),
		showHelp: false,
	}
	return m
}

func (m *hostPickerModel) Init() tea.Cmd { return nil }

func (m *hostPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w := msg.Width
		h := msg.Height
		m.width = w
		m.height = h
		innerW, innerH := frameInnerSize(w, h)
		m.list.SetSize(innerW, max(1, innerH-5))
		m.search.Width = max(10, innerW-len(m.search.Prompt))
		return m, nil
	case tea.KeyMsg:
		if m.showHelp {
			if key.Matches(msg, m.keymap.Help) || msg.String() == "esc" {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if key.Matches(msg, m.keymap.Esc) {
			if m.focus == focusSearch {
				if m.search.Value() != "" {
					m.search.SetValue("")
					m.applyFilter("")
					m.prevSearch = ""
					return m, nil
				}
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
				return m, nil
			}
			if m.search.Value() != "" {
				m.search.SetValue("")
				m.applyFilter("")
				m.prevSearch = ""
				return m, nil
			}
			return m, func() tea.Msg { return hostPickerCancelMsg{} }
		}
		if key.Matches(msg, m.keymap.FocusSearch) {
			m.focus = focusSearch
			m.search.Focus()
			setSearchBarFocused(&m.search, true)
			return m, nil
		}
		if key.Matches(msg, m.keymap.ToggleFocus) {
			if m.focus == focusSearch {
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
			} else {
				m.focus = focusSearch
				m.search.Focus()
				setSearchBarFocused(&m.search, true)
			}
			return m, nil
		}
		if key.Matches(msg, m.keymap.ToggleSel) && m.focus == focusList {
			m.toggleCurrentSelection()
			return m, nil
		}
		if key.Matches(msg, m.keymap.SelectAll) && m.focus == focusList {
			for _, h := range m.filtered {
				m.selected[h] = true
			}
			m.refreshVisibleSelection()
			return m, nil
		}
		if key.Matches(msg, m.keymap.ClearSel) && m.focus == focusList {
			m.selected = make(map[string]bool)
			m.refreshVisibleSelection()
			return m, nil
		}
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
				return m, nil
			}
			picked := m.selectedHosts()
			if len(picked) == 0 {
				row, ok := m.list.SelectedItem().(hostRow)
				if ok && row.host != "" {
					picked = []string{row.host}
				}
			}
			return m, func() tea.Msg { return hostPickerDoneMsg{hosts: picked} }
		}
	}

	return m, updateSearchOrList(m.focus, &m.search, &m.list, &m.prevSearch, msg, m.applyFilter)
}

func (m *hostPickerModel) View() string {
	if m.showHelp {
		return renderHelpModal(m.width, m.height, "Add Hosts", m.help, m.helpKeys())
	}
	innerW, _ := frameInnerSize(m.width, m.height)
	sep := dim.Render(strings.Repeat("─", innerW))
	searchLine := m.search.View()
	listView := strings.TrimRight(m.list.View(), "\n")
	body := strings.TrimRight(searchLine+"\n"+sep+"\n"+listView+"\n"+sep, "\n")
	return renderFocusedFrame(m.width, m.height, breadcrumbTitle(m.parentCrumb, "Add hosts"), "", body, m.statusLine())
}

func (m *hostPickerModel) helpKeys() helpMap {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	add := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "add"),
	)

	return helpMap{
		short: []key.Binding{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.keymap.FocusSearch,
			m.keymap.ToggleSel,
			add,
			esc,
			m.keymap.Help,
		},
		full: [][]key.Binding{{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.list.KeyMap.PrevPage,
			m.list.KeyMap.NextPage,
		}, {
			m.keymap.FocusSearch,
			m.keymap.ToggleFocus,
			esc,
		}, {
			m.keymap.ToggleSel,
			m.keymap.SelectAll,
			m.keymap.ClearSel,
			add,
		}, {
			esc,
			m.keymap.Help,
		}},
		sections: []helpSection{
			{title: "Navigation", keys: []key.Binding{
				m.list.KeyMap.CursorUp,
				m.list.KeyMap.CursorDown,
				m.list.KeyMap.PrevPage,
				m.list.KeyMap.NextPage,
				m.keymap.FocusSearch,
				m.keymap.ToggleFocus,
			}},
			{title: "Selection", keys: []key.Binding{
				m.keymap.ToggleSel,
				m.keymap.SelectAll,
				m.keymap.ClearSel,
				add,
			}},
			{title: "General", keys: []key.Binding{
				esc,
				m.keymap.Help,
			}},
		},
	}
}

func (m *hostPickerModel) statusLine() string {
	shown := len(m.list.Items())
	total := len(m.allHosts)
	sel := len(m.selected)
	pg := ""
	if m.list.Paginator.TotalPages > 1 {
		pg = fmt.Sprintf("pg:%d/%d", m.list.Paginator.Page+1, m.list.Paginator.TotalPages)
	}

	q := strings.TrimSpace(m.search.Value())
	searchInfo := "search"
	if q != "" {
		if len(q) > 40 {
			q = q[:40] + "..."
		}
		searchInfo = "search: " + q
	}

	left := fmt.Sprintf("hosts: %d/%d  sel:%d", shown, total, sel)
	if pg != "" {
		left += "  " + dim.Render(pg)
	}
	if !m.toast.empty() {
		left += "  " + renderToast(m.toast)
	}
	return left + "  " + statusOK.Render(searchInfo)
}

func (m *hostPickerModel) applyFilter(query string) {
	m.hostSelectList.applyFilter(&m.list, query, m.opts.Inventory, nil, nil)
}

func (m *hostPickerModel) refreshVisibleSelection() {
	m.hostSelectList.refreshVisibleSelection(&m.list)
}

func (m *hostPickerModel) refreshVisibleBadges() {
	m.hostSelectList.refreshVisibleBadges(&m.list, m.opts.Inventory)
}

func (m *hostPickerModel) toggleCurrentSelection() {
	m.hostSelectList.toggleCurrentSelection(&m.list)
}
