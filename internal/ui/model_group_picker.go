package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sahilm/fuzzy"
)

type groupPickRow struct {
	index int
	name  string
}

func (i groupPickRow) Title() string       { return i.name }
func (i groupPickRow) Description() string { return "" }
func (i groupPickRow) FilterValue() string { return i.name }

type groupPickerDelegate struct{}

func (d groupPickerDelegate) Height() int                             { return 1 }
func (d groupPickerDelegate) Spacing() int                            { return 0 }
func (d groupPickerDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d groupPickerDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	row, ok := item.(groupPickRow)
	if !ok {
		fmt.Fprint(w, item.FilterValue())
		return
	}
	fmt.Fprint(w, renderSimpleRow(m.Width(), index == m.Index(), row.name))
}

type groupPickerModel struct {
	opts        Options
	parentCrumb string

	width  int
	height int

	all         []groupPickRow
	keymap      keyMap
	help        help.Model
	showHelp    bool
	toast       toast
	confirmQuit bool

	list       list.Model
	search     textinput.Model
	focus      focusState
	prevSearch string
}

func newGroupPickerModel(opts Options) *groupPickerModel {
	all := make([]groupPickRow, 0, len(opts.Inventory.Groups))
	items := make([]list.Item, 0, len(opts.Inventory.Groups))
	for i, g := range opts.Inventory.Groups {
		row := groupPickRow{index: i, name: g.Name}
		all = append(all, row)
		items = append(items, row)
	}

	l := list.New(items, groupPickerDelegate{}, 0, 0)
	l.Title = "Select group"
	configureList(&l)

	search := textinput.New()
	search.Placeholder = "search"
	search.Prompt = "/ "
	search.CharLimit = 256
	search.Width = 40
	configureSearch(&search)
	setSearchBarFocused(&search, false)

	m := &groupPickerModel{
		opts:     opts,
		all:      all,
		keymap:   defaultKeyMap(),
		help:     help.New(),
		showHelp: false,
		list:     l,
		search:   search,
		focus:    focusList,
	}
	return m
}

func (m *groupPickerModel) Init() tea.Cmd { return nil }

func (m *groupPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		if m.confirmQuit {
			s := msg.String()
			switch s {
			case "y", "Y", "enter":
				return m, tea.Quit
			case "n", "N", "esc":
				m.confirmQuit = false
				m.toast = toast{}
				return m, nil
			default:
				return m, nil
			}
		}
		if key.Matches(msg, m.keymap.Quit) {
			if !m.opts.Config.Defaults.ConfirmQuit {
				return m, tea.Quit
			}
			m.confirmQuit = true
			m.toast = toast{text: "quit? (y/n)", level: toastWarn}
			return m, nil
		}
		if key.Matches(msg, m.keymap.Help) {
			m.showHelp = !m.showHelp
			return m, nil
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
			return m, func() tea.Msg { return groupPickerCancelMsg{} }
		}
		if key.Matches(msg, m.keymap.Connect) {
			if m.focus == focusSearch {
				m.focus = focusList
				m.search.Blur()
				setSearchBarFocused(&m.search, false)
				return m, nil
			}
			row, ok := m.list.SelectedItem().(groupPickRow)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg { return groupPickerDoneMsg{groupIndex: row.index} }
		}
	}

	var cmd tea.Cmd
	if m.focus == focusSearch {
		m.search, cmd = m.search.Update(msg)
		cur := m.search.Value()
		if cur != m.prevSearch {
			m.applyFilter(cur)
			m.prevSearch = cur
		}
		return m, cmd
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *groupPickerModel) View() string {
	if m.showHelp {
		return renderHelpModal(m.width, m.height, "Select Group", m.help, m.helpKeys())
	}
	if m.confirmQuit {
		return renderQuitConfirm(m.width, m.height)
	}
	innerW, _ := frameInnerSize(m.width, m.height)
	sep := dim.Render(strings.Repeat("─", innerW))
	searchLine := m.search.View()
	listView := strings.TrimRight(m.list.View(), "\n")
	body := strings.TrimRight(searchLine+"\n"+sep+"\n"+listView+"\n"+sep, "\n")
	return renderFrame(m.width, m.height, breadcrumbTitle(m.parentCrumb, "Select group"), "", body, m.statusLine())
}

func (m *groupPickerModel) helpKeys() helpMap {
	esc := key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back/clear"),
	)
	selectGroup := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	)

	return helpMap{
		short: []key.Binding{
			m.list.KeyMap.CursorUp,
			m.list.KeyMap.CursorDown,
			m.keymap.FocusSearch,
			selectGroup,
			esc,
			m.keymap.Help,
			m.keymap.Quit,
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
			selectGroup,
			esc,
		}, {
			m.keymap.Help,
			m.keymap.Quit,
		}},
	}
}

func (m *groupPickerModel) statusLine() string {
	left := fmt.Sprintf("groups: %d", len(m.all))
	if m.list.Paginator.TotalPages > 1 {
		left += "  " + dim.Render(fmt.Sprintf("pg:%d/%d", m.list.Paginator.Page+1, m.list.Paginator.TotalPages))
	}
	if !m.toast.empty() {
		left += "  " + renderToast(m.toast)
	}
	q := strings.TrimSpace(m.search.Value())
	searchInfo := "search"
	if q != "" {
		if len(q) > 40 {
			q = q[:40] + "..."
		}
		searchInfo = "search: " + q
	}
	return left + "  " + statusOK.Render(searchInfo)
}

func (m *groupPickerModel) applyFilter(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		items := make([]list.Item, 0, len(m.all))
		for _, r := range m.all {
			items = append(items, r)
		}
		m.list.SetItems(items)
		if len(items) > 0 {
			m.list.Select(0)
		}
		return
	}

	names := make([]string, 0, len(m.all))
	for _, r := range m.all {
		names = append(names, r.name)
	}
	matches := fuzzy.Find(query, names)
	items := make([]list.Item, 0, len(matches))
	for _, mt := range matches {
		items = append(items, groupPickRow{index: mt.Index, name: mt.Str})
	}
	m.list.SetItems(items)
	if len(items) > 0 {
		m.list.Select(0)
	}
}
